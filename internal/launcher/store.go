package launcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
)

// 应用库扫描的并发度：条目之间互相独立，耗时由文件 I/O 主导，
// 8 路足以摊平每次打开文件的等待，又不会把杀毒软件的文件扫描压得太挤。
const scanWorkers = 8

type Store struct {
	mu        sync.Mutex
	library   string
	apps      map[string]App
	imports   map[string]appImport
	preloaded atomic.Pointer[Snapshot]
	scanned   atomic.Bool
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

// Preload 在窗口初始化期间先扫描一次，首个 Scan 直接复用结果，
// 让应用库的读取与 WebView2 的启动重叠而不是串行等待。
func (s *Store) Preload() {
	snapshot, err := s.scan(false)
	if err != nil || s.scanned.Load() {
		return
	}
	s.preloaded.Store(&snapshot)
}

func (s *Store) Scan() (Snapshot, error) {
	if ready := s.preloaded.Swap(nil); ready != nil {
		s.scanned.Store(true)
		return *ready, nil
	}
	return s.scan(true)
}

type scanCandidate struct {
	directory    string
	name         string
	profile      Profile
	profileError string
	warning      string
	raw          []byte
	modified     time.Time
	entryValid   bool
	version      string
	iconModTime  int64
	iconSize     int64
	hasIcon      bool
}

func (s *Store) scan(mark bool) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := Snapshot{Library: s.library, Apps: []App{}, Warnings: []string{}}
	entries, err := os.ReadDir(s.library)
	if err != nil {
		return result, fmt.Errorf("无法读取应用库：%w", err)
	}
	recent := s.readState()
	s.reclaimStaging(entries)

	// 每个应用的元数据互不依赖，先并行读取，串行阶段再分配身份与使用记录。
	candidates := make([]scanCandidate, len(entries))
	valid := make([]bool, len(entries))
	limit := make(chan struct{}, scanWorkers)
	var wg sync.WaitGroup
	for index, item := range entries {
		if !item.IsDir() || item.Type()&os.ModeSymlink != 0 || strings.HasPrefix(strings.ToLower(item.Name()), ".app-runner-new-") {
			continue
		}
		wg.Add(1)
		go func(index int, item os.DirEntry) {
			defer wg.Done()
			limit <- struct{}{}
			defer func() { <-limit }()
			if candidate, ok := s.readCandidate(item); ok {
				candidates[index] = candidate
				valid[index] = true
			}
		}(index, item)
	}
	wg.Wait()

	next := map[string]App{}
	for index := range entries {
		if !valid[index] {
			continue
		}
		candidate := candidates[index]
		if candidate.warning != "" {
			result.Warnings = append(result.Warnings, candidate.warning)
		}
		p := candidate.profile
		id, parseErr := uuid.Parse(p.UUID)
		_, duplicate := next[id.String()]
		if parseErr != nil || id == uuid.Nil || duplicate {
			id = uuid.New()
			if candidate.profileError == "" {
				updated, patchErr := insertUUID(candidate.raw, id.String())
				if patchErr == nil {
					patchErr = atomicWrite(filepath.Join(candidate.directory, "app.toml"), updated)
				}
				if patchErr != nil {
					result.Warnings = append(result.Warnings, "无法写入 UUID："+filepath.Join(candidate.directory, "app.toml")+": "+patchErr.Error())
					id = uuid.NewSHA1(uuid.NameSpaceURL, []byte(candidate.directory))
				}
			} else {
				id = uuid.NewSHA1(uuid.NameSpaceURL, []byte(candidate.directory))
			}
		}
		p.UUID = id.String()
		if strings.TrimSpace(p.Title) == "" {
			p.Title = candidate.name
		}
		if p.RelatedLinks == nil {
			p.RelatedLinks = []Link{}
		}
		if p.Updater == nil {
			p.Updater = []string{}
		}
		modified := candidate.modified
		a := App{Profile: p, Directory: candidate.directory, Modified: &modified, ProfileError: candidate.profileError,
			EntryValid: candidate.entryValid, Version: candidate.version}
		if candidate.hasIcon {
			a.IconURL = fmt.Sprintf("%s%s?v=%d-%d", iconPrefix, a.UUID, candidate.iconModTime, candidate.iconSize)
		}
		for key, value := range recent.LastUsed {
			if strings.EqualFold(key, a.UUID) {
				a.LastUsed = value.LastOpen
				break
			}
			if a.LastUsed == nil && (strings.EqualFold(key, p.Title) || strings.EqualFold(key, candidate.name) || strings.EqualFold(value.App, p.Title)) {
				a.LastUsed = value.LastOpen
			}
		}
		next[a.UUID] = a
		result.Apps = append(result.Apps, a)
	}
	sort.Slice(result.Apps, func(i, j int) bool {
		return strings.ToLower(result.Apps[i].Title) < strings.ToLower(result.Apps[j].Title)
	})
	s.apps = next
	if mark {
		s.scanned.Store(true)
		// 真实扫描已经拿到最新结果，之前预热的结果作废。
		s.preloaded.Store(nil)
	}
	return result, nil
}

// readCandidate 读取单个应用目录的配置、图标与入口信息，不修改任何文件。
func (s *Store) readCandidate(item os.DirEntry) (scanCandidate, bool) {
	directory := filepath.Join(s.library, item.Name())
	profilePath := filepath.Join(directory, "app.toml")
	if st, err := os.Lstat(profilePath); err != nil || !st.Mode().IsRegular() {
		return scanCandidate{}, false
	}
	st, err := os.Lstat(filepath.Join(directory, "content"))
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return scanCandidate{}, false
	}
	s.reclaimUpdateStaging(directory)
	candidate := scanCandidate{directory: directory, name: item.Name(), modified: st.ModTime(),
		profile: Profile{RelatedLinks: []Link{}, Updater: []string{}}}
	raw, readErr := os.ReadFile(profilePath)
	if readErr == nil {
		readErr = toml.Unmarshal([]byte(strings.TrimPrefix(string(raw), string(rune(0xfeff)))), &candidate.profile)
	}
	if readErr != nil {
		candidate.profileError = readErr.Error()
		candidate.warning = profilePath + ": " + readErr.Error()
		candidate.profile = Profile{RelatedLinks: []Link{}, Updater: []string{}}
	}
	candidate.raw = raw
	if _, info, iconErr := iconFile(directory); iconErr == nil {
		candidate.hasIcon = true
		candidate.iconModTime = info.ModTime().UnixNano()
		candidate.iconSize = info.Size()
	}
	if entry, resolveErr := ResolveEntry(App{Directory: directory, Profile: candidate.profile}, candidate.profile.Entry); resolveErr == nil {
		candidate.entryValid = true
		candidate.version = fileVersion(entry)
	}
	return candidate, true
}

// Locate the actual UUID assignment via the TOML parser: comments, unknown fields,
// nested uuid keys, and multiline strings must survive identity insertion verbatim.
func insertUUID(raw []byte, id string) ([]byte, error) {
	text := strings.TrimPrefix(string(raw), string(rune(0xfeff)))
	var values map[string]any
	if err := toml.Unmarshal([]byte(text), &values); err != nil {
		return nil, err
	}
	line := "uuid = " + strconv.Quote(id)
	if _, exists := values["uuid"]; exists {
		pattern := regexp.MustCompile(`(?m)^[ \t]*(?:uuid|"uuid"|'uuid')[ \t]*=`)
		for _, match := range pattern.FindAllStringIndex(text, -1) {
			end := strings.IndexByte(text[match[1]:], 10)
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

// setProfileValue replaces a top-level string key without rewriting the rest
// of the document; a missing key is prepended so it stays outside any table.
func setProfileValue(path, key, value string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	bom := ""
	text := string(raw)
	if strings.HasPrefix(text, "\ufeff") {
		bom = "\ufeff"
		text = strings.TrimPrefix(text, "\ufeff")
	}
	line := key + " = " + strconv.Quote(value)
	pattern := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(key) + `[ \t]*=`)
	if loc := pattern.FindStringIndex(text); loc != nil {
		end := strings.IndexByte(text[loc[1]:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += loc[1]
		}
		text = text[:loc[0]] + line + text[end:]
	} else {
		newline := "\n"
		if strings.Contains(text, "\r\n") {
			newline = "\r\n"
		}
		text = line + newline + text
	}
	return atomicWrite(path, []byte(bom+text))
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
	p.LastUpdateAt = a.Profile.LastUpdateAt
	p.Category = normalizeCategory(p.Category)
	p.Title = strings.TrimSpace(p.Title)
	p.Entry = strings.TrimSpace(p.Entry)
	if err := Validate(p); err != nil {
		return a, err
	}
	path := filepath.Join(a.Directory, "app.toml")
	values := map[string]any{}
	if raw, e := os.ReadFile(path); e == nil {
		_ = toml.Unmarshal([]byte(strings.TrimPrefix(string(raw), string(rune(0xfeff)))), &values)
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
