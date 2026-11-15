package launcher

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bodgit/sevenzip"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// UpdateProgress reports installation progress to the UI. Stage is one of
// "cleanup", "backup" and "install"; Done/Total stay zero while a step has no
// countable work.
type UpdateProgress struct {
	Stage string `json:"stage"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

type UpdateResult struct {
	App    App    `json:"app"`
	Backup string `json:"backup"`
}

// All destructive paths derive from a scanned app, never a frontend path.
//
// An update never copies the installed tree: a replace update (RMC) moves
// content aside with one rename and moves pinned paths back, while an overlay
// update (no RMC) backs up only the files the package overwrites.
func (s *Store) Update(id, packagePath string, report func(UpdateProgress)) (UpdateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := UpdateResult{}
	emit := func(stage string, done, total int) {
		if report != nil {
			report(UpdateProgress{Stage: stage, Done: done, Total: total})
		}
	}
	a, ok := s.apps[id]
	if !ok {
		return result, errors.New("应用不存在")
	}
	if len(a.Updater) == 0 {
		return result, errors.New("未定义更新方式")
	}
	if err := ValidateUpdater(a.Updater); err != nil {
		return result, err
	}
	input, inputInfo, err := sourcePath(packagePath)
	if err != nil {
		return result, err
	}
	content := filepath.Join(a.Directory, "content")
	backup := filepath.Join(a.Directory, "content.bak")
	if err = assertDirectory(content); err != nil {
		return result, err
	}
	actualContent, err := filepath.EvalSymlinks(content)
	if err != nil {
		return result, err
	}
	actualApp, err := filepath.EvalSymlinks(a.Directory)
	if err != nil {
		return result, err
	}
	actualBackup := filepath.Join(actualApp, "content.bak")
	if inside(actualContent, input) || inside(actualBackup, input) || (inputInfo.IsDir() && (inside(input, actualContent) || inside(input, actualBackup))) {
		return result, errors.New("更新来源不能位于 content / content.bak 内，也不能包含这些目录")
	}
	// The package is validated in full before content is touched, so a broken
	// archive fails without any change to the installed files.
	pkg, err := openPackage(input, inputInfo)
	if err != nil {
		return result, err
	}
	defer pkg.close()
	replace := false
	pins := []string{}
	restores := []string{}
	for _, command := range a.Updater {
		action, param, _ := strings.Cut(command, ":")
		switch strings.TrimSpace(action) {
		case "RMC":
			replace = true
		case "PIN":
			relative, e := safeRelative(strings.TrimSpace(param))
			if e != nil {
				return result, e
			}
			pins = append(pins, filepath.Join(content, relative))
			restores = append(restores, relative)
		}
	}
	// Pinned paths are declarations, not steps: restore the shallowest ones
	// first so a nested pin never collides with an already restored parent.
	sort.SliceStable(restores, func(i, j int) bool {
		return strings.Count(restores[i], string(filepath.Separator)) < strings.Count(restores[j], string(filepath.Separator))
	})
	if replace {
		if err = removeTreeReport(backup, emit); err != nil {
			return result, fmt.Errorf("无法替换旧备份，原内容未修改：%w", err)
		}
		if err = os.Rename(content, backup); err != nil {
			return result, fmt.Errorf("无法备份原内容：%w", err)
		}
		if err = os.Mkdir(content, 0755); err != nil {
			return result, err
		}
		result.Backup = backup
		for _, relative := range restores {
			if err = restorePin(backup, content, relative); err != nil {
				return result, fmt.Errorf("更新失败：%w。更新前的文件已保存在 %s", err, backup)
			}
		}
	} else {
		if err = backupOverwritten(pkg, content, backup, emit); err != nil {
			return result, err
		}
		result.Backup = backup
	}
	for _, command := range a.Updater {
		action, _, _ := strings.Cut(command, ":")
		switch strings.TrimSpace(action) {
		case "RMC":
			err = clearContent(content, pins)
		case "UNPKG":
			err = pkg.install(content, emit)
		}
		if err != nil {
			return result, fmt.Errorf("更新失败：%w。更新前的文件已保存在 %s", err, backup)
		}
	}
	if info, e := os.Stat(content); e == nil {
		now := info.ModTime()
		a.Modified = &now
	}
	refreshMetadata(&a)
	s.apps[id] = a
	result.App = a
	return result, nil
}

func (s *Store) DeleteBackup(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[id]
	if !ok {
		return errors.New("应用不存在")
	}
	return removeTree(filepath.Join(a.Directory, "content.bak"))
}

func assertDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("拒绝操作链接或非目录：" + path)
	}
	return nil
}

// backupOverwritten copies the files an overlay update replaces, leaving every
// other file in place; the copy becomes content.bak so the update is revertible.
func backupOverwritten(pkg *packageFiles, content, backup string, emit func(string, int, int)) error {
	stage, err := os.MkdirTemp(filepath.Dir(content), ".content-backup-")
	if err != nil {
		return err
	}
	kept := false
	defer func() {
		if !kept {
			_ = removeTree(stage)
		}
	}()
	total := len(pkg.files)
	for index, file := range pkg.files {
		target := filepath.Join(content, file.relative)
		info, e := os.Lstat(target)
		if e != nil {
			if !os.IsNotExist(e) {
				return e
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if e = copyFile(target, filepath.Join(stage, file.relative), info.Mode().Perm()); e != nil {
			return e
		}
		if index%16 == 0 {
			emit("backup", index+1, total)
		}
	}
	if err = removeTreeReport(backup, emit); err != nil {
		return fmt.Errorf("无法替换旧备份，原内容未修改：%w", err)
	}
	if err = os.Rename(stage, backup); err != nil {
		return err
	}
	kept = true
	return nil
}

// restorePin moves a pinned path from the backup into the fresh content tree.
func restorePin(backup, content, relative string) error {
	source := filepath.Join(backup, relative)
	if _, err := os.Lstat(source); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	target := filepath.Join(content, relative)
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("固定路径 %s 已被安装内容占用，固定操作需要排在解包之前", relative)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.Rename(source, target)
}

func removeTree(path string) error { return removeTreeReport(path, nil) }

// removeTreeReport deletes a directory tree depth first and reports the number
// of removed entries. Read-only entries are retried after clearing attributes.
func removeTreeReport(path string, emit func(string, int, int)) error {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	}
	if err := assertDirectory(path); err != nil {
		return err
	}
	names := []string{}
	if err := filepath.WalkDir(path, func(name string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		names = append(names, name)
		return nil
	}); err != nil {
		return err
	}
	total := len(names)
	for index := len(names) - 1; index >= 0; index-- {
		name := names[index]
		if err := os.Remove(name); err != nil {
			if chmodErr := os.Chmod(name, 0755); chmodErr != nil {
				return err
			}
			if err = os.Remove(name); err != nil {
				return err
			}
		}
		if emit != nil && total > 1 && index%64 == 0 {
			emit("cleanup", total-index, total)
		}
	}
	if emit != nil && total > 1 {
		emit("cleanup", total, total)
	}
	return nil
}

func copyFile(source, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	if st, e := os.Lstat(target); e == nil {
		if !st.Mode().IsRegular() {
			return errors.New("不能覆盖目录或链接：" + target)
		}
		if err = os.Chmod(target, 0600); err != nil {
			return err
		}
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, input)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func clearContent(root string, pins []string) error {
	var visit func(string) error
	visit = func(directory string) error {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, item := range entries {
			path := filepath.Join(directory, item.Name())
			preserve, ancestor := false, false
			for _, pin := range pins {
				lowerPath, lowerPin := strings.ToLower(path), strings.ToLower(pin)
				if lowerPath == lowerPin || strings.HasPrefix(lowerPath, lowerPin+string(filepath.Separator)) {
					preserve = true
				}
				if strings.HasPrefix(lowerPin, lowerPath+string(filepath.Separator)) {
					ancestor = true
				}
			}
			if preserve {
				continue
			}
			if item.IsDir() && item.Type()&os.ModeSymlink == 0 {
				if ancestor {
					if err = visit(path); err != nil {
						return err
					}
				} else {
					if err = removeTree(path); err != nil {
						return err
					}
				}
			} else {
				if item.Type()&os.ModeSymlink == 0 {
					_ = os.Chmod(path, 0600)
				}
				if err = os.Remove(path); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(root)
}

// packageFile is one validated file to install, relative to content.
type packageFile struct {
	relative string
	mode     os.FileMode
	open     func() (io.ReadCloser, error)
}

// packageFiles is a validated install payload: entries are checked before any
// file is written, and a wrapping directory is stripped.
type packageFiles struct {
	dirs  []string
	files []packageFile
	close func() error
}

func openPackage(path string, info os.FileInfo) (*packageFiles, error) {
	if info.IsDir() {
		return openDirectoryPackage(path)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".zip":
		reader, err := zip.OpenReader(path)
		if err != nil {
			return nil, err
		}
		result := &packageFiles{close: reader.Close}
		for _, entry := range reader.File {
			name := entry.Name
			if entry.NonUTF8 {
				if decoded, e := simplifiedchinese.GBK.NewDecoder().String(name); e == nil {
					name = decoded
				}
			}
			if err = result.add(name, entry.Mode(), entry.Open); err != nil {
				reader.Close()
				return nil, err
			}
		}
		return result.stripWrapper(), nil
	case ".7z":
		reader, err := sevenzip.OpenReader(path)
		if err != nil {
			return nil, err
		}
		result := &packageFiles{close: reader.Close}
		for _, file := range reader.File {
			if err = result.add(file.Name, file.Mode(), file.Open); err != nil {
				reader.Close()
				return nil, err
			}
		}
		return result.stripWrapper(), nil
	}
	return nil, errors.New("只支持 .zip 和 .7z 安装包")
}

func openDirectoryPackage(root string) (*packageFiles, error) {
	result := &packageFiles{close: func() error { return nil }}
	err := filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if item.Type()&os.ModeSymlink != 0 {
			return errors.New("更新来源包含符号链接，无法安全安装：" + path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if item.IsDir() {
			result.dirs = append(result.dirs, relative)
			return nil
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("更新来源包含特殊文件：" + path)
		}
		target, mode := path, info.Mode().Perm()
		result.files = append(result.files, packageFile{
			relative: relative,
			mode:     mode,
			open:     func() (io.ReadCloser, error) { return os.Open(target) },
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.stripWrapper(), nil
}

func (p *packageFiles) add(name string, mode os.FileMode, open func() (io.ReadCloser, error)) error {
	relative, err := safeRelative(name)
	if err != nil {
		return err
	}
	switch {
	case mode.IsDir():
		p.dirs = append(p.dirs, relative)
	case mode.IsRegular():
		p.files = append(p.files, packageFile{relative: relative, mode: 0644, open: open})
	default:
		return errors.New("安装包包含链接或特殊文件：" + name)
	}
	return nil
}

// stripWrapper drops a single directory shared by every entry, matching the
// common "Release/app.exe" layout; root-level files keep the archive as is.
func (p *packageFiles) stripWrapper() *packageFiles {
	if len(p.files) == 0 {
		return p
	}
	first := ""
	shares := func(relative string, allowRoot bool) bool {
		parts := strings.Split(filepath.ToSlash(relative), "/")
		if len(parts) == 1 && allowRoot {
			return first == "" || parts[0] == first
		}
		if len(parts) < 2 {
			return false
		}
		if first == "" {
			first = parts[0]
		}
		return parts[0] == first
	}
	for _, relative := range p.dirs {
		if !shares(relative, true) {
			return p
		}
	}
	for _, file := range p.files {
		if !shares(file.relative, false) {
			return p
		}
	}
	if first == "" {
		return p
	}
	result := &packageFiles{close: p.close}
	for _, relative := range p.dirs {
		if trimmed, err := safeRelative(strings.TrimPrefix(filepath.ToSlash(relative), first+"/")); err == nil {
			result.dirs = append(result.dirs, trimmed)
		}
	}
	for _, file := range p.files {
		trimmed, err := safeRelative(strings.TrimPrefix(filepath.ToSlash(file.relative), first+"/"))
		if err != nil {
			return p
		}
		file.relative = trimmed
		result.files = append(result.files, file)
	}
	return result
}

func (p *packageFiles) install(content string, emit func(string, int, int)) error {
	for _, relative := range p.dirs {
		if err := os.MkdirAll(filepath.Join(content, relative), 0755); err != nil {
			return err
		}
	}
	buffer := make([]byte, 256<<10)
	total := len(p.files)
	for index, file := range p.files {
		target := filepath.Join(content, file.relative)
		if info, err := os.Lstat(target); err == nil && !info.Mode().IsRegular() {
			return errors.New("不能覆盖目录或链接：" + target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		input, err := file.open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.mode)
		if err != nil {
			input.Close()
			return err
		}
		_, err = io.CopyBuffer(output, input, buffer)
		input.Close()
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		if emit != nil && (index%16 == 0 || index == total-1) {
			emit("install", index+1, total)
		}
	}
	return nil
}

// unpack installs a package into an empty content directory (new applications).
func unpack(packagePath, content string) error {
	info, err := os.Stat(packagePath)
	if err != nil {
		return err
	}
	pkg, err := openPackage(packagePath, info)
	if err != nil {
		return err
	}
	defer pkg.close()
	return pkg.install(content, nil)
}
