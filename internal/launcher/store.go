package launcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
)

type Store struct {
	mu      sync.Mutex
	library string
	apps    map[string]App
	imports map[string]appImport
}

// Library returns the application library directory.
func (s *Store) Library() string { return s.library }

func NewStore(library string) *Store { return &Store{library: library, apps: map[string]App{}} }

type usage struct {
	App      string     `json:"app"`
	LastOpen *time.Time `json:"lastOpen"`
}
type state struct {
	LastUsed map[string]usage `json:"lastUsed"`
}

func (s *Store) readState() state {
	result := state{LastUsed: map[string]usage{}}
	data, err := os.ReadFile(filepath.Join(s.library, "state.json"))
	if err == nil {
		if json.Unmarshal(data, &result) != nil {
			result.LastUsed = map[string]usage{}
		}
	}
	if result.LastUsed == nil {
		result.LastUsed = map[string]usage{}
	}
	return result
}

func (s *Store) Scan() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := Snapshot{Library: s.library, Apps: []App{}, Warnings: []string{}}
	entries, err := os.ReadDir(s.library)
	if err != nil {
		return result, fmt.Errorf("无法读取应用库：%w", err)
	}
	recent := s.readState()
	next := map[string]App{}
	for _, item := range entries {
		if !item.IsDir() || item.Type()&os.ModeSymlink != 0 || strings.HasPrefix(strings.ToLower(item.Name()), ".app-runner-new-") {
			continue
		}
		directory := filepath.Join(s.library, item.Name())
		profilePath := filepath.Join(directory, "app.toml")
		content := filepath.Join(directory, "content")
		if st, e := os.Lstat(profilePath); e != nil || !st.Mode().IsRegular() {
			continue
		}
		st, e := os.Lstat(content)
		if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			continue
		}
		p := Profile{RelatedLinks: []Link{}, Updater: []string{}}
		raw, e := os.ReadFile(profilePath)
		if e == nil {
			e = toml.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\ufeff")), &p)
		}
		profileError := ""
		if e != nil {
			profileError = e.Error()
			result.Warnings = append(result.Warnings, profilePath+": "+e.Error())
			p = Profile{RelatedLinks: []Link{}, Updater: []string{}}
		}
		id, parseErr := uuid.Parse(p.UUID)
		_, duplicate := next[id.String()]
		if parseErr != nil || id == uuid.Nil || duplicate {
			id = uuid.New()
			if e == nil {
				updated, patchErr := insertUUID(raw, id.String())
				if patchErr == nil {
					patchErr = atomicWrite(profilePath, updated)
				}
				if patchErr != nil {
					result.Warnings = append(result.Warnings, "无法写入 UUID："+profilePath+": "+patchErr.Error())
					id = uuid.NewSHA1(uuid.NameSpaceURL, []byte(directory))
				}
			} else {
				id = uuid.NewSHA1(uuid.NameSpaceURL, []byte(directory))
			}
		}
		p.UUID = id.String()
		if strings.TrimSpace(p.Title) == "" {
			p.Title = item.Name()
		}
		if p.RelatedLinks == nil {
			p.RelatedLinks = []Link{}
		}
		if p.Updater == nil {
			p.Updater = []string{}
		}
		modified := st.ModTime()
		a := App{Profile: p, Directory: directory, Modified: &modified, ProfileError: profileError}
		for key, value := range recent.LastUsed {
			if strings.EqualFold(key, a.UUID) {
				a.LastUsed = value.LastOpen
				break
			}
			if a.LastUsed == nil && (strings.EqualFold(key, p.Title) || strings.EqualFold(key, item.Name()) || strings.EqualFold(value.App, p.Title)) {
				a.LastUsed = value.LastOpen
			}
		}
		refreshMetadata(&a)
		next[a.UUID] = a
		result.Apps = append(result.Apps, a)
	}
	sort.Slice(result.Apps, func(i, j int) bool {
		return strings.ToLower(result.Apps[i].Title) < strings.ToLower(result.Apps[j].Title)
	})
	s.apps = next
	return result, nil
}

// Locate the actual UUID assignment via the TOML parser: comments, unknown fields,
// nested uuid keys, and multiline strings must survive identity insertion verbatim.
func insertUUID(raw []byte, id string) ([]byte, error) {
	text := strings.TrimPrefix(string(raw), "\ufeff")
	var values map[string]any
	if err := toml.Unmarshal([]byte(text), &values); err != nil {
		return nil, err
	}
	line := "uuid = \"" + id + "\""
	if _, exists := values["uuid"]; exists {
		pattern := regexp.MustCompile(`(?m)^[ \t]*(?:uuid|"uuid"|'uuid')[ \t]*=`)
		for _, match := range pattern.FindAllStringIndex(text, -1) {
			end := strings.IndexByte(text[match[1]:], '\n')
			if end < 0 {
				end = len(text)
			} else {
				end += match[1]
			}
			candidate := text[:match[0]] + line + text[end:]
			var check map[string]any
			if toml.Unmarshal([]byte(candidate), &check) == nil && check["uuid"] == id {
				return []byte(candidate), nil
			}
		}
		return nil, errors.New("无法安全替换 UUID 字段")
	}
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	return []byte(line + newline + text), nil
}

func (s *Store) Get(id string) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[id]
	if !ok {
		return a, errors.New("应用不存在，请刷新列表")
	}
	// Never let caller-owned slices modify the stored profile before a save succeeds.
	a.RelatedLinks = append([]Link{}, a.RelatedLinks...)
	a.Updater = append([]string{}, a.Updater...)
	return a, nil
}

func (s *Store) Save(id string, p Profile) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[id]
	if !ok {
		return a, errors.New("应用不存在")
	}
	p.UUID = id
	p.Category = normalizeCategory(p.Category)
	p.Title = strings.TrimSpace(p.Title)
	p.Entry = strings.TrimSpace(p.Entry)
	if err := Validate(p); err != nil {
		return a, err
	}
	path := filepath.Join(a.Directory, "app.toml")
	values := map[string]any{}
	if raw, e := os.ReadFile(path); e == nil {
		_ = toml.Unmarshal([]byte(strings.TrimPrefix(string(raw), "\ufeff")), &values)
	}
	// Saving repairs malformed profiles and keeps parseable unknown keys.
	if values == nil {
		values = map[string]any{}
	}
	values["uuid"] = id
	values["category"] = p.Category
	values["description"] = p.Description
	values["title"] = p.Title
	values["type"] = p.Type
	values["entry"] = p.Entry
	values["run_as_admin"] = p.RunAsAdmin
	values["related_links"] = p.RelatedLinks
	values["updater"] = p.Updater
	data, err := toml.Marshal(values)
	if err != nil {
		return a, err
	}
	if err = atomicWrite(path, data); err != nil {
		return a, err
	}
	a.Profile = p
	a.ProfileError = ""
	refreshMetadata(&a)
	s.apps[id] = a
	return a, nil
}

func (s *Store) RecordLaunch(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[id]
	if !ok {
		return errors.New("应用不存在")
	}
	data := s.readState()
	now := time.Now()
	data.LastUsed[id] = usage{App: a.Title, LastOpen: &now}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWrite(filepath.Join(s.library, "state.json"), bytes); err != nil {
		return err
	}
	a.LastUsed = &now
	s.apps[id] = a
	return nil
}

func (s *Store) RefreshApp(id string) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[id]
	if !ok {
		return a, errors.New("应用不存在")
	}
	if st, err := os.Stat(filepath.Join(a.Directory, "content")); err == nil {
		modified := st.ModTime()
		a.Modified = &modified
	}
	refreshMetadata(&a)
	s.apps[id] = a
	return a, nil
}

func refreshMetadata(a *App) {
	refreshIcon(a)
	entry, err := ResolveEntry(*a, a.Entry)
	a.EntryValid = err == nil
	a.Version = ""
	if err == nil {
		a.Version = fileVersion(entry)
	}
}

func atomicWrite(path string, data []byte) error {
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
		if mode&0200 == 0 {
			return errors.New("文件只读：" + path)
		}
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".app-runner-*")
	if err != nil {
		return err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Chmod(temporary, mode); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
