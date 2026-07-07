import { createEffect } from "solid-js";
import katex from "katex";
import "katex/dist/contrib/mhchem";
import hljs from "highlight.js";

// Card renders a single flashcard: deck name plus front/back content.
// It owns rendering of that content (KaTeX math, highlight.js code blocks)
// since both are intrinsic to the card, not to the drill session around it.
export default function Card(props) {
  const card = () => props.card;

  // Re-run KaTeX and highlight.js over the freshly injected card content
  // whenever the card data changes. Solid runs effects after the DOM has
  // been committed, so the elements are guaranteed to exist here.
  createEffect(() => {
    renderMathAndCode(card().macros);
  });

  return (
    <div class="flex flex-1 flex-col items-center justify-center overflow-hidden bg-[var(--color-surface)]">
      <div class="flex h-full w-full flex-col shadow-none bg-[var(--color-card)] md:min-h-[400px] md:max-h-[90%] md:w-[900px] md:border md:border-[var(--color-border)] md:shadow-[0_0_48px_16px_var(--color-shadow)]">
        <div class="border-b border-[var(--color-border)] p-6">
          <h1 class="text-4xl">{card().deckName}</h1>
        </div>
        <div
          class="card-content flex flex-1 flex-col overflow-x-hidden"
          innerHTML={card().revealed ? card().back : card().front}
        />
         <Show when={card().revealed && card().lastReviewedAt}>
          <div class="px-6 pb-4 text-sm text-[var(--color-border-soft)] text-right">
            Last reviewed: {card().lastReviewedAt}
          </div>
        </Show>
      </div>
    </div>
  );
}

// renderMathAndCode runs KaTeX and highlight.js over the freshly injected
// card content.
function renderMathAndCode(macros) {
  document.querySelectorAll(".math-inline").forEach((el) => {
    katex.render(el.textContent, el, {
      displayMode: false,
      throwOnError: false,
      macros: macros || {},
    });
  });
  document.querySelectorAll(".math-display").forEach((el) => {
    katex.render(el.textContent, el, {
      displayMode: true,
      throwOnError: false,
      macros: macros || {},
    });
  });
  hljs.highlightAll();

  const content = document.querySelector(".card-content");
  if (content) content.style.opacity = "1";
}
