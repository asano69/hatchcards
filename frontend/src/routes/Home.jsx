import { createResource, For, Show } from "solid-js";
import { A } from "@solidjs/router";

// Session data (including the retrievability percentage used for the
// progress-bar background) comes entirely from GET /api/sessions.
async function fetchSessions() {
  const res = await fetch("/api/sessions");
  if (!res.ok) {
    throw new Error(`GET /api/sessions failed: ${res.status}`);
  }
  return res.json();
}

function SessionItem(props) {
  const pct = props.session.retri_pct.toFixed(1);
  return (
    <li>
      <A
        href={props.session.drill_url}
        class="relative flex items-center justify-between overflow-hidden rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] px-5 py-4 text-lg font-semibold text-[var(--color-text)] shadow-[0_1px_3px_0_var(--color-shadow)] transition-colors hover:bg-[var(--color-hover-bg)] hover:border-[var(--color-hover-border)]"
      >
        {/* Retrievability fill, drawn behind the text as a simple overlay. */}
        <div
          class="absolute inset-y-0 left-0"
          style={{
            width: `${pct}%`,
            background: "var(--color-progress)",
          }}
        />
        <span class="relative">{props.session.name}</span>
        <span class="relative text-sm font-normal opacity-60">{pct}%</span>
      </A>
    </li>
  );
}

export default function Home() {
  const [sessions] = createResource(fetchSessions);

  return (
    <div class="mx-auto flex min-h-screen w-full max-w-xl flex-col items-center bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      <h1 class="mb-10 font-serif text-4xl">Hashcards</h1>
      {/* Nothing is shown while the initial request is in flight, matching
          the previous vanilla-JS behaviour. */}
      <Show when={!sessions.loading}>
        <Show
          when={!sessions.error}
          fallback={<p class="text-[var(--color-border-soft)]">Failed to load sessions.</p>}
        >
          <Show
            when={sessions().length > 0}
            fallback={<p class="text-[var(--color-border-soft)]">No sessions configured.</p>}
          >
            <ul class="flex w-full flex-col gap-3">
              <For each={sessions()}>{(s) => <SessionItem session={s} />}</For>
            </ul>
          </Show>
        </Show>
      </Show>
    </div>
  );
}
