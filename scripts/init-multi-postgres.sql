-- Initialize isolated databases for each microservice
CREATE DATABASE user_db;
CREATE DATABASE product_db;
CREATE DATABASE order_db;
CREATE DATABASE payment_db;

-- Enable pgcrypto on all databases
\c user_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c product_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c order_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c payment_db;
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
