package launcher

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func decodePowerShell(t *testing.T, args string) string {
	t.Helper()
	const prefix = "-NoExit -EncodedCommand "
	if !strings.HasPrefix(args, prefix) {
		t.Fatalf("unexpected PowerShell arguments: %s", args)
	}
	bytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(args, prefix))
	if err != nil {
		t.Fatal(err)
	}
	units := make([]uint16, len(bytes)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(bytes[i*2:])
	}
	return string(utf16.Decode(units))
}

func TestPowerShellCommandPreservesUnicodeAndQuote(t *testing.T) {
	path := `C:\测试目录\O'Brien\tool.cmd`
	quoted := psQuote(path)
	if quoted != `'C:\测试目录\O''Brien\tool.cmd'` {
		t.Fatal("PowerShell single-quote escape failed")
	}
	script := "& " + quoted
	bytes, err := base64.StdEncoding.DecodeString(encodedCommand(script))
	if err != nil {
		t.Fatal(err)
	}
	units := make([]uint16, len(bytes)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(bytes[i*2:])
	}
	decoded := string(utf16.Decode(units))
	if decoded != script || !strings.Contains(decoded, "测试目录") {
		t.Fatal("encoded command corrupted path")
	}
}

// CLI 应用只打开注入了入口目录的 shell，不执行入口；工作目录是用户目录。
func TestLaunchCommandOpensShellForCLI(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "tool")
	content := filepath.Join(directory, "content")
	if err := os.MkdirAll(filepath.Join(content, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "bin", "tool.exe"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	app := App{Directory: directory, Profile: Profile{Title: "Tool", Type: "CLI", Entry: "bin/tool.exe"}}
	file, args, cwd, err := launchCommand(app)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.Base(file), "pwsh.exe") && !strings.EqualFold(filepath.Base(file), "powershell.exe") {
		t.Fatal("CLI application did not resolve a PowerShell binary: " + file)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if cwd != home {
		t.Fatalf("shell working directory = %s, want the user directory %s", cwd, home)
	}
	decoded := decodePowerShell(t, args)
	const prefix = "$env:Path = '"
	start := strings.Index(decoded, prefix)
	if start < 0 || !strings.HasSuffix(decoded, ";' + $env:Path") {
		t.Fatal("unexpected shell script: " + decoded)
	}
	rest := decoded[start+len(prefix):]
	end := strings.Index(rest, ";'")
	if end < 0 {
		t.Fatal("unexpected shell script: " + decoded)
	}
	injected := rest[:end]
	if !strings.EqualFold(filepath.Base(injected), "bin") || !strings.EqualFold(filepath.Base(filepath.Dir(injected)), "content") {
		t.Fatalf("entry directory is not on PATH: %s", injected)
	}
	if strings.Contains(decoded, "& ") {
		t.Fatal("CLI application still executes the entry: " + decoded)
	}
}

func TestLaunchCommandRunsGUIEntry(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "editor")
	content := filepath.Join(directory, "content")
	if err := os.MkdirAll(content, 0755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(content, "app.exe")
	if err := os.WriteFile(entry, nil, 0644); err != nil {
		t.Fatal(err)
	}
	app := App{Directory: directory, Profile: Profile{Title: "Editor", Type: "GUI", Entry: "app.exe"}}
	file, args, cwd, err := launchCommand(app)
	if err != nil {
		t.Fatal(err)
	}
	if args != "" || !strings.EqualFold(filepath.Base(file), "app.exe") || !strings.EqualFold(filepath.Base(filepath.Dir(file)), "content") || !strings.EqualFold(cwd, filepath.Dir(file)) {
		t.Fatalf("GUI launch changed: file=%s args=%q cwd=%s", file, args, cwd)
	}
}

func TestScanRecordsContentModificationTime(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "app", "title = 'App'\ntype = 'GUI'\nentry = 'app.exe'\n")
	apps := scan(t, NewStore(root)).Apps
	if len(apps) != 1 || apps[0].Modified == nil {
		t.Fatalf("scan lost content modification time: %+v", apps)
	}
}
