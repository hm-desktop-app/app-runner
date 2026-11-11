package launcher

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

func normalizeCategory(category string) string {
	category = strings.TrimSpace(category)
	if category == "未分类" {
		return ""
	}
	return category
}

func validateCategory(category string) error {
	if strings.ContainsAny(category, "\r\n\x00") {
		return errors.New("分类名称不能包含换行或空字符")
	}
	return nil
}

// Patch just the top-level value, preserving comments, notes and unknown fields.
func patchCategory(raw []byte, category string) ([]byte, error) {
	text := bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	encoded, err := toml.Marshal(map[string]string{"category": category})
	if err != nil {
		return nil, err
	}
	value := bytes.TrimSpace(encoded[bytes.IndexByte(encoded, '=')+1:])
	parser := unstable.Parser{}
	parser.Reset(text)
	for parser.NextExpression() {
		node := parser.Expression()
		if node.Kind == unstable.Table || node.Kind == unstable.ArrayTable {
			break
		}
		if node.Kind != unstable.KeyValue {
			continue
		}
		keys := node.Key()
		if !keys.Next() || string(keys.Node().Data) != "category" || keys.Next() {
			continue
		}
		region := node.Value().Raw
		start := len(raw) - len(text) + int(region.Offset)
		end := start + int(region.Length)
		result := append([]byte{}, raw[:start]...)
		result = append(result, value...)
		return append(result, raw[end:]...), nil
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}
	newline := "\n"
	if bytes.Contains(text, []byte("\r\n")) {
		newline = "\r\n"
	}
	prefix := raw[:len(raw)-len(text)]
	result := append([]byte{}, prefix...)
	result = append(result, []byte("category = ")...)
	result = append(result, value...)
	result = append(result, []byte(newline)...)
	return append(result, text...), nil
}

func (s *Store) SetCategory(id, category string) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[id]
	if !ok {
		return a, errors.New("应用不存在，请刷新列表")
	}
	category = normalizeCategory(category)
	if err := validateCategory(category); err != nil {
		return a, err
	}
	path := filepath.Join(a.Directory, "app.toml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return a, err
	}
	var profile Profile
	if err = toml.Unmarshal(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), &profile); err != nil {
		return a, err
	}
	data, err := patchCategory(raw, category)
	if err != nil {
		return a, err
	}
	if err = atomicWrite(path, data); err != nil {
		return a, err
	}
	profile.Category = category
	profile.UUID = a.UUID
	if strings.TrimSpace(profile.Title) == "" {
		profile.Title = filepath.Base(a.Directory)
	}
	if profile.RelatedLinks == nil {
		profile.RelatedLinks = []Link{}
	}
	if profile.Updater == nil {
		profile.Updater = []string{}
	}
	a.Profile = profile
	a.ProfileError = ""
	refreshMetadata(&a)
	s.apps[id] = a
	return a, nil
}

type categoryChange struct {
	app     App
	profile Profile
	path    string
	before  []byte
	after   []byte
}

func (s *Store) RenameCategory(oldName, newName string) ([]App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldName, newName = normalizeCategory(oldName), normalizeCategory(newName)
	if oldName == "" || newName == "" {
		return nil, errors.New("请填写非空分组名称")
	}
	if err := validateCategory(newName); err != nil {
		return nil, err
	}
	if oldName != newName {
		for _, app := range s.apps {
			if normalizeCategory(app.Category) == newName {
				return nil, errors.New("目标分组已存在，请使用其他名称")
			}
		}
	}
	return s.rewriteCategory(oldName, newName)
}

func (s *Store) DeleteCategory(name string) ([]App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = normalizeCategory(name)
	if name == "" {
		return nil, errors.New("未分类是默认分组")
	}
	return s.rewriteCategory(name, "")
}

// Prepare every affected profile before changing files; roll back completed writes
// on a commit error. Empty groups live in frontend preferences and return no apps.
func (s *Store) rewriteCategory(oldName, newName string) ([]App, error) {
	members := []App{}
	for _, app := range s.apps {
		if normalizeCategory(app.Category) == oldName {
			members = append(members, app)
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Directory < members[j].Directory })
	changes := []categoryChange{}
	for _, app := range members {
		path := filepath.Join(app.Directory, "app.toml")
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0200 == 0 {
			return nil, fmt.Errorf("配置不可写：%s", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var profile Profile
		if err = toml.Unmarshal(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")), &profile); err != nil {
			return nil, fmt.Errorf("%s：%w", path, err)
		}
		if normalizeCategory(profile.Category) != oldName {
			return nil, errors.New("应用分组已在外部更改，请刷新列表后重试")
		}
		data, err := patchCategory(raw, newName)
		if err != nil {
			return nil, err
		}
		changes = append(changes, categoryChange{app: app, profile: profile, path: path, before: raw, after: data})
	}
	for index, change := range changes {
		if err := atomicWrite(change.path, change.after); err != nil {
			rollbackErrors := []error{fmt.Errorf("保存分组失败：%s：%w", change.path, err)}
			for rollback := index - 1; rollback >= 0; rollback-- {
				previous := changes[rollback]
				if restoreErr := atomicWrite(previous.path, previous.before); restoreErr != nil {
					rollbackErrors = append(rollbackErrors, fmt.Errorf("还原失败，请刷新检查：%s：%w", previous.path, restoreErr))
				}
			}
			return nil, errors.Join(rollbackErrors...)
		}
	}
	result := []App{}
	for _, change := range changes {
		app, profile := change.app, change.profile
		profile.Category = newName
		profile.UUID = app.UUID
		if strings.TrimSpace(profile.Title) == "" {
			profile.Title = filepath.Base(app.Directory)
		}
		if profile.RelatedLinks == nil {
			profile.RelatedLinks = []Link{}
		}
		if profile.Updater == nil {
			profile.Updater = []string{}
		}
		app.Profile = profile
		app.ProfileError = ""
		refreshMetadata(&app)
		s.apps[app.UUID] = app
		result = append(result, app)
	}
	return result, nil
}
