// Admin section of the Settings page: a single link to the PocketBased
// admin dashboard.
export default function Admin() {
  return (
    <div class="flex flex-col items-center justify-center py-6">
      <a
        href="/_/"
        target="_blank"
        rel="noopener noreferrer"
        class="rounded-md border border-[var(--color-border-soft)] bg-[var(--color-field)] px-5 py-3 text-lg font-semibold text-[var(--color-text)] shadow-[0_1px_3px_0_var(--color-shadow)] transition-colors hover:bg-[var(--color-hover-bg)] hover:border-[var(--color-hover-border)]"
      >
        PocketBase↗
      </a>
    </div>
  );
}
