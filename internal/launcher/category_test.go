package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCategoryAndDescriptionRoundTrip(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\ntype='GUI'\nentry='app.exe'\ncustom='keep'\n")
	store := NewStore(root)
	app := scan(t, store).Apps[0]
	if app.Category != "" || app.Description != "" {
		t.Fatal("legacy profile defaults changed")
	}
	profile := app.Profile
	profile.Category = "  开发工具  "
	profile.Description = "第一行备注\n第二行：自由文本 \"引号\"\n"
	saved, err := store.Save(app.UUID, profile)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Category != "开发工具" || saved.Description != profile.Description {
		t.Fatal("metadata save mismatch")
	}
	rescanned := scan(t, store).Apps[0]
	if rescanned.Category != saved.Category || rescanned.Description != saved.Description {
		t.Fatal("metadata did not round-trip")
	}
	if !strings.Contains(read(t, filepath.Join(dir, "app.toml")), "custom = 'keep'") {
		t.Fatal("unknown field lost")
	}
}
func TestMoveCategoryPreservesProfileText(t *testing.T) {
	root := t.TempDir()
	original := "# keep comment\r\ntitle='App'\r\ncategory = '''old\ncategory = 'inside old group'\n''' # group comment\r\ndescription = '''notes\ncategory = 'inside notes'\n'''\r\n[nested]\r\ncategory='nested value'\r\ncustom=123\r\n"
	dir := fixture(t, root, "app", original)
	store := NewStore(root)
	app := scan(t, store).Apps[0]
	path := filepath.Join(dir, "app.toml")
	before := read(t, path)
	saved, err := store.SetCategory(app.UUID, "  图形工具  ")
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.Replace(before, "'''old\ncategory = 'inside old group'\n'''", "'图形工具'", 1)
	after := read(t, path)
	// Both basic and literal TOML strings are acceptable replacements.
	if after != expected && after != strings.Replace(expected, "'图形工具'", "\"图形工具\"", 1) {
		t.Fatalf("other text changed:\n%s", after)
	}
	if saved.Category != "图形工具" || saved.Description != "notes\ncategory = 'inside notes'\n" {
		t.Fatal("category move overwrote notes")
	}
	if saved.Type != "" || saved.Entry != "" {
		t.Fatal("unconfigured application was repaired unexpectedly")
	}
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0644)
	if _, err := store.SetCategory(app.UUID, "只读"); err == nil {
		t.Fatal("read-only move succeeded")
	}
	current, _ := store.Get(app.UUID)
	if current.Category != "图形工具" || read(t, path) != after {
		t.Fatal("failed move altered disk or memory")
	}
}
func TestCategoryPatchInsertionAndValidation(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "\ufeff# comment\r\ntitle='App'\r\ndescription='saved notes'\r\n[nested]\r\ncategory='keep'\r\n")
	store := NewStore(root)
	app := scan(t, store).Apps[0]
	path := filepath.Join(dir, "app.toml")
	before := read(t, path)
	saved, err := store.SetCategory(app.UUID, "引号 \" / 工具")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Category != "引号 \" / 工具" {
		t.Fatal("category text changed")
	}
	after := read(t, path)
	if !strings.Contains(after, strings.TrimPrefix(before, "\ufeff")) {
		t.Fatal("insertion changed original profile text")
	}
	if _, err := store.SetCategory(app.UUID, "bad\nname"); err == nil {
		t.Fatal("multiline category accepted")
	}
	if read(t, path) != after {
		t.Fatal("invalid category modified profile")
	}
	write(t, path, strings.Replace(after, "saved notes", "external notes", 1))
	saved, err = store.SetCategory(app.UUID, "未分类")
	if err != nil || saved.Category != "" || saved.Description != "external notes" {
		t.Fatalf("did not preserve latest profile: %+v %v", saved, err)
	}
	write(t, path, "broken = [")
	if _, err := store.SetCategory(app.UUID, "Tools"); err == nil || read(t, path) != "broken = [" {
		t.Fatal("malformed profile was silently overwritten")
	}
}

func TestRenameAndDeleteCategoryPreserveAllProfiles(t *testing.T) {
	root := t.TempDir()
	first := fixture(t, root, "a", "title='First'\ncategory='开发工具' # retain category comment\ndescription='notes A'\ncustom=123\n")
	second := fixture(t, root, "b", "title='Second'\ncategory=' 开发工具 '\ndescription='notes B'\n")
	other := fixture(t, root, "c", "title='Other'\ncategory='图形工具'\n")
	store := NewStore(root)
	scan(t, store)
	beforeA := read(t, filepath.Join(first, "app.toml"))
	beforeB := read(t, filepath.Join(second, "app.toml"))
	beforeOther := read(t, filepath.Join(other, "app.toml"))
	renamed, err := store.RenameCategory("开发工具", "  编程工具  ")
	if err != nil || len(renamed) != 2 {
		t.Fatalf("rename failed: %v %+v", err, renamed)
	}
	for _, app := range renamed {
		if app.Category != "编程工具" || app.Description == "" {
			t.Fatal("rename lost metadata")
		}
	}
	afterA := read(t, filepath.Join(first, "app.toml"))
	afterB := read(t, filepath.Join(second, "app.toml"))
	if !strings.Contains(afterA, "# retain category comment") || !strings.Contains(afterA, "custom=123") {
		t.Fatal("comments/unknown fields lost")
	}
	if strings.Replace(afterA, "'编程工具'", "'开发工具'", 1) != beforeA {
		t.Fatal("rename changed unrelated text")
	}
	if strings.Replace(afterB, "'编程工具'", "' 开发工具 '", 1) != beforeB {
		t.Fatal("normalized rename changed unrelated text")
	}
	if read(t, filepath.Join(other, "app.toml")) != beforeOther {
		t.Fatal("other group changed")
	}
	if _, err := store.RenameCategory("编程工具", "图形工具"); err == nil {
		t.Fatal("rename into existing group accepted")
	}
	if _, err := store.RenameCategory("", "Named"); err == nil {
		t.Fatal("renamed default group")
	}
	if _, err := store.RenameCategory("编程工具", "bad\nname"); err == nil {
		t.Fatal("invalid rename accepted")
	}
	deleted, err := store.DeleteCategory("编程工具")
	if err != nil || len(deleted) != 2 {
		t.Fatal("delete failed", err)
	}
	for _, app := range deleted {
		if app.Category != "" || app.Description == "" {
			t.Fatal("delete lost app or notes")
		}
		if _, err := os.Stat(filepath.Join(app.Directory, "content", "app.exe")); err != nil {
			t.Fatal("delete removed application files")
		}
	}
	if _, err := store.DeleteCategory(""); err == nil {
		t.Fatal("deleted default group")
	}
	empty, err := store.RenameCategory("Empty", "New empty")
	if err != nil || len(empty) != 0 {
		t.Fatal("empty group rename failed")
	}
}
func TestGroupReadOnlyAndExternalChangeLeaveOtherMembersUnchanged(t *testing.T) {
	root := t.TempDir()
	first := fixture(t, root, "a", "title='First'\ncategory='Tools'\ndescription='keep'\n")
	second := fixture(t, root, "b", "title='Second'\ncategory='Tools'\n")
	store := NewStore(root)
	scan(t, store)
	firstPath, secondPath := filepath.Join(first, "app.toml"), filepath.Join(second, "app.toml")
	beforeFirst, beforeSecond := read(t, firstPath), read(t, secondPath)
	if err := os.Chmod(secondPath, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(secondPath, 0644)
	for _, action := range []func() error{
		func() error { _, err := store.RenameCategory("Tools", "Other"); return err },
		func() error { _, err := store.DeleteCategory("Tools"); return err },
	} {
		if err := action(); err == nil {
			t.Fatal("read-only bulk update succeeded")
		}
		if read(t, firstPath) != beforeFirst || read(t, secondPath) != beforeSecond {
			t.Fatal("preflight failure partially changed files")
		}
	}
	if err := os.Chmod(secondPath, 0644); err != nil {
		t.Fatal(err)
	}
	write(t, secondPath, strings.Replace(beforeSecond, "'Tools'", "'Externally changed'", 1))
	if _, err := store.RenameCategory("Tools", "Other"); err == nil {
		t.Fatal("stale group update accepted")
	}
	if read(t, firstPath) != beforeFirst {
		t.Fatal("external change failure partially changed group")
	}
}
