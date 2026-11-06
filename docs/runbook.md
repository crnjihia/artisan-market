# path: docs/runbook.md
# Artisan Market — Production Operations & Incident Runbook

- **Platform**: Artisan Market Microservices E-Commerce
- **On-Call Engineer**: Primary SRE / Lead Architect (Christopher Njihia, Nairobi)
- **Timezone**: `Africa/Nairobi` (EAT / UTC+3)
- **Deployment**: Kubernetes + Istio Service Mesh on Cloud / Hybrid Infrastructure

---

## 1. System Component Overview & Health Endpoints

| Service | Port | Protocol | Health Check | Primary Log Identifier |
| :--- | :--- | :--- | :--- | :--- |
| **API Gateway** | `4000` | HTTP / REST | `GET /api/health` | `[Gateway]` |
| **User Service** | `50051` | gRPC | TCP / gRPC probe | `[User Service]` |
| **Product Service** | `50052` | gRPC | TCP / gRPC probe | `[Product Service]` |
| **Order Service** | `50053` | gRPC | TCP / gRPC probe | `[Order Service]` |
| **Payment Service** | `50054` / `8084`| gRPC / HTTP | `GET /healthz` (8084) | `[Payment Service]` |
| **Notification** | `3001` | HTTP / Events | `GET /healthz` (3001) | `[Notification Service]` |
| **Redpanda/Kafka** | `9092` / `8081`| Kafka / Schema | `rpk cluster health` | `redpanda` |
| **PostgreSQL** | `5432` | TCP / SQL | `pg_isready -U postgres` | `postgres` |

---

## 2. Emergency Escalation & Fast-Check Cheatsheet

```bash
# 1. Quick Cluster Status Check
kubectl get pods -n artisan-market -o wide

# 2. Check Istio Ingress Gateway Status
kubectl get svc -n istio-system istio-ingressgateway

# 3. Check Microservice Error Rates in Realtime
kubectl logs -n artisan-market -l app=order --tail=100 -f
kubectl logs -n artisan-market -l app=payment --tail=100 -f

# 4. Inspect Kafka Consumer Lag across Saga Groups
rpk group describe order-saga-service-group
rpk group describe payment-saga-service-group
rpk group describe notification-consumer-group
```

---

## 3. Disaster Recovery & Emergency Procedures

### Scenario A: Safaricom Daraja STK Push Downtime / Callback Failure
**Symptoms**: Orders remain stuck in `reserved` status; customers report no M-Pesa prompt on their phones; error spike in Payment Service logs with `DARAJA_ERROR` or `HTTP 500`.

**Triage & Resolution Steps**:
1. **Check Safaricom Developer Portal Status**: Verify whether Safaricom Daraja Sandbox/Production is undergoing scheduled maintenance or throttling.
2. **Inspect Webhook Callback Ingress**:
   ```bash
   kubectl logs -n istio-system -l app=istio-ingressgateway | grep mpesa
   ```
   Ensure the Istio VirtualService routes `/mpesa/callback` to `payment:8084` without TLS interception (port 8084 is in `PERMISSIVE` mTLS).
3. **Emergency Fallback (Switch to Card/Stripe or Mock Simulator)**:
   If Daraja is completely down, update the storefront notice or toggle mock fallback for testing:
   ```bash
   kubectl set env deployment/payment -n artisan-market USE_MOCKS=true
   ```
4. **Reprocessing Pending STK Prompts**:
   Pending payments older than 60 seconds will be marked `timed_out` by the payment scanner, triggering `payment.timeout` events to release inventory automatically.

---

### Scenario B: Kafka / Redpanda Cluster Outage
**Symptoms**: Outbox worker logs error: `Failed to write messages to Kafka: broker connection refused`.

**System Resilience Behavior**:
- **Zero Data Loss**: Microservice operations continue; orders and payments are committed to PostgreSQL, and events are queued in the `outbox` table with `status = 'pending'`.
- The database connection pool is safeguarded using `SKIP LOCKED` queries.

**Recovery Steps**:
1. Check Redpanda pod status:
   ```bash
   kubectl describe pod -n kafka artisan-kafka-0
   ```
2. Restart broker pods if storage disk is full or restart was hung:
   ```bash
   kubectl rollout restart statefulset artisan-kafka -n kafka
   ```
3. Once the broker accepts connections:
   - The Go Outbox Relay goroutines (`StartOutboxRelay`) in `order` and `payment` will automatically resume draining the pending backlog in FIFO order (`ORDER BY created_at ASC`).
   - Monitor the drain rate:
     ```sql
     SELECT status, count(*) FROM outbox GROUP BY status;
     ```

---

### Scenario C: PostgreSQL Connection Pool Exhaustion (`too many clients`)
**Symptoms**: Database queries fail with `pq: sorry, too many clients already`.

**Triage & Resolution**:
1. Identify leaking connections:
   ```sql
   SELECT client_addr, usename, state, count(*) 
   FROM pg_stat_activity 
   GROUP BY client_addr, usename, state;
   ```
2. Each Go microservice is configured with a strict connection pool limit:
   - `db.SetMaxOpenConns(25)`
   - `db.SetMaxIdleConns(5)`
   - `db.SetConnMaxLifetime(5 * time.Minute)`
3. If total pods exceed Postgres `max_connections` (100):
   ```sql
   ALTER SYSTEM SET max_connections = 200;
   -- Requires reload
   SELECT pg_reload_conf();
   ```
4. Terminate hanging idle in transaction queries:
   ```sql
   SELECT pg_terminate_backend(pid) 
   FROM pg_stat_activity 
   WHERE state = 'idle in transaction' 
   AND state_change < now() - INTERVAL '5 minutes';
   ```

---

## 4. Istio Canary Deployment & Traffic Shifting

Artisan Market supports zero-downtime canary rollouts for microservices.

### Step 1: Deploy Canary Pods (`v2`)
Deploy the canary deployment alongside stable `v1`:
```bash
kubectl apply -f deploy/k8s/istio/canary-example.yaml
```

### Step 2: Test Canary with Internal Metadata Header
Send an HTTP request with the header `x-canary: artisan-beta`:
```bash
curl -H "x-canary: artisan-beta" https://market.artisan.co.ke/api/products
```
The Istio VirtualService routes 100% of matching header traffic to `subset: v2`.

### Step 3: Progressive Rollout (90/10 -> 50/50 -> 100% v2)
Edit `deploy/k8s/istio/canary-example.yaml` to shift weights:
```yaml
- destination:
    host: product.artisan-market.svc.cluster.local
    subset: v1
  weight: 50
- destination:
    host: product.artisan-market.svc.cluster.local
    subset: v2
  weight: 50
```

### Step 4: Rollback Procedure (Emergency Abort)
If error rates spike on `v2`:
```bash
# Immediately revert traffic 100% to v1
kubectl patch virtualservice product-service-canary -n artisan-market --type merge -p '{"spec":{"http":[{"route":[{"destination":{"host":"product.artisan-market.svc.cluster.local","subset":"v1"},"weight":100}]}]}}'
```

---

## 5. Circuit Breaker & Outlier Detection Tuning

Istio DestinationRules configure active outlier detection for each microservice:
```yaml
outlierDetection:
  consecutive5xxErrors: 3
  interval: 10s
  baseEjectionTime: 30s
  maxEjectionPercent: 50
```
- If a pod fails 3 consecutive requests with 5xx errors within 10s, Envoy ejects it from the load balancing pool for 30s.
- Up to 50% of instances in a service cluster can be ejected simultaneously, preserving service continuity.

---

## 6. Prometheus RED Metrics & Alerting Thresholds

| Metric | Prometheus Query | Warning Threshold | Critical Threshold |
| :--- | :--- | :--- | :--- |
| **HTTP/gRPC Error Rate** | `sum(rate(artisan_errors_total[5m])) / sum(rate(artisan_requests_total[5m])) * 100` | `> 1.0%` | `> 5.0%` |
| **p95 Latency** | `histogram_quantile(0.95, sum(rate(artisan_request_duration_seconds_bucket[5m])) by (le, service))` | `> 500ms` | `> 1500ms` |
| **Outbox Pending Backlog** | `artisan_outbox_pending_records` | `> 50` | `> 250` |
| **Kafka DLQ Growth** | `sum(increase(artisan_notification_dlq_total[15m]))` | `> 5` | `> 25` |
| **Stock Lock Wait Time** | Postgres row lock wait | `> 200ms` | `> 1000ms` |

---

## 7. mTLS Certificate Renewal Procedure

Development certificates reside in `certs/` and are generated via `scripts/generate-certs.go`:
```powershell
# Renew all microservice and CA certificates (valid for 365 days)
go run scripts/generate-certs.go
```
In Kubernetes production, certs are managed automatically via cert-manager and Istio's internal Citadel/CA with automatic 90-day rotation.

