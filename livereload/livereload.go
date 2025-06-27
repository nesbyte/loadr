package livereload

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nesbyte/loadr/registry"
)

//go:embed liveReloader.html
var liveReloaderHTML embed.FS

// Allow for custom Data to be injected on load

type clientChan chan string

var (
	liveServerStarted bool = false
	liveServerMu      sync.Mutex
	clientsMu         sync.Mutex
	clientsRegister   = make(map[clientChan]struct{})
)

// Broadcasts a message to all connected clients
func broadcast(msg string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	for ch := range clientsRegister {
		select {
		case ch <- msg:
		default:
		}
	}
}

type ReloadHandler func(fsnotify.Event, error)

var customReloadHandler ReloadHandler

func LiveReloadCustomErrorHandler(err error) {
	customReloadHandler(fsnotify.Event{}, err)
}

func RunLiveReload(handlePattern string, handleReload ReloadHandler, pathsToWatch ...string) (http.HandlerFunc, context.CancelFunc, error) {

	liveServerMu.Lock()
	defer liveServerMu.Unlock()

	if liveServerStarted {
		return nil, nil, errors.New("live reload is already running")
	}
	liveServerStarted = true

	if handlePattern == "" {
		return nil, nil, errors.New("handlePattern can not be empty")
	}

	if handleReload == nil {
		return nil, nil, errors.New("handleChange must be set in order to propagate errors, feel free to use loadr.HandleChange as a helper")
	} else {
		customReloadHandler = handleReload
	}

	bs, err := liveReloaderHTML.ReadFile("liveReloader.html")
	if err != nil {
		return nil, nil, err
	}
	registry.SetJSToInject(bs)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	// Recursively adds directories to the watcher
	err = walkDirsAndAddPaths(watcher, pathsToWatch)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go runWatcher(ctx, watcher, handleReload)

	// Build up the HTTP handler function
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Register the current client
		broadcastChannel := make(clientChan, 1)
		clientsMu.Lock()
		clientsRegister[broadcastChannel] = struct{}{}
		clientsMu.Unlock()

		// Unregister the client
		defer func() {
			clientsMu.Lock()
			delete(clientsRegister, broadcastChannel)
			clientsMu.Unlock()
		}()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Notify the client of the live server start
		w.Write([]byte("data: live server is running\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Listen for events from the broadcast channel, client requests, or context cancellation
		for {
			select {
			case msg := <-broadcastChannel:
				w.Write([]byte(msg))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				return
			case <-ctx.Done():
				return
			}
		}

	})

	// Register live reloading with the validator
	registry.SetLiveReload(true)

	return handlerFunc, cancel, nil
}

// fsnotify does not support recursive directory watching,
// so we need to walk through the directories and add them to the watcher manually.
func walkDirsAndAddPaths(watcher *fsnotify.Watcher, pathsToWatch []string) error {
	for _, path := range pathsToWatch {
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// If it's a directory, add it to the watcher
			if d.IsDir() {
				err := watcher.Add(path)
				if err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil

}

// The runWatcher function listens for file system events, debounces
// them to avoid multiple notifications for the same file change, and
// broadcasts changes to all connected clients
func runWatcher(ctx context.Context, watcher *fsnotify.Watcher, handleChange func(fsnotify.Event, error)) {
	var (
		batchDelay = 100 * time.Millisecond // Delay for batching events
		batchTimer *time.Timer
	)

	defer watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// If the event was to create a folder, we need to add it to the watcher
			// regardless of the timer
			if event.Has(fsnotify.Create) {
				fi, err := os.Stat(event.Name)
				if err != nil {
					if handleChange != nil {
						handleChange(fsnotify.Event{}, err)
					}
					continue
				}

				if fi.IsDir() && len(fi.Name()) != 0 {
					// Ignore dotfiles
					if (len(event.Name) > 0) && (event.Name[0] != '.') {
						walkDirsAndAddPaths(watcher, []string{event.Name})
					}
				}
			}

			// Avoid multiple notifications for the same file change
			if batchTimer != nil {
				batchTimer.Stop()
			}

			batchTimer = time.AfterFunc(batchDelay, func() {
				if handleChange != nil {
					handleChange(event, nil)
				}

				// Trigger a reload event
				broadcast("data: reload\n\n")
			})
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			if handleChange != nil {
				handleChange(fsnotify.Event{}, err)
			}

			// Trigger a reload event
			broadcast(fmt.Sprintf("data: live reload error: %s\n\n", err.Error()))
		}
	}
}
