# App Runner

使用 **Go + Wails v3** 重写的 Windows 本地应用启动器，原 WPF 项目仍保留在 `../AppRunner`。业务逻辑在 Go 中实现，界面使用 TypeScript 和 Vite，通过 Wails 自动生成的类型化绑定调用 Go 服务。

界面使用暖白底与砂岩蓝强调色，没有系统标题栏，窗口默认 920×640（最小 740×470）。命令中心负责搜索和启动：输入即时过滤，上下键选择，Enter 运行，右侧预览显示选中项。应用库负责分类、筛选和批量管理，编辑与新建通过右侧面板完成。主题在设置中切换并记住选择；按 Esc、点击关闭按钮或窗口失去焦点（仅命令中心）都会收起窗口，热键再次唤起。

## 构建与运行

需要 Go 1.25+、Node.js 22.18+、npm，以及 Windows WebView2 Runtime。项目固定使用 Wails `v3.0.0-beta.28`，前端运行时使用相同版本。

在本目录执行：

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28
go mod download
npm --prefix frontend ci
wails3 build
```

发布程序位于 `bin/app-runner.exe`。界面资源和字体已嵌入程序，无需另外分发前端目录。

```powershell
.\bin\app-runner.exe --app-dir "D:\Apps"
# 或
$env:USER_LOCAL_APP = "D:\Apps"
.\bin\app-runner.exe
```

`--app-dir=<路径>` 也可用，命令行路径优先于 `USER_LOCAL_APP`。未提供路径或路径不可用时，显示中文错误并以状态码 2 退出。

启动器在当前用户的本地缓存目录持有系统文件锁（Windows：`%LOCALAPPDATA%\app-runner\instance.lock`）。同一用户再次启动时，已有实例会显示窗口，新进程随即退出。进程退出或崩溃后系统会释放锁；保留的锁文件不会阻止重启。`--help` 和 `--selftest` 不参与单实例互斥。

开发模式：

```powershell
$env:USER_LOCAL_APP = "D:\Apps"
wails3 dev
```

## 应用库和兼容性

直接子目录同时包含 `app.toml` 和 `content` 目录，就会出现在列表中：

```text
D:\Apps\
  editor\
    app.toml
    app.ico       # 可选，用作列表和详情图标
    content\
      editor.exe
  state.json
```

```toml
uuid = "3103402d-29ee-49b8-b8cc-d0c625e50faf"
title = "编辑器"
category = "开发工具"
description = "自由备注；支持多行文本"
type = "GUI"
entry = "editor.exe"
run_as_admin = false
updater = ["PIN:settings", "RMC", "UNPKG"]

[[related_links]]
title = "官网"
url = "https://example.com"
```

- 读取每个软件目录下、与 `app.toml` 同级的 `app.ico`，支持标准 ICO 内的多尺寸图像。添加或替换后按 F5 刷新；缺失、损坏或超过 4 MiB 时保留字母占位。图标不会写入 TOML。
- 沿用 WPF 版本的 TOML 字段和 `state.json` 的 `lastUsed / app / lastOpen` 格式，原应用库可直接使用。
- 空配置也会列出；损坏配置显示目录名和错误，扫描不会覆盖它，用户可在详情中修复并保存。
- 扫描为缺失、无效或重复的 UUID 生成 v4 身份，并安全插入或替换实际顶层 UUID 字段，保留其他文本。原有合法 UUID 保持不变。
- 无法写回 UUID 或无法解析配置时，以目录生成临时的稳定身份并报告原因；这种身份在目录重命名后会改变。
- 保存采用临时文件替换，保留可解析的未知字段，但会重排 TOML 并移除注释。只读文件保存失败时，界面中的编辑保留。
- 入口为相对 `content` 的文件路径，也支持 `re:` 开头的文件名正则匹配。Go 的 RE2 正则不支持 .NET 的后行断言和反向引用，需要相应调整这类入口表达式。
- GUI 应用通过 Windows Shell 启动；CLI 应用通过 `pwsh -NoExit` 启动，无 `pwsh` 时退回 Windows PowerShell。入口目录加入 PATH，并使用 UTF-16 编码命令正确处理空格、中文和引号。
- 管理员选项只提升被启动的应用；Windows 要求提升时自动重试。

## 操作

| 操作 | 行为 |
| --- | --- |
| 搜索 | 按名称、分类和备注过滤，不区分大小写；自动展开匹配分类 |
| 名称排序 | 中文排序，数字采用自然排序 |
| 更新时间排序 | 按 `content` 目录本身的修改时间逆序，显示 30 / 90 天分界的状态点 |
| 最近使用 | 按实际启动成功后的使用时间逆序，未使用过的应用排后 |
| 排序切换 / F5 | 重读应用库，保留选择和未保存的编辑 |
| 双击 / Enter | 使用已保存的配置启动应用 |
| 上下方向键 | 搜索框或列表中移动选择 |
| Ctrl+F / Ctrl+S | 搜索 / 保存 |
| 右键 | 快速设置分组、新建分组并移动、运行、打开目录、编辑配置、安装更新 |
| 切换应用 | 有未保存内容时，询问是否放弃 |
| 关闭窗口 | 隐藏到托盘；托盘菜单可显示窗口或退出 |
| 再次启动 | 唤回已运行的实例 |

链接的标题和地址均可编辑，旁边的箭头在默认浏览器打开地址。缺少协议时补充 `https://`。

## 分类与备注

`category` 与 `description` 均为可选字符串，旧配置无需迁移。`category` 未填写或为空时显示在“未分类”；分类为一级目录，应用在各分类内按当前排序方式排列。`description` 用作自由备注，详情页支持多行编辑，鼠标悬停应用时也能查看。

点击分类标题折叠或展开。拖动应用到目标分类标题会立即保存分类；拖到侧栏“＋ 新建分类”按钮，输入名称即可创建分类并移动应用。也可以先点击该按钮创建空分类，或在详情页输入新的分类名称后保存。拖到“未分类”可清除分类。

右键应用后选择“设置分组”，可直接选择已有分组、“未分类”，或新建分组并移动；操作不会切走当前正在编辑的应用。支持 Shift+F10 打开菜单、方向键选择和 Escape 关闭。

分组标题右侧的编辑图标可打开管理窗口：重命名同步修改全部成员的 category；删除分组将成员移至“未分类”，保留应用及其文件。空分组同样支持重命名、删除。“未分类”作为默认分组保留。重命名遇到重复名称会提示；批量修改会预先检查全部成员配置，保存失败时尝试还原已写入的配置。

拖拽和分组操作仅修改 TOML 的 category 值，保留备注、注释和其他配置；详情中未保存的编辑保持原样。空分类和折叠状态按应用库保存在本机。搜索时临时展开匹配分类，清空搜索后恢复折叠状态。

## 新建应用

侧栏“＋ 新建应用”打开来源窗口，可选择目录或 ZIP / 7z 压缩包，也可粘贴完整来源路径。目录名会按来源名称建议为小写、空格转短横线；手动修改目录名后，后续选择来源不再覆盖这个名字。

点击“填写详情”后显示空的应用名称和启动入口，默认类型为图形应用。入口可从候选路径选择，也支持 `re:` 正则。填写名称、有效入口和所需配置后，点击保存，将文件部署至应用库的 `<应用目录名>/content`，配置写入同级 `app.toml`；同名目录不会覆盖。保存前不会创建该最终应用目录，也不会出现在应用列表中。

准备文件使用应用库中的独立暂存目录。可点击“取消新建”放弃；正常退出也会清理尚未保存的暂存文件。来源文件保持原样。

## 安装更新

在已保存的配置中定义更新操作，点击安装更新，选择本地目录或 ZIP / 7z 文件。按声明顺序执行：

| 操作 | 参数 | 含义 |
| --- | --- | --- |
| `PIN` | 必填，相对 content 的路径 | 清空时保留该文件或目录，包括空目录 |
| `RMC` | 无 | 清除未固定的旧内容 |
| `UNPKG` | 无 | 解包或复制所选目录内容，覆盖安装内容 |

更新前完整复制 `content`；新备份复制成功后才替换旧的 `content.bak`。备份失败时不修改原内容；更新执行失败时保留备份，并在错误中显示恢复路径。成功后询问是否删除备份。更新过程中禁止退出，避免中断文件操作。

目录来源直接复制其内容，保留目录层级；更新前先暂存来源、再完整备份，避免清空步骤破坏来源。来源不能位于 content / content.bak 内，或包含这些目录。更新完成后可保留备份，或点击删除备份。

ZIP 仅使用 Go 标准库 `archive/zip`，7z 仅使用编入 EXE 的 `github.com/bodgit/sevenzip`；不查找或调用外部解压工具。解包先验证路径和文件类型，再写入独立暂存目录；只有一个顶层目录时自动去掉包装层，再覆盖到 `content`。拒绝越界路径和符号链接；安装包应放在 `content` 目录之外。当前不提供加密压缩包的密码输入。

## 验证

```powershell
go test ./...
go vet ./...
npm --prefix frontend test
npm --prefix frontend run check
wails3 build
```

Go 测试使用独立临时应用库，覆盖扫描容错、UUID、只读配置、保存、最近使用、入口、备份、PIN 路径边界、ZIP / 7z、包装目录、覆盖安装和失败保留备份。前端测试覆盖筛选排序、时间边界、草稿隔离和更新参数校验。

无界面扫描自检：

```powershell
.\bin\app-runner.exe --app-dir "D:\Apps" --selftest
```

报告写入当前工作目录的 `selfcheck/selftest.json`。自检不会启动应用，但和正常扫描一样会补齐可写配置的 UUID。

正式运行和发布只需要 `bin/app-runner.exe`。浏览器集成测试按需编译服务端 `app-runner-ui-test.exe`，放在 `selfcheck/test-bin/`。原生窗口测试可直接使用正式 EXE，在独立的测试缓存中运行。

浏览器集成测试通过 Wails 的本地 server transport 调用真实 Go 服务，使用自行创建的测试应用库；服务仅用于验证：

```powershell
New-Item -ItemType Directory -Force selfcheck/test-bin | Out-Null
go build -tags server -o selfcheck/test-bin/app-runner-ui-test.exe .
npm --prefix frontend install --no-save playwright
# Windows 上使用已安装的 Edge；其他平台需要 npx playwright install chromium
$env:NODE_PATH = (Join-Path $PWD "frontend\node_modules")
node scripts/ui-test.cjs
```

集成报告输出到 `selfcheck/`。运行测试前应完成前端构建。

原生窗口测试脚本：`powershell -File scripts/native-test.ps1`，验证启动、关闭后驻留和第二实例唤回。

本次验证记录：Go 测试（包括 7z 内置解包）、静态检查、前端测试、类型检查、Windows 发布构建和浏览器集成测试通过。原生测试也已在普通权限下通过：WebView2 窗口与前端加载、关闭后驻留托盘、第二实例唤回；`selfcheck/native-test.json` 中 `elevated` 为 `false`。

### Windows 启动与缓存

进一步排查确认：修复前，本机项目目录带有可继承的 `Low Mandatory Level` 标签，构建的 EXE 也继承了 Low 标签。从 Medium 完整性的资源管理器直接启动后，启动器实际进程完整性为 `S-1-16-4096`（Low）；原来的 AppData 缓存没有完整性标签，Windows 将其视为 Medium。Windows 的强制完整性控制禁止 Low 进程写入 Medium 目录，即使当前账号拥有完全控制权限。参见 [Microsoft Mandatory Integrity Control](https://learn.microsoft.com/en-us/windows/win32/secauthz/mandatory-integrity-control)。

对照测试使用同一份诊断 EXE 和原来的 AppData 缓存，二进制 SHA256 不变，仅在诊断副本上临时将文件标签从 Low 调整为 Medium：Low 启动失败，Medium 无需管理员权限即可启动。对照测试结束时已恢复诊断副本标签，该阶段未改正式 EXE 和工作目录标签。随后已按用户要求将整个 AppStart 项目恢复桌面的正常 ACL 继承和 Medium 完整性：1803 个条目处理成功，无残留 Low 标签。恢复后，未提权的 Medium 进程使用原来的 AppData 缓存启动成功；验证报告位于 `selfcheck/permissions-restored-test.json`。原权限备份位于 `selfcheck/permissions-before-20261012-012914/`。日志见 `selfcheck/integrity-low-stderr.log` 和 `selfcheck/integrity-medium-stderr.log`。

已移除之前的便携缓存、目录回退和管理员缓存隔离逻辑，Windows 版恢复使用 Wails 的默认缓存 `%APPDATA%/app-runner.exe`。应用清单保持 `asInvoker`。原生测试同时检查启动器进程为 Medium 完整性且没有管理员提权，避免将 Low 进程误判为普通桌面进程。
原生测试等待前端运行时就绪后再检查关闭与唤回，诊断日志位于 `selfcheck/native-stderr.log`。当前 Wails Go WebView2 loader 会清空 `WEBVIEW2_USER_DATA_FOLDER` 和 `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS`，因此应通过 `application.WindowsOptions` 配置缓存或调试参数。

## 代码结构

```text
main.go                       入口、窗口、托盘、单实例、自检
launcherservice.go             Wails 服务、原生对话框和文件选择
internal/launcher/            扫描、配置、启动、使用状态、更新与测试
frontend/src/main.ts           界面和交互
frontend/src/state.ts          筛选、排序、草稿和验证
frontend/src/style.css         界面样式
frontend/bindings/             Wails 自动生成的类型绑定
scripts/ui-test.cjs            真实 Go 服务的浏览器集成测试
build/                        图标、Windows 资源、Wails 构建配置
```

Wails v3 的 API 参考：[Application API](https://v3.wails.io/reference/application/)、[安装说明](https://v3.wails.io/quick-start/installation/)。

