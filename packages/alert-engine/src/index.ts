import { KAFKA_TOPICS } from "@price-alert/shared";

console.log("[Alert Engine] Service starting...");
console.log(`[Alert Engine] Will consume from: ${KAFKA_TOPICS.PRICE_UPDATES}`);
console.log(`[Alert Engine] Will produce to: ${KAFKA_TOPICS.ALERT_TRIGGERS}`);

// TODO: Consume price updates from Kafka
// TODO: Load active alerts from Redis cache (fallback to PostgreSQL)
// TODO: Match prices against alert conditions
// TODO: Publish triggered alerts to Kafka

process.on("SIGTERM", () => {
  console.log("[Alert Engine] Shutting down...");
  process.exit(0);
});
