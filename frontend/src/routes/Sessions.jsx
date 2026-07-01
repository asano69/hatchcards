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
  return (
    <li>
      <A
        href={props.session.drill_url}
        class="session-link"
        style={{ "--retri-pct": `${props.session.retri_pct.toFixed(1)}%` }}
      >
        {props.session.name}
      </A>
    </li>
  );
}

export default function Sessions() {
  const [sessions] = createResource(fetchSessions);

  return (
    <div class="index-wrap">
      <h1>Hashcards</h1>
      {/* Nothing is shown while the initial request is in flight, matching
          the previous vanilla-JS behaviour. */}
      <Show when={!sessions.loading}>
        <Show
          when={!sessions.error}
          fallback={<p class="index-message">Failed to load sessions.</p>}
        >
          <Show
            when={sessions().length > 0}
            fallback={<p class="index-message">No sessions configured.</p>}
          >
            <ul class="session-list">
              <For each={sessions()}>
                {(s) => <SessionItem session={s} />}
              </For>
            </ul>
          </Show>
        </Show>
      </Show>
    </div>
  );
}
