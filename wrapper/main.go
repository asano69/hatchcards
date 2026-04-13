// hashwrap - on-demand reverse proxy launcher for hashcards
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type Config struct {
	Listen            string        `json:"listen"`
	StartupTimeoutSec int           `json:"startup_timeout_sec"`
	PublicBaseURL     string        `json:"public_base_url"`
	PWADir            string        `json:"pwa_dir"`
	Style             StyleConfig   `json:"style"`
	Routes            []RouteConfig `json:"routes"`
}

// StyleConfig controls injected CSS behaviour.
type StyleConfig struct {
	// DarkBrightness is the CSS brightness() value applied in dark mode (default: 0.70).
	// Lower = darker text. Typical range: 0.60–0.85.
	DarkBrightness float64 `json:"dark_brightness"`
	// MobileRootSize sets html font-size on mobile, scaling all rem-based text
	// proportionally. Default: "90%". Example: "85%", "14px".
	MobileRootSize string `json:"mobile_root_size"`
	// MobileH1Size overrides font-size for every <h1> on mobile.
	// Leave empty to inherit from MobileRootSize. Example: "1.3rem", "110%".
	MobileH1Size string `json:"mobile_h1_size"`
	// MobilePSize overrides font-size for every <p> on mobile.
	// Leave empty to inherit from MobileRootSize. Example: "0.9rem", "95%".
	MobilePSize string `json:"mobile_p_size"`
}

type RouteConfig struct {
	Path string `json:"path"`
	// Command to run. {port} is replaced with the actual port number.
	// Example: "./hashcards drill --host=0.0.0.0 --port={port} --open-browser=false"
	Command string `json:"command"`
	// Port to bind hashcards to. 0 = auto-assign a free port.
	Port int `json:"port"`
}

// stripPrefix reports whether the path prefix should be stripped before
// forwarding requests to the backend. Always true for sub-path routes;
// false only for the root route "/". hashcards always expects to run at
// the root, so any prefix must be removed before the request is forwarded.
func (r RouteConfig) stripPrefix() bool {
	return r.Path != "/"
}

// ---------------------------------------------------------------------------
// Route state
// ---------------------------------------------------------------------------

type routeState struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	port  int
	proxy *httputil.ReverseProxy
}

func (s *routeState) isAlive() bool {
	return s.cmd != nil && s.cmd.ProcessState == nil
}

// ---------------------------------------------------------------------------
// Early exit error
// ---------------------------------------------------------------------------

const earlyExitHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>hashcards</title>
  <style>
    body {
      margin: 0; padding: 0;
      font-family: "TeX Gyre Termes", "Nimbus Roman No9 L", "Times New Roman", "Times", serif;
      font-weight: normal;
      box-sizing: border-box;
    }
    .wrap {
      display: flex; flex-direction: column;
      align-items: center; justify-content: center;
      min-height: 80vh; gap: 3em;
      text-align: center; padding: 2em;
    }
    .msg { margin: 0; font-size: 32px; opacity: 0.4; }
    .path { margin: 0; font-size: 56px; white-space: pre-wrap; }
    a     { font-size: 32px; color: inherit; opacity: 0.5; text-underline-offset: 8px; }
    a:hover { opacity: 0.8; }
    @media (prefers-color-scheme: dark) {
      body { background: #1a1a1a; color: #e0e0e0; }
    }
  </style>
</head>
<body>
  <div class="wrap">
    <p class="path">%s</p>
    <p class="msg">%s</p>
    <a href="%s">check again</a>
  </div>
</body>
</html>`

// EarlyExitError is returned when the process exits before the port becomes available.
// Output holds the combined stdout+stderr of the process.
type EarlyExitError struct {
	Output string
}

func (e *EarlyExitError) Error() string {
	return fmt.Sprintf("process exited early: %s", e.Output)
}

// ---------------------------------------------------------------------------
// Process lifecycle
// ---------------------------------------------------------------------------

// ensureRunning starts the hashcards process if it is not already running,
// then waits for it to begin listening on the configured port.
func (s *routeState) ensureRunning(cfg RouteConfig, wrapperPort int, configBase string, pwaEnabled bool, styleCfg StyleConfig, timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isAlive() {
		log.Printf("[%s] process running (port=%d), reusing", cfg.Path, s.port)
		return nil
	}

	// Determine port.
	port := cfg.Port
	if port == 0 {
		var err error
		port, err = getFreePort()
		if err != nil {
			return fmt.Errorf("failed to get free port: %w", err)
		}
	}

	// Build command.
	cmdStr := strings.ReplaceAll(cfg.Command, "{port}", fmt.Sprintf("%d", port))
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("command is empty")
	}

	// Capture stdout/stderr into buffer (also tee to terminal).
	var outBuf bytes.Buffer
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &outBuf)

	log.Printf("[%s] starting: %s", cfg.Path, cmdStr)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Single goroutine owns the one and only cmd.Wait() call.
	// The channel carries the Wait error (nil on clean exit).
	exited := make(chan error, 1)
	go func() {
		exited <- cmd.Wait()
	}()

	// Wait until the port is listening, the process exits, or timeout.
	if err := waitForPortOrExit(exited, port, timeout); err != nil {
		output := strings.TrimSpace(outBuf.String())
		if output == "" {
			output = "(no output)"
		}
		return &EarlyExitError{Output: output}
	}

	log.Printf("[%s] ready (port=%d)", cfg.Path, port)

	stripPrefix := cfg.stripPrefix()

	// Build reverse proxy.
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Save the client-facing host and scheme BEFORE original() clears req.Host.
		// Priority: configBase > X-Forwarded-* (set by nginx/Caddy) > direct Host header.
		var publicBase string
		if configBase != "" {
			publicBase = configBase
		} else {
			proto := req.Header.Get("X-Forwarded-Proto")
			if proto == "" {
				proto = "http"
			}
			host := req.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = req.Host
			}
			if host == "" {
				host = fmt.Sprintf("localhost:%d", wrapperPort)
			}
			publicBase = proto + "://" + host
		}

		original(req)

		// Pass the resolved public base to ModifyResponse via a private header.
		// This header is for internal use only and must not reach hashcards.
		req.Header.Set("X-Hashwrap-Base", strings.TrimRight(publicBase, "/"))

		// Disable compression so ModifyResponse can read and rewrite the response
		// body as plain text. Without this, hashcards may return gzip-encoded
		// responses that bytes.Index / ReplaceAll cannot process.
		req.Header.Del("Accept-Encoding")

		// Strip all headers that could leak the public hostname to hashcards.
		// hashcards must only see Host: 127.0.0.1:{backendPort} so that the URLs
		// it generates are predictable localhost URLs we can reliably rewrite.
		req.Host = ""
		req.Header.Del("X-Forwarded-Host")
		req.Header.Del("X-Forwarded-Proto")
		req.Header.Del("X-Forwarded-For")

		if stripPrefix {
			trimmed := strings.TrimPrefix(req.URL.Path, cfg.Path)
			if trimmed == "" {
				trimmed = "/"
			}
			req.URL.Path = trimmed
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, cfg.Path)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[%s] proxy error: %v", cfg.Path, err)
		http.Error(w, "failed to connect to hashcards", http.StatusBadGateway)
	}

	// Rewrite URLs in responses (always enabled; needed for full-URL asset refs).
	proxy.ModifyResponse = makeRewriter(cfg.Path, stripPrefix, port, wrapperPort, pwaEnabled, styleCfg)

	s.cmd = cmd
	s.port = port
	s.proxy = proxy

	// Monitor process exit using the same channel; reset state when done.
	go func() {
		waitErr := <-exited
		s.mu.Lock()
		defer s.mu.Unlock()
		if waitErr != nil {
			log.Printf("[%s] process exited with error (port=%d): %v", cfg.Path, port, waitErr)
		} else {
			log.Printf("[%s] process exited cleanly (port=%d) — will restart on next request", cfg.Path, port)
		}
		s.cmd = nil
		s.port = 0
		s.proxy = nil
	}()

	return nil
}

// ---------------------------------------------------------------------------
// Response rewriting
// ---------------------------------------------------------------------------

// makeRewriter returns a ModifyResponse function that rewrites backend URLs in
// HTML/CSS/JS responses so that assets resolve through the public-facing URL.
//
// The public base URL is passed in via the X-Hashwrap-Base header, which
// the Director sets on every proxied request before forwarding to the backend.
//
//	Backend generates: http://127.0.0.1:8002/file/img.jpg
//	After rewrite:     https://cards.example.com/greek/file/img.jpg
//
// When pwaEnabled is true, PWA meta tags, service worker registration, and
// dark-mode styles are injected into every HTML response just before </head>.
func makeRewriter(prefix string, stripPrefix bool, backendPort, wrapperPort int, pwaEnabled bool, styleCfg StyleConfig) func(*http.Response) error {
	rewritableTypes := []string{
		"text/html",
		"text/css",
		"application/javascript",
		"text/javascript",
		"application/json",
	}

	return func(resp *http.Response) error {
		// Read the public base URL saved by the Director.
		publicBase := resp.Request.Header.Get("X-Hashwrap-Base")
		if publicBase == "" {
			publicBase = fmt.Sprintf("http://localhost:%d", wrapperPort)
		}

		// Rewrite Location header for redirects.
		if loc := resp.Header.Get("Location"); loc != "" {
			if strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, prefix) {
				newLoc := prefix + loc
				resp.Header.Set("Location", newLoc)
				log.Printf("[%s] Location rewrite: %s → %s", prefix, loc, newLoc)
			}
		}

		ct := resp.Header.Get("Content-Type")
		rewritable := false
		for _, t := range rewritableTypes {
			if strings.Contains(ct, t) {
				rewritable = true
				break
			}
		}
		if !rewritable {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		replaced := rewriteBody(body, prefix, stripPrefix, backendPort, publicBase)

		// Always inject common styles (dark mode + mobile) into every HTML response.
		// Additionally inject PWA tags when pwa_dir is configured.
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			replaced = injectStyles(replaced, pwaEnabled, styleCfg)
		}

		resp.Body = io.NopCloser(bytes.NewReader(replaced))
		resp.ContentLength = int64(len(replaced))
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(replaced)))
		return nil
	}
}

// buildCommonStyles generates the CSS block injected into every HTML response.
func buildCommonStyles(s StyleConfig) string {
	br := s.DarkBrightness
	if br == 0 {
		br = 0.70
	}

	var mobile strings.Builder

	rootSize := s.MobileRootSize
	if rootSize == "" {
		rootSize = "90%"
	}
	fmt.Fprintf(&mobile, "  html { font-size: %s !important; }\n", rootSize)

	if s.MobileH1Size != "" {
		fmt.Fprintf(&mobile, "  h1 { font-size: %s !important; }\n", s.MobileH1Size)
	}
	if s.MobilePSize != "" {
		fmt.Fprintf(&mobile, "  p { font-size: %s !important; }\n", s.MobilePSize)
	}

	return fmt.Sprintf(`<style>
/* ---- Dark mode (invert trick, no white flash) ---- */
@media (prefers-color-scheme: dark) {
  html {
    background: #000 !important;
  }
  body {
    filter: invert(1) hue-rotate(180deg) brightness(%.2f) !important;
    min-height: 100vh;
  }
  img, video, canvas, picture, svg,
  [style*="background-image"] {
    filter: invert(1) hue-rotate(180deg) brightness(%.2f) !important;
  }
}
/* ---- Mobile layout (tag selectors only — class-name independent) ---- */
@media (max-width: 768px) {
%s}
</style>`, br, br, mobile.String())
}

// pwaOnlyTags are injected in addition to commonStyles when pwa_dir is configured.
const pwaOnlyTags = `<link rel="manifest" href="/manifest.json">
<meta name="theme-color" content="#000000">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
<link rel="apple-touch-icon" href="/icon-192.png">
<script>
if ("serviceWorker" in navigator)
  navigator.serviceWorker.register("/sw.js");
</script>`

// faviconTags are always injected so that sub-path pages (/art/, /math/, etc.)
// resolve the favicon correctly. Without this, browsers fall back to requesting
// /art/favicon.ico which hashwrap does not serve.
const faviconTags = `<link rel="icon" href="/favicon.ico" sizes="any">
<link rel="icon" href="/icon-192.png" type="image/png">`

// injectStyles inserts favicon links, common styles, and optionally PWA tags
// just before </head> in every HTML response.
func injectStyles(body []byte, pwaEnabled bool, styleCfg StyleConfig) []byte {
	const marker = "</head>"
	idx := bytes.Index(bytes.ToLower(body), []byte(marker))
	if idx < 0 {
		return body
	}
	inject := buildCommonStyles(styleCfg)
	if pwaEnabled {
		inject = pwaOnlyTags + "\n" + inject
	}
	inject = faviconTags + "\n" + inject
	var buf bytes.Buffer
	buf.Grow(len(body) + len(inject))
	buf.Write(body[:idx])
	buf.WriteString(inject + "\n")
	buf.Write(body[idx:])
	return buf.Bytes()
}

// rewriteBody rewrites absolute URLs and paths in a response body.
//
// Step 1 — port-based URL rewrite (always applied):
//
//	Replaces any URL that explicitly references backendPort (regardless of
//	hostname) with the public wrapper URL, prepending the path prefix when needed.
//
//	  http://127.0.0.1:8002/file/... → https://example.com/greek/file/...
//	  http://localhost:8000/file/...  → https://example.com/file/...  (prefix="/")
//
// Step 1b — hostname-based URL normalisation (always applied):
//
//	Replaces any URL — http:// or https://, with or without a port — that uses
//	the public hostname with the canonical publicBase. This catches two cases
//	that Step 1 misses:
//
//	  (a) hashcards generated a port-less absolute URL from the Host header
//	      (e.g. http://hashcards.app.internal/art/file/...), which Step 1 cannot
//	      match because its pattern requires an explicit ":port" segment.
//	  (b) publicBase is https:// but Step 1's replacement still produced http://
//	      (e.g. because X-Forwarded-Proto was absent); the scheme is corrected here.
//
// Step 2 — absolute path rewrite (only for sub-path routes):
//
//	href="/style.css" → href="/greek/style.css"
func rewriteBody(body []byte, prefix string, stripPrefix bool, backendPort int, publicBase string) []byte {
	result := body

	// When prefix is "/", appending it would produce double slashes ("//").
	pathPrefix := prefix
	if pathPrefix == "/" {
		pathPrefix = ""
	}

	// Step 1: replace any full URL pointing at backendPort with the public wrapper URL.
	portPattern := regexp.MustCompile(
		fmt.Sprintf(`https?://[^/"']+:%d/`, backendPort),
	)
	to := publicBase + pathPrefix + "/"
	result = portPattern.ReplaceAll(result, []byte(to))

	// Step 1b: normalise any URL that uses the public hostname to the canonical
	// publicBase, correcting both scheme and port in one pass.
	if u, err := url.Parse(publicBase); err == nil && u.Hostname() != "" {
		hostPattern := regexp.MustCompile(
			fmt.Sprintf(`https?://%s(?::\d+)?/`, regexp.QuoteMeta(u.Hostname())),
		)
		canonical := strings.TrimRight(publicBase, "/") + "/"
		result = hostPattern.ReplaceAll(result, []byte(canonical))
	}

	// Step 2: rewrite absolute paths for sub-path routes so that root-relative
	// references generated by hashcards resolve through the correct prefix.
	if stripPrefix {
		type rep struct{ from, to []byte }
		patterns := []rep{
			{[]byte(`href="/`), []byte(`href="` + prefix + `/`)},
			{[]byte(`href='/`), []byte(`href='` + prefix + `/`)},
			{[]byte(`src="/`), []byte(`src="` + prefix + `/`)},
			{[]byte(`src='/`), []byte(`src='` + prefix + `/`)},
			{[]byte(`action="/`), []byte(`action="` + prefix + `/`)},
			{[]byte(`action='/`), []byte(`action='` + prefix + `/`)},
			{[]byte("url('/"), []byte("url('" + prefix + "/")},
			{[]byte(`url("/`), []byte(`url("` + prefix + `/`)},
			{[]byte("url(/"), []byte("url(" + prefix + "/")},
			{[]byte(`fetch("/`), []byte(`fetch("` + prefix + `/`)},
			{[]byte(`fetch('/`), []byte(`fetch('` + prefix + `/`)},
			{[]byte(`"/api/`), []byte(`"` + prefix + `/api/`)},
			{[]byte(`'/api/`), []byte(`'` + prefix + `/api/`)},
		}
		for _, p := range patterns {
			result = bytes.ReplaceAll(result, p.from, p.to)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Network helpers
// ---------------------------------------------------------------------------

// getFreePort asks the OS for an available TCP port.
func getFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForPortOrExit polls until the given port accepts connections, the process
// exits (signalled via exited), or the deadline is reached.
// It does NOT call cmd.Wait() itself; the caller owns the exited channel.
func waitForPortOrExit(exited <-chan error, port int, timeout time.Duration) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			return fmt.Errorf("process exited before port %d became available", port)
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil // port is ready
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for port %d", port)
}

// listenPort parses the port number from a listen address such as ":8080".
func listenPort(listen string) int {
	_, portStr, err := net.SplitHostPort(listen)
	if err != nil {
		return 8080
	}
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	return port
}

// ---------------------------------------------------------------------------
// HTTP handler
// ---------------------------------------------------------------------------

func buildHandler(cfg Config) http.Handler {
	wrapperPort := listenPort(cfg.Listen)
	timeout := time.Duration(cfg.StartupTimeoutSec) * time.Second
	pwaEnabled := cfg.PWADir != ""

	// Normalize configBase: strip any number of trailing slashes.
	// The scheme must be provided explicitly — hashwrap cannot safely infer whether
	// the deployment is HTTP or HTTPS. Omitting it is a fatal configuration error.
	// Valid examples:
	//   "https://cards.example.com"
	//   "http://localhost:3001"
	configBase := strings.TrimRight(cfg.PublicBaseURL, "/")
	if configBase != "" && !strings.Contains(configBase, "://") {
		log.Fatalf("public_base_url must include a scheme (e.g. \"http://%s\")", configBase)
	}

	states := make(map[string]*routeState, len(cfg.Routes))
	for _, r := range cfg.Routes {
		states[r.Path] = &routeState{}
	}

	mux := http.NewServeMux()

	// --- PWA static files ---
	// Served directly by hashwrap; highest priority so they are never proxied.
	if pwaEnabled {
		pwaFiles := map[string]string{
			"/manifest.json": "application/manifest+json",
			"/sw.js":         "application/javascript",
			"/icon-192.png":  "image/png",
			"/icon-512.png":  "image/png",
			"/favicon.ico":   "image/x-icon",
		}
		for path, ct := range pwaFiles {
			path, ct := path, ct // capture
			filePath := cfg.PWADir + path
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", ct)
				if path == "/manifest.json" || path == "/sw.js" {
					w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				}
				if path == "/sw.js" {
					w.Header().Set("Service-Worker-Allowed", "/")
				}
				http.ServeFile(w, r, filePath)
			})
		}
		log.Printf("PWA enabled: serving static files from %s", cfg.PWADir)
	}

	for _, route := range cfg.Routes {
		route := route
		state := states[route.Path]

		pattern := route.Path
		if !strings.HasSuffix(pattern, "/") {
			pattern += "/"
		}

		handler := func(w http.ResponseWriter, r *http.Request) {
			log.Printf("-> %s %s", r.Method, r.URL.Path)

			if err := state.ensureRunning(route, wrapperPort, configBase, pwaEnabled, cfg.Style, timeout); err != nil {
				var earlyExit *EarlyExitError
				if errors.As(err, &earlyExit) {
					log.Printf("[%s] early exit: %s", route.Path, earlyExit.Output)
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, earlyExitHTML, route.Path, earlyExit.Output, route.Path)
					return
				}
				log.Printf("[%s] startup error: %v", route.Path, err)
				http.Error(w, fmt.Sprintf("failed to start hashcards: %v", err), http.StatusInternalServerError)
				return
			}

			state.mu.Lock()
			proxy := state.proxy
			state.mu.Unlock()

			if proxy == nil {
				http.Error(w, "proxy not initialized", http.StatusInternalServerError)
				return
			}

			proxy.ServeHTTP(w, r)
		}

		mux.HandleFunc(pattern, handler)
		if route.Path != pattern {
			mux.HandleFunc(route.Path, handler)
		}
		log.Printf("route registered: %s -> %s (strip_prefix=%v)", route.Path, route.Command, route.stripPrefix())
	}

	return mux
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func main() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	f, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("cannot open config file (%s): %v", configPath, err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.StartupTimeoutSec == 0 {
		cfg.StartupTimeoutSec = 10
	}

	log.Printf("hashwrap listening on %s", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, buildHandler(cfg)); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
