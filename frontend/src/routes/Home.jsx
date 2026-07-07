import { createResource, For, Show } from "solid-js";
import { A } from "@solidjs/router";
import NavBar from "../components/NavBar";
import pb from "../lib/pb";


async function loadHomeData() {
  const [sessions, status, counts] = await Promise.all([
    fetchSessions(),
    fetchTodayStatus(),
    fetchCardCounts(),
  ]);
  return { sessions, status, counts };
}


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

// card_counts is keyed by deck_name; card_counts_all has no deck_name and
// always contains exactly one row (the aggregate across all decks).
async function fetchCardCounts() {
  const [perDeck, all] = await Promise.all([
    pb.collection("card_counts").getFullList({ sort: "deck_name" }),
    pb.collection("card_counts_all").getFullList(),
  ]);
  return {
    byDeck: new Map(perDeck.map((r) => [r.deck_name, r])),
    all: all[0] ?? null,
  };
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
  // The "All Decks" session (path === "") has no deck_name in today_status
  // or card_counts, so it reads from the "_all" view instead.
  const stat = () =>
    props.session.path === ""
      ? props.status.all
      : props.status.byDeck.get(props.session.name);
  const count = () =>
    props.session.path === ""
      ? props.counts.all
      : props.counts.byDeck.get(props.session.name);
  const pct = () => reviewedPct(stat()).toFixed(1);

  const isEmpty = () =>
    (stat()?.new_count ?? 0) === 0 && (stat()?.due_count ?? 0) === 0;

  return (
    <li>
      <A
        href={props.session.drill_url}
        class="relative flex items-center justify-between overflow-hidden rounded-md border border-[var(--color-border-soft)] px-5 py-4 text-lg font-semibold text-[var(--color-text)] shadow-[0_1px_3px_0_var(--color-shadow)] transition-colors hover:bg-[var(--color-hover-bg)] hover:border-[var(--color-hover-border)]"
        style={{ background: isEmpty() ? "var(--color-muted)" : "var(--color-field)" }}
      >
        <div
          class="absolute inset-y-0 left-0"
          style={{ width: `${pct()}%`, background: "var(--color-progress)" }}
        />
        <span class="relative">
          {props.session.name}
          <span class="ml-2 font-normal text-[var(--color-border-soft)]">
            ({count()?.card_count ?? 0})
          </span>
        </span>
        <span class="relative flex gap-2 text-sm font-serif tabular-nums">
          <span className="text-gray-500">{Math.round((stat()?.new_count ?? 0) + (stat()?.due_count ?? 0))}</span>
          <span className="text-green-500">{Math.round(stat()?.reviewed_today_count ?? 0)}</span>
        </span>
      </A>
    </li>
  );
}

export default function Home() {
  const [data, { refetch }] = createResource(loadHomeData);

  return (
    <div class="mx-auto flex min-h-screen w-full max-w-xl flex-col items-center bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      <NavBar onRefresh={refetch} />
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
                {(s) => <SessionItem session={s} status={data().status} counts={data().counts} />}
              </For>
            </ul>
          </Show>
        </Show>
      </Show>
    </div>
  );
}
