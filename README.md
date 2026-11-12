# App Runner

Windows 本地应用启动器，Go + Wails v3 实现。单实例常驻托盘，按 `Ctrl+Alt+Space` 唤出命令中心；应用以目录加 `app.toml` 描述，按分类浏览、启动与更新。窗口 920×640，无系统标题栏。

![命令中心](docs/command-center.png)

![应用库](docs/library.png)

## 构建与运行

依赖 Go 1.25+、Node.js 22.18+、npm 和 WebView2 Runtime，Wails 固定 `v3.0.0-beta.28`。

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28
go mod download
npm --prefix frontend ci
wails3 build
```

产物为 `bin/app-runner.exe`，前端资源与字体已嵌入。图标、应用清单和版本信息来自构建时生成的 `wails_windows_amd64.syso`；`wails3 build` 自动处理，单独 `go build` 不含这些资源。

```powershell
.\bin\app-runner.exe --app-dir "D:\Apps"
# 或
$env:USER_LOCAL_APP = "D:\Apps"
.\bin\app-runner.exe
```

`--app-dir` 优先于 `USER_LOCAL_APP`；两者都缺失或路径无效时显示中文错误并以状态码 2 退出。`--help`、`--selftest` 不参与单实例互斥。开发模式：

```powershell
$env:USER_LOCAL_APP = "D:\Apps"
wails3 dev
```

## 应用库

同时含有 `app.toml` 与 `content` 的子目录即一个应用：

```text
D:\Apps\
  editor\
    app.toml
    app.ico          # 可选，同级 ICO 作为列表与详情图标
    content\
      editor.exe
  state.json         # 最近使用记录
```

```toml
uuid = "3103402d-29ee-49b8-b8cc-d0c625e50faf"
title = "编辑器"
category = "开发工具"
description = "自由备注，支持多行"
type = "GUI"            # GUI | CLI
entry = "editor.exe"    # 相对 content，或以 re: 开头按文件名匹配
run_as_admin = false
updater = ["PIN:settings", "RMC", "UNPKG"]

[[related_links]]
title = "官网"
url = "https://example.com"
```

- 扫描为缺失、重复或非法的 UUID 生成 v4 身份并写回；配置损坏时显示目录名与错误，不覆盖原文件。
- 保存通过临时文件替换，保留可解析的未知字段，但会重排 TOML 并移除注释；只读文件保存失败时保留界面编辑。
- `entry` 必须指向 `content` 内的普通文件，越界或缺失即入口无效；`re:` 使用 Go RE2 正则，不支持后行断言与反向引用。
- GUI 应用经 Shell 启动，CLI 应用经 `pwsh -NoExit` 启动（缺失时用 Windows PowerShell），入口目录加入 PATH；管理员选项只提升被启动的应用。
- 图标缺失、损坏或超过 4 MiB 时使用字母占位。

## 配置与持久化

没有独立设置文件，状态落在四处：

| 位置 | 内容 |
| --- | --- |
| `<应用库>\<应用>\app.toml` | 分类、备注、入口、管理员、链接、更新操作与 UUID |
| `<应用库>\state.json` | 最近使用的 UUID、标题和最后启动时间 |
| `%APPDATA%\app-runner.exe\EBWebView\` | 界面偏好：`app-runner-theme`、`app-runner-autohide`、`app-runner-categories:<应用库路径>` |
| `%LOCALAPPDATA%\app-runner\instance.lock` | 单实例锁，进程退出后由系统释放 |

窗口尺寸、无边框、热键 `Ctrl+Alt+Space` 和关闭隐藏到托盘均为固定行为，不提供配置项。删除 `%APPDATA%\app-runner.exe` 只会重置主题、运行后隐藏和空分类，不影响应用库。

应用库中的临时文件：`.app-runner-new-*` 是新建应用的暂存目录，取消、正常退出或下次启动时清理；`.content-source-*`、`.content-unpack-*`、`.content-backup-*` 是更新时复制来源、解包与备份的暂存目录，操作结束后删除；`.app-runner-*` 是原子写入配置时的临时文件；`content.bak` 是保留的上一版本备份，可在界面删除。

## 操作

| 操作 | 行为 |
| --- | --- |
| 搜索 | 按名称、分类、备注和入口过滤，名称与分类支持拼音全拼和首字母；`>` 开头只列命令 |
| ↑ / ↓ | 在命令中心或应用库中移动选择 |
| Enter | 启动选中应用或执行选中命令 |
| Esc | 清空搜索；命令中心再按隐藏窗口，应用库再按返回命令中心 |
| 双击 | 启动应用 |
| `E` | 编辑选中应用 |
| `F5` | 重新扫描应用库 |
| 窗口重新激活 | 清空命令中心的搜索词并聚焦搜索框 |
| `Ctrl+F` / `Ctrl+K` | 聚焦搜索框 |
| 关闭按钮 / 失焦 | 隐藏到托盘；失焦自动隐藏仅命令中心生效 |
| 托盘菜单 | 显示窗口、退出 |
| 再次启动 | 唤回已运行实例 |

应用库的筛选项为全部应用、最近使用（7 天内启动）、可更新和分类，排序支持最近使用、名称（中文与数字自然排序）和更新时间。分类为一级目录，未填写时归入“未分类”。

拖动应用行到侧栏分类即修改分类，拖到“＋ 新建分类”可新建并移入；右键分类可重命名或删除，删除后成员回到“未分类”，应用文件与配置保留。编辑面板可修改名称、分类、备注、类型、入口、管理员、链接和更新操作，行尾按钮提供运行与编辑。

## 新建与更新

新建应用可从目录、ZIP / 7z 或完整路径导入：先选来源，再确认名称、入口和分类。文件先暂存于应用库的隐藏目录，保存后复制为 `<应用目录名>`，同名目录不覆盖，取消、退出或下次启动时清理暂存。

更新按 `updater` 声明顺序执行：

| 操作 | 参数 | 含义 |
| --- | --- | --- |
| `PIN` | 相对 content 的路径 | 清空时保留该文件或目录 |
| `RMC` | 无 | 清除未固定的旧内容 |
| `UNPKG` | 无 | 解包压缩包或复制目录，覆盖安装内容 |

更新前完整复制 `content` 到 `content.bak`，失败时保留备份并给出恢复路径。ZIP 使用标准库，7z 使用内置 `github.com/bodgit/sevenzip`，不调用外部解压工具；拒绝越界路径和符号链接，单层包装目录自动去除，不支持加密压缩包。

## 验证

```powershell
go test ./...
go vet ./...
npm --prefix frontend test
npm --prefix frontend run check
wails3 build
```

Go 测试覆盖扫描容错、暂存回收、UUID、只读配置、保存、最近使用、入口、备份、PIN 边界、ZIP / 7z 和失败保留；前端测试覆盖筛选排序、关键字与拼音搜索、时间边界、草稿隔离与更新参数校验。

无界面扫描：

```powershell
.\bin\app-runner.exe --app-dir "D:\Apps" --selftest
```

报告写入当前目录的 `selfcheck/selftest.json`。浏览器集成测试经 Wails 本地 server transport 调用真实 Go 服务：

```powershell
New-Item -ItemType Directory -Force selfcheck/test-bin | Out-Null
go build -tags server -o selfcheck/test-bin/app-runner-ui-test.exe .
npm --prefix frontend install --no-save playwright
$env:NODE_PATH = (Join-Path $PWD "frontend\node_modules")
node scripts/ui-test.cjs
```

Windows 使用已安装的 Edge，其他平台需先 `npx playwright install chromium`，报告写入 `selfcheck/`。原生窗口测试：`powershell -File scripts/native-test.ps1`。

## 代码结构

```text
main.go                       入口、窗口、托盘、单实例、自检
launcherservice.go            Wails 服务、原生对话框与文件选择
internal/launcher/            扫描、配置、启动、使用记录、更新
frontend/src/main.ts          界面与交互
frontend/src/state.ts         筛选、排序、草稿与校验
frontend/src/style.css        界面样式
frontend/bindings/            Wails 生成的类型绑定
scripts/ui-test.cjs           Go 服务的浏览器集成测试
build/                        图标、Windows 资源与 Wails 配置
docs/                         README 截图
```
