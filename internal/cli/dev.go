package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/server"
)

// dev brings the site up to date the way `shugo update` does, shows it, and
// brings it up to date again whenever a file of the project changes: a
// change of the project compiles it again and computes every page, a change
// of the data computes only what read the rows that changed. Pages open in
// a browser reload themselves.
func (c *command) dev() error {
	st, err := c.open()
	if err != nil {
		return err
	}
	defer st.Close()
	// SQLite removes the files it keeps beside a database when its last
	// connection closes, and makes them again for the next one. Every update
	// would then look like a change of the data and set off another. One
	// connection held open keeps the files where they are; it also tells
	// whether a write to the database has changed the data at all.
	var held *data.DB
	if path := c.project.Data(); path != "" {
		if held, err = data.Open(path); err == nil {
			defer held.Close()
		}
	}
	live := &server.Live{}
	pages := server.NewPages(st)
	pages.Mounts = map[string]string{"body": server.LiveScript}

	// A mistake in the project is not the end of development: it is
	// printed, and the last good site stays up until the next change.
	renew := func() {
		if err := c.renew(st); err != nil {
			c.report(err)
			return
		}
		live.Reload()
	}
	renew()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	if err := watch(watcher, c.project.Root); err != nil {
		return err
	}
	go func() {
		// Editors write a file in several steps; wait until they are done.
		var settle <-chan time.Time
		files := false
		for {
			select {
			case event, open := <-watcher.Events:
				if !open {
					return
				}
				if ignored(event.Name) || c.writesTo(event.Name) {
					continue
				}
				if event.Has(fsnotify.Create) {
					watch(watcher, event.Name) // a new folder
				}
				files = files || !c.isData(event.Name)
				settle = time.After(80 * time.Millisecond)
			case <-settle:
				settle = nil
				// Shugo writes to the data database itself: it makes an
				// index, it drops the lines of the journal it has read.
				// Only what the journal has not told yet is a change.
				if files || c.moved(held) {
					renew()
				}
				files = false
			case err, open := <-watcher.Errors:
				if !open {
					return
				}
				fmt.Fprintln(c.stderr, "error: watching files:", err)
			}
		}
	}()
	return c.listen(&server.Server{Pages: pages, Live: live})
}

// moved tells whether the data database holds changes that the site does
// not show yet.
func (c *command) moved(held *data.DB) bool {
	if held == nil {
		return true
	}
	moved, err := held.Moved(c.position)
	return moved || err != nil
}

// isData tells whether a file is the data database of the project, or one
// of the files SQLite keeps beside it.
func (c *command) isData(path string) bool {
	database := c.project.Data()
	if database == "" {
		return false
	}
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		if path == database+suffix {
			return true
		}
	}
	return false
}

// writesTo tells whether the command itself writes to the file: its output
// may be sent to a file inside the project, and every build would then set
// off the next one.
func (c *command) writesTo(path string) bool {
	changed, err := os.Stat(path)
	if err != nil {
		return false
	}
	for _, output := range []io.Writer{c.stdout, c.stderr} {
		if file, ok := output.(*os.File); ok {
			if info, err := file.Stat(); err == nil && os.SameFile(changed, info) {
				return true
			}
		}
	}
	return false
}

// watch watches a folder and every folder inside it.
func watch(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil // a file, or something that vanished meanwhile
		}
		if path != root && ignored(path) {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

// ignored tells whether a path is not a source of the project: what the
// framework writes itself, installed packages and hidden files.
func ignored(path string) bool {
	name := filepath.Base(path)
	return name == project.Output || name == "node_modules" || strings.HasPrefix(name, ".") ||
		strings.Contains(filepath.ToSlash(path), "/"+project.Output+"/")
}
