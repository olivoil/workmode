package backend

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"
)

// WatchMsg is sent when a watched file changes.
type WatchMsg struct {
	// Path is the file that changed.
	Path string
	// Kind distinguishes history vs log updates.
	Kind WatchKind
}

// WatchKind identifies the type of file change.
type WatchKind int

const (
	WatchHistory WatchKind = iota
	WatchLog
)

// Sender can receive messages (matches *tea.Program).
type Sender interface {
	Send(msg tea.Msg)
}

// Watcher monitors history.jsonl and log files via fsnotify.
type Watcher struct {
	w       *fsnotify.Watcher
	sender  Sender
	client  *Client
	mu      sync.Mutex
	logFile string // currently watched log file (if any)
}

// NewWatcher creates a file watcher for history.jsonl.
func NewWatcher(client *Client, sender Sender) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		w:      fw,
		sender: sender,
		client: client,
	}

	// Watch the directory containing history.jsonl (to catch creates + writes).
	dir := filepath.Dir(client.HistoryPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fw.Close()
		return nil, err
	}
	if err := fw.Add(dir); err != nil {
		fw.Close()
		return nil, err
	}

	go watcher.loop()
	return watcher, nil
}

// WatchLog starts watching a specific log file for live tail.
func (w *Watcher) WatchLog(shortID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Remove old log watch.
	if w.logFile != "" {
		_ = w.w.Remove(filepath.Dir(w.logFile))
		w.logFile = ""
	}

	if shortID == "" {
		return
	}

	path := w.client.LogPath(shortID)
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	if err := w.w.Add(dir); err != nil {
		log.Printf("watcher: watch log dir %s: %v", dir, err)
		return
	}
	w.logFile = path
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	return w.w.Close()
}

func (w *Watcher) loop() {
	historyFile := filepath.Base(w.client.HistoryPath())

	for {
		select {
		case event, ok := <-w.w.Events:
			if !ok {
				return
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			base := filepath.Base(event.Name)
			if base == historyFile {
				w.sender.Send(WatchMsg{Path: event.Name, Kind: WatchHistory})
				continue
			}

			w.mu.Lock()
			isLog := w.logFile != "" && event.Name == w.logFile
			w.mu.Unlock()
			if isLog {
				w.sender.Send(WatchMsg{Path: event.Name, Kind: WatchLog})
			}

		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}
