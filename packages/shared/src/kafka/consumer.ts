import { Kafka, Consumer, EachMessagePayload } from "kafkajs";

export interface ConsumerConfig {
  groupId: string;
  topics: string[];
  onMessage: (payload: EachMessagePayload) => Promise<void>;
}

export async function createConsumer(config: ConsumerConfig): Promise<Consumer> {
  const kafka = new Kafka({
    clientId: "price-alert-system",
    brokers: (process.env.KAFKA_BROKERS ?? "localhost:9092").split(","),
  });

  const consumer = kafka.consumer({ groupId: config.groupId });
  await consumer.connect();

  for (const topic of config.topics) {
    await consumer.subscribe({ topic, fromBeginning: false });
  }

  await consumer.run({
    eachMessage: config.onMessage,
  });

  return consumer;
}
