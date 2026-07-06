import { createSignal } from "solid-js";
import { A } from "@solidjs/router";
import pb from "../lib/pb";

// Shared top navigation bar shown on Home, Stats, and Admin.
// The "Hashcards" title itself links back to Home, so only Stats and
// Admin need explicit links here.
//
// Refresh re-scans the deck directory on the server (POST /api/rescan),
// then calls `onRefresh` (if given) so the current page can reload
// whatever data it displays.
export default function NavBar(props) {
  const [refreshing, setRefreshing] = createSignal(false);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await pb.send("/api/rescan", { method: "POST" });
      await props.onRefresh?.();
    } finally {
      setRefreshing(false);
    }
  };

  // Clearing the auth store is enough: AuthGate in main.jsx watches
  // pb.authStore and swaps the app for the Login screen automatically.
  const handleLogout = () => pb.authStore.clear();

  return (
    <div class="mb-10 flex w-full items-center justify-between">
<A href="/" class="font-serif text-4xl flex items-center gap-2 transition-opacity hover:opacity-80">
  <img src="/favicon.svg" alt="" class="h-12 w-12" />#
</A>
      <nav class="flex items-center gap-3">
        <button type="button" class="btn" disabled={refreshing()} onClick={handleRefresh}>
          {refreshing() ? "Refreshing…" : "Refresh"}
        </button>
        <A href="/stats" class="btn">Stats</A>
        <A href="/admin" class="btn">Admin</A>
        <A href="/connections" class="btn">Connections</A>
        <button type="button" class="btn" onClick={handleLogout}>Log out</button>
      </nav>
    </div>
  );
}
