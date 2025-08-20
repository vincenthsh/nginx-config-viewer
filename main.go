package main

import (
	"bufio"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

//go:generate pnpm --dir website build

var (
	// Build information (injected by GoReleaser)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	// Command line flags
	addr        = flag.String("addr", ":8080", "listen address")
	confPath    = flag.String("path", "/etc/nginx/nginx.conf", "nginx.conf path")
	allowCORS   = flag.Bool("cors", false, "allow CORS on /raw (off by default)")
	showVersion = flag.Bool("version", false, "show version information")
)

//go:embed website/build
var staticFiles embed.FS

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("nginx-config-viewer %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		return
	}

	absPath, err := filepath.Abs(*confPath)
	if err != nil {
		log.Fatal(err)
	}

	// Simple bus for SSE clients
	hub := newSSEHub()
	go hub.run()

	// Watcher (watch the file *and* its directory so we catch atomic renames)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	dir := filepath.Dir(absPath)
	if err := watcher.Add(dir); err != nil {
		log.Fatalf("watch dir: %v", err)
	}
	_ = watcher.Add(absPath) // ignore error if file isn't present yet; dir watch will catch create

	// Debounced notifier
	go func() {
		var timer *time.Timer
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !isEventFor(ev, absPath) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(200*time.Millisecond, func() {
					hub.broadcast([]byte("reload"))
				})
			case err := <-watcher.Errors:
				log.Printf("watch error: %v", err)
			}
		}
	}()

	// Serve React app
	staticFS, _ := fs.Sub(staticFiles, "website/build")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve React app for root and non-API routes
		if r.URL.Path == "/" || (!strings.HasPrefix(r.URL.Path, "/raw") && !strings.HasPrefix(r.URL.Path, "/events")) {
			// Try to serve the requested file, fallback to index.html for SPA routing
			file := strings.TrimPrefix(r.URL.Path, "/")
			if file == "" {
				file = "index.html"
			}

			if content, err := staticFS.Open(file); err == nil {
				defer content.Close()
				if stat, err := content.Stat(); err == nil && !stat.IsDir() {
					// Set content type based on file extension
					ext := filepath.Ext(file)
					switch ext {
					case ".js":
						w.Header().Set("Content-Type", "application/javascript")
					case ".css":
						w.Header().Set("Content-Type", "text/css")
					case ".html":
						w.Header().Set("Content-Type", "text/html")
					case ".json":
						w.Header().Set("Content-Type", "application/json")
					}
					io.Copy(w, content)
					return
				}
			}

			// Fallback to index.html for SPA routing
			if indexFile, err := staticFS.Open("index.html"); err == nil {
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html")
				io.Copy(w, indexFile)
				return
			}
		}
		http.NotFound(w, r)
	})

	http.HandleFunc("/raw", func(w http.ResponseWriter, r *http.Request) {
		// single fixed file; no path traversal
		f, err := os.Open(absPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		etag := ""
		if h, err := hashOfFile(f); err == nil {
			etag = `W/"` + h + `"`
			w.Header().Set("ETag", etag)
			_, _ = f.Seek(0, io.SeekStart)
		}
		if stat != nil {
			w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
		}

		if inm := r.Header.Get("If-None-Match"); etag != "" && strings.Contains(inm, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if *allowCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		br := bufio.NewReader(f)
		io.Copy(w, br)
	})

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}
		// register client
		ch := hub.register()
		defer hub.unregister(ch)

		// initial ping
		fmt.Fprintf(w, "data: hello\n\n")
		flusher.Flush()

		// heartbeat
		ctx := r.Context()
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			case <-tick.C:
				fmt.Fprintf(w, ": ping\n\n") // comment heartbeat
				flusher.Flush()
			}
		}
	})

	log.Printf("listening on %s, serving %s", *addr, absPath)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func isEventFor(ev fsnotify.Event, target string) bool {
	// We watch the directory; detect writes, creates, renames affecting the target file
	if ev.Name == target && (ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) || ev.Has(fsnotify.Rename) || ev.Has(fsnotify.Remove)) {
		return true
	}
	// On some editors: write temp then rename over target; catch rename of any file in dir to target
	if filepath.Base(ev.Name) == filepath.Base(target) {
		return true
	}
	return false
}

func hashOfFile(f *os.File) (string, error) {
	h := sha256.New()
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil // short ETag
}

// --- SSE hub ---

type sseHub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newSSEHub() *sseHub { return &sseHub{clients: map[chan []byte]struct{}{}} }
func (h *sseHub) run()   {}
func (h *sseHub) register() chan []byte {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}
func (h *sseHub) unregister(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}
func (h *sseHub) broadcast(msg []byte) {
	h.mu.Lock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
	h.mu.Unlock()
}

// --- Static file serving for React app ---
