package evaluate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/search"
	"github.com/grimalschi/shugo/internal/store"
)

// searchPreparation keeps the indices and default results for one evaluation.
// A temporary snapshot is published only after the journal check succeeds.
type searchPreparation struct {
	dir, path string
	source    *data.DB
	previous  []string
	indices   map[string]*searchSnapshot
	bindings  map[string]searchValue
	changed   []string
	built     int
}

type searchSnapshot struct {
	name, stage string
	index       *search.Index
}

type searchValue struct {
	binding store.SearchBinding
	result  []byte
	dirty   bool
}

func prepareSearch(p *project.Project, u *store.Update, source *data.DB, changes []data.Change, whole bool) (*searchPreparation, error) {
	ids, err := u.SearchIDs()
	if err != nil {
		return nil, err
	}
	s := &searchPreparation{
		dir: p.Path(project.Output), path: p.Data(), source: source, previous: ids,
		indices: map[string]*searchSnapshot{}, bindings: map[string]searchValue{},
	}
	if whole {
		return s, nil
	}

	changed := map[string][]data.Change{}
	for _, change := range changes {
		changed[change.Table.Name] = append(changed[change.Table.Name], change)
	}
	// Resolve all bindings before rebuilding so two bindings of one index
	// use the same decision and the same source snapshot.
	stale := map[string]bool{}
	for _, id := range ids {
		binding, found, err := u.Search(id)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("search %s has no published binding", id)
		}
		result, err := u.SearchDefault(id)
		if err != nil {
			return nil, err
		}
		def := binding.Definition
		s.bindings[id] = searchValue{binding: binding, result: result}
		if snapshot := s.indices[def.IndexID]; snapshot != nil {
			stale[def.IndexID] = stale[def.IndexID] || snapshot.name != binding.Snapshot
			continue
		}
		s.indices[def.IndexID] = &searchSnapshot{name: binding.Snapshot}
		table := source.TableFold(def.Table)
		if table == nil {
			return nil, fmt.Errorf("search table %q is absent from the data journal", def.Table)
		}
		stale[def.IndexID] = !searchSnapshotExists(s.dir, binding.Snapshot) || searchAffected(def, table, changed[table.Name])
	}
	for _, id := range ids {
		old := s.bindings[id]
		def := old.binding.Definition
		if !stale[def.IndexID] && len(old.result) != 0 {
			continue
		}
		snapshot := s.indices[def.IndexID]
		if stale[def.IndexID] && snapshot.stage == "" {
			snapshot.name = ""
		}
		value, err := s.defaultResult(def)
		if err != nil {
			s.discard()
			return nil, err
		}
		s.bindings[id] = value
		if !bytes.Equal(old.result, value.result) {
			s.changed = append(s.changed, id)
		}
	}
	return s, nil
}

func searchSnapshotExists(dir, name string) bool {
	if !filepath.IsLocal(name) || filepath.Base(name) != name {
		return false
	}
	info, err := os.Lstat(filepath.Join(dir, name))
	return err == nil && info.Mode().IsRegular()
}

func searchAffected(def search.Definition, table *data.Table, changes []data.Change) bool {
	if len(changes) == 0 {
		return false
	}
	columns := []int{table.Key}
	for _, field := range def.Columns() {
		i := slices.IndexFunc(table.Columns, func(name string) bool {
			return len(name) == len(field) && strings.EqualFold(name, field)
		})
		// SQLite computes generated columns from the journaled columns of
		// this row. Their dependencies are not recorded separately.
		if i < 0 {
			return true
		}
		columns = append(columns, i)
	}
	for _, change := range changes {
		for _, i := range columns {
			if change.Differs(i) {
				return true
			}
		}
	}
	return false
}

func (s *searchPreparation) request(body []byte) ([]byte, error) {
	var declaration search.Declaration
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return nil, err
	}
	def, err := search.Normalize(declaration)
	if err != nil {
		return nil, err
	}
	value, found := s.bindings[def.ID]
	if !found {
		if s.source.TableFold(def.Table) == nil {
			return nil, fmt.Errorf("search table %q is absent from the data database", def.Table)
		}
		value, err = s.defaultResult(def)
		if err != nil {
			return nil, err
		}
		s.bindings[def.ID] = value
	}
	return json.Marshal(struct {
		ID     string          `json:"id"`
		Result json.RawMessage `json:"result"`
	}{def.ID, value.result})
}

func (s *searchPreparation) defaultResult(def search.Definition) (searchValue, error) {
	ctx := context.Background()
	snapshot := s.indices[def.IndexID]
	if snapshot == nil {
		snapshot = &searchSnapshot{}
		s.indices[def.IndexID] = snapshot
	}
	if snapshot.name == "" {
		file, err := os.CreateTemp(s.dir, ".search-*.tmp")
		if err != nil {
			return searchValue{}, err
		}
		path := file.Name()
		defer func() {
			if snapshot.stage != path {
				os.Remove(path)
			}
		}()
		if err := file.Close(); err != nil {
			return searchValue{}, err
		}
		if err := search.Build(ctx, s.path, path, def); err != nil {
			return searchValue{}, fmt.Errorf("build search index for %q: %w", def.Table, err)
		}
		snapshot.name, err = searchSnapshotName(path)
		if err != nil {
			return searchValue{}, err
		}
		snapshot.stage = path
		s.built++
	}
	if snapshot.index == nil {
		path := snapshot.stage
		if path == "" {
			path = filepath.Join(s.dir, snapshot.name)
		}
		index, err := search.Open(path)
		if err != nil {
			return searchValue{}, err
		}
		snapshot.index = index
	}
	result, err := snapshot.index.Search(ctx, def, nil)
	if err != nil {
		return searchValue{}, fmt.Errorf("initial search for %q: %w", def.Table, err)
	}
	raw, err := json.Marshal(result)
	return searchValue{
		binding: store.SearchBinding{Definition: def, Snapshot: snapshot.name}, result: raw, dirty: true,
	}, err
}

func searchSnapshotName(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("search-%x.sqlite", hash.Sum(nil)[:16]), nil
}

func (s *searchPreparation) discard() {
	for _, snapshot := range s.indices {
		if snapshot.index != nil {
			snapshot.index.Close()
		}
		if snapshot.stage != "" {
			os.Remove(snapshot.stage)
		}
	}
}

func (s *searchPreparation) publish(u *store.Update) error {
	ids := make([]string, 0, len(s.bindings))
	published := false
	for id, value := range s.bindings {
		hasReaders, err := u.HasSearchReaders(id)
		if err != nil {
			return err
		}
		if !hasReaders {
			continue
		}
		ids = append(ids, id)
		if !value.dirty {
			continue
		}
		snapshot := s.indices[value.binding.Definition.IndexID]
		if snapshot.stage != "" {
			if err := os.Rename(snapshot.stage, filepath.Join(s.dir, snapshot.name)); err != nil {
				return err
			}
			snapshot.stage = ""
			published = true
		}
		if err := u.SetSearch(value.binding, value.result); err != nil {
			return err
		}
	}
	if published {
		dir, err := os.Open(s.dir)
		if err != nil {
			return err
		}
		defer dir.Close()
		if err := dir.Sync(); err != nil {
			return err
		}
	}
	slices.Sort(ids)
	for _, id := range s.previous {
		if _, found := slices.BinarySearch(ids, id); !found {
			if err := u.RemoveSearch(id); err != nil {
				return err
			}
		}
	}
	if !slices.Equal(ids, s.previous) {
		return u.SetSearchIDs(ids)
	}
	return nil
}

func clearSearch(u *store.Update) error {
	ids, err := u.SearchIDs()
	if err != nil || len(ids) == 0 {
		return err
	}
	for _, id := range ids {
		if err := u.RemoveSearch(id); err != nil {
			return err
		}
	}
	return u.SetSearchIDs(nil)
}
