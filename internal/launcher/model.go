package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Link struct {
	Title string `json:"title" toml:"title"`
	URL   string `json:"url" toml:"url"`
}

type Profile struct {
	Category     string   `json:"category" toml:"category"`
	Description  string   `json:"description" toml:"description"`
	UUID         string   `json:"uuid" toml:"uuid"`
	Title        string   `json:"title" toml:"title"`
	Type         string   `json:"type" toml:"type"`
	Entry        string   `json:"entry" toml:"entry"`
	RunAsAdmin   bool     `json:"runAsAdmin" toml:"run_as_admin"`
	RelatedLinks []Link   `json:"relatedLinks" toml:"related_links"`
	Updater      []string `json:"updater" toml:"updater"`
}

type App struct {
	IconURL string `json:"iconURL"`
	Profile
	Directory    string     `json:"directory"`
	Modified     *time.Time `json:"modified"`
	LastUsed     *time.Time `json:"lastUsed"`
	Version      string     `json:"version"`
	EntryValid   bool       `json:"entryValid"`
	ProfileError string     `json:"profileError"`
}

type Snapshot struct {
	Library  string   `json:"library"`
	Apps     []App    `json:"apps"`
	Warnings []string `json:"warnings"`
}

type Options struct {
	Library  string
	SelfTest bool
	Help     bool
}

func ParseOptions(args []string, env string) (Options, error) {
	result := Options{Library: strings.TrimSpace(env)}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			result.Help = true
		case arg == "--selftest":
			result.SelfTest = true
		case arg == "--app-dir":
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return result, errors.New("--app-dir 需要应用库路径")
			}
			result.Library = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--app-dir="):
			result.Library = strings.TrimSpace(strings.TrimPrefix(arg, "--app-dir="))
		default:
			return result, errors.New("未知参数：" + arg)
		}
	}
	if result.Help {
		return result, nil
	}
	if result.Library == "" {
		return result, errors.New("请通过 --app-dir <路径> 或 USER_LOCAL_APP 环境变量指定应用库目录")
	}
	path, err := filepath.Abs(result.Library)
	if err != nil {
		return result, err
	}
	result.Library = path
	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	if !info.IsDir() {
		return result, errors.New("应用库路径必须是目录")
	}
	return result, nil
}

func Validate(p Profile) error {
	if err := validateCategory(p.Category); err != nil {
		return err
	}
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("标题不能为空")
	}
	if p.Type != "GUI" && p.Type != "CLI" {
		return errors.New("请选择图形应用或命令行应用")
	}
	if strings.TrimSpace(p.Entry) == "" {
		return errors.New("入口不能为空")
	}
	return ValidateUpdater(p.Updater)
}

func ValidateUpdater(commands []string) error {
	for _, command := range commands {
		action, param, has := strings.Cut(command, ":")
		action = strings.TrimSpace(action)
		switch action {
		case "PIN":
			if !has || strings.TrimSpace(param) == "" {
				return errors.New("固定文件/目录需要参数")
			}
			if _, err := safeRelative(strings.TrimSpace(param)); err != nil {
				return err
			}
		case "RMC", "UNPKG":
			if has {
				return errors.New(action + " 不接受参数")
			}
		default:
			return errors.New("未知更新操作：" + action)
		}
	}
	return nil
}

// Archive names use both slash styles, even when tests run outside Windows.
func safeRelative(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, ":") || strings.ContainsRune(name, 0) {
		return "", errors.New("路径必须位于 content 目录内：" + name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", errors.New("路径不能包含上级目录：" + name)
		}
	}
	path := filepath.Clean(filepath.FromSlash(name))
	if path == "." || filepath.IsAbs(path) {
		return "", errors.New("无效的相对路径：" + name)
	}
	return path, nil
}
