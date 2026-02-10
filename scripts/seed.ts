/**
 * Database seed script
 * Run: npx tsx scripts/seed.ts
 *
 * Creates tables and optionally seeds with sample data.
 */

import pg from "pg";

const { Pool } = pg;

const pool = new Pool({
  host: process.env.POSTGRES_HOST ?? "localhost",
  port: Number(process.env.POSTGRES_PORT ?? 5432),
  database: process.env.POSTGRES_DB ?? "price_alerts",
  user: process.env.POSTGRES_USER ?? "postgres",
  password: process.env.POSTGRES_PASSWORD ?? "postgres",
});

async function seed() {
  console.log("Creating tables...");

  await pool.query(`
    CREATE TABLE IF NOT EXISTS users (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      email       VARCHAR(255) UNIQUE NOT NULL,
      password    VARCHAR(255) NOT NULL,
      created_at  TIMESTAMP DEFAULT NOW(),
      updated_at  TIMESTAMP DEFAULT NOW()
    );

    CREATE TABLE IF NOT EXISTS alerts (
      id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      symbol        VARCHAR(20) NOT NULL,
      condition     VARCHAR(30) NOT NULL,
      threshold     DECIMAL(20, 8) NOT NULL,
      status        VARCHAR(20) DEFAULT 'ACTIVE',
      created_at    TIMESTAMP DEFAULT NOW(),
      triggered_at  TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_alerts_symbol_status ON alerts(symbol, status);
    CREATE INDEX IF NOT EXISTS idx_alerts_user_id ON alerts(user_id);

    CREATE TABLE IF NOT EXISTS alert_history (
      id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      alert_id        UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
      triggered_price DECIMAL(20, 8) NOT NULL,
      notification    JSONB,
      created_at      TIMESTAMP DEFAULT NOW()
    );

    CREATE INDEX IF NOT EXISTS idx_alert_history_alert_id ON alert_history(alert_id);

    CREATE TABLE IF NOT EXISTS price_history (
      id        BIGSERIAL PRIMARY KEY,
      symbol    VARCHAR(20) NOT NULL,
      price     DECIMAL(20, 8) NOT NULL,
      volume    DECIMAL(20, 8),
      timestamp TIMESTAMP NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_price_history_symbol_ts ON price_history(symbol, timestamp DESC);
  `);

  console.log("Tables created successfully.");
  await pool.end();
}

seed().catch((err) => {
  console.error("Seed failed:", err);
  process.exit(1);
});
