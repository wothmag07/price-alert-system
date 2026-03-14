import { useAuth } from "../hooks/useAuth";
import { useWebSocket } from "../hooks/useWebSocket";
import PriceTicker from "../components/PriceTicker";
import AlertManager from "../components/AlertManager";
import TrendingDrops from "../components/TrendingDrops";
import Notifications from "../components/Notifications";

export default function Dashboard() {
  const { user, logout } = useAuth();
  const { prices, notifications, connected, dismissNotification } =
    useWebSocket();

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <Notifications
        notifications={notifications}
        onDismiss={dismissNotification}
      />

      <header className="border-b border-gray-800 px-6 py-4">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <h1 className="text-2xl font-bold">Price Alert System</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-400">{user?.email}</span>
            <button
              onClick={logout}
              className="rounded border border-gray-700 px-3 py-1 text-sm text-gray-300 hover:bg-gray-800"
            >
              Logout
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl p-6">
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <PriceTicker prices={prices} connected={connected} />
          <AlertManager />
          <TrendingDrops />
        </div>
      </main>
    </div>
  );
}
