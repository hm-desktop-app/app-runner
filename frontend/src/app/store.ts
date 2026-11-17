import { reactive } from 'vue';
import { Events } from '@wailsio/runtime';
import * as API from '../../bindings/app-runner/launcherservice';
import {
 categoryOf,
 filterLibrary,
 filterKey,
 groupApps,
 parseFilter,
 rankApps,
 recentApps,
 relativeTime,
 statusOf,
 suggestsUpdate,
 visibleApps
} from '../state';
import type { App, LibraryFilter, SortMode } from '../state';
import type { SettingsInfo, SheetContext, Theme } from '../sheet/state';
import * as sheet from '../sheet/state';
import { $, escape, icons, statusName } from '../ui';

export interface Command {
 id: string;
 name: string;
 sub: string;
 note: string;
 key: string;
 icon: string;
 run: () => void;
}
export interface AppItem {
 kind: 'app';
 key: string;
 app: App;
}
export interface CommandItem {
 kind: 'command';
 key: string;
 command: Command;
}
export type Item = AppItem | CommandItem;
export type Mode = 'command' | 'library';
export interface FilterRow {
 key: string;
 label: string;
 icon: string;
 count: number;
 category: boolean;
 tone?: string;
}

export const shell = reactive({
 apps: [] as App[],
 libraryPath: '',
 mode: 'command' as Mode,
 commandQuery: '',
 libraryQuery: '',
 sort: 'recent' as SortMode,
 filter: { kind: 'all' } as LibraryFilter,
 commandSelected: '',
 librarySelected: '',
 theme: 'light' as Theme,
 autohide: false,
 busy: false,
 loading: false,
 libraryVisited: false,
 extraCategories: [] as string[]
});

let libraryKey = '';
let toastTimer = 0;
let tableTimer = 0;
let draggedID = '';
let categoryCreated: ((name?: string) => void) | null = null;

export type SourceKind = 'archive' | 'directory';
export const overlays = reactive({
 toast: { message: '', visible: false },
 activity: '',
 category: { open: false, appID: '', message: '', name: '', error: '', saving: false },
 group: { open: false, original: '', summary: '', name: '', error: '', saving: false },
 update: { open: false, appID: '', title: '', kind: 'archive' as SourceKind, source: '', error: '', installed: false, picking: false }
});

/* ---------- preferences ---------- */
function loadPreferences(): void {
 try {
  if (localStorage.getItem('app-runner-autohide') === '1') shell.autohide = true;
  if (localStorage.getItem('app-runner-theme') === 'dark') shell.theme = 'dark';
 } catch {}
 document.documentElement.dataset.theme = shell.theme;
}
function saveTheme(value: Theme): void {
 try {
  localStorage.setItem('app-runner-theme', value);
 } catch {}
}
function saveAutohide(value: boolean): void {
 try {
  localStorage.setItem('app-runner-autohide', value ? '1' : '0');
 } catch {}
}
function loadLibraryPreferences(path: string): void {
 if (path === libraryKey) return;
 libraryKey = path;
 shell.extraCategories = [];
 try {
  const saved = JSON.parse(localStorage.getItem('app-runner-categories:' + path) || '{}');
  for (const name of saved.categories || []) if (typeof name === 'string' && name.trim() && name.trim() !== '未分类') shell.extraCategories.push(name.trim());
 } catch {}
}
function saveCategories(): void {
 try {
  localStorage.setItem('app-runner-categories:' + libraryKey, JSON.stringify({ categories: shell.extraCategories }));
 } catch {}
}
export function categoryNames(): string[] {
 return groupApps(shell.apps, '', 'name', new Set(shell.extraCategories))
  .map((group) => group.category)
  .filter(Boolean) as string[];
}
export function suggestedCount(): number {
 return shell.apps.filter((app) => suggestsUpdate(app)).length;
}
export function currentApp(id = shell.librarySelected): App | undefined {
 return shell.apps.find((app) => app.uuid === id);
}

/* ---------- chrome ---------- */
export function setTheme(value: Theme): void {
 shell.theme = value;
 document.documentElement.dataset.theme = value;
 saveTheme(value);
 void API.SetTheme(value === 'dark').catch(console.error);
}
export function setMode(next: Mode): void {
 shell.mode = next;
 if (next === 'library') shell.libraryVisited = true;
 else {
  clearTimeout(tableTimer);
  tableTimer = 0;
 }
 setTimeout(focusSearch, 30);
}
export function focusSearch(): void {
 if (sheet.state.open !== null) return;
 const input = (shell.mode === 'command' ? $('#cmd-search') : $('#lib-search')) as HTMLInputElement | null;
 input?.focus();
 input?.select();
}
export function clearSearch(): void {
 if (shell.mode !== 'command' || !shell.commandQuery) return;
 shell.commandQuery = '';
}
export function setBusy(value: boolean, text = ''): void {
 shell.busy = value;
 overlays.activity = value ? text : '';
}
export function showToast(message: string): void {
 overlays.toast.message = message;
 overlays.toast.visible = true;
 clearTimeout(toastTimer);
 toastTimer = window.setTimeout(() => {
  overlays.toast.visible = false;
 }, 2100);
}
async function perform<T>(action: () => PromiseLike<T>): Promise<T | undefined> {
 try {
  return await action();
 } catch (error) {
  await API.ShowError(error instanceof Error ? error.message : String(error)).catch(console.error);
  return undefined;
 }
}

/* ---------- commands ---------- */
const commands: Command[] = [
 {
  id: 'new',
  name: '新建应用…',
  sub: '从目录或压缩包导入',
  note: '分两步：先选来源，再确认名称与分类。导入后应用复制进库中，可随时编辑。',
  key: 'N',
  icon: icons.plus,
  run: () => openCreate()
 },
 {
  id: 'library',
  name: '打开应用库',
  sub: '浏览、搜索与管理所有应用',
  note: '分类筛选、状态过滤、批量操作与拖拽整理都在这边；按 Esc 或左上角箭头返回命令中心。',
  key: 'L',
  icon: icons.grid,
  run: () => setMode('library')
 },
 {
  id: 'refresh',
  name: '刷新应用库',
  sub: '重新扫描本地应用目录',
  note: '只读扫描，不会修改应用文件；重新读取每个 app.toml 并同步源与本地文件的变化。',
  key: 'F5',
  icon: icons.refresh,
  run: () => void refresh()
 },
 {
  id: 'settings',
  name: '设置',
  sub: '热键、启动与外观',
  note: '改动立即生效并写入配置，不需要重启应用。',
  key: ',',
  icon: icons.settings,
  run: () => openSettings()
 }
];
export function commandItems(): Item[] {
 let query = shell.commandQuery.trim();
 const commandsOnly = query.startsWith('>');
 if (commandsOnly) query = query.slice(1).trim();
 const needle = query.toLocaleLowerCase();
 const recent = recentApps(shell.apps, 5);
 const appList = commandsOnly ? [] : query ? rankApps(shell.apps, query) : recent.length ? recent : visibleApps(shell.apps, '', 'name').slice(0, 5);
 const commandList = commands.filter((command) => !needle || (command.name + ' ' + command.sub).toLocaleLowerCase().includes(needle));
 const items: Item[] = [
  ...appList.map((app): AppItem => ({ kind: 'app', key: 'app:' + app.uuid, app })),
  ...commandList.map((command): CommandItem => ({ kind: 'command', key: 'command:' + command.id, command }))
 ];
 if (!items.some((item) => item.key === shell.commandSelected)) shell.commandSelected = items[0]?.key || '';
 return items;
}
export function commandGroups(): { kind: 'app' | 'command'; label: string; items: Item[] }[] {
 const items = commandItems();
 const query = shell.commandQuery.trim();
 const appItems = items.filter((item): item is AppItem => item.kind === 'app');
 const commandList = items.filter((item): item is CommandItem => item.kind === 'command');
 const groups: { kind: 'app' | 'command'; label: string; items: Item[] }[] = [];
 if (appItems.length) groups.push({ kind: 'app', label: query ? '应用' : recentApps(shell.apps, 1).length ? '最近使用' : '全部应用', items: appItems });
 if (commandList.length) groups.push({ kind: 'command', label: '命令', items: commandList });
 return groups;
}
export function peekItem(): Item | undefined {
 return commandItems().find((item) => item.key === shell.commandSelected);
}
export function execute(key: string): void {
 const item = commandItems().find((candidate) => candidate.key === key);
 if (!item) return;
 if (item.kind === 'command') item.command.run();
 else void launch(item.app.uuid);
}
export function moveCommandSelection(step: number): void {
 const items = commandItems();
 if (!items.length) return;
 const index = items.findIndex((item) => item.key === shell.commandSelected);
 const next = Math.max(0, Math.min(items.length - 1, index + step));
 shell.commandSelected = items[next].key;
}
export function hoverCommand(key: string): void {
 shell.commandSelected = key;
}

/* ---------- library ---------- */
export function libraryRows(): App[] {
 return filterLibrary(shell.apps, shell.filter, shell.libraryQuery, shell.sort);
}
export function filterRows(): { browsing: FilterRow[]; categories: FilterRow[] } {
 const active = filterKey(shell.filter);
 const count = (f: LibraryFilter): number => filterLibrary(shell.apps, f, '', 'name').length;
 return {
  browsing: [
   { key: 'all', label: '全部应用', icon: icons.grid, count: shell.apps.length, category: false },
   { key: 'recent', label: '最近使用', icon: icons.clock, count: count({ kind: 'recent' }), category: false },
   { key: 'suggested', label: '建议更新', icon: icons.update, count: count({ kind: 'suggested' }), category: false }
  ].map((row) => ({ ...row, tone: active === row.key ? 'active' : '' })),
  categories: categoryNames().map((name) => ({
   key: 'category:' + name,
   label: name,
   icon: '',
   count: count({ kind: 'category', name }),
   category: true,
   tone: active === 'category:' + name ? 'active' : ''
  }))
 };
}
export function categoryTone(name: string): string {
 let hash = 0;
 for (const character of name) hash = (hash * 37 + character.codePointAt(0)!) % 360;
 return 'hsl(' + hash + ' 26% 52%)';
}
export function selectFilter(key: string): void {
 shell.filter = parseFilter(key);
}
export function selectApp(id: string): void {
 shell.librarySelected = id;
}
export function moveLibrarySelection(step: number): void {
 const rows = libraryRows();
 if (!rows.length) return;
 const index = rows.findIndex((app) => app.uuid === shell.librarySelected);
 const next = rows[Math.max(0, Math.min(rows.length - 1, (index < 0 ? 0 : index) + step))];
 shell.librarySelected = next.uuid;
}
export function scheduleTable(): void {
 clearTimeout(tableTimer);
 tableTimer = window.setTimeout(() => {
  tableTimer = 0;
 }, 150);
}
export function openLibrary(): void {
 setMode('library');
}
export function goBack(): void {
 shell.commandQuery = '';
 setMode('command');
}
export function hideWindow(): void {
 void API.HideWindow().catch(console.error);
}

/* ---------- data ---------- */
function replaceApp(saved: App): void {
 const index = shell.apps.findIndex((app) => app.uuid === saved.uuid);
 if (index >= 0) shell.apps[index] = saved;
 else shell.apps.push(saved);
}
export async function refresh(): Promise<void> {
 if (shell.busy || shell.loading) return;
 shell.loading = true;
 const data = await perform(() => API.Scan());
 if (data) {
  shell.apps = (data.apps || []) as App[];
  shell.libraryPath = data.library || '';
  loadLibraryPreferences(shell.libraryPath);
  if (shell.librarySelected && !currentApp()) shell.librarySelected = '';
  if (data.warnings?.length) await API.ShowError('应用库已加载，以下配置需要检查：\n' + data.warnings.join('\n'));
 }
 shell.loading = false;
}
export async function launch(id: string): Promise<void> {
 const app = currentApp(id);
 if (!app || shell.busy || shell.loading || !app.entryValid || !['GUI', 'CLI'].includes(app.type)) return;
 setBusy(true, '正在启动应用…');
 const started = (await perform(() => API.Launch(id))) as App | undefined;
 if (started) replaceApp(started);
 setBusy(false);
 if (started && shell.autohide) await API.HideWindow().catch(console.error);
}
export async function changeCategory(id: string, category: string): Promise<boolean> {
 if (shell.busy || shell.loading) return false;
 const app = currentApp(id);
 if (!app) return false;
 category = category.trim() === '未分类' ? '' : category.trim();
 if (categoryOf(app) === category) return true;
 setBusy(true, '正在保存分类…');
 const saved = (await perform(() => API.SetCategory(id, category))) as App | undefined;
 if (saved) {
  replaceApp(saved);
  if (category && !shell.extraCategories.includes(category)) shell.extraCategories.push(category);
  saveCategories();
 }
 setBusy(false);
 return !!saved;
}
export function setSort(value: SortMode): void {
 shell.sort = value;
}

/* ---------- sheet ---------- */
export function openEdit(id: string): void {
 sheet.openEdit(id);
}
export function openCreate(): void {
 sheet.openCreate();
}
export function openSettings(): void {
 sheet.openSettings();
}
const sheetCtx: SheetContext = {
 apps: () => shell.apps,
 categories: () => categoryNames(),
 newCategory: () => new Promise<string | undefined>((resolve) => openCategoryDialog('', resolve)),
 library: () => shell.libraryPath,
 settings: async () => (await API.Settings()) as unknown as SettingsInfo,
 theme: () => shell.theme,
 setTheme: (value) => setTheme(value),
 autohide: () => shell.autohide,
 setAutohide: (value) => {
  shell.autohide = value;
  saveAutohide(value);
 },
 busy: (value, text) => setBusy(value, text),
 isBusy: () => shell.busy,
 changed: (saved) => {
  replaceApp(saved);
  shell.librarySelected = saved.uuid;
 },
 created: (created) => {
  shell.apps.push(created);
  const category = categoryOf(created);
  if (category && !shell.extraCategories.includes(category)) shell.extraCategories.push(category);
  saveCategories();
  shell.librarySelected = created.uuid;
  shell.filter = { kind: 'all' };
  shell.libraryQuery = '';
  setMode('library');
 },
 toast: showToast,
 closed: () => focusSearch(),
 openUpdate: (id) => openUpdateDialog(id)
};
sheet.setContext(sheetCtx);

/* ---------- overlays ---------- */
const errorText = (error: unknown): string => (error instanceof Error ? error.message : String(error));

export function openCategoryDialog(id = '', onCreated?: (name?: string) => void): void {
 if (shell.busy) {
  onCreated?.(undefined);
  return;
 }
 categoryCreated = onCreated || null;
 overlays.category = {
  open: true,
  appID: id,
  message: id ? '将“' + (currentApp(id)?.title || '') + '”移入新分类' : '创建后可以在编辑面板里给应用指定分类',
  name: '',
  error: '',
  saving: false
 };
}
export function closeCategoryDialog(): void {
 overlays.category.open = false;
}
export function categoryDialogClosed(): void {
 const done = categoryCreated;
 categoryCreated = null;
 overlays.category.open = false;
 done?.();
}
export async function submitCategory(): Promise<void> {
 const state = overlays.category;
 if (state.saving) return;
 const name = state.name.trim();
 if (!name || name === '未分类' || /[\r\n\0]/.test(name)) {
  state.error = '请输入有效的分类名称；留空分类的应用会归入“未分类”';
  return;
 }
 state.saving = true;
 state.error = '';
 try {
  if (state.appID && !(await changeCategory(state.appID, name))) {
   state.error = '分类保存失败，请检查应用配置是否可写';
   return;
  }
  if (!shell.extraCategories.includes(name)) shell.extraCategories.push(name);
  saveCategories();
  const done = categoryCreated;
  categoryCreated = null;
  state.open = false;
  done?.(name);
 } finally {
  state.saving = false;
 }
}

export function openGroupEditor(name: string): void {
 if (!name || shell.busy) return;
 overlays.group = {
  open: true,
  original: name,
  summary: '“' + name + '”包含 ' + shell.apps.filter((app) => categoryOf(app) === name).length + ' 个应用',
  name,
  error: '',
  saving: false
 };
}
export function closeGroupDialog(): void {
 overlays.group.open = false;
}
export async function submitGroup(remove: boolean): Promise<void> {
 const state = overlays.group;
 const oldName = state.original;
 if (shell.busy || !state.open || !oldName) return;
 const name = state.name.trim();
 if (!remove) {
  if (!name || name === '未分类' || /[\r\n\0]/.test(name)) {
   state.error = '请输入有效的分类名称';
   return;
  }
  if (name === oldName) {
   state.open = false;
   return;
  }
  if (categoryNames().includes(name)) {
   state.error = '分类名称已存在，请使用其他名称';
   return;
  }
 }
 state.saving = true;
 setBusy(true, remove ? '正在删除分类…' : '正在重命名分类…');
 try {
  const changed = (remove ? await API.DeleteCategory(oldName) : await API.RenameCategory(oldName, name)) as App[];
  for (const app of changed || []) replaceApp(app);
  shell.extraCategories = shell.extraCategories.filter((item) => item !== oldName);
  if (!remove && !shell.extraCategories.includes(name)) shell.extraCategories.push(name);
  if (shell.filter.kind === 'category' && shell.filter.name === oldName) shell.filter = remove ? { kind: 'all' } : { kind: 'category', name };
  saveCategories();
  state.open = false;
 } catch (error) {
  state.error = errorText(error);
 } finally {
  state.saving = false;
  setBusy(false);
 }
}

export function openUpdateDialog(id: string): void {
 const app = currentApp(id);
 if (!app || shell.busy || !app.updater.length) return;
 overlays.update = { open: true, appID: id, title: '更新“' + app.title + '”', kind: 'archive', source: '', error: '', installed: false, picking: false };
}
export function closeUpdateDialog(): void {
 overlays.update.open = false;
}
export async function browseUpdateSource(): Promise<void> {
 const state = overlays.update;
 if (state.picking || shell.busy) return;
 state.picking = true;
 try {
  const picked = await API.PickSource(state.kind);
  if (picked) state.source = picked;
 } catch (error) {
  state.error = errorText(error);
 } finally {
  state.picking = false;
 }
}
export async function submitUpdate(): Promise<void> {
 const state = overlays.update;
 if (shell.busy || shell.loading) return;
 const source = state.source.trim();
 if (!source) {
  state.error = '请选择更新目录或压缩包';
  return;
 }
 state.error = '';
 setBusy(true, '正在安装更新，请稍候…');
 try {
  const result = await API.Update(state.appID, source);
  if (result) {
   replaceApp(result.app as App);
   state.title = '更新完成。备份保存在：' + result.backup;
   state.installed = true;
  }
 } catch (error) {
  state.error = errorText(error);
 } finally {
  setBusy(false);
 }
}
export async function deleteUpdateBackup(): Promise<void> {
 const state = overlays.update;
 if (shell.busy) return;
 setBusy(true, '正在删除备份…');
 try {
  await API.DeleteBackup(state.appID);
  state.open = false;
 } catch (error) {
  state.error = errorText(error);
 } finally {
  setBusy(false);
 }
}

/* ---------- events ---------- */
let blurTimer = 0;
function scheduleAutoHide(): void {
 window.clearTimeout(blurTimer);
 blurTimer = window.setTimeout(() => {
  if (shell.busy || sheet.state.open !== null || shell.mode !== 'command') return;
  if (document.querySelector('dialog[open]')) return;
  hideWindow();
 }, 180);
}
const updateStages: Record<string, string> = { cleanup: '正在清理旧备份', backup: '正在准备备份', install: '正在安装文件' };

export function start(): void {
 loadPreferences();
 setTheme(shell.theme);
 commandItems();

 window.addEventListener('blur', scheduleAutoHide);
 window.addEventListener('focus', () => window.clearTimeout(blurTimer));
 Events.On('app-runner:window-lost-focus', scheduleAutoHide);
 Events.On('app-runner:update-progress', (event) => {
  if (!shell.busy) return;
  const data = (event.data || {}) as { stage?: string; done?: number; total?: number };
  const label = updateStages[data.stage || ''] || '正在安装更新';
  const done = typeof data.done === 'number' ? data.done : 0,
   total = typeof data.total === 'number' ? data.total : 0;
  overlays.activity = total > 0 ? label + ' ' + done.toLocaleString('zh-CN') + ' / ' + total.toLocaleString('zh-CN') + '…' : label + '…';
 });

 document.addEventListener('keydown', (event) => {
  const target = event.target as HTMLElement;
  if (document.querySelector('dialog[open]')) return;
  if (sheet.handleKey(event)) return;
  if (sheet.state.open !== null || shell.busy) return;
  if (event.isComposing) return;
  const typing = target.matches('input,textarea,select') || target.isContentEditable;
  const modifier = event.ctrlKey || event.metaKey;
  if (modifier && event.key.toLowerCase() === 'k') {
   event.preventDefault();
   focusSearch();
   return;
  }
  if (modifier && event.key.toLowerCase() === 'f') {
   event.preventDefault();
   focusSearch();
   return;
  }
  if (event.key === 'F5') {
   event.preventDefault();
   void refresh();
   return;
  }
  if (event.key === 'Escape') {
   if (shell.mode === 'command' && shell.commandQuery) {
    shell.commandQuery = '';
    return;
   }
   if (shell.mode === 'library' && shell.libraryQuery) {
    shell.libraryQuery = '';
    scheduleTable();
    return;
   }
   if (shell.mode === 'library') {
    goBack();
    return;
   }
   void API.HideWindow().catch(console.error);
   return;
  }
  if (shell.mode === 'command') {
   if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    if (typing && target.id !== 'cmd-search') return;
    event.preventDefault();
    moveCommandSelection(event.key === 'ArrowDown' ? 1 : -1);
    return;
   }
   if (event.key === 'Enter') {
    if (target.tagName === 'BUTTON') return;
    if (shell.commandSelected && !typing) {
     event.preventDefault();
     execute(shell.commandSelected);
    }
    return;
   }
   if (!typing && event.key.toLowerCase() === 'e') {
    const item = commandItems().find((candidate) => candidate.key === shell.commandSelected);
    if (item?.kind === 'app') {
     event.preventDefault();
     openEdit(item.app.uuid);
    }
    return;
   }
  } else {
   if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    if (typing && target.id !== 'lib-search') return;
    event.preventDefault();
    moveLibrarySelection(event.key === 'ArrowDown' ? 1 : -1);
    return;
   }
   if (event.key === 'Enter' && !typing && target.tagName !== 'BUTTON' && shell.librarySelected) {
    event.preventDefault();
    void launch(shell.librarySelected);
    return;
   }
   if (!typing && event.key.toLowerCase() === 'e' && shell.librarySelected) {
    event.preventDefault();
    openEdit(shell.librarySelected);
    return;
   }
  }
  if (!typing && !modifier && !event.altKey && event.key.length === 1) {
   if (shell.mode === 'command') shell.commandQuery += event.key;
   else shell.libraryQuery += event.key;
   const input = (shell.mode === 'command' ? $('#cmd-search') : $('#lib-search')) as HTMLInputElement | null;
   input?.focus();
   event.preventDefault();
  }
 });

 window.addEventListener('focus', () => {
  if (sheet.state.open !== null || document.querySelector('dialog[open]')) return;
  clearSearch();
  focusSearch();
 });

 void refresh().then(() => focusSearch());
}

/* ---------- drag helpers ---------- */

export function startDrag(id: string): void {
 draggedID = id;
}
export function endDrag(): void {
 draggedID = '';
 for (const element of Array.from(document.querySelectorAll<HTMLElement>('.drop-target'))) element.classList.remove('drop-target');
}
export function draggedApp(): string {
 return draggedID;
}
export async function dropOnCategory(event: DragEvent, name: string, element: HTMLElement): Promise<void> {
 event.preventDefault();
 element.classList.remove('drop-target');
 const id = draggedID || event.dataTransfer?.getData('text/plain') || '';
 if (id) await changeCategory(id, name);
}
export async function dropOnNewCategory(event: DragEvent, element: HTMLElement): Promise<void> {
 event.preventDefault();
 element.classList.remove('drop-target');
 const id = draggedID || event.dataTransfer?.getData('text/plain') || '';
 if (id) openCategoryDialog(id);
}
export function dragOverCategory(event: DragEvent, element: HTMLElement): void {
 if (!draggedID) return;
 event.preventDefault();
 element.classList.add('drop-target');
}
export function dragLeaveCategory(element: HTMLElement): void {
 element.classList.remove('drop-target');
}
export function dragOverTable(event: DragEvent): void {
 if (draggedID) event.preventDefault();
}
export function openDirectory(id: string): void {
 void perform(() => API.OpenDirectory(id));
}
export function openLinkUrl(url: string): void {
 void perform(() => API.OpenLink(url));
}

export { statusName, statusOf, categoryOf, relativeTime, escape, icons };
