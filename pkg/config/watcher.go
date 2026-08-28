package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type ConfigWatcher struct {
	watcher    *fsnotify.Watcher
	configPath string
	callback   func(*TOMLConfig)
	watched    map[string]bool
	mu         sync.Mutex
	done       chan struct{}
}

func NewConfigWatcher() (*ConfigWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	return &ConfigWatcher{
		watcher: w,
		watched: make(map[string]bool),
		done:    make(chan struct{}),
	}, nil
}

func (cw *ConfigWatcher) Start(path string, callback func(*TOMLConfig)) {
	cw.configPath = path
	cw.callback = callback

	cw.updateWatchedFiles()

	go cw.loop()
}

func (cw *ConfigWatcher) Stop() {
	close(cw.done)
	cw.watcher.Close()
}

func (cw *ConfigWatcher) loop() {
	var debounceTimer *time.Timer

	for {
		select {
		case <-cw.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
				cw.reload()
			})

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("[axctl-config] Watcher error: %v\n", err)
		}
	}
}

func (cw *ConfigWatcher) reload() {
	cfg, err := LoadConfig(cw.configPath)
	if err != nil {
		fmt.Printf("[axctl-config] Error reloading config: %v\n", err)
		return
	}

	fmt.Printf("[axctl-config] Config reloaded from %s\n", cw.configPath)

	cw.updateWatchedFiles()

	if cw.callback != nil {
		cw.callback(cfg)
	}
}

func (cw *ConfigWatcher) updateWatchedFiles() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	paths := ResolveIncludePaths(cw.configPath)
	newWatched := make(map[string]bool)

	for _, p := range paths {
		newWatched[p] = true
		if !cw.watched[p] {
			if err := cw.watcher.Add(p); err != nil {
				fmt.Printf("[axctl-config] Warning: cannot watch %s: %v\n", p, err)
			}
		}
	}

	for p := range cw.watched {
		if !newWatched[p] {
			cw.watcher.Remove(p)
		}
	}

	cw.watched = newWatched
}
