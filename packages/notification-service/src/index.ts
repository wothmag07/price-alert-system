import { KAFKA_TOPICS } from "@price-alert/shared";

console.log("[Notification Service] Service starting...");
console.log(`[Notification Service] Will consume from: ${KAFKA_TOPICS.ALERT_TRIGGERS}`);

// TODO: Consume alert triggers from Kafka
// TODO: Check per-user rate limits (Redis sliding window)
// TODO: Send email notifications via Resend
// TODO: Push WebSocket notifications via Redis pub/sub
// TODO: Track delivery status in alert_history

process.on("SIGTERM", () => {
  console.log("[Notification Service] Shutting down...");
  process.exit(0);
});
