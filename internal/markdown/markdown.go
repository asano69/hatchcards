// Package markdown renders card content to HTML using goldmark.
//
// HTMLFront and HTMLBack are the primary entry points. They accept a
// types.Card and the absolute path of the deck file that contains it.
//
// Image and audio src attributes are rewritten to <fileMountBase>/file/<path>
// URLs so the drill server can serve them directly.
//
// Math syntax ($...$ and $$...$$) is preprocessed into spans with the
// "math-inline" and "math-display" CSS classes that KaTeX.js expects.
package markdown

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/asano69/hashcards/internal/types"
)

// renderer is a configured goldmark instance shared across all calls.
// WithUnsafe is required so that raw HTML injected by preprocessMath
// and rewriteAudioMarkdown passes through without being escaped.
var renderer = goldmark.New(
	goldmark.WithExtensions(extension.Table, extension.Strikethrough),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// imgSrcRe matches the src attribute of any <img> element in rendered HTML.
var imgSrcRe = regexp.MustCompile(`(<img\b[^>]*?\bsrc=")([^"]+)"`)

// audioSrcRe matches the src attribute of any <audio> element in rendered HTML.
var audioSrcRe = regexp.MustCompile(`(<audio\b[^>]*?\bsrc=")([^"]+)"`)

// audioExtensions is the set of file extensions treated as audio.
var audioExtensions = map[string]bool{
	".mp3": true,
	".wav": true,
	".ogg": true,
	".m4a": true,
}

// displayMathRe matches $$...$$ display math blocks (including newlines).
var displayMathRe = regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)

// inlineMathRe matches $...$ inline math (no newlines, non-greedy).
var inlineMathRe = regexp.MustCompile(`\$([^$\n]+?)\$`)

// clozePlaceholder is substituted into the markdown source before rendering,
// then replaced with the actual HTML span after rendering. This matches the
// Rust implementation's approach and ensures the placeholder is treated as
// plain text by the markdown renderer (e.g. inside backtick code spans).
const clozePlaceholder = "XHASHCARDSCLOZEX"

// HTMLFront returns the HTML for the front face of a card.
// fileMountBase is the URL prefix used when constructing /file/ paths,
// e.g. "/drill/geo". Pass "" to use the bare path "/file/...".
func HTMLFront(card types.Card, deckFilePath string, fileMountBase string) (string, error) {
	cc := card.Content()
	switch cc.Kind() {
	case types.CardTypeBasic:
		return renderMarkdown(cc.Question, deckFilePath, fileMountBase)
	default: // CardTypeCloze
		src := clozeWithPlaceholder(cc.Text, cc.Start, cc.End)
		rendered, err := renderMarkdown(src, deckFilePath, fileMountBase)
		if err != nil {
			return "", err
		}
		return strings.ReplaceAll(rendered, clozePlaceholder, `<span class="cloze">[...]</span>`), nil
	}
}

// HTMLBack returns the HTML for the back face of a card.
// fileMountBase is the URL prefix used when constructing /file/ paths.
func HTMLBack(card types.Card, deckFilePath string, fileMountBase string) (string, error) {
	cc := card.Content()
	switch cc.Kind() {
	case types.CardTypeBasic:
		return renderMarkdown(cc.Answer, deckFilePath, fileMountBase)
	default: // CardTypeCloze
		textBytes := []byte(cc.Text)
		end := cc.End
		if end >= len(textBytes) {
			end = len(textBytes) - 1
		}
		// Render the deleted content separately as inline markdown so that
		// any markup inside the deletion (e.g. math) is properly rendered.
		deleted := string(textBytes[cc.Start : end+1])
		renderedDeleted, err := renderMarkdownInline(deleted, deckFilePath, fileMountBase)
		if err != nil {
			return "", err
		}
		src := clozeWithPlaceholder(cc.Text, cc.Start, cc.End)
		rendered, err := renderMarkdown(src, deckFilePath, fileMountBase)
		if err != nil {
			return "", err
		}
		revealSpan := `<span class="cloze-reveal">` + renderedDeleted + `</span>`
		return strings.ReplaceAll(rendered, clozePlaceholder, revealSpan), nil
	}
}

// clozeWithPlaceholder replaces the bytes [start, end] in text with
// clozePlaceholder and returns the resulting string.
func clozeWithPlaceholder(text string, start, end int) string {
	textBytes := []byte(text)
	if end >= len(textBytes) {
		end = len(textBytes) - 1
	}
	return string(textBytes[:start]) + clozePlaceholder + string(textBytes[end+1:])
}

// renderMarkdownInline renders markdown and strips the outer <p>...</p>
// wrapper when the result is a single paragraph, matching the Rust
// markdown_to_html_inline behaviour.
func renderMarkdownInline(src, deckFilePath, fileMountBase string) (string, error) {
	html, err := renderMarkdown(src, deckFilePath, fileMountBase)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(html, "<p>") && strings.HasSuffix(html, "</p>\n") {
		return html[3 : len(html)-5], nil
	}
	return html, nil
}

// renderMarkdown converts a Markdown string to HTML, then rewrites image and
// audio src attributes to <fileMountBase>/file/<relative-path> URLs.
func renderMarkdown(src, deckFilePath, fileMountBase string) (string, error) {
	src = rewriteAudioMarkdown(src)
	src = preprocessMath(src)

	var buf bytes.Buffer
	if err := renderer.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	result := rewriteSrcs(buf.String(), deckFilePath, fileMountBase)
	return result, nil
}

// preprocessMath converts $...$ and $$...$$ syntax into raw HTML spans that
// KaTeX.js will find and render.
func preprocessMath(src string) string {
	src = displayMathRe.ReplaceAllString(src, `<span class="math math-display">$1</span>`)
	src = inlineMathRe.ReplaceAllString(src, `<span class="math math-inline">$1</span>`)
	return src
}

// rewriteAudioMarkdown replaces Markdown image syntax that points to audio
// files with a raw <audio> tag before Markdown rendering.
func rewriteAudioMarkdown(src string) string {
	imageRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	return imageRe.ReplaceAllStringFunc(src, func(match string) string {
		sub := imageRe.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		path := sub[2]
		ext := strings.ToLower(filepath.Ext(path))
		if audioExtensions[ext] {
			return `<audio controls src="` + path + `"></audio>`
		}
		return match
	})
}

// rewriteSrcs rewrites every relative src attribute in <img> and <audio>
// elements to a <fileMountBase>/file/<path> URL so the drill server can serve them.
// Absolute URLs and data URIs are left unchanged.
func rewriteSrcs(rawHTML, deckFilePath, fileMountBase string) string {
	rewrite := func(src string) string {
		if isURL(src) {
			return src
		}
		if strings.HasPrefix(src, "@/") {
			return fileMountBase + "/file/" + src[2:]
		}
		return fileMountBase + "/file/" + filepath.ToSlash(src)
	}

	result := imgSrcRe.ReplaceAllStringFunc(rawHTML, func(match string) string {
		parts := imgSrcRe.FindStringSubmatch(match)
		if parts == nil {
			return match
		}
		prefix, src := parts[1], parts[2]
		return prefix + rewrite(src) + `"`
	})

	result = audioSrcRe.ReplaceAllStringFunc(result, func(match string) string {
		parts := audioSrcRe.FindStringSubmatch(match)
		if parts == nil {
			return match
		}
		prefix, src := parts[1], parts[2]
		return prefix + rewrite(src) + `"`
	})

	return result
}

// isURL reports whether src is an external URL, data URI, or already-rewritten path.
func isURL(src string) bool {
	return strings.HasPrefix(src, "http://") ||
		strings.HasPrefix(src, "https://") ||
		strings.HasPrefix(src, "data:") ||
		strings.Contains(src, "/file/")
}
