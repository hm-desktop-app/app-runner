# App Runner 评审响应

## 元信息

| 项目 | 内容 |
| --- | --- |
| 对应评审 | [`code-reviews/2026-11-13T01-31-14+0800_130f89/REVIEW.md`](REVIEW.md) |
| 评审对象提交 | `130f899b1aa659613e42dbbd4ffdf2cd5d484df7` |
| 响应时间 | 2026-11-13T01:57:52+08:00（Asia/Shanghai, UTC+8） |
| 修复者 | lead（fixer） |
| 修复范围 | F1–F9 全部处理；本轮无遗留未修复的评审项 |

修复后工作区：`REVIEW.md`、本文件与 `scripts/` 探针脚本一并纳入版本库，生产与测试代码的改动与评审记录同一个提交。

## 逐项响应

### [P2][F1] 拖到「＋ 新建分类」新建并移入

- 位置：`frontend/src/main.ts`（`renderFilters`）、`frontend/src/style.css`
- 修复方式：为 `#new-category` 绑定 `ondragover`（存在 `draggedID` 时 `preventDefault` 并高亮）、`ondragleave` 与 `ondrop`；放置时以被拖行的 ID 调用已有的 `openCategoryDialog(id)`，提交对话框时复用 `changeCategory(pendingCategoryApp, name)` 管道。原先不可达的 `pendingCategoryApp` 路径因此恢复可达，无需删除。`.filter-add.drop-target` 复用分类项的放置高亮样式。
- 验证：浏览器集成测试新增用例——把 `Alpha Changed` 行拖到「＋ 新建分类」，断言对话框文案为“将“Alpha Changed”移入新分类”、创建后 `app.toml` 的分类变更、侧栏出现新分类项。测试中的拖放经 `dragUntil` 辅助函数最多重试 3 次：浏览器合成拖放在负载下偶发丢事件，重试只影响自动化稳定性，处理器本身回归时三次均失败、断言仍会报错。

### [P2][F2] 只填名称的相关链接被保存后成为必然报错的按钮

- 位置：`frontend/src/detail.ts`（`save`）、`frontend/src/main.ts`（`renderPeek`）
- 修复方式：保存时只保留有地址的链接（`.filter(link => link.url)`），与编辑面板“地址为空则禁用打开”的规则一致；历史 `app.toml` 中已存在的空地址链接在命令中心预览面板中被跳过，不再渲染成可点击按钮。
- 验证：UI 测试新增两段断言——只填名称的链接行保存后 `[[related_links]]` 数量不变；向 `app.toml` 注入空地址链接并刷新后，预览面板不渲染 `data-open-url=""` 的按钮，有效链接数量不变。修复前同一用例在旧构建上报 3 条链接（已实际复现）。

### [P3][F3] 忙碌态未覆盖编辑面板，保存可重入并提前清除忙碌指示

- 位置：`frontend/src/detail.ts`、`frontend/src/main.ts`
- 修复方式：`SheetContext` 增加 `isBusy()`，`save`、`prepare`、`submitCreate` 在忙碌时直接返回（与 `launch`/`changeCategory` 一致），从源头避免第二次并发服务调用与提前 `busy(false)`；`requestClose` 在忙碌时返回 `false`，Esc／遮罩点击不再在保存途中弹出“放弃更改”并关面板。
- 验证：UI 测试将保存 RPC 延迟 700 ms 后双击「保存」，断言只发出 1 个 Save 请求、保存期间 Esc 不关闭面板。反向验证：临时移除 `isBusy()` 守卫后，同一用例报告“a double click on save issued 2 saves”，确认测试能捕获该缺陷。

### [P3][F4] 残留死代码与只被测试使用的生产代码

- 位置：`internal/launcher/store.go`、`frontend/src/state.ts`、`frontend/src/state.test.ts`、`frontend/src/ui.ts`、`frontend/index.html`、`frontend/src/style.css`
- 修复方式：删除无调用点的 `Store.RefreshApp`、产品代码未使用的 `age()` 及其测试、`ui.dateTime`、`#context-menu` 元素、`.hotkey-hint` 与 `.brand-name` 样式；`pendingCategoryApp`／`openCategoryDialog(id)` 因 F1 接通实时交互而保留。
- 验证：`go vet`、`tsc --noEmit`、单元测试与 UI 测试均通过；全仓库已无上述符号引用。

### [P3][F5] 命令中心与库内搜索两套逻辑、字段与排序分叉

- 位置：`frontend/src/state.ts`
- 修复方式：抽出唯一匹配入口 `matches()` 与统一字段集 `searchText()`（名称、分类、备注、入口，以及名称与分类的拼音全拼/首字母）；`visibleApps`（库与命令中心共用）与 `rankApps` 都用它决定“是否命中”，`score()` 只负责命令中心的排序。命令中心原有的入口匹配成为两端一致的行为，并补进 README。
- 验证：新增单测断言 `visibleApps` 与 `rankApps` 对同一查询返回相同应用集合（含入口、拼音、分类与无命中查询），确保两处不会再出现字段或排序口径分叉。

### [P3][F6] README / AGENTS 与当前实现不一致

- 位置：`README.md`、`AGENTS.md`、`frontend/index.html`
- 修复方式：搜索说明补充拼音全拼/首字母与入口字段，两个搜索框占位文案同步；临时文件清单补全 `.content-source-*`、`.content-unpack-*`，并修正 `.app-runner-*` 是原子写入临时文件而非目录；补充“窗口重新激活清空命令中心搜索”和行为与暂存清理时机；AGENTS 的 CI 描述补上 `go vet`。
- 验证：逐条对照本次改动与 `.github/workflows/build.yml` 的实际内容。

### [P3][F7] 异常退出遗留的 `.app-runner-new-*` 暂存目录永不回收

- 位置：`internal/launcher/create.go`、`internal/launcher/store.go`
- 修复方式：`Scan` 时调用 `reclaimStaging`：应用库根下匹配 `.app-runner-new-` 的目录（跳过符号链接）在通过 `inside()` 边界校验后删除；仍在 `s.imports` 中的待提交暂存目录按目录名排除，不会误删当前会话的新建草稿。README 相应说明“取消、正常退出或下次启动时清理”。
- 验证：新增 Go 测试 `TestScanReclaimsCrashedStagingWithoutTouchingPendingImports`：伪造崩溃遗留目录后 `Scan` 予以回收、不出现为应用；随后 `PrepareNewApp` 的待提交暂存经 `Scan` 后仍可 `CheckNewEntry`。

### [P3][F8] 窗口激活时清空搜索的范围过宽，会清掉库视图正在输入的查询

- 位置：`frontend/src/main.ts`
- 修复方式：`clearSearch` 只在命令中心生效；应用库视图（失焦时窗口保持可见）不再因切回窗口而丢失查询，仅 Esc 清空。
- 验证：UI 测试改为在库视图输入查询后派发窗口 focus 事件，断言查询保留、搜索框获得焦点；命令中心原有的激活清空断言保留。

### [P3][F9] 新增 UI 测试断言消息与实际断言相反

- 位置：`scripts/ui-test.cjs`
- 修复方式：改正 4 条反转消息（配置链接可打开、空链接不可打开、更新期间来源/安装/取消按钮应禁用），并把一处误传为 `waitForFunction` 参数的失败消息改为真正的消息参数。
- 验证：逐条核对断言表达式与消息语义；测试全部通过。

## 待确认项处理

1. **真实 WebView2 窗口激活事件**：仍未能自动验证。UI 测试使用合成的 `window.dispatchEvent(new Event('focus'))`，只证明前端逻辑；真实热键/托盘/第二实例唤回是否派发 DOM `focus` 以及 Alt-Tab 切回的频率，需要手工或原生测试确认。本轮未执行 `scripts/native-test.ps1`（会打开真实窗口）。F8 已把清除范围收窄到命令中心，即使该事件在切回窗口时触发，影响也限于命令中心。
2. **`LauncherService.Confirm` 的非阻塞 `select`**：本轮未改动。改为阻塞等待（可带超时）会改变退出/放弃更改/放弃新建的确认语义，且当前 Wails `v3.0.0-beta.28` 的 Windows 实现确实在 `MessageBox` 返回后同步调用按钮回调；保留现状，若后续升级 Wails 版本需重新核对。
3. **发布资产 provenance**：无需代码动作，评审记录中的推测保留。
4. **未执行的验证**：`wails3 build` 在本机 shell 下不可用（`sh` 不在 PATH，go-task 的 `sh: npm version` 前置条件失败；同时 `wails3 generate icons` 尝试生成不存在的 `build/darwin/icon.icns` 后失败），与本次改动无关；改用等价的 `wails3 generate syso` + `go build -tags production` 验证生产构建通过。`scripts/native-test.ps1` 未执行。

## 构建与测试结果

在仓库根目录、修复后的工作区执行：

| 命令 | 结果 |
| --- | --- |
| `gofmt -l .` | 无输出 |
| `go vet ./...` | exit 0 |
| `go test ./...` | exit 0：`internal/launcher` ok，新增暂存回收测试通过 |
| `npm --prefix frontend test` | exit 0：11 tests / 11 pass（原 `age()` 边界测试移除，新增统一搜索测试，数量不变） |
| `npm --prefix frontend run check` | exit 0（`tsc --noEmit`） |
| `npm --prefix frontend run build` | exit 0（Vite 生产构建，364.01 kB / gzip 168.14 kB） |
| `go build -tags server -o selfcheck/test-bin/app-runner-ui-test.exe .` | exit 0 |
| `node scripts/ui-test.cjs` | exit 0，`OK: all UI checks passed`，新增 F1/F2/F3/F8 用例后连续 6 次通过 |
| `wails3 generate syso …` + `go build -tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui" -o bin/app-runner.exe .` | exit 0，产物 21,030,912 B |

说明：

- 浏览器集成测试中的 HTML5 拖放在负载下偶发丢失（与业务逻辑无关），两次拖拽前各加 200 ms 稳定等待，并引入 `dragUntil` 重试辅助函数；最终连续 6 次运行全部通过。
- 浏览器集成测试会启动真实 Go server（server 构建）并用 Edge headless 驱动；测试库位于被 gitignore 的 `selfcheck/`。
