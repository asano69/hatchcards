import { createResource, createSignal, For, Show } from "solid-js";
import Button from "../../components/Button";
import pb from "../../lib/pb";

const emptyForm = { id: null, name: "", remote_url: "", username: "", token: "", enabled: true, hook_name: "" };

// How long the "just synced" checkmark stays visible after a successful sync.
const SYNC_CHECKMARK_DURATION_MS = 2000;

async function fetchConnections() {
  return pb.collection("connections").getFullList({ sort: "name" });
}

async function fetchHooks() {
  const res = await pb.send("/api/hooks", { method: "GET" });
  return res.hooks ?? [];
}

// formatLastSynced turns a PocketBase timestamp string (or "") into a
// locale-formatted display string, or a fallback for connections that have
// never been synced.
function formatLastSynced(lastSyncedAt) {
  if (!lastSyncedAt) return "Never synced";
  const d = new Date(lastSyncedAt.replace(" ", "T") + "Z");
  if (Number.isNaN(d.getTime())) return lastSyncedAt;
  return d.toLocaleString();
}

export default function Connections() {
  const [connections, { refetch }] = createResource(fetchConnections);
  const [hooks] = createResource(fetchHooks);
  const [form, setForm] = createSignal(emptyForm);
  const [error, setError] = createSignal("");
  // Tracks which connection's mirror sync (including its post-sync hook,
  // if any) is currently in flight, so only that row shows a spinner.
  const [syncingId, setSyncingId] = createSignal(null);
  // Tracks which connection most recently finished a successful sync, so
  // its row can flash a checkmark briefly before reverting to normal.
  const [syncedId, setSyncedId] = createSignal(null);

  const startCreate = () => { setForm(emptyForm); setError(""); };
  const startEdit = (c) =>
    setForm({
      id: c.id,
      name: c.name,
      remote_url: c.remote_url,
      username: c.username,
      token: "",
      enabled: c.enabled,
      hook_name: c.hook_name ?? "",
    });

  const set = (key) => (e) =>
    setForm({ ...form(), [key]: e.target.type === "checkbox" ? e.target.checked : e.target.value });

  const save = async (e) => {
    e.preventDefault();
    setError("");
    const f = form();
    const path = f.id ? `/api/connections/${f.id}` : "/api/connections";
    const method = f.id ? "PATCH" : "POST";
    try {
      await pb.send(path, { method, body: f });
      startCreate();
      await refetch();
    } catch (err) {
      setError(err?.message || "Save failed.");
    }
  };

  const remove = async (id) => {
    if (!confirm("Delete this connection? This cannot be undone.")) return;
    await pb.collection("connections").delete(id);
    await refetch();
  };

  // Runs the mirror sync for one connection. The server-side call blocks
  // until the git sync AND any configured post-sync hook have finished, so
  // the spinner naturally stays up for the whole thing. On success, flash
  // a checkmark for a couple seconds so the user gets clear feedback that
  // the sync (and the refreshed last_synced_at) actually landed.
  const sync = async (id) => {
    setSyncingId(id);
    try {
      await pb.send(`/api/connections/${id}/mirror`, { method: "POST" });
      await refetch();
      setSyncedId(id);
      setTimeout(() => {
        // Only clear if another sync hasn't already started/finished for
        // a different connection in the meantime.
        setSyncedId((current) => (current === id ? null : current));
      }, SYNC_CHECKMARK_DURATION_MS);
    } finally {
      setSyncingId(null);
    }
  };

  return (
    <div class="flex flex-col gap-6">
      <h2 class="font-serif text-3xl">Mirror Connections</h2>

      <ul class="flex flex-col gap-3">
        <For each={connections()}>
          {(c) => (
            <li class="flex flex-wrap items-center justify-between gap-y-2 rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] px-4 py-3">
  <div>
    <div class="font-semibold">{c.name}</div>
    <div class="text-sm text-[var(--color-border-soft)]">{c.remote_url}</div>

    <Show when={c.hook_name}>
      <div class="text-sm text-[var(--color-border-soft)]">hook: {c.hook_name}</div>
    </Show>

    {/* Always-visible last sync time. */}
    <div class="text-sm text-[var(--color-border-soft)]">
      Last synced: {formatLastSynced(c.last_synced_at)}
    </div>

    <Show when={c.last_error}>
      <div class="text-sm text-[#dc3545]">{c.last_error}</div>
    </Show>
  </div>
  <div class="flex flex-wrap items-center gap-2">
    <Show when={syncingId() === c.id}>
      <span
        class="h-4 w-4 animate-spin rounded-full border-2 border-[var(--color-border-soft)] border-t-transparent"
        aria-label="Syncing"
      />
    </Show>
    {/* Brief checkmark confirming the sync just succeeded. */}
    <Show when={syncedId() === c.id}>
      <span class="text-lg text-[#28a745]" aria-label="Synced">✓</span>
    </Show>
    <Button
      value={syncingId() === c.id ? "Syncing…" : "Sync"}
      disabled={syncingId() !== null}
      onClick={() => sync(c.id)}
    />
    <Button value="Edit" onClick={() => startEdit(c)} disabled={syncingId() !== null} />
    <Button variant="danger" value="Delete" onClick={() => remove(c.id)} disabled={syncingId() !== null} />
  </div>
</li>
              
          )}
        </For>
      </ul>




      <form onSubmit={save} class="flex flex-col gap-3 rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] p-6">
        <h3 class="text-xl">{form().id ? "Edit Connection" : "New Connection"}</h3>
        <input placeholder="Name" value={form().name} onInput={set("name")} required
          class="rounded-md border border-[var(--color-border-soft)] bg-[var(--color-bg)] px-3 py-2" />
        <input placeholder="Remote URL (https://github.com/org/repo.git)" value={form().remote_url} onInput={set("remote_url")} required
          class="rounded-md border border-[var(--color-border-soft)] bg-[var(--color-bg)] px-3 py-2" />
        <input placeholder="Username (leave empty for public repositories)" value={form().username} onInput={set("username")}
          class="rounded-md border border-[var(--color-border-soft)] bg-[var(--color-bg)] px-3 py-2" />
        <input type="password" value={form().token} onInput={set("token")}
          placeholder={form().id ? "Token (leave blank to keep existing / public repo)" : "Token (leave empty for public repositories)"}
          class="rounded-md border border-[var(--color-border-soft)] bg-[var(--color-bg)] px-3 py-2" />
        <label class="flex flex-col gap-1">
          <span class="text-sm text-[var(--color-border-soft)]">Post-sync hook (optional)</span>
          <select
            value={form().hook_name}
            onInput={set("hook_name")}
            class="rounded-md border border-[var(--color-border-soft)] bg-[var(--color-bg)] px-3 py-2"
          >
            <option value="">None</option>
            <For each={hooks()}>{(name) => <option value={name}>{name}</option>}</For>
          </select>
        </label>
        <label class="flex items-center gap-2">
          <input type="checkbox" checked={form().enabled} onInput={set("enabled")} />
          Enabled
        </label>
        {error() && <p class="text-sm text-[#dc3545]">{error()}</p>}
        <div class="flex gap-2">
          <button type="submit" class="btn">{form().id ? "Save" : "Create"}</button>
          <Show when={form().id}>
            <Button value="Cancel" onClick={startCreate} />
          </Show>
        </div>
      </form>
    </div>
  );
}
