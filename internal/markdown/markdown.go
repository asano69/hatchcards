// Package markdown renders card content to HTML using goldmark.
//
// HTMLFront and HTMLBack are the primary entry points. They accept a
// types.Card and the absolute path of the deck file that contains it.
//
// Image and audio src attributes are rewritten to /file/<path> URLs so the
// drill server can serve them directly, matching the Rust implementation.
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
// WithUnsafe is required so that the raw HTML spans injected by preprocessMath
// and rewriteAudioMarkdown pass through without being escaped.
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

// HTMLFront returns the HTML for the front face of a card.
func HTMLFront(card types.Card, deckFilePath string) (string, error) {
	cc := card.Content()
	switch cc.Kind() {
	case types.CardTypeBasic:
		return renderMarkdown(cc.Question, deckFilePath)
	default: // CardTypeCloze
		processed := processClozeText(cc.Text, cc.Start, cc.End, true)
		return renderMarkdown(processed, deckFilePath)
	}
}

// HTMLBack returns the HTML for the back face of a card.
func HTMLBack(card types.Card, deckFilePath string) (string, error) {
	cc := card.Content()
	switch cc.Kind() {
	case types.CardTypeBasic:
		return renderMarkdown(cc.Answer, deckFilePath)
	default: // CardTypeCloze
		processed := processClozeText(cc.Text, cc.Start, cc.End, false)
		return renderMarkdown(processed, deckFilePath)
	}
}

// renderMarkdown converts a Markdown string to HTML, then rewrites image and
// audio src attributes to /file/<relative-path> URLs.
func renderMarkdown(src, deckFilePath string) (string, error) {
	// Order matters: audio must be rewritten before math preprocessing, and
	// math must be preprocessed before goldmark renders the document (so the
	// injected HTML spans are not escaped).
	src = rewriteAudioMarkdown(src)
	src = preprocessMath(src)

	var buf bytes.Buffer
	if err := renderer.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	result := rewriteSrcs(buf.String(), deckFilePath)
	return result, nil
}

// preprocessMath converts $...$ and $$...$$ syntax into raw HTML spans that
// KaTeX.js will find and render. Display math is processed first to avoid
// partial matches by the inline pattern.
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
// elements to a /file/<path> URL so the drill server can serve them.
// Absolute URLs and data URIs are left unchanged.
//
// The @/ prefix is the Rust media resolver's convention for collection-root-
// relative paths (e.g. @/thetempest.webp). It is stripped so that the URL
// becomes /file/thetempest.webp rather than the invalid /file/@/thetempest.webp.
func rewriteSrcs(rawHTML, deckFilePath string) string {
	rewrite := func(src string) string {
		if isURL(src) {
			return src
		}
		// @/ means collection-root-relative. Strip the prefix so the /file/
		// handler (which serves relative to the collection root) finds the file.
		if strings.HasPrefix(src, "@/") {
			return "/file/" + src[2:]
		}
		return "/file/" + filepath.ToSlash(src)
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

// isURL reports whether src is an external URL or data URI.
func isURL(src string) bool {
	return strings.HasPrefix(src, "http://") ||
		strings.HasPrefix(src, "https://") ||
		strings.HasPrefix(src, "data:") ||
		strings.HasPrefix(src, "/file/")
}

// processClozeText rewrites raw cloze text into HTML suitable for rendering.
//
// The span at targetStart/targetEnd is the deletion being tested:
//   - isFront=true: replaced with a blank placeholder span.
//   - isFront=false: wrapped in a highlight span.
func processClozeText(rawText string, targetStart, targetEnd int, isFront bool) string {
	textBytes := []byte(rawText)
	if targetEnd >= len(textBytes) {
		targetEnd = len(textBytes) - 1
	}

	var sb strings.Builder
	sb.Write(textBytes[:targetStart])
	content := string(textBytes[targetStart : targetEnd+1])
	if isFront {
		sb.WriteString(`<span class="cloze-blank">[...]</span>`)
	} else {
		sb.WriteString(`<span class="cloze-answer">`)
		sb.WriteString(content)
		sb.WriteString(`</span>`)
	}
	sb.Write(textBytes[targetEnd+1:])
	return sb.String()
}
