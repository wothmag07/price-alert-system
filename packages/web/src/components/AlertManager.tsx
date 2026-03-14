import { useState, useEffect, useCallback, type FormEvent } from "react";
import { alerts as alertsApi, type Alert } from "../lib/api";

const SYMBOLS = ["BTCUSDT", "ETHUSDT", "SOLUSDT", "DOGEUSDT", "AVAXUSDT", "ADAUSDT"];
const CONDITIONS = [
  { value: "PRICE_ABOVE", label: "Price Above" },
  { value: "PRICE_BELOW", label: "Price Below" },
  { value: "PCT_CHANGE_ABOVE", label: "% Change Above" },
  { value: "PCT_CHANGE_BELOW", label: "% Change Below" },
];

export default function AlertManager() {
  const [alertList, setAlertList] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [symbol, setSymbol] = useState(SYMBOLS[0] ?? "BTCUSDT");
  const [condition, setCondition] = useState(CONDITIONS[0]?.value ?? "PRICE_ABOVE");
  const [threshold, setThreshold] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const fetchAlerts = useCallback(async () => {
    try {
      const res = await alertsApi.list();
      setAlertList(res.alerts);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAlerts();
  }, [fetchAlerts]);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await alertsApi.create({
        symbol,
        condition,
        threshold: parseFloat(threshold),
      });
      setThreshold("");
      setShowForm(false);
      fetchAlerts();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create alert");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await alertsApi.delete(id);
      setAlertList((prev) => prev.filter((a) => a.id !== id));
    } catch {
      // ignore
    }
  }

  function conditionLabel(cond: string) {
    return CONDITIONS.find((c) => c.value === cond)?.label ?? cond;
  }

  function statusColor(status: string) {
    if (status === "ACTIVE") return "text-green-400";
    if (status === "TRIGGERED") return "text-yellow-400";
    return "text-gray-500";
  }

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-300">My Alerts</h2>
        <button
          onClick={() => setShowForm(!showForm)}
          className="rounded bg-blue-600 px-3 py-1 text-sm text-white hover:bg-blue-700"
        >
          {showForm ? "Cancel" : "+ New"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="mb-4 space-y-2 rounded bg-gray-800 p-3">
          {error && (
            <p className="text-sm text-red-400">{error}</p>
          )}
          <select
            value={symbol}
            onChange={(e) => setSymbol(e.target.value)}
            className="w-full rounded border border-gray-600 bg-gray-700 px-2 py-1.5 text-sm text-white"
          >
            {SYMBOLS.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
          <select
            value={condition}
            onChange={(e) => setCondition(e.target.value)}
            className="w-full rounded border border-gray-600 bg-gray-700 px-2 py-1.5 text-sm text-white"
          >
            {CONDITIONS.map((c) => (
              <option key={c.value} value={c.value}>{c.label}</option>
            ))}
          </select>
          <input
            type="number"
            step="any"
            placeholder="Threshold value"
            value={threshold}
            onChange={(e) => setThreshold(e.target.value)}
            required
            className="w-full rounded border border-gray-600 bg-gray-700 px-2 py-1.5 text-sm text-white placeholder-gray-500"
          />
          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded bg-green-600 py-1.5 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
          >
            {submitting ? "Creating..." : "Create Alert"}
          </button>
        </form>
      )}

      {loading ? (
        <p className="text-gray-500">Loading alerts...</p>
      ) : alertList.length === 0 ? (
        <p className="text-gray-500">No alerts configured yet.</p>
      ) : (
        <div className="space-y-2">
          {alertList.map((alert) => (
            <div
              key={alert.id}
              className="flex items-center justify-between rounded bg-gray-800/50 px-3 py-2"
            >
              <div>
                <span className="font-mono text-sm text-white">
                  {alert.symbol}
                </span>
                <span className="ml-2 text-xs text-gray-400">
                  {conditionLabel(alert.condition)}{" "}
                  {alert.threshold}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className={`text-xs font-medium ${statusColor(alert.status)}`}>
                  {alert.status}
                </span>
                {alert.status === "ACTIVE" && (
                  <button
                    onClick={() => handleDelete(alert.id)}
                    className="rounded px-2 py-0.5 text-xs text-red-400 hover:bg-red-900/30"
                  >
                    Delete
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
