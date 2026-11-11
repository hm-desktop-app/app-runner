package launcher

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipPackage(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, text := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestUpdateBackupPinAndWrappedArchive(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\ntype='GUI'\nentry='app.exe'\nupdater=['PIN:settings','RMC','UNPKG']")
	content := filepath.Join(dir, "content")
	write(t, filepath.Join(content, "settings", "user.json"), "user data")
	write(t, filepath.Join(content, "settings-other", "remove.txt"), "remove")
	write(t, filepath.Join(content, "obsolete.txt"), "obsolete")
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, map[string]string{"Release-v2/app.exe": "new version", "Release-v2/new.txt": "new"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	result, err := s.Update(a.UUID, packagePath)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(content, "app.exe")) != "new version" || read(t, filepath.Join(content, "settings", "user.json")) != "user data" {
		t.Fatal("installation/pin mismatch")
	}
	for _, name := range []string{"obsolete.txt", "settings-other", "Release-v2"} {
		if _, err = os.Stat(filepath.Join(content, name)); !os.IsNotExist(err) {
			t.Fatal("obsolete content or wrapper left behind: " + name)
		}
	}
	if read(t, filepath.Join(result.Backup, "app.exe")) != "original" || read(t, filepath.Join(result.Backup, "obsolete.txt")) != "obsolete" {
		t.Fatal("backup incomplete")
	}
	if err = s.DeleteBackup(a.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(result.Backup); !os.IsNotExist(err) {
		t.Fatal("backup not deleted")
	}
}
func TestFailedUpdateRetainsBackupAndRejectsTraversal(t *testing.T) {
	for _, entry := range []string{"../escape.txt", `..\escape.txt`, `C:\escape.txt`} {
		t.Run(entry, func(t *testing.T) {
			root := t.TempDir()
			dir := fixture(t, root, "app", "title='App'\nupdater=['RMC','UNPKG']")
			packagePath := filepath.Join(root, "unsafe.zip")
			zipPackage(t, packagePath, map[string]string{entry: "unsafe"})
			s := NewStore(root)
			a := scan(t, s).Apps[0]
			result, err := s.Update(a.UUID, packagePath)
			if err == nil || !strings.Contains(err.Error(), result.Backup) {
				t.Fatal("failure omitted backup")
			}
			if read(t, filepath.Join(dir, "content.bak", "app.exe")) != "original" {
				t.Fatal("failed update lost backup")
			}
			if _, err = os.Stat(filepath.Join(dir, "escape.txt")); !os.IsNotExist(err) {
				t.Fatal("archive escaped staging")
			}
		})
	}
}
func TestBackupFailureLeavesContentAndPreviousBackup(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['RMC','UNPKG']")
	// A file at the backup path cannot be silently deleted as a directory.
	write(t, filepath.Join(dir, "content.bak"), "previous backup marker")
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, map[string]string{"app.exe": "new"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	if _, err := s.Update(a.UUID, packagePath); err == nil {
		t.Fatal("bad backup target accepted")
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "original" || read(t, filepath.Join(dir, "content.bak")) != "previous backup marker" {
		t.Fatal("backup failure modified original content")
	}
}
func TestUpdateOverwritesWithoutClear(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['UNPKG']")
	write(t, filepath.Join(dir, "content", "keep.txt"), "keep")
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, map[string]string{"app.exe": "replaced", "new.txt": "new"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	if _, err := s.Update(a.UUID, packagePath); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "replaced" || read(t, filepath.Join(dir, "content", "keep.txt")) != "keep" {
		t.Fatal("overlay mismatch")
	}
	for _, command := range []string{"PIN:../outside", "PIN:settings/../outside", "PIN:", "RMC:parameter", "BOGUS"} {
		if ValidateUpdater([]string{command}) == nil {
			t.Fatal("invalid updater accepted: " + command)
		}
	}
}

func TestSevenZipExtraction(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	if err := os.Mkdir(content, 0755); err != nil {
		t.Fatal(err)
	}
	if err := unpack(filepath.Join("testdata", "update.7z"), content); err != nil {
		t.Fatal(err)
	}
	if strings.TrimPrefix(read(t, filepath.Join(content, "app.exe")), "\ufeff") != "sevenzip version" {
		t.Fatal("7z extraction/wrapper failed")
	}
}

func TestDirectoryUpdateBacksUpAndPreservesPins(t *testing.T) {
	library := t.TempDir()
	source := t.TempDir()
	dir := fixture(t, library, "app", "title='App'\ntype='GUI'\nentry='app.exe'\nupdater=['PIN:settings','RMC','UNPKG']")
	write(t, filepath.Join(dir, "content", "settings", "user.txt"), "personal settings")
	write(t, filepath.Join(dir, "content", "obsolete.txt"), "old")
	write(t, filepath.Join(source, "app.exe"), "directory update")
	write(t, filepath.Join(source, "bin", "helper.exe"), "helper")
	store := NewStore(library)
	app := scan(t, store).Apps[0]
	result, err := store.Update(app.UUID, source)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "directory update" || read(t, filepath.Join(dir, "content", "settings", "user.txt")) != "personal settings" {
		t.Fatal("directory update/pin mismatch")
	}
	if read(t, filepath.Join(result.Backup, "app.exe")) != "original" {
		t.Fatal("backup missing")
	}
	if _, err := os.Stat(filepath.Join(dir, "content", "obsolete.txt")); !os.IsNotExist(err) {
		t.Fatal("obsolete file retained")
	}
	if read(t, filepath.Join(source, "app.exe")) != "directory update" {
		t.Fatal("source changed")
	}
	for _, path := range []string{filepath.Join(dir, "content"), filepath.Join(dir, "content", "bin"), dir, result.Backup, library} {
		if _, err := store.Update(app.UUID, path); err == nil {
			t.Fatalf("overlapping update directory accepted: %s", path)
		}
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "directory update" {
		t.Fatal("rejected update modified content")
	}
}
