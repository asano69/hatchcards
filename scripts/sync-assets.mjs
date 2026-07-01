#!/usr/bin/env node
// Copies vendored frontend assets from node_modules into the Go embed
// tree (internal/assets/static/), so `go:embed` picks up npm-managed
// packages without committing their source to git.
import { cpSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, "..");

const targets = [
  {
    name: "katex",
    src: path.join(root, "frontend", "node_modules", "katex", "dist"),
    dest: path.join(root, "internal", "assets", "static", "katex"),
  },
];

for (const t of targets) {
  if (!existsSync(t.src)) {
    console.error(
      `sync-assets: ${t.name} not found at ${t.src}; did you run "npm install"?`,
    );
    process.exit(1);
  }
  rmSync(t.dest, { recursive: true, force: true });
  cpSync(t.src, t.dest, { recursive: true });
  console.log(
    `sync-assets: copied ${t.name} -> ${path.relative(root, t.dest)}`,
  );
}
