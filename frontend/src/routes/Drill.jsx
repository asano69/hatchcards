import { createSignal, onMount, onCleanup, createEffect, Switch, Match } from "solid-js";
import { useSearchParams } from "@solidjs/router";
import katex from "katex";
import "katex/dist/contrib/mhchem";
import hljs from "highlight.js";

async function fetchState(deck) {
  const res = await fetch(`/api/drill/state?deck=${encodeURIComponent(deck)}`);
  return res.json();
}

async function postAction(deck, action) {
  const res = await fetch(
    `/api/drill/action?deck=${encodeURIComponent(deck)}`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action }),
    },
  );
  return res.json();
}

export default function Drill() {
  const [searchParams] = useSearchParams();
  const deck = () => searchParams.deck || "";

  const [state, setState] = createSignal(null);

  const run = async (action) => setState(await postAction(deck(), action));

  onMount(() => {
    fetchState(deck()).then(setState);

    const handler = (event) => {
      if (event.target.tagName === "INPUT" && event.target.type === "text") return;
      if (event.shiftKey || event.ctrlKey || event.altKey || event.metaKey) return;

      const keybindings = { " ": "reveal", u: "undo", 1: "forgot", 2: "hard", 3: "good", 4: "easy" };
      const id = keybindings[event.key];
      if (id) {
        event.preventDefault();
        const node = document.getElementById(id);
        if (node) node.click();
      }
    };
    document.addEventListener("keydown", handler);
    onCleanup(() => document.removeEventListener("keydown", handler));
  });

  const goHome = async (finishAction) => {
    await run(finishAction);
    location.href = "/";
  };

  return (
    <Switch>
      <Match when={state()?.status === "no_cards"}>
        <NoCards onHome={() => goHome("Reset")} />
      </Match>
      <Match when={state()?.status === "done"}>
        <Done done={state().done} onHome={() => goHome("ReturnHome")} />
      </Match>
      <Match when={state()?.status === "card"}>
        <Card card={state().card} onAction={run} onReset={() => goHome("Reset")} />
      </Match>
    </Switch>
  );
}

function NoCards(props) {
  return (
    <div class="finished">
      <h1>No cards due today.</h1>
      <div class="shutdown-container">
        <button class="btn btn-primary" onClick={props.onHome}>Home</button>
      </div>
    </div>
  );
}

function Done(props) {
  const d = props.done;
  return (
    <div class="finished">
      <h1>Session Completed 🎉</h1>
      <div class="summary">Reviewed {d.reviewed} cards in {d.durationSec} seconds.</div>
      <h2>Session Stats</h2>
      <div class="stats">
        <table>
          <tbody>
            <tr><td class="key">Total Cards</td><td class="val">{d.total}</td></tr>
            <tr><td class="key">Cards Reviewed</td><td class="val">{d.reviewed}</td></tr>
            <tr><td class="key">Duration (seconds)</td><td class="val">{d.durationSec}</td></tr>
          </tbody>
        </table>
      </div>
      <div class="shutdown-container">
        <button class="btn btn-primary" onClick={props.onHome}>Home</button>
      </div>
    </div>
  );
}

function GradeButtons(props) {
  return (
    <Switch>
      <Match when={props.card.answerControls === "binary"}>
        <input id="forgot" class="btn" type="button" value="Forgot" title="Mark card as forgotten. Shortcut: 1." onClick={() => props.onAction("Forgot")} />
        <input id="good" class="btn" type="button" value="Good" title="Mark card as remembered well. Shortcut: 3." onClick={() => props.onAction("Good")} />
      </Match>
      <Match when={true}>
        <input id="forgot" class="btn" type="button" value="Forgot" title="Forgot. Shortcut: 1." onClick={() => props.onAction("Forgot")} />
        <input id="hard" class="btn" type="button" value="Hard" title="Hard. Shortcut: 2." onClick={() => props.onAction("Hard")} />
        <input id="good" class="btn" type="button" value="Good" title="Good. Shortcut: 3." onClick={() => props.onAction("Good")} />
        <input id="easy" class="btn" type="button" value="Easy" title="Easy. Shortcut: 4." onClick={() => props.onAction("Easy")} />
      </Match>
    </Switch>
  );
}

function Card(props) {
  const card = () => props.card;

  // Re-run KaTeX and highlight.js over the freshly injected card content
  // whenever the card data changes. Solid runs effects after the DOM has
  // been committed, so the elements are guaranteed to exist here.
  createEffect(() => {
    renderMathAndCode(card().macros);
  });

  return (
    <div class="root">
      <div class="header">
        <div class="reset-form">
          <button class="btn" title="Discard session and return home" onClick={props.onReset}>Reset</button>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" style={{ width: `${card().progressPct}%` }} />
        </div>
      </div>
      <div class="card-container">
        <div class="card">
          <div class="card-header"><h1>{card().deckName}</h1></div>
          <div class="card-content" innerHTML={card().revealed ? card().back : card().front} />
        </div>
      </div>
<div class="controls">
        <form onSubmit={(e) => e.preventDefault()}>
          <input 
            id="undo" 
            class="btn" 
            type="button" 
            value="Undo" 
            title="Undo last action. Shortcut: u."
            disabled={!card().canUndo}
            onClick={() => props.onAction("Undo")}
          />
          
          <div class="spacer" />
          
          <Switch>
            <Match when={card().revealed}>
              <div class="grades">
                <GradeButtons card={card()} onAction={props.onAction} />
              </div>
            </Match>
            <Match when={true}>
              <input 
                id="reveal" 
                class="btn" 
                type="button" 
                value="Reveal" 
                title="Show the answer. Shortcut: space." 
                onClick={() => props.onAction("Reveal")} 
              />
            </Match>
          </Switch>
          
          <div class="spacer" />
          
          <input 
            id="end" 
            class="btn" 
            type="button" 
            value="End" 
            title="End the session (changes are saved)" 
            onClick={() => props.onAction("End")} 
          />
        </form>
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
