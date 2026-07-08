// A single reusable button used across the drill screens (Undo, Reveal,
// grade buttons, End, Reset, etc). variant="danger" is used for the
// destructive "Home" action on the session-summary screens.

// 1. ボタンの「骨格」（サイズ、マージン、フォント、アニメーション、無効化挙動などすべて維持）
const baseLayout =
  "my-1.5 cursor-pointer appearance-none whitespace-nowrap rounded-md px-3 py-[7px] " +
  "font-sans text-base font-semibold " +
  "transition-colors transition-shadow duration-150 md:mx-3 md:my-0 " +
  "disabled:cursor-not-allowed disabled:opacity-40";

// 2. 通常時の「色」
const defaultColors =
  "shadow-[0_1px_3px_0_var(--color-shadow)] " +
  "border border-[var(--color-border-soft)] bg-[var(--color-field)] text-[var(--color-text)] " +
  "enabled:hover:bg-[var(--color-hover-bg)] enabled:hover:border-[var(--color-hover-border)] " +
  "enabled:active:bg-[var(--color-active-bg)] enabled:active:border-[var(--color-active-border)]";

// 3. Danger時の「赤色」（baseLayoutのサイズ感のまま、色だけを置き換える）
const dangerColors =
  "text-white " +
  "shadow-[0_1px_3px_0_rgb(220_53_69_/_0.3)] " + 
  "border border-[#c82333] bg-[#dc3545] " +
  "enabled:hover:bg-[#c82333] enabled:hover:border-[#c82333] " +
  "enabled:active:bg-[#bd2130] enabled:active:border-[#bd2130]";

export default function Button(props) {
  // 骨格（baseLayout）に対して、通常色か危険色かを組み合わせる
  const buttonClass = props.variant === "danger"
    ? `${baseLayout} ${dangerColors}`
    : `${baseLayout} ${defaultColors}`;

  return (
    <input
      id={props.id}
      className={buttonClass} // Reactの規約に合わせて class から className に修正しています
      type="button"
      value={props.value}
      title={props.title}
      disabled={props.disabled}
      onClick={props.onClick}
    />
  );
}
