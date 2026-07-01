import { render } from "solid-js/web";
import { Router, Route } from "@solidjs/router";

// Order matters: tokens.css defines the CSS custom properties every other
// stylesheet consumes via var().
import "./styles/tokens.css";
import "./styles/base.css";
import "./styles/components.css";
import "./styles/index.css";
import "./styles/drill.css";
import "./styles/done.css";
import "highlight.js/styles/github.css";


window.katex = katex; // Drill.jsx が window.katex/window.hljs 経由で参照するため
window.hljs = hljs;


import Sessions from "./routes/Sessions";
import Drill from "./routes/Drill";

render(
  () => (
    <Router>
      <Route path="/" component={Sessions} />
      <Route path="/drill" component={Drill} />
    </Router>
  ),
  document.getElementById("app"),
);
