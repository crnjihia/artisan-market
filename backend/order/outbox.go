package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// OutboxEntry represents a domain event queued in PostgreSQL for reliable Kafka dispatch.
type OutboxEntry struct {
	ID          string
	AggregateID string
	EventType   string
	Topic       string
	Payload     []byte
	Headers     map[string]string
}

// StartOutboxRelay runs a background loop polling the outbox table and producing to Kafka.
func StartOutboxRelay(ctx context.Context, db *sql.DB, client *kgo.Client, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Printf("[Order Outbox] Relay worker started (polling every %v)", pollInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Order Outbox] Relay worker stopping...")
			return
		case <-ticker.C:
			if err := processOutboxBatch(ctx, db, client); err != nil {
				log.Printf("[Order Outbox] Error processing batch: %v", err)
			}
		}
	}
}

func processOutboxBatch(ctx context.Context, db *sql.DB, client *kgo.Client) error {
	if client == nil {
		return nil
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		SELECT id, aggregate_id, event_type, topic, payload, headers
		FROM outbox
		WHERE published = false
		ORDER BY created_at ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var entries []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var rawHeaders sql.NullString
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.EventType, &e.Topic, &e.Payload, &rawHeaders); err != nil {
			return err
		}
		if rawHeaders.Valid && rawHeaders.String != "" {
			_ = json.Unmarshal([]byte(rawHeaders.String), &e.Headers)
		}
		entries = append(entries, e)
	}

	if len(entries) == 0 {
		return nil
	}

	for _, entry := range entries {
		var kgoHeaders []kgo.RecordHeader
		for k, v := range entry.Headers {
			kgoHeaders = append(kgoHeaders, kgo.RecordHeader{Key: k, Value: []byte(v)})
		}
		kgoHeaders = append(kgoHeaders, kgo.RecordHeader{Key: "event_type", Value: []byte(entry.EventType)})

		record := &kgo.Record{
			Topic:   entry.Topic,
			Key:     []byte(entry.AggregateID),
			Value:   entry.Payload,
			Headers: kgoHeaders,
		}

		if err := client.ProduceSync(ctx, record).FirstErr(); err != nil {
			log.Printf("[Order Outbox] Failed to produce event %s (%s): %v", entry.ID, entry.EventType, err)
			_, _ = tx.ExecContext(ctx, "UPDATE outbox SET retry_count = retry_count + 1 WHERE id = $1", entry.ID)
			continue
		}

		updateQuery := `UPDATE outbox SET published = true, published_at = now() WHERE id = $1`
		if _, err := tx.ExecContext(ctx, updateQuery, entry.ID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// QueueOutboxEvent persists a saga event atomically within an existing database transaction.
func QueueOutboxEvent(ctx context.Context, tx *sql.Tx, aggregateID, eventType, topic string, payload any, correlationID string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"correlation_id": correlationID,
		"event_type":     eventType,
		"schema_version": "1.0",
		"service":        "order",
	}
	headerBytes, _ := json.Marshal(headers)

	query := `
		INSERT INTO outbox (aggregate_id, event_type, topic, payload, headers)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.ExecContext(ctx, query, aggregateID, eventType, topic, payloadBytes, headerBytes)
	return err
}

