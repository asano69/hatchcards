import { createResource, createSignal, For, Show } from "solid-js";
import NavBar from "../components/NavBar";
import Button from "../components/Button";
import pb from "../lib/pb";

const emptyForm = { id: null, name: "", remote_url: "", username: "", token: "", enabled: true, hook_name: "" };

async function fetchConnections() {
  return pb.collection("connections").getFullList({ sort: "name" });
}

// The hook picker only ever offers names the server resolves against its
// hooks directory (see GET /api/hooks), so this stays a plain <select>
// instead of a free-text field.
async function fetchHooks() {
  const res = await pb.send("/api/hooks", { method: "GET" });
  return res.hooks ?? [];
}

export default function Connections() {
  const [connections, { refetch }] = createResource(fetchConnections);
  const [hooks] = createResource(fetchHooks);
  const [form, setForm] = createSignal(emptyForm);
  const [error, setError] = createSignal("");

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

  return (
    <div class="mx-auto flex min-h-screen w-full max-w-2xl flex-col gap-6 bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      <NavBar onRefresh={refetch} />
      <h1 class="font-serif text-4xl">Mirror Connections</h1>

      <ul class="flex flex-col gap-3">
        <For each={connections()}>
          {(c) => (
            <li class="flex items-center justify-between rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] px-4 py-3">
              <div>
                <div class="font-semibold">{c.name}</div>
                <div class="text-sm text-[var(--color-border-soft)]">{c.remote_url}</div>
                <div class="text-sm text-[var(--color-border-soft)]">local: {c.local_path}</div>
                <Show when={c.hook_name}>
                  <div class="text-sm text-[var(--color-border-soft)]">hook: {c.hook_name}</div>
                </Show>
                <Show when={c.last_error}>
                  <div class="text-sm text-[#dc3545]">{c.last_error}</div>
                </Show>
              </div>
              <div class="flex gap-2">
                <Button value="Sync" onClick={async () => { await pb.send(`/api/connections/${c.id}/mirror`, { method: "POST" });  await refetch();}} />
                <Button value="Edit" onClick={() => startEdit(c)} />
                <Button variant="danger" value="Delete" onClick={() => remove(c.id)} />
              </div>
            </li>
          )}
        </For>
      </ul>

      <form onSubmit={save} class="flex flex-col gap-3 rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] p-6">
        <h2 class="text-xl">{form().id ? "Edit Connection" : "New Connection"}</h2>
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
