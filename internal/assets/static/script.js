document.addEventListener("DOMContentLoaded", function () {
  // MACROS is defined by the inline <script> block in card.html, which is
  // loaded before this file. Fall back to an empty object when not present
  // (e.g. during standalone testing).
  var macros = typeof MACROS !== "undefined" ? MACROS : {};

  // Render inline math
  document.querySelectorAll(".math-inline").forEach(function (element) {
    katex.render(element.textContent, element, {
      displayMode: false,
      throwOnError: false,
      macros: macros,
    });
  });
  // Render display math
  document.querySelectorAll(".math-display").forEach(function (element) {
    katex.render(element.textContent, element, {
      displayMode: true,
      throwOnError: false,
      macros: macros,
    });
  });
  // Initialize syntax highlighting
  if (typeof hljs !== "undefined") {
    hljs.highlightAll();
  }
  // Fade in card content after rendering
  const cardContent = document.querySelector(".card-content");
  if (cardContent) {
    cardContent.style.opacity = "1";
  }
});

document.addEventListener("keydown", function (event) {
  // Skip during text input.
  if (event.target.tagName === "INPUT" && event.target.type === "text") {
    return;
  }
  // Ignore modifiers.
  if (event.shiftKey || event.ctrlKey || event.altKey || event.metaKey) {
    return;
  }

  const keybindings = {
    " ": "reveal", // Space
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
    if (node) {
      node.click();
    }
  }
});
