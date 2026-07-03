import { createSignal, onMount, onCleanup, Switch, Match } from "solid-js";
import { useSearchParams, useNavigate } from "@solidjs/router";
import "../style.css";
import Button from "../components/Button";
import Card from "../components/Card";
import Done from "./Done";
import NoCards from "./NoCards";

async function fetchState(deck) {
  return pb.send("/api/drill/state", { query: { deck } });
}

async function postAction(deck, action) {
  return pb.send("/api/drill/action", {
    method: "POST",
    query: { deck },
    body: { action },
  });
}

export default function Drill() {
  const [searchParams] = useSearchParams();
  const deck = () => searchParams.deck || "";
  const navigate = useNavigate();

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

  // Runs a server-side action first, then navigates within the SPA router
  // (no full page reload, unlike the previous location.href approach).
  const goHome = async (finishAction) => {
    await run(finishAction);
    navigate("/");
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
        <DrillSession card={state().card} onAction={run} onReset={() => goHome("Reset")} />
      </Match>
    </Switch>
  );
}

// DrillSession owns everything around a single card that isn't the card
// itself: the reset button, the progress bar, and the answer/grade controls.
function DrillSession(props) {
  const card = () => props.card;
return (
    <div class="flex h-screen w-screen flex-col">
      <div class="flex flex-row items-center justify-center p-8 bg-[var(--color-panel)] border-b border-[var(--color-border)]">
        <div class="reset-form">
          <Button title="Discard session and return home" value="Reset" onClick={props.onReset} />
        </div>
        <div class="h-6 w-[300px] overflow-hidden rounded-full bg-[var(--color-card)] border border-[var(--color-border-soft)]">
          <div class="h-full bg-[var(--color-progress)]" style={{ width: `${card().progressPct}%` }} />
        </div>
      </div>

      <Card card={card()} />

      <div class="p-3 md:p-8 border-t border-[var(--color-border)]">
        <form onSubmit={(e) => e.preventDefault()} class="flex flex-col md:flex-row md:justify-between">
          <Button
            id="undo"
            value="Undo"
            title="Undo last action. Shortcut: u."
            disabled={!card().canUndo}
            onClick={() => props.onAction("Undo")}
          />

          <div class="flex-1" />

          <Switch>
            <Match when={card().revealed}>
              <div class="flex flex-row justify-between">
                <GradeButtons card={card()} onAction={props.onAction} />
              </div>
            </Match>
            <Match when={true}>
              <Button
                id="reveal"
                value="Reveal"
                title="Show the answer. Shortcut: space."
                onClick={() => props.onAction("Reveal")}
              />
            </Match>
          </Switch>

          <div class="flex-1" />
          <Button
            id="end"
            value="End"
            title="End the session (changes are saved)"
            onClick={() => props.onAction("End")}
          />
        </form>
      </div>
    </div>
  );

}

function GradeButtons(props) {
  return (
    <Switch>
      <Match when={props.card.answerControls === "binary"}>
        <Button id="forgot" value="Forgot" title="Mark card as forgotten. Shortcut: 1." onClick={() => props.onAction("Forgot")} />
        <Button id="good" value="Good" title="Mark card as remembered well. Shortcut: 3." onClick={() => props.onAction("Good")} />
      </Match>
      <Match when={true}>
        <Button id="forgot" value="Forgot" title="Forgot. Shortcut: 1." onClick={() => props.onAction("Forgot")} />
        <Button id="hard" value="Hard" title="Hard. Shortcut: 2." onClick={() => props.onAction("Hard")} />
        <Button id="good" value="Good" title="Good. Shortcut: 3." onClick={() => props.onAction("Good")} />
        <Button id="easy" value="Easy" title="Easy. Shortcut: 4." onClick={() => props.onAction("Easy")} />
      </Match>
    </Switch>
  );
}
