package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T, root, name, profile string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	write(t, filepath.Join(dir, "app.toml"), profile)
	write(t, filepath.Join(dir, "content", "app.exe"), "original")
	return dir
}
func read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
func scan(t *testing.T, s *Store) Snapshot {
	t.Helper()
	data, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestScanIdentityAndMalformedProfiles(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)
	original := "# retained comment\r\ntitle = '编辑器'\r\ntype = 'GUI'\r\nentry = 'app.exe'\r\nunknown = 123\r\n"
	directory := fixture(t, root, "editor", original)
	broken := fixture(t, root, "broken", "title = [")
	fixture(t, root, "empty", "")
	write(t, filepath.Join(root, "not-an-app", "app.toml"), "")
	first := scan(t, s)
	if len(first.Apps) != 3 || len(first.Warnings) != 1 {
		t.Fatalf("unexpected scan: %+v", first)
	}
	ids := map[string]string{}
	for _, a := range first.Apps {
		id, err := uuid.Parse(a.UUID)
		if err != nil {
			t.Fatal(err)
		}
		if a.ProfileError == "" && id.Version() != 4 {
			t.Fatal("new UUID must be v4")
		}
		ids[a.Directory] = a.UUID
	}
	raw := read(t, filepath.Join(directory, "app.toml"))
	if !strings.HasSuffix(raw, original) {
		t.Fatal("scan modified profile contents")
	}
	if read(t, filepath.Join(broken, "app.toml")) != "title = [" {
		t.Fatal("scan rewrote malformed profile")
	}
	second := scan(t, s)
	for _, a := range second.Apps {
		if ids[a.Directory] != a.UUID {
			t.Fatal("identity changed on rescan")
		}
	}
	renamed := filepath.Join(root, "renamed")
	if err := os.Rename(directory, renamed); err != nil {
		t.Fatal(err)
	}
	third := scan(t, s)
	for _, a := range third.Apps {
		if a.Directory == renamed && a.UUID != ids[directory] {
			t.Fatal("rename changed UUID")
		}
	}
}

func TestDuplicateIdentityAndUUIDInMultilineString(t *testing.T) {
	root := t.TempDir()
	id := uuid.NewString()
	fixture(t, root, "a", "uuid = '"+id+"'\ntitle = 'A'")
	fixture(t, root, "b", "uuid = '"+id+"'\ntitle = 'B'")
	data := scan(t, NewStore(root))
	if data.Apps[0].UUID == data.Apps[1].UUID {
		t.Fatal("duplicate identities")
	}
	original := "notes = '''\nuuid = 'inside string'\n'''\nuuid = 'bad'\n[nested]\nuuid = 'keep'\n"
	fixed, err := insertUUID([]byte(original), id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fixed), "uuid = 'inside string'") || !strings.Contains(string(fixed), "uuid = 'keep'") {
		t.Fatal("modified nested/string UUID")
	}
	var values map[string]any
	if err = toml.Unmarshal(fixed, &values); err != nil {
		t.Fatal(err)
	}
	if values["uuid"] != id {
		t.Fatal("UUID not fixed")
	}
}

func TestSaveAndUsageCompatibility(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "app", "title = 'Before'\ntype = 'GUI'\nentry = 'app.exe'\ncustom = 'keep'\n")
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	when := time.Now().Add(-time.Hour).Truncate(time.Second)
	raw, _ := json.Marshal(state{LastUsed: map[string]usage{"Before": {App: "Before", LastOpen: &when}}})
	write(t, filepath.Join(root, "state.json"), string(raw))
	a = scan(t, s).Apps[0]
	if a.LastUsed == nil || !a.LastUsed.Equal(when) {
		t.Fatal("legacy usage not loaded")
	}
	p := a.Profile
	p.Title = "After"
	p.RelatedLinks = []Link{{Title: "主页", URL: "example.com"}}
	p.Updater = []string{"PIN:settings", "RMC", "UNPKG"}
	saved, err := s.Save(a.UUID, p)
	if err != nil {
		t.Fatal(err)
	}
	if saved.UUID != a.UUID || saved.Title != "After" {
		t.Fatal("save mismatch")
	}
	var profile map[string]any
	if err = toml.Unmarshal([]byte(read(t, filepath.Join(a.Directory, "app.toml"))), &profile); err != nil {
		t.Fatal(err)
	}
	if profile["custom"] != "keep" {
		t.Fatal("unknown key lost")
	}
	if err = s.RecordLaunch(a.UUID); err != nil {
		t.Fatal(err)
	}
	current := scan(t, s).Apps[0]
	if current.LastUsed == nil || current.LastUsed.Before(when) {
		t.Fatal("usage not persisted")
	}
	p.Title = ""
	before := read(t, filepath.Join(a.Directory, "app.toml"))
	if _, err = s.Save(a.UUID, p); err == nil {
		t.Fatal("invalid save accepted")
	}
	if read(t, filepath.Join(a.Directory, "app.toml")) != before {
		t.Fatal("failed save touched profile")
	}
	write(t, filepath.Join(root, "state.json"), "invalid json")
	if scan(t, s).Apps[0].LastUsed != nil {
		t.Fatal("corrupt usage not ignored")
	}
}

func TestEntryAndOptions(t *testing.T) {
	root := t.TempDir()
	directory := fixture(t, root, "app", "title = 'App'")
	a := App{Directory: directory}
	write(t, filepath.Join(directory, "content", "nested", "程序.exe"), "")
	for _, entry := range []string{"app.exe", `nested\程序.exe`, `re:^程序\.exe$`} {
		if _, err := ResolveEntry(a, entry); err != nil {
			t.Fatalf("%s: %v", entry, err)
		}
	}
	for _, entry := range []string{"", "missing.exe", "../app.toml", `C:\Windows\notepad.exe`, "re:[", "re:", "nested"} {
		if _, err := ResolveEntry(a, entry); err == nil {
			t.Fatalf("accepted entry: %s", entry)
		}
	}
	for _, args := range [][]string{{"--app-dir", root}, {"--app-dir=" + root}} {
		option, err := ParseOptions(args, "ignored")
		if err != nil || option.Library != root {
			t.Fatalf("options: %+v %v", option, err)
		}
	}
	if _, err := ParseOptions([]string{"--app-dir"}, root); err == nil {
		t.Fatal("missing switch value accepted")
	}
	if _, err := ParseOptions(nil, ""); err == nil {
		t.Fatal("missing library accepted")
	}
}

func TestReadOnlyProfileAndRepair(t *testing.T) {
	root := t.TempDir()
	directory := fixture(t, root, "readonly", "title = 'Readonly'")
	path := filepath.Join(directory, "app.toml")
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0644)
	s := NewStore(root)
	data := scan(t, s)
	if len(data.Apps) != 1 || len(data.Warnings) != 1 {
		t.Fatal("read-only profile should be listed with warning")
	}
	if read(t, path) != "title = 'Readonly'" {
		t.Fatal("read-only profile changed")
	}
	if scan(t, s).Apps[0].UUID != data.Apps[0].UUID {
		t.Fatal("unwritten identity unstable")
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	write(t, path, "bad = [")
	a := scan(t, s).Apps[0]
	p := a.Profile
	p.Title = "Repaired"
	p.Type = "GUI"
	p.Entry = "app.exe"
	if _, err := s.Save(a.UUID, p); err != nil {
		t.Fatal(err)
	}
	if scan(t, s).Apps[0].ProfileError != "" {
		t.Fatal("repair failed")
	}
}
