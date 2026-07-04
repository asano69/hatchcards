import { createResource, createSignal, For, Show } from "solid-js";
import { A } from "@solidjs/router";
import pb from "../lib/pb";

async function fetchSessions() {
  return pb.send("/api/sessions", { method: "GET" });
}

// today_status is keyed by deck_name; today_status_all has no deck_name and
// always contains exactly one row (the aggregate across all decks).
async function fetchTodayStatus() {
  const [perDeck, all] = await Promise.all([
    pb.collection("today_status").getFullList({ sort: "deck_name" }),
    pb.collection("today_status_all").getFullList(),
  ]);
  return {
    byDeck: new Map(perDeck.map((r) => [r.deck_name, r])),
    all: all[0] ?? null,
  };
}

async function loadHomeData() {
  const [sessions, status] = await Promise.all([fetchSessions(), fetchTodayStatus()]);
  return { sessions, status };
}

// reviewedPct returns what fraction of (new + due + reviewed) is reviewed,
// as a percentage in [0, 100]. Returns 0 when there's nothing to show yet.
function reviewedPct(stat) {
  if (!stat) return 0;
  const total = (stat.new_count ?? 0) + (stat.due_count ?? 0) + (stat.reviewed_today_count ?? 0);
  if (total <= 0) return 0;
  return (stat.reviewed_today_count / total) * 100;
}

function SessionItem(props) {
  // The "All Decks" session (path === "") has no deck_name in today_status,
  // so it reads from today_status_all instead.
  const stat = () =>
    props.session.path === ""
      ? props.status.all
      : props.status.byDeck.get(props.session.name);
  const pct = () => reviewedPct(stat()).toFixed(1);

  return (
    <li>
      <A
        href={props.session.drill_url}
        class="relative flex items-center justify-between overflow-hidden rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] px-5 py-4 text-lg font-semibold text-[var(--color-text)] shadow-[0_1px_3px_0_var(--color-shadow)] transition-colors hover:bg-[var(--color-hover-bg)] hover:border-[var(--color-hover-border)]"
      >
        {/* Reviewed-today fill, drawn behind the text. New/due cards are
            left uncolored — only the reviewed share is highlighted. */}
        <div
          class="absolute inset-y-0 left-0"
          style={{ width: `${pct()}%`, background: "var(--color-progress)" }}
        />
        <span class="relative">{props.session.name}</span>
        <span class="relative flex gap-3 text-sm font-normal opacity-70 tabular-nums">
        <span className="font-mono font-bold">{Math.round(stat()?.new_count ?? 0)}</span>
        <span className="text-red-600 font-mono font-bold">{Math.round(stat()?.due_count ?? 0)}</span>
        <span className="text-green-600 font-mono font-bold">{Math.round(stat()?.reviewed_today_count ?? 0)}</span>
        </span>
      </A>
    </li>
  );
}

export default function Home() {
  const [data, { refetch }] = createResource(loadHomeData);
  const [refreshing, setRefreshing] = createSignal(false);

  // Ask the server to rescan the deck directory for decks added since
  // startup, then reload sessions + today's stats to reflect any changes.
  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await pb.send("/api/rescan", { method: "POST" });
      await refetch();
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <div class="mx-auto flex min-h-screen w-full max-w-xl flex-col items-center bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      <div class="mb-10 flex w-full items-center justify-between">
        <h1 class="font-serif text-4xl">Hashcards</h1>
        <button
          type="button"
          class="btn"
          disabled={refreshing()}
          onClick={handleRefresh}
        >
          {refreshing() ? "Refreshing…" : "Refresh"}
        </button>
        <A href="/admin" class="btn">Admin</A>
      </div>
      <Show when={!data.loading}>
        <Show
          when={!data.error}
          fallback={<p class="text-[var(--color-border-soft)]">Failed to load sessions.</p>}
        >
          <Show
            when={data().sessions.length > 0}
            fallback={<p class="text-[var(--color-border-soft)]">No sessions configured.</p>}
          >
            <ul class="flex w-full flex-col gap-3">
              <For each={data().sessions}>
                {(s) => <SessionItem session={s} status={data().status} />}
              </For>
            </ul>
          </Show>
        </Show>
      </Show>
    </div>
  );
}
