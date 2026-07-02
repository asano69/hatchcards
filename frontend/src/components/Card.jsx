import { createEffect } from "solid-js";
import katex from "katex";
import "katex/dist/contrib/mhchem";
import hljs from "highlight.js";
import "./Card.css";

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
    <div class="card-container">
      <div class="card">
        <div class="card-header"><h1>{card().deckName}</h1></div>
        <div class="card-content" innerHTML={card().revealed ? card().back : card().front} />
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
