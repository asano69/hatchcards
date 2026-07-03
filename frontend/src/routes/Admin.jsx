
// Admin route: shows a single link to the PocketBase admin dashboard.
export default function Admin() {
  return (
    <div class="mx-auto flex min-h-screen w-full max-w-xl flex-col items-center justify-center bg-[var(--color-bg)] px-6 py-12 text-[var(--color-text)]">
      {/* 2. 壊れていたタグを <A> に置き換えます */}
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
