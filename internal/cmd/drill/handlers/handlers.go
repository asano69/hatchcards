// Package handlers implements the HTTP handlers for the drill server.
//
// Actions handled by POST /:
//
//	Reveal   — flip the card to show the answer
//	Forgot   — grade the card and advance
//	Hard     — grade the card and advance
//	Good     — grade the card and advance
//	Easy     — grade the card and advance
//	Undo     — reverse the most recent grade
//	End      — end the session early (saves progress)
//	Shutdown — shut the server down (only valid after session ends)
package handlers

import (
	"bufio"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	drillcache "github.com/asano69/hashcards/internal/cmd/drill/cache"
	"github.com/asano69/hashcards/internal/cmd/drill/state"
	"github.com/asano69/hashcards/internal/db"
	"github.com/asano69/hashcards/internal/fsrs"
	"github.com/asano69/hashcards/internal/types"
)

// mimeTypes maps lowercase file extensions to Content-Type values.
// An explicit map is used instead of mime.TypeByExtension so that the correct
// type is returned on every OS regardless of the system MIME database
// (e.g. image/webp is not always registered on macOS).
var mimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

// Register attaches all drill routes to r.
//   - staticDir: directory containing style.css, script.js, card.html, done.html, katex/
//   - collectionRoot: root directory of the card collection (for /file/ serving)
//   - answerControls: "full" (4 buttons) or "binary" (2 buttons)
func Register(
	r *mux.Router,
	mu *sync.Mutex,
	sess *state.State,
	cache *drillcache.Cache,
	database *db.Database,
	done chan<- struct{},
	staticDir string,
	collectionRoot string,
	answerControls string,
) {
	macros := loadMacros(filepath.Join(collectionRoot, "macros.tex"))

	h := &handler{
		mu:             mu,
		sess:           sess,
		cache:          cache,
		db:             database,
		done:           done,
		staticDir:      staticDir,
		collectionRoot: collectionRoot,
		macros:         macros,
		answerControls: answerControls,
	}

	r.HandleFunc("/", h.getRoot).Methods(http.MethodGet)
	r.HandleFunc("/", h.postRoot).Methods(http.MethodPost)

	// Serve media files from the collection root at /file/<relative-path>.
	r.PathPrefix("/file/").HandlerFunc(h.serveFile)

	// Serve KaTeX assets at /katex/ so that the absolute font paths baked into
	// katex.min.css (e.g. /katex/fonts/KaTeX_Main-Regular.woff2) resolve correctly.
	katexFS := http.FileServer(http.Dir(filepath.Join(staticDir, "katex")))
	r.PathPrefix("/katex/").Handler(http.StripPrefix("/katex/", katexFS))

	// Serve remaining static assets (CSS, JS, templates).
	staticFS := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFS))
}

// loadMacros reads a macros.tex file and returns (name, definition) pairs,
// skipping lines that start with '%'.
func loadMacros(path string) [][2]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var macros [][2]string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "%") {
			continue
		}
		idx := strings.IndexByte(line, ' ')
		if idx < 0 {
			continue
		}
		name := line[:idx]
		definition := strings.TrimSpace(line[idx+1:])
		if name != "" && definition != "" {
			macros = append(macros, [2]string{name, definition})
		}
	}
	return macros
}

type handler struct {
	mu             *sync.Mutex
	sess           *state.State
	cache          *drillcache.Cache
	db             *db.Database
	done           chan<- struct{}
	sessionSaved   bool
	startedAt      time.Time
	endedAt        time.Time
	staticDir      string
	collectionRoot string
	macros         [][2]string
	// answerControls is "full" (Forgot/Hard/Good/Easy) or "binary" (Forgot/Good).
	answerControls string
}

// -------------------------------------------------------------------------
// GET /
// -------------------------------------------------------------------------

func (h *handler) getRoot(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.sess.IsFinished() {
		h.renderDone(w)
		return
	}
	h.renderCard(w)
}

// -------------------------------------------------------------------------
// POST /
// -------------------------------------------------------------------------

func (h *handler) postRoot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form data", http.StatusBadRequest)
		return
	}
	action := r.FormValue("action")

	h.mu.Lock()
	defer h.mu.Unlock()

	switch action {
	case "Reveal":
		if !h.sess.IsFinished() {
			h.sess.Reveal()
		}

	case "Undo":
		h.sess.Undo()

	case "End":
		h.finishSession()

	case "Shutdown":
		if h.sess.IsFinished() {
			select {
			case h.done <- struct{}{}:
			default:
			}
		}

	case "Forgot", "Hard", "Good", "Easy":
		if !h.sess.IsFinished() && h.sess.Revealed {
			grade := actionToGrade(action)
			newPerf, ok := h.sess.Grade(grade)
			if ok {
				last := h.sess.Done[len(h.sess.Done)-1]
				fmt.Printf("[debug] hash=%s grade=%s stability=%.2f difficulty=%.2f interval=%d due=%s\n",
					last.Card.Hash(),
					action,
					newPerf.Stability,
					newPerf.Difficulty,
					newPerf.IntervalDays,
					newPerf.DueDate.String(),
				)
			}
			if h.sess.Remaining() == 0 {
				h.finishSession()
			}
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// finishSession saves the session to the database and marks it as finished.
func (h *handler) finishSession() {
	if h.sessionSaved {
		return
	}
	h.sess.Finish()
	h.sessionSaved = true
	h.endedAt = time.Now()
	if err := h.db.SaveSession(
		h.sess.StartedAt,
		types.NewTimestamp(h.endedAt),
		h.sess.ToReviewRecords(),
	); err != nil {
		fmt.Printf("warning: save session: %v\n", err)
	}
	for _, cr := range h.sess.Done {
		if err := h.db.UpdateCardPerformance(
			cr.Card.Hash(),
			types.ReviewedCardPerformance(cr.Performance),
		); err != nil {
			fmt.Printf("warning: update card performance: %v\n", err)
		}
	}
}

func actionToGrade(action string) fsrs.Grade {
	switch action {
	case "Forgot":
		return fsrs.GradeForgot
	case "Hard":
		return fsrs.GradeHard
	case "Good":
		return fsrs.GradeGood
	case "Easy":
		return fsrs.GradeEasy
	default:
		return fsrs.GradeGood
	}
}

// -------------------------------------------------------------------------
// /file/ handler
// -------------------------------------------------------------------------

// serveFile serves a media file from the collection root directory.
// Directory traversal via ".." components is rejected.
func (h *handler) serveFile(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/file/")

	if strings.Contains(relPath, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	absPath := filepath.Join(h.collectionRoot, filepath.FromSlash(relPath))

	info, err := os.Stat(absPath)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	ct, ok := mimeTypes[ext]
	if !ok {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)

	http.ServeFile(w, r, absPath)
}

// -------------------------------------------------------------------------
// Rendering
// -------------------------------------------------------------------------

// macrosJS returns a JavaScript snippet that sets up the KaTeX MACROS object.
func (h *handler) macrosJS() template.JS {
	var sb strings.Builder
	sb.WriteString("var MACROS = {};\n")
	for _, m := range h.macros {
		name := escapeJSString(m[0])
		def := escapeJSString(m[1])
		sb.WriteString(fmt.Sprintf("MACROS['%s'] = '%s';\n", name, def))
	}
	return template.JS(sb.String())
}

func escapeJSString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "$", "\\$")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func (h *handler) renderCard(w http.ResponseWriter) {
	dc, _ := h.sess.Current()
	entry, ok := h.cache.Get(dc.Card.Hash())
	if !ok {
		http.Error(w, "card not found in cache", http.StatusInternalServerError)
		return
	}

	total := h.sess.Total()
	done := h.sess.ReviewedCount()
	percent := 0
	if total > 0 {
		percent = done * 100 / total
	}

	data := cardData{
		DeckName:       dc.Card.DeckName(),
		Front:          template.HTML(entry.Front),
		Back:           template.HTML(entry.Back),
		Revealed:       h.sess.Revealed,
		CanUndo:        len(h.sess.Done) > 0,
		Done:           done,
		Total:          total,
		ProgressPct:    percent,
		MacrosJS:       h.macrosJS(),
		AnswerControls: h.answerControls,
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.staticDir, "card.html"))
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		fmt.Printf("template error: %v\n", err)
	}
}

func (h *handler) renderDone(w http.ResponseWriter) {
	reviewed := h.sess.ReviewedCount()
	var durationSec int64
	if !h.endedAt.IsZero() {
		durationSec = int64(h.endedAt.Sub(h.sess.StartedAt.Time()).Seconds())
	}

	data := doneData{
		Reviewed:    reviewed,
		Total:       h.sess.Total(),
		DurationSec: durationSec,
	}

	tmpl, err := template.ParseFiles(filepath.Join(h.staticDir, "done.html"))
	if err != nil {
		http.Error(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		fmt.Printf("template error: %v\n", err)
	}
}

// -------------------------------------------------------------------------
// Template data types
// -------------------------------------------------------------------------

type cardData struct {
	DeckName    string
	Front       template.HTML
	Back        template.HTML
	Revealed    bool
	CanUndo     bool
	Done        int
	Total       int
	ProgressPct int
	MacrosJS    template.JS
	// AnswerControls is "full" or "binary", consumed by the card template.
	AnswerControls string
}

type doneData struct {
	Reviewed    int
	Total       int
	DurationSec int64
}
