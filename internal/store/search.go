package store

import (
	"encoding/json"

	"github.com/grimalschi/shugo/internal/search"
)

const (
	metaSearchIDs     = "search:ids"
	metaSearchBinding = "search:binding:"
	metaSearchDefault = "search:default:"
)

// SearchResultShape uses a nonempty binding ID as its key. The legacy
// catalog uses the empty key in the same reserved shape.
const SearchResultShape = CatalogResultShape

// SearchBinding identifies the index snapshot for one published search.
type SearchBinding struct {
	Definition search.Definition `json:"definition"`
	Snapshot   string            `json:"snapshot"`
}

func (r *reader) Search(id string) (binding SearchBinding, found bool, err error) {
	raw, found, err := r.blob(r.meta, metaSearchBinding+id)
	if err == nil && found {
		err = json.Unmarshal(raw, &binding)
	}
	return
}

func (u *Update) SearchIDs() (ids []string, err error) {
	raw, found, err := u.blob(u.meta, metaSearchIDs)
	if err == nil && found {
		err = json.Unmarshal(raw, &ids)
	}
	return
}

func (u *Update) SetSearchIDs(ids []string) error {
	raw, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	_, err = u.write.Exec(put, metaSearchIDs, raw)
	return err
}

func (u *Update) SearchDefault(id string) ([]byte, error) {
	raw, _, err := u.blob(u.meta, metaSearchDefault+id)
	return raw, err
}

func (u *Update) SetSearch(binding SearchBinding, result []byte) error {
	raw, err := json.Marshal(binding)
	if err != nil {
		return err
	}
	id := binding.Definition.ID
	if _, err := u.write.Exec(put, metaSearchBinding+id, raw); err != nil {
		return err
	}
	_, err = u.write.Exec(put, metaSearchDefault+id, result)
	return err
}

func (u *Update) RemoveSearch(id string) error {
	_, err := u.write.Exec("DELETE FROM meta WHERE name IN (?, ?)", metaSearchBinding+id, metaSearchDefault+id)
	return err
}

func (u *Update) HasSearchReaders(id string) (has bool, err error) {
	err = u.write.QueryRow("SELECT EXISTS(SELECT 1 FROM reads WHERE shape = ? AND key = ?)", SearchResultShape, []byte(id)).Scan(&has)
	return
}
