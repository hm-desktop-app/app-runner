package launcher

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ResolveEntry(a App, entry string) (string, error) {
	root := filepath.Join(a.Directory, "content")
	if strings.HasPrefix(entry, "re:") {
		if len(entry) == 3 {
			return "", errors.New("正则入口不能为空")
		}
		pattern, err := regexp.Compile(entry[3:])
		if err != nil {
			return "", err
		}
		var found string
		err = filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if item.Type()&os.ModeSymlink != 0 {
				if item.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !item.IsDir() && pattern.MatchString(item.Name()) {
				found = path
				return fs.SkipAll
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		if found == "" {
			return "", errors.New("没有匹配的入口文件")
		}
		return checkedFile(root, found)
	}
	relative, err := safeRelative(entry)
	if err != nil {
		return "", err
	}
	return checkedFile(root, filepath.Join(root, relative))
}

func checkedFile(root, path string) (string, error) {
	actual, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("入口文件不存在：" + path)
	}
	actualRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(actualRoot, actual)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("入口超出 content 目录")
	}
	info, err := os.Stat(actual)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("入口必须是文件")
	}
	return actual, nil
}
