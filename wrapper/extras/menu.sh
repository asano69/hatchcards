#!/bin/sh
PORT="${1:-8001}"
HTML_FILE=/tmp/index.html

# Extract only the origin part (scheme + host[:port]) from public_base_url
ORIGIN=$(grep -o '"public_base_url": "[^"]*"' /app/config.json |
	grep -o 'https\?://[^/"]*')

LINKS=$(grep -o '"path": "[^"]*"' /app/config.json |
	grep -o '"/[^"]*"' |
	sed 's/"//g' |
	while read path; do
		echo "<li><a href=\"${ORIGIN}${path}\">${path}</a></li>"
	done)
cat >"$HTML_FILE" <<HTML
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>hashcards</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    :root {
      --fg:     #1a1a1a;
      --fg-sub: #888;
      --border: #ddd;
      --bg:     #fff;
    }
    @media (prefers-color-scheme: dark) {
      html {
        background: #000;
      }
      body {
        filter: invert(1) hue-rotate(180deg) brightness(0.9);
        min-height: 100vh;
      }
    }
    body {
      font-family: "TeX Gyre Termes", "Nimbus Roman No9 L", "Times New Roman", Times, serif;
      color: var(--fg);
      background: var(--bg);
      min-height: 100vh;
      display: flex;
      align-items: flex-start;
      justify-content: center;
      padding: 4em 2em;
    }
    .wrap {
      width: 100%;
      max-width: 560px;
      margin: 0 auto;
    }
    .title {
      font-size: 12px;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      color: var(--fg-sub);
      margin-bottom: 3em;
    }
    ul {
      list-style: none;
      border-top: 1px solid var(--border);
    }
    li {
      border-bottom: 1px solid var(--border);
    }
    li a {
      display: block;
      padding: 0.8em 0;
      font-size: 26px;
      color: var(--fg);
      text-decoration: none;
      word-break: break-all;
    }
    li a:hover {
      text-decoration: underline;
      text-underline-offset: 6px;
    }
  </style>
</head>
<body>
  <div class="wrap">
    <p class="title">index</p>
    <ul>${LINKS}</ul>
  </div>
</body>
</html>
HTML

httpd -f -p "$PORT" -h /tmp
