package launcher

import (
	"archive/zip"
	"fmt"
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

func absent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path should not exist: %s", path)
	}
}

func TestUpdateReplacesContentAndKeepsPinnedPaths(t *testing.T) {
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
	result, err := s.Update(a.UUID, packagePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(content, "app.exe")) != "new version" || read(t, filepath.Join(content, "settings", "user.json")) != "user data" {
		t.Fatal("installation/pin mismatch")
	}
	for _, name := range []string{"obsolete.txt", "settings-other", "Release-v2"} {
		absent(t, filepath.Join(content, name))
	}
	// Replaced files are moved into the backup; pinned paths stay in content and
	// must not be duplicated into the backup.
	if read(t, filepath.Join(result.Backup, "app.exe")) != "original" || read(t, filepath.Join(result.Backup, "obsolete.txt")) != "obsolete" {
		t.Fatal("backup incomplete")
	}
	absent(t, filepath.Join(result.Backup, "settings"))
	if err = s.DeleteBackup(a.UUID); err != nil {
		t.Fatal(err)
	}
	absent(t, result.Backup)
}

func TestUpdateRejectsUnsafeArchiveBeforeTouchingContent(t *testing.T) {
	for _, entry := range []string{"../escape.txt", "..\\escape.txt", "C:\\escape.txt"} {
		t.Run(entry, func(t *testing.T) {
			root := t.TempDir()
			dir := fixture(t, root, "app", "title='App'\nupdater=['RMC','UNPKG']")
			packagePath := filepath.Join(root, "unsafe.zip")
			zipPackage(t, packagePath, map[string]string{entry: "unsafe"})
			s := NewStore(root)
			a := scan(t, s).Apps[0]
			result, err := s.Update(a.UUID, packagePath, nil)
			if err == nil || result.Backup != "" {
				t.Fatal("unsafe archive accepted or produced a backup")
			}
			if read(t, filepath.Join(dir, "content", "app.exe")) != "original" {
				t.Fatal("rejected archive modified content")
			}
			absent(t, filepath.Join(dir, "content.bak"))
			absent(t, filepath.Join(dir, "escape.txt"))
		})
	}
}

func TestFailedUpdateKeepsBackupAndPinnedContent(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['PIN:settings','RMC','UNPKG']")
	content := filepath.Join(dir, "content")
	write(t, filepath.Join(content, "settings", "user.json"), "user data")
	packagePath := filepath.Join(root, "update.zip")
	// A file cannot replace the pinned directory.
	zipPackage(t, packagePath, map[string]string{"settings": "not a directory"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	result, err := s.Update(a.UUID, packagePath, nil)
	if err == nil || !strings.Contains(err.Error(), result.Backup) {
		t.Fatal("failure omitted backup")
	}
	if read(t, filepath.Join(content, "settings", "user.json")) != "user data" {
		t.Fatal("failed update lost pinned content")
	}
	if read(t, filepath.Join(result.Backup, "app.exe")) != "original" {
		t.Fatal("failed update lost backup")
	}
}

func TestUpdateBackupFailureLeavesContentAndPreviousBackup(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['RMC','UNPKG']")
	// A file at the backup path cannot be silently deleted as a directory.
	write(t, filepath.Join(dir, "content.bak"), "previous backup marker")
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, map[string]string{"app.exe": "new"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	_, err := s.Update(a.UUID, packagePath, nil)
	if err == nil || !strings.Contains(err.Error(), "无法替换旧备份") {
		t.Fatal("bad backup target accepted")
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "original" || read(t, filepath.Join(dir, "content.bak")) != "previous backup marker" {
		t.Fatal("backup failure modified original content")
	}
}

func TestUpdateOverlayBacksUpOnlyOverwrittenFiles(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['UNPKG']")
	content := filepath.Join(dir, "content")
	write(t, filepath.Join(content, "keep.txt"), "keep")
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, map[string]string{"app.exe": "replaced", "new.txt": "new"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	result, err := s.Update(a.UUID, packagePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(content, "app.exe")) != "replaced" || read(t, filepath.Join(content, "keep.txt")) != "keep" {
		t.Fatal("overlay mismatch")
	}
	if read(t, filepath.Join(result.Backup, "app.exe")) != "original" {
		t.Fatal("overwritten file not backed up")
	}
	absent(t, filepath.Join(result.Backup, "keep.txt"))
	absent(t, filepath.Join(result.Backup, "new.txt"))
	for _, command := range []string{"PIN:../outside", "PIN:settings/../outside", "PIN:", "RMC:parameter", "BOGUS"} {
		if ValidateUpdater([]string{command}) == nil {
			t.Fatal("invalid updater accepted: " + command)
		}
	}
}

func TestUpdateRestoresNestedPinsRegardlessOfOrder(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['PIN:data/user','PIN:data','RMC','UNPKG']")
	content := filepath.Join(dir, "content")
	write(t, filepath.Join(content, "data", "user", "prefs.txt"), "user prefs")
	write(t, filepath.Join(content, "data", "cache.txt"), "cache")
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, map[string]string{"app.exe": "new entry", "data/seed.txt": "new seed"})
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	result, err := s.Update(a.UUID, packagePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(content, "data", "user", "prefs.txt")) != "user prefs" || read(t, filepath.Join(content, "data", "cache.txt")) != "cache" {
		t.Fatal("pinned content lost")
	}
	if read(t, filepath.Join(content, "data", "seed.txt")) != "new seed" {
		t.Fatal("package content missing")
	}
	absent(t, filepath.Join(result.Backup, "data"))
}

func TestUpdateReportsProgress(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'\nupdater=['PIN:data','RMC','UNPKG']")
	write(t, filepath.Join(dir, "content", "data", "user.txt"), "user data")
	files := map[string]string{}
	for index := 0; index < 40; index++ {
		files["app/file-"+fmt.Sprintf("%02d", index)+".txt"] = "payload"
	}
	packagePath := filepath.Join(root, "update.zip")
	zipPackage(t, packagePath, files)
	s := NewStore(root)
	a := scan(t, s).Apps[0]
	progress := []UpdateProgress{}
	if _, err := s.Update(a.UUID, packagePath, func(step UpdateProgress) { progress = append(progress, step) }); err != nil {
		t.Fatal(err)
	}
	last := UpdateProgress{}
	for _, step := range progress {
		if step.Stage == "install" {
			last = step
		}
	}
	if last.Done != 40 || last.Total != 40 {
		t.Fatalf("install progress = %+v, want 40/40", last)
	}
	if read(t, filepath.Join(dir, "content", "data", "user.txt")) != "user data" {
		t.Fatal("pinned content lost")
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
	result, err := store.Update(app.UUID, source, nil)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "directory update" || read(t, filepath.Join(dir, "content", "settings", "user.txt")) != "personal settings" {
		t.Fatal("directory update/pin mismatch")
	}
	if read(t, filepath.Join(result.Backup, "app.exe")) != "original" {
		t.Fatal("backup missing")
	}
	absent(t, filepath.Join(result.Backup, "settings"))
	absent(t, filepath.Join(dir, "content", "obsolete.txt"))
	if read(t, filepath.Join(source, "app.exe")) != "directory update" {
		t.Fatal("source changed")
	}
	for _, path := range []string{filepath.Join(dir, "content"), filepath.Join(dir, "content", "bin"), dir, result.Backup, library} {
		if _, err := store.Update(app.UUID, path, nil); err == nil {
			t.Fatalf("overlapping update directory accepted: %s", path)
		}
	}
	if read(t, filepath.Join(dir, "content", "app.exe")) != "directory update" {
		t.Fatal("rejected update modified content")
	}
}

func TestScanReclaimsInterruptedUpdateBackup(t *testing.T) {
	root := t.TempDir()
	dir := fixture(t, root, "app", "title='App'")
	stale := filepath.Join(dir, ".content-backup-1234")
	write(t, filepath.Join(stale, "app.exe"), "stale")
	s := NewStore(root)
	scan(t, s)
	absent(t, stale)
}
