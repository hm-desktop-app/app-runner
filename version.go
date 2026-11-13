package main

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// buildInfoJSON 与 Windows 版本资源共用同一份版本来源，避免多处手写版本号。
//
//go:embed build/windows/info.json
var buildInfoJSON []byte

// appVersion 返回构建资源中的版本号（例如 "0.2.2"）；解析失败时返回空字符串。
func appVersion() string {
	var info struct {
		Fixed struct {
			FileVersion string `json:"file_version"`
		} `json:"fixed"`
	}
	if err := json.Unmarshal(buildInfoJSON, &info); err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(info.Fixed.FileVersion), "v")
}
