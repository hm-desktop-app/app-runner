# App Runner 代码评审

## 元信息

| 项目 | 内容 |
| --- | --- |
| 仓库路径 | `S:\local\repos\app-runner` |
| 完整提交 | `130f899b1aa659613e42dbbd4ffdf2cd5d484df7` |
| 分支 | `main` |
| 评审开始时间 | 2026-11-13T01:31:14+08:00（Asia/Shanghai, UTC+8） |
| 评审范围 | 全项目评审 + 变更级评审；变更范围为 `67f303c..130f899` |
| 重点提交 | 130f899、8de18de、4a39c72、1daf0dc、566d579、7243db7、f99edfc、d4792da、30fe166、cb799d7 |
| 起始工作区状态 | 干净：`git status --porcelain` 无输出，HEAD = 130f899 |
| 评审期间工作区变化 | 未修改任何受版本控制的文件；仅新增未跟踪的 `code-reviews/`，以及被 .gitignore 忽略的 `selfcheck/` 测试产物。评审结束时 `git status --porcelain` 仅有 `?? code-reviews/` |
| 行为契约 | `README.md` + `AGENTS.md`；`DESIGN.md` 不存在（已确认） |
| 历史评审 | `code-reviews/` 此前不存在，本目录是首轮评审，没有上一轮修复项需要复核 |

评审方式：先读 README/AGENTS 与全部产品代码，再逐提交检查 `67f303c..130f899`；对可复现的问题用真实 Go 服务（`go build -tags server`）+ Playwright/Edge 探针脚本验证（脚本见 `scripts/`）。未安装任何全局包。

## 变更级评审结论

| 提交 | 结论 |
| --- | --- |
| 130f899 清空窗口激活时的搜索 | 实现与提交说明一致，UI 集成测试覆盖；未发现功能性缺陷。但触发范围过宽，见 F8（P3）。真实 WebView2 激活是否派发 DOM focus 事件未能验证，见「待确认」。 |
| 8de18de / 7243db7 发布资产命名 | workflow 与 AGENTS.md 一致：资产固定为 `app-runner-windows-amd64.exe`；沙箱内 `gh release view v0.2.1` 显示已发布资产确为无版本名。未发现缺陷。 |
| 4a39c72 版本号 0.2.1 | `build/config.yml` 与 `build/windows/info.json` 两处一致；未发现缺陷。 |
| 1daf0dc 拼音搜索 | 行为正确（单测 + UI 测试均通过），但文档未同步、与命令中心排序实现分叉，见 F5、F6。 |
| 566d579 记录仓库约定 | AGENTS.md 内容与仓库现状基本一致，仅 CI 描述遗漏 `go vet`，见 F6。 |
| f99edfc 忙碌时禁用对话框控件 | 机制在真实 UI 测试中生效；但覆盖不到编辑面板（非 `<dialog>`），见 F3；新增测试断言消息写反，见 F9。 |
| d4792da 忙碌指示器置于顶层 | popover 实现被 UI 测试覆盖（安装期间可见、结束后关闭），未发现缺陷。 |
| 30fe166 预览面板展示相关链接 | 展示与打开正常；但与 cb799d7 的过滤规则组合后，空 URL 链接会被暴露为可点击按钮，见 F2。 |
| cb799d7 相关链接卡片列表 | 交互符合提交说明；但"未填完的行不落盘"只实现了"两栏都空才丢弃"，见 F2。 |

## 发现

### [P2][F1] README 记载的"拖到「＋ 新建分类」新建并移入"并未实现，相关代码是死路径

- 位置：[README.md:104](../../README.md#L104)（操作说明）、[frontend/src/main.ts:269](../../frontend/src/main.ts#L269)、[frontend/src/main.ts:287](../../frontend/src/main.ts#L287)、[frontend/src/main.ts:446-464](../../frontend/src/main.ts#L446-L464)、[frontend/src/main.ts:579-583](../../frontend/src/main.ts#L579-L583)
- 触发条件：在应用库视图把某个应用行拖到侧栏底部的「＋ 新建分类」按钮上。
- 实际结果：没有任何反应——不弹新建分类对话框，不创建分类，`app.toml` 不变。只有分类项本身实现了 `ondragover/ondrop`（main.ts:277-284），`#new-category` 按钮只绑定了 `onclick`（main.ts:287），document 级 `dragover`（main.ts:579-583）也只对 `[data-filter^="category:"]` 调用 `preventDefault()`，因此浏览器不会把它当作合法放置目标。
- 证据：探针 `scripts/probe-ui.cjs` 对真实 Go 服务执行 `dragTo('#new-category')` 后：`category-dialog[open]` 计数 0、分类项计数 0、alpha 的 `app.toml` 前后完全一致。
- 连带影响：`openCategoryDialog(id = '')` 的 `id` 参数从未被传入过（唯一调用点 main.ts:287 不传参），因此 `pendingCategoryApp`（main.ts:446）、main.ts:450 的"将…移入新分类"文案、main.ts:464 的 `changeCategory(pendingCategoryApp, name)` 分支都是不可达的死代码——这正说明该交互原本设计过但没接上。
- 修复建议：在 `renderFilters()` 里为 `#new-category` 绑定 `ondragover`（有 `draggedID` 时 `preventDefault`）与 `ondrop`（`openCategoryDialog(draggedID)`），复用现有管道；或者从 README 删除该描述，并删除 `openCategoryDialog` 的 `id` 参数与 `pendingCategoryApp` 死代码。

### [P2][F2] 只填名称的相关链接会被保存，随后在命令中心预览里成为必然报错的按钮

- 位置：[frontend/src/detail.ts:229](../../frontend/src/detail.ts#L229)（保存过滤）、[frontend/src/detail.ts:105](../../frontend/src/detail.ts#L105)（编辑面板对空 URL 禁用"打开"）、[frontend/src/main.ts:202-204](../../frontend/src/main.ts#L202-L204)（预览面板无条件渲染链接按钮）、[frontend/src/main.ts:231-233](../../frontend/src/main.ts#L231-L233)（点击打开）、[launcherservice.go:66-76](../../launcherservice.go#L66-L76)（OpenLink 校验）
- 触发条件：编辑应用 → 相关链接 → 「添加链接」，只填"链接名称"、留空"链接地址" → 保存；然后在命令中心选中该应用，点击预览面板里的这条相关链接。
- 实际结果：
  1. 保存时 `draft.relatedLinks.map(...).filter(link => link.title || link.url)` 只丢弃"两栏都空"的行，只有名称的行被写进 `app.toml`（探针实测写入 `[[related_links]] title = '只有名字' url = ''`）。
  2. 编辑面板对这条链接的"打开"按钮是禁用的（detail.ts:105），但命令中心预览面板对所有 `relatedLinks` 无条件生成 `data-open-url` 按钮（实测 `.peek-link` 计数 1，`dataset.openUrl === ''`）。
  3. 点击后 `OpenLink('')` 会把地址补成 `https://` 再解析，Host 为空 → 返回"请输入有效的 HTTP 或 HTTPS 链接"（探针实测），前端 `perform()` 随即弹出原生"操作失败"对话框。
- 与提交说明的差距：cb799d7 的说明是"untouched rows are dropped on save"，但实现只丢弃完全空行；30fe166 把链接搬到预览面板后，这条坏数据第一次有了可点击入口。
- 修复建议：保存时要求链接同时具备名称与地址（`filter(link => link.title && link.url)`），或在预览面板跳过/禁用 `url` 为空的链接（与 detail.ts:105 的行为保持一致）。

### [P3][F3] 忙碌态没有覆盖编辑面板：`保存`/`创建应用` 可重入，双击会产生两次并发服务调用，并提前清掉忙碌指示

- 位置：[frontend/src/main.ts:103-115](../../frontend/src/main.ts#L103-L115)（`setDialogBusy` 只处理 `dialog`）、[frontend/src/detail.ts:226-241](../../frontend/src/detail.ts#L226-L241)（`save` 无重入保护、不禁用 `#sheet-save`）、[frontend/src/detail.ts:348-361](../../frontend/src/detail.ts#L348-L361)（`submitCreate` 同样无保护）、[frontend/src/main.ts:93-102](../../frontend/src/main.ts#L93-L102)（`busy` 是布尔量而非计数）
- 触发条件：编辑应用后快速双击「保存」（或在保存途中再按 Ctrl+S）／新建应用第 2 步双击「创建应用」。
- 实际结果：第一次调用把 `busy=true` 后，按钮仍可点击，第二次 `save()` 通过 `dirty()` 检查再次发起 `API.Save`；探针 `scripts/probe-dblclick.cjs` 实测双击保存产生 2 个并发的 Save 请求（同一 app、同一内容，无数据损坏，但会重复 toast、重复 `ctx.changed`）。任一调用结束都会执行 `ctx.busy(false)`，因此先结束的那次会把全局忙碌指示提前关掉，后一次仍在写盘。同类问题：保存过程中按 Esc 会走 `requestClose` → 原生"有未保存的更改，确定放弃吗？"，确认后界面关闭但已发出的保存仍会提交并提示"已保存"。
- 说明：f99edfc 只把真实 `<dialog>` 内的控件设为禁用，编辑面板是 `<aside id="sheet" role="dialog">`（frontend/index.html:57），不在该机制覆盖范围内，所以没有兜住这条路径。
- 修复建议：`save` / `submitCreate` 开头加 `if (busy) return;`（与 `launch`/`changeCategory` 一致），或把忙碌控制扩展到 `#sheet-layer`（禁用 sheet 内控件 / 拦截 Esc），或把 `busy` 改成计数并在 `setBusy(false)` 时只在计数归零后清除。

### [P3][F4] 残留死代码与只被测试使用的生产代码

- 位置与证据：
  - [internal/launcher/store.go:251-265](../../internal/launcher/store.go#L251-L265) `Store.RefreshApp`：全仓库无任何调用点（仅定义）。
  - [frontend/src/state.ts:65-69](../../frontend/src/state.ts#L65-L69) `age()`：产品代码从不使用，只被 [frontend/src/state.test.ts:3](../../frontend/src/state.test.ts#L3) 引入并测试。属于"仅为测试存在的生产代码"。
  - [frontend/src/ui.ts:7](../../frontend/src/ui.ts#L7) `dateTime`：全仓库无引用。
  - [frontend/index.html:68](../../frontend/index.html#L68) `#context-menu` 元素：JS 中无任何引用。
  - [frontend/src/style.css:382](../../frontend/src/style.css#L382) `.hotkey-hint`、[frontend/src/style.css:385](../../frontend/src/style.css#L385) `.brand-name`：只出现在 media query 里，模板残留、无对应元素。
  - 分类新建死路径（`pendingCategoryApp`/`openCategoryDialog(id)`）见 F1。
- 影响：增加阅读与维护成本，容易让后续修改误以为这些分支仍然可达。
- 修复建议：删除无引用项；`age()` 若确无产品用途，连同测试一起删除（或改用真实的筛选/状态逻辑覆盖同样的边界）。`RefreshApp` 若本来是给 F5 刷新用的，应接入 `Scan` 之外的增量刷新路径，否则删除。

### [P3][F5] 命令中心与库内搜索是两套逻辑，已出现字段/排序分叉，且 pinyin 逻辑两处各写一遍

- 位置：[frontend/src/state.ts:29-32](../../frontend/src/state.ts#L29-L32)（库搜索用的 `searchText`，字段：标题、分类、备注、标题/分类拼音）、[frontend/src/state.ts:121-133](../../frontend/src/state.ts#L121-L133)（命令中心 `score`，额外匹配 `entry` 且带排序）、[README.md:90](../../README.md#L90)（约定搜索字段为名称、分类、备注）
- 触发条件与差异：
  - 命令中心输入入口文件名（如 `app.exe`、`editor`）会命中应用（score 第 6 档，state.ts:131），库内搜索同样输入却不会命中——README 并未声明入口参与搜索。
  - 命令中心对命中分档排序（精确>前缀>包含>拼音>分类>备注），库内搜索只过滤不排序；1daf0dc 需要同时改两处才能保持"拼音可搜"这一事实一致，后续很容易只改一处。
- 影响：行为不一致且缺少单一事实来源；属于"同一功能多套实现"。
- 修复建议：抽出统一的匹配键（`title/category/description/entry + pinyin keys`）与可选的打分函数，库搜索复用同一匹配集合；若决定保留入口匹配，请补进 README。

### [P3][F6] README / AGENTS 与当前实现不一致（文档是行为契约）

- 位置：
  - [README.md:90](../../README.md#L90)：搜索仍写"按名称、分类和备注过滤"，未提 1daf0dc 新增的拼音全拼/首字母；[frontend/index.html:16](../../frontend/index.html#L16) 与 [frontend/index.html:34](../../frontend/index.html#L34) 的搜索占位文案同样未提，用户无从发现该功能。
  - [README.md:84](../../README.md#L84)：临时文件清单不完整/不准确——实际还会产生 `content` 更新的暂存目录 `.content-source-*`（[internal/launcher/update.go:60](../../internal/launcher/update.go#L60)）与解包暂存目录 `.content-unpack-*`（[internal/launcher/update.go:272](../../internal/launcher/update.go#L272)），而 `.app-runner-*` 是 `atomicWrite` 生成的临时**文件**（[internal/launcher/store.go:285](../../internal/launcher/store.go#L285)），文档写成"中间目录"。
  - [AGENTS.md:28-29](../../AGENTS.md#L28-L29)：写"CI 只做构建、Go 测试与前端单测"，实际 [.github/workflows/build.yml:51-55](../../.github/workflows/build.yml#L51-L55) 还执行了 `go vet ./...`。
- 影响：维护者/使用者按文档操作会得到错误预期；本仓库 AGENTS.md 明确"文档只描述当前实现"。
- 修复建议：补一句拼音搜索说明（README 操作表 + 两个 placeholder），补全临时目录清单，修正 CI 描述。

### [P3][F7] 异常退出遗留的 `.app-runner-new-*` 暂存目录永远不会被清理（磁盘泄漏）

- 位置：[internal/launcher/create.go:101-110](../../internal/launcher/create.go#L101-L110)（暂存目录建在应用库根下）、[internal/launcher/create.go:223-233](../../internal/launcher/create.go#L223-L233)（`CleanupImports` 只遍历内存里的 `s.imports`）、[main.go:65](../../main.go#L65)（只在正常退出时调用）、[internal/launcher/store.go:64](../../internal/launcher/store.go#L64)（扫描时静默跳过该前缀）
- 触发条件：新建应用走到第 2 步（文件已复制进 `.app-runner-new-*`），此时进程被强杀/崩溃（或系统重启），之后再次启动。
- 实际结果：该暂存目录（可能是整份应用内容的副本）永久留在应用库中；下次启动 `CleanupImports` 的内存表为空，`Scan` 又按前缀跳过，没有任何代码会回收它。README.md:84/108 声称"退出时清理暂存"，但异常退出路径没有兜底。
- 修复建议：启动扫描时顺带列出应用库根下匹配 `.app-runner-new-*` 的目录并按安全规则（`inside(library, …)` + 前缀校验，已有类似逻辑）删除，或为每个暂存目录写一个超时标记；同时把 README 的措辞改成"正常退出时清理"。

### [P3][F8] 130f899 的搜索清空在库视图下触发范围过宽：仅切回窗口就会清掉正在输入的查询

- 位置：[frontend/src/main.ts:734-738](../../frontend/src/main.ts#L734-L738)（任何 window `focus` 都调用 `clearSearch()`）、[frontend/src/main.ts:79-92](../../frontend/src/main.ts#L79-L92)、[frontend/src/main.ts:620-627](../../frontend/src/main.ts#L620-L627)（失焦自动隐藏只在命令中心生效，库视图失焦时窗口保持可见）
- 触发条件：命令中心 → 打开应用库 → 在库搜索框输入关键词 → 点击别的窗口再切回 App Runner。
- 实际结果：窗口始终未隐藏、用户也没有"重新唤出"，但库视图的搜索词被静默清空（150ms 后表格恢复为全量列表），正在进行的筛选丢失。命令中心因为失焦会隐藏，问题不明显；库视图是刻意保持可见的，因此这条路径比提交说明中的"唤出时清空"更宽。
- 说明：该行为与提交说明"by hotkey or focus … in both the command center and the library"一致，所以按 UX 折衷记为 P3；若产品确实希望"切回窗口也重置"，可忽略。
- 修复建议：只在实际"隐藏→显示"时清空（例如 Go 侧 `show()` 通过事件通知前端，或前端用 `document.visibilityState`/标记位区分失去焦点与隐藏），或在库视图下不清空。

### [P3][F9] f99edfc 新增的 UI 测试断言消息与实际断言相反

- 位置：[scripts/ui-test.cjs:238](../../scripts/ui-test.cjs#L238)、[scripts/ui-test.cjs:245-247](../../scripts/ui-test.cjs#L245-L247)，以及 [scripts/ui-test.cjs:211](../../scripts/ui-test.cjs#L211)
- 触发条件：断言失败时阅读输出。
- 实际结果：消息与断言语义相反，例如断言 `#update-install` 处于禁用态却写 `'the install button stays enabled while installing'`，断言输入框可用却写 `'the source field starts disabled'`，断言空链接的打开按钮禁用却写 `'a blank link can be opened'`。失败时会把人引向错误结论。
- 修复建议：改正这些消息（例如 `'the source field must be disabled while installing'`、`'a blank link must not be openable'`）。

## 待确认

1. **130f899 的真实窗口激活事件**：自动化测试（scripts/ui-test.cjs:93、357）是 `window.dispatchEvent(new Event('focus'))` 合成事件，不能证明 WebView2 在热键/托盘/第二实例唤回、以及 `window.Hide()→Show()` 后真的派发 DOM `focus` 事件；也不能证明反向情况（只是 Alt-Tab 切回也派发）。这决定 F8 的触发频率与 130f899 是否真的生效，需要手工或原生测试验证。`scripts/native-test.ps1` 也不覆盖该行为。
2. **`LauncherService.Confirm` 依赖第三方实现的时序**：[launcherservice.go:80-93](../../launcherservice.go#L80-L93) 用 `select { case … default: return false }` 非阻塞取值。当前 Wails v3.0.0-beta.28 的 Windows 实现是在 `MessageBox` 返回后同步调用按钮回调（模块缓存 `pkg/application/dialogs_windows.go:78-85`），所以现在成立；但它依赖库内部实现，换版本/平台后若回调改为异步，所有确认框（退出、放弃更改、放弃新建）都会恒返回 false。建议改为等待回调（可带超时）以消除隐式依赖。
3. **发布资产状态**：`gh release view v0.2.1` 显示已发布资产为无版本名 `app-runner-windows-amd64.exe`，但 v0.2.1 标签指向的提交（4a39c72）中的 workflow 仍使用 `app-runner_<version>_windows_amd64.exe`；推测发布后做过人工改名/重传（8de18de 在发布之后）。不影响后续 tag 的发布，仅记录无法从仓库状态自证的部分。
4. **未运行的验证**：`wails3 build`（完整产物/图标/版本资源）、`npm --prefix frontend run build`（Vite 生产打包，本轮只跑了 `tsc --noEmit`）、`scripts/native-test.ps1`（原生窗口/托盘/单实例）——这些需要 Wails CLI/原生 GUI，会打开真实窗口，本轮未执行；UI 集成测试使用 server 构建 + Edge headless 验证。

## 验证记录（实际执行）

在仓库根目录、HEAD=130f899、工作区干净时执行：

| 命令 | 结果 |
| --- | --- |
| `go vet ./...` | exit 0，无输出 |
| `go test ./...` | exit 0：`app-runner`（no test files）、`app-runner/internal/launcher` ok 2.247s、`app-runner/internal/singleinstance` ok（cached） |
| `npm --prefix frontend test` | exit 0：11 tests / 11 pass / 0 fail |
| `npm --prefix frontend run check` | exit 0（`tsc --noEmit`） |
| `go build -tags server -o selfcheck/test-bin/app-runner-ui-test.exe .` | exit 0 |
| `$env:NODE_PATH=frontend\node_modules; node scripts/ui-test.cjs` | exit 0，输出 `OK: all UI checks passed` |
| `node code-reviews/…/scripts/probe-ui.cjs` | 见 F1/F2：title-only 链接写入 `url = ''`；预览面板出现 `data-openUrl === ''` 的链接；拖到「＋ 新建分类」无任何变化 |
| `node code-reviews/…/scripts/probe-dblclick.cjs` | 双击「保存」产生 2 个并发 Save 服务调用（不同 call-id），见 F3 |
| `node code-reviews/…/scripts/probe-description.cjs` | 对照验证：正常保存备注会正确落盘（排除"编辑面板保存异常"的疑点） |
| `node code-reviews/…/scripts/probe-pinyin.cjs` | 输出 pinyin-pro 3.29.5 的数组/全拼/首字母，确认拼音索引行为与 state.ts 实现一致 |
| 临时 Go 探针（已删除，仅记录结果） | 复刻 `OpenLink` 校验：`OpenLink("")` 与 `OpenLink("   ")` 均返回"请输入有效的 HTTP 或 HTTPS 链接"，佐证 F2 |

测试/探针产物均位于被 gitignore 的 `selfcheck/` 与本次评审目录内，未改动产品/测试代码，未提交任何内容。
