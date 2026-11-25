// Package manifest describes which background videos exist in the media
// bucket and which one the site plays. It is stored as manifest.json at the
// bucket root: r2sync writes it, the frontend (and later the Go server) reads it.
package manifest

import (
	"fmt"
	"sort"
	"time"
)

// Key is the object key of the manifest in the bucket.
const Key = "manifest.json"

// Manifest is the whole manifest.json document.
type Manifest struct {
	// Current names the entry in Videos the site plays.
	Current string            `json:"current"`
	Updated time.Time         `json:"updated"`
	Videos  map[string]*Video `json:"videos"`
}

// Video is one uploaded background: a landscape cut, a portrait cut for
// phones, and a blurred still for the reading pages.
type Video struct {
	Landscape Cut       `json:"landscape"`
	Portrait  Cut       `json:"portrait"`
	Blur      string    `json:"blur"` // object key of the blurred still
	Uploaded  time.Time `json:"uploaded"`
	// PortraitCropped is true when the portrait cut was cropped from the
	// landscape video instead of supplied.
	PortraitCropped bool `json:"portraitCropped,omitempty"`
}

// Cut is one video file and its poster frame. Src and Poster are object keys,
// relative to the bucket's public URL.
type Cut struct {
	Src    string `json:"src"`
	Poster string `json:"poster"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int64  `json:"bytes"`
}

// New returns an empty manifest.
func New() *Manifest {
	return &Manifest{Videos: map[string]*Video{}}
}

// Keys returns every object key the video points at.
func (v *Video) Keys() []string {
	return []string{v.Landscape.Src, v.Landscape.Poster, v.Portrait.Src, v.Portrait.Poster, v.Blur}
}

// Put adds or replaces an entry and makes it current. It reports whether an
// entry with that name already existed.
func (m *Manifest) Put(name string, v *Video, now time.Time) (replaced bool) {
	if m.Videos == nil {
		m.Videos = map[string]*Video{}
	}
	_, replaced = m.Videos[name]
	m.Videos[name] = v
	m.Current = name
	m.Updated = now
	return replaced
}

// Use makes an existing entry current.
func (m *Manifest) Use(name string, now time.Time) error {
	if _, ok := m.Videos[name]; !ok {
		return fmt.Errorf("no video named %q (have: %v)", name, m.Names())
	}
	m.Current = name
	m.Updated = now
	return nil
}

// Remove deletes an entry. The current entry can't be removed.
func (m *Manifest) Remove(name string, now time.Time) error {
	if _, ok := m.Videos[name]; !ok {
		return fmt.Errorf("no video named %q (have: %v)", name, m.Names())
	}
	if name == m.Current {
		return fmt.Errorf("%q is live; switch to another video with `use` first", name)
	}
	delete(m.Videos, name)
	m.Updated = now
	return nil
}

// Referenced returns the set of object keys any entry points at, plus the
// manifest itself. Anything else under the media prefix is an orphan.
func (m *Manifest) Referenced() map[string]bool {
	refs := map[string]bool{Key: true}
	for _, v := range m.Videos {
		for _, k := range v.Keys() {
			if k != "" {
				refs[k] = true
			}
		}
	}
	return refs
}

// Names returns the entry names, newest upload first.
func (m *Manifest) Names() []string {
	names := make([]string, 0, len(m.Videos))
	for n := range m.Videos {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := m.Videos[names[i]], m.Videos[names[j]]
		if !a.Uploaded.Equal(b.Uploaded) {
			return a.Uploaded.After(b.Uploaded)
		}
		return names[i] < names[j]
	})
	return names
}
