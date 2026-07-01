// drill.js — client-side driver for the drill screen.
//
// Replaces the old server-rendered card.html / done.html / no_cards.html
// templates. All session state still lives on the server; this file only
// talks to the JSON API and re-renders the DOM.

const deck = new URLSearchParams(location.search).get("deck") || "";
const app = document.getElementById("app");

async function fetchState() {
  const res = await fetch(`/api/drill/state?deck=${encodeURIComponent(deck)}`);
  return res.json();
}

async function postAction(action) {
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

function render(s) {
  if (s.status === "no_cards") return renderNoCards();
  if (s.status === "done") return renderDone(s.done);
  return renderCard(s.card);
}

function renderNoCards() {
  app.innerHTML = `
    <div class="finished">
      <h1>No cards due today.</h1>
      <div class="shutdown-container">
        <button class="btn btn-primary" id="home-btn">Home</button>
      </div>
    </div>`;
  document.getElementById("home-btn").addEventListener("click", async () => {
    await postAction("Reset");
    location.href = "/";
  });
}

function renderDone(done) {
  app.innerHTML = `
    <div class="finished">
      <h1>Session Completed 🎉</h1>
      <div class="summary">Reviewed ${done.reviewed} cards in ${done.durationSec} seconds.</div>
      <h2>Session Stats</h2>
      <div class="stats">
        <table>
          <tr><td class="key">Total Cards</td><td class="val">${done.total}</td></tr>
          <tr><td class="key">Cards Reviewed</td><td class="val">${done.reviewed}</td></tr>
          <tr><td class="key">Duration (seconds)</td><td class="val">${done.durationSec}</td></tr>
        </table>
      </div>
      <div class="shutdown-container">
        <button class="btn btn-primary" id="home-btn">Home</button>
      </div>
    </div>`;
  document.getElementById("home-btn").addEventListener("click", async () => {
    await postAction("ReturnHome");
    location.href = "/";
  });
}

function gradeButtonsHTML(card) {
  if (card.answerControls === "binary") {
    return `
      <input id="forgot" class="btn" type="button" value="Forgot" title="Mark card as forgotten. Shortcut: 1.">
      <input id="good"   class="btn" type="button" value="Good"   title="Mark card as remembered well. Shortcut: 3.">`;
  }
  return `
    <input id="forgot" class="btn" type="button" value="Forgot" title="Forgot. Shortcut: 1.">
    <input id="hard"   class="btn" type="button" value="Hard"   title="Hard. Shortcut: 2.">
    <input id="good"   class="btn" type="button" value="Good"   title="Good. Shortcut: 3.">
    <input id="easy"   class="btn" type="button" value="Easy"   title="Easy. Shortcut: 4.">`;
}

function renderCard(card) {
  app.innerHTML = `
    <div class="root">
      <div class="header">
        <div class="reset-form">
          <button class="btn" id="reset-btn" title="Discard session and return home">Reset</button>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" style="width: ${card.progressPct}%;"></div>
        </div>
      </div>
      <div class="card-container">
        <div class="card">
          <div class="card-header"><h1>${card.deckName}</h1></div>
          <div class="card-content">${card.revealed ? card.back : card.front}</div>
        </div>
      </div>
      <div class="controls">
        <input id="undo" class="btn" type="button" value="Undo"
               title="Undo last action. Shortcut: u." ${card.canUndo ? "" : "disabled"}>
        <div class="spacer"></div>
        ${
          card.revealed
            ? `<div class="grades">${gradeButtonsHTML(card)}</div>`
            : `<input id="reveal" class="btn" type="button" value="Reveal" title="Show the answer. Shortcut: space.">`
        }
        <div class="spacer"></div>
        <input id="end" class="btn" type="button" value="End" title="End the session (changes are saved)">
      </div>
    </div>`;

  renderMathAndCode(card.macros);
  wireCardActions();
}

// renderMathAndCode runs KaTeX and highlight.js over the freshly injected
// card content. Equivalent to the old script.js DOMContentLoaded handler,
// but re-run on every render instead of once at page load.
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
  if (typeof hljs !== "undefined") hljs.highlightAll();

  const content = document.querySelector(".card-content");
  if (content) content.style.opacity = "1";
}

function wireCardActions() {
  const bind = (id, action) => {
    const el = document.getElementById(id);
    if (el) el.addEventListener("click", () => run(action));
  };
  bind("reveal", "Reveal");
  bind("undo", "Undo");
  bind("end", "End");
  bind("forgot", "Forgot");
  bind("hard", "Hard");
  bind("good", "Good");
  bind("easy", "Easy");

  document.getElementById("reset-btn").addEventListener("click", async () => {
    await postAction("Reset");
    location.href = "/";
  });
}

async function run(action) {
  render(await postAction(action));
}

// Keyboard shortcuts, unchanged from the old script.js.
document.addEventListener("keydown", (event) => {
  if (event.target.tagName === "INPUT" && event.target.type === "text") return;
  if (event.shiftKey || event.ctrlKey || event.altKey || event.metaKey) return;

  const keybindings = {
    " ": "reveal",
    u: "undo",
    1: "forgot",
    2: "hard",
    3: "good",
    4: "easy",
  };
  const id = keybindings[event.key];
  if (id) {
    event.preventDefault();
    const node = document.getElementById(id);
    if (node) node.click();
  }
});

fetchState().then(render);
