function App() {
  return (
    <div className="min-h-screen bg-gray-950 text-white">
      <header className="border-b border-gray-800 px-6 py-4">
        <h1 className="text-2xl font-bold">Price Alert System</h1>
      </header>

      <main className="mx-auto max-w-7xl p-6">
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Live Prices */}
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h2 className="mb-4 text-lg font-semibold text-gray-300">
              Live Prices
            </h2>
            <p className="text-gray-500">Connecting to WebSocket...</p>
          </div>

          {/* My Alerts */}
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h2 className="mb-4 text-lg font-semibold text-gray-300">
              My Alerts
            </h2>
            <p className="text-gray-500">No alerts configured yet.</p>
          </div>

          {/* Trending Drops */}
          <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
            <h2 className="mb-4 text-lg font-semibold text-gray-300">
              Trending Drops (Top-K)
            </h2>
            <p className="text-gray-500">Loading analytics...</p>
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;
