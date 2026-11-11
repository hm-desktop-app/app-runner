package launcher

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type UpdateResult struct {
	App    App    `json:"app"`
	Backup string `json:"backup"`
}

// All destructive paths derive from a scanned app, never a frontend path.
func (s *Store) Update(id, packagePath string) (UpdateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := UpdateResult{}
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
	packagePath = input
	content := filepath.Join(a.Directory, "content")
	backup := filepath.Join(a.Directory, "content.bak")
	if err := assertDirectory(content); err != nil {
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
	if inputInfo.IsDir() {
		sourceStage, err := os.MkdirTemp(a.Directory, ".content-source-")
		if err != nil {
			return result, err
		}
		defer removeTree(sourceStage)
		if err = copyTree(input, sourceStage); err != nil {
			return result, fmt.Errorf("准备更新目录失败，原内容未修改：%w", err)
		}
		packagePath = sourceStage
	}

	temporary, err := os.MkdirTemp(a.Directory, ".content-backup-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(temporary)
	if err = copyTree(content, temporary); err != nil {
		return result, fmt.Errorf("备份失败，原内容未修改：%w", err)
	}
	if err = removeTree(backup); err != nil {
		return result, fmt.Errorf("无法替换旧备份，原内容未修改：%w", err)
	}
	if err = os.Rename(temporary, backup); err != nil {
		return result, err
	}
	result.Backup = backup
	pins := []string{}
	for _, command := range a.Updater {
		action, param, _ := strings.Cut(command, ":")
		switch strings.TrimSpace(action) {
		case "PIN":
			var relative string
			relative, err = safeRelative(strings.TrimSpace(param))
			pins = append(pins, filepath.Join(content, relative))
		case "RMC":
			err = clearContent(content, pins)
		case "UNPKG":
			err = unpack(packagePath, content)
		}
		if err != nil {
			return result, fmt.Errorf("更新失败：%w。备份已保留在 %s，可将其中的文件复制回 content 恢复", err, backup)
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

func removeTree(path string) error {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	}
	if err := assertDirectory(path); err != nil {
		return err
	}
	if err := filepath.WalkDir(path, func(name string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.Type()&os.ModeSymlink == 0 {
			return os.Chmod(name, 0755)
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func copyTree(source, target string) error {
	return filepath.WalkDir(source, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.Type()&os.ModeSymlink != 0 {
			return errors.New("内容包含符号链接，无法安全备份：" + path)
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if item.IsDir() {
			if err := os.MkdirAll(destination, 0755); err != nil {
				return err
			}
			return assertDirectory(destination)
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("内容包含特殊文件：" + path)
		}
		return copyFile(path, destination, info.Mode().Perm())
	})
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

type archiveEntry struct {
	name string
	mode os.FileMode
	open func() (io.ReadCloser, error)
}

func unpack(packagePath, content string) error {
	if info, err := os.Stat(packagePath); err != nil {
		return err
	} else if info.IsDir() {
		return copyTree(packagePath, content)
	}
	stage, err := os.MkdirTemp(filepath.Dir(content), ".content-unpack-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	entries := []archiveEntry{}
	switch strings.ToLower(filepath.Ext(packagePath)) {
	case ".zip":
		reader, e := zip.OpenReader(packagePath)
		if e != nil {
			return e
		}
		defer reader.Close()
		for _, f := range reader.File {
			name := f.Name
			if f.NonUTF8 {
				if decoded, e := simplifiedchinese.GBK.NewDecoder().String(name); e == nil {
					name = decoded
				}
			}
			entries = append(entries, archiveEntry{name: name, mode: f.Mode(), open: f.Open})
		}
	case ".7z":
		reader, e := sevenzip.OpenReader(packagePath)
		if e != nil {
			return e
		}
		defer reader.Close()
		for _, f := range reader.File {
			entries = append(entries, archiveEntry{name: f.Name, mode: f.Mode(), open: f.Open})
		}
	default:
		return errors.New("只支持 .zip 和 .7z 安装包")
	}
	// Validate all names and links before writing extracted files.
	for _, entry := range entries {
		if _, err = safeRelative(entry.name); err != nil {
			return err
		}
		if entry.mode&os.ModeSymlink != 0 || (!entry.mode.IsRegular() && !entry.mode.IsDir()) {
			return errors.New("安装包包含链接或特殊文件：" + entry.name)
		}
	}
	for _, entry := range entries {
		relative, _ := safeRelative(entry.name)
		target := filepath.Join(stage, relative)
		if entry.mode.IsDir() {
			if err = os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		input, e := entry.open()
		if e != nil {
			return e
		}
		output, e := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if e != nil {
			input.Close()
			return e
		}
		_, e = io.Copy(output, input)
		input.Close()
		closeErr := output.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
	}
	top, err := os.ReadDir(stage)
	if err != nil {
		return err
	}
	source := stage
	if len(top) == 1 && top[0].IsDir() {
		source = filepath.Join(stage, top[0].Name())
	}
	// copyTree rejects links in the extracted tree and copyFile rejects target links.
	return copyTree(source, content)
}
