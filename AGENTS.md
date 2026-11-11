# AGENTS.md

App Runner 是 Go + Wails v3 的 Windows 桌面启动器，前端为 TypeScript + Vite，
前端资源在编译时嵌入可执行文件。本文件记录本仓库的协作约定。

## 构建与验证

`wails3 build` 生成完整产物（前端、图标与版本资源，输出 `bin/app-runner.exe`）；
单独 `go build` 需先构建前端，且不含 `.syso` 资源。

改动后至少执行：

```powershell
go test ./...
go vet ./...
npm --prefix frontend test
npm --prefix frontend run check
```

涉及界面的改动，再跑浏览器集成测试（真实 Go 服务 + Playwright，Windows 使用已安装的 Edge）：

```powershell
go build -tags server -o selfcheck/test-bin/app-runner-ui-test.exe .
$env:NODE_PATH = (Join-Path $PWD "frontend\node_modules")
node scripts/ui-test.cjs
```

这些测试目前只在本地执行，CI 只做构建、Go 测试与前端单测。原生窗口行为用
`powershell -File scripts/native-test.ps1` 验证。

## 发版

1. 同步版本号：`build/config.yml` 的 `info.version`，以及 `build/windows/info.json` 的
   `file_version` 与 `ProductVersion`，统一改为 `X.Y.Z`（不带 `v`）。
2. 提交后打 annotated tag：`git tag -a vX.Y.Z -m "App Runner vX.Y.Z"`。
3. 推送分支与 tag：`git push && git push local`，再 `git push origin vX.Y.Z && git push local vX.Y.Z`。
4. CI 在 tag 上构建并自动创建 GitHub Release，资产命名为 `app-runner-windows-amd64.exe`；
   文件名不带版本号，版本以 tag 为准。
5. 用 `gh release view vX.Y.Z` 确认资产已上传且大小正常。

## 部署本机 live

用户已授权：更新后默认替换正在运行的 live 版本，可以直接强制结束旧进程。

- 路径：`S:\local\bin\app-runner.exe`。
- 结束进程前确认其路径确实是该文件；复制后校验 SHA256 一致，再通过 Shell 重新启动。
- 发版时优先部署 GitHub Release 里的资产，而不是本地构建产物。

## 仓库

- 两个远端：`origin`（github.com）、`local`（git.daveny.cn），推送时两边都推。
- `bin/`、`selfcheck/`、`*.syso`、`.ref/` 不纳入版本库。