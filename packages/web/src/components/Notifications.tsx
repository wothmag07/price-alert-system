import type { AlertTriggered } from "../hooks/useWebSocket";

interface Props {
  notifications: AlertTriggered[];
  onDismiss: (index: number) => void;
}

export default function Notifications({ notifications, onDismiss }: Props) {
  if (notifications.length === 0) return null;

  return (
    <div className="fixed right-4 top-4 z-50 flex flex-col gap-2">
      {notifications.slice(0, 5).map((n, i) => (
        <div
          key={`${n.alertId}-${n.timestamp}`}
          className="flex items-start gap-3 rounded-lg border border-yellow-700 bg-yellow-900/90 px-4 py-3 shadow-lg backdrop-blur"
        >
          <div className="flex-1">
            <p className="text-sm font-medium text-yellow-200">
              Alert Triggered
            </p>
            <p className="text-xs text-yellow-300/80">
              {n.symbol} — {n.condition} {n.threshold} — Price: {n.triggeredPrice}
            </p>
          </div>
          <button
            onClick={() => onDismiss(i)}
            className="text-yellow-400 hover:text-yellow-200"
          >
            &times;
          </button>
        </div>
      ))}
    </div>
  );
}
