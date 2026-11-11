package launcher

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

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
