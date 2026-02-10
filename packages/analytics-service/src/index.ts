import { KAFKA_TOPICS } from "@price-alert/shared";

console.log("[Analytics Service] Service starting...");
console.log(`[Analytics Service] Will consume from: ${KAFKA_TOPICS.PRICE_UPDATES}`);

// TODO: Consume price updates from Kafka
// TODO: Calculate rolling window price changes
// TODO: Update Redis sorted sets for Top-K trending drops
// TODO: Expose analytics data via Redis for API Server queries

process.on("SIGTERM", () => {
  console.log("[Analytics Service] Shutting down...");
  process.exit(0);
});
