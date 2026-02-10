import express from "express";
import cors from "cors";

const app = express();
const PORT = Number(process.env.API_PORT ?? 3000);

app.use(cors());
app.use(express.json());

app.get("/health", (_req, res) => {
  res.json({ status: "ok", service: "api-server" });
});

// TODO: Auth routes (register, login, refresh)
// TODO: Alert CRUD routes
// TODO: Price routes
// TODO: Analytics routes
// TODO: WebSocket server for real-time updates
// TODO: Rate limiting middleware
// TODO: JWT auth middleware

app.listen(PORT, () => {
  console.log(`[API Server] Running on http://localhost:${PORT}`);
});
