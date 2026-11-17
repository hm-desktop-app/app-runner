import {nextTick, reactive} from 'vue';
import * as API from '../../bindings/app-runner/launcherservice';
import {categoryOf, profileOf, statusOf, validation} from '../state';
import type {App, Link, NewAppInfo, Profile} from '../state';
import {escape, statusName, tile} from '../ui';

export type Theme = 'light' | 'dark';
export interface SettingsInfo {hotkey: string; hotkeyError: string; library: string; autoStart: boolean}
export interface SheetContext {
 apps(): App[];
 categories(): string[];
 newCategory(): Promise<string | undefined>;
 library(): string;
 settings(): Promise<SettingsInfo>;
 theme(): Theme;
 setTheme(value: Theme): void;
 autohide(): boolean;
 setAutohide(value: boolean): void;
 busy(value: boolean, text?: string): void;
 isBusy(): boolean;
 changed(app: App): void;
 created(app: App): void;
 toast(message: string): void;
 closed(): void;
 openUpdate(id: string): void;
}

export type SheetName = 'edit' | 'create' | 'settings';
export type EditTab = 'basic' | 'links' | 'update';
export const newCategoryOption = '__new-category__';
export const tabList: {id: EditTab; label: string}[] = [
 {id: 'basic', label: '基本信息'},
 {id: 'links', label: '相关链接'},
 {id: 'update', label: '更新方式'},
];
export const knownActions = ['PIN', 'RMC', 'UNPKG'];

export const state = reactive({
 open: null as SheetName | null,
 editingId: '',
 draft: null as Profile | null,
 baseline: '',
 tab: 'basic' as EditTab,
 createStep: 1,
 createKind: 'directory',
 createSource: '',
 createName: '',
 createNameEdited: false,
 pendingNew: null as NewAppInfo | null,
 newEntryValid: false,
 entryInvalid: false,
 entryHint: '',
 createError: '',
 saveError: '',
 saving: false,
 preparing: false,
 settingsInfo: null as SettingsInfo | null,
 settingsAutohide: false,
 settingsThemeDark: false,
});

let current: SheetContext | null = null;
export const setContext = (value: SheetContext): void => {current = value;};
export const context = (): SheetContext => {
 if (!current) throw new Error('sheet context is not ready');
 return current;
};

export const currentApp = (): App | undefined => state.editingId ? context().apps().find(item => item.uuid === state.editingId) : undefined;
export const dirty = (): boolean => !!state.draft && JSON.stringify(state.draft) !== state.baseline;
export const editError = (): string => state.draft ? validation(state.draft) : '';
export const canSave = (): boolean => !state.saving && !editError() && dirty();
export const createValidation = (): string => {
 if (!state.draft) return '';
 return validation(state.draft) || (state.draft.entry && !state.newEntryValid ? '请选择来源中的有效启动入口' : '');
};
export const categoryNames = (): string[] => {
 const names = current ? current.categories() : [];
 const selected = state.draft?.category;
 return selected && !names.includes(selected) ? [...names, selected] : names;
};
export const summaryHtml = (): string => {
 const app = currentApp();
 if (!app) return '';
 const status = statusName(statusOf(app));
 return tile(app) + '<div><strong>' + escape(app.title) + '</strong><span>' + escape(categoryOf(app) || '未分类') + ' · ' + escape(app.type === 'GUI' ? '图形应用' : app.type === 'CLI' ? '命令行应用' : '待配置') + (app.version ? ' · ' + escape(app.version) : '') + '</span></div><span class="chip ' + status.tone + '">' + status.label + '</span>';
};

let entryTimer = 0;
let entryRequest = 0;

function resetSheet(): void {
 state.open = null;
 state.editingId = '';
 state.draft = null;
 state.baseline = '';
 state.createStep = 1;
 state.pendingNew = null;
 state.saveError = '';
 state.createError = '';
 state.entryInvalid = false;
 state.entryHint = '';
}

export function openEdit(id: string): void {
 const app = context().apps().find(item => item.uuid === id);
 if (!app) return;
 state.open = 'edit';
 state.editingId = id;
 state.draft = profileOf(app);
 state.baseline = JSON.stringify(state.draft);
 state.tab = 'basic';
 state.saveError = '';
 state.entryInvalid = false;
 state.entryHint = '';
 scheduleEntryCheck();
 void nextTick(() => {
  const field = document.getElementById('edit-title');
  if (field instanceof HTMLInputElement) {field.focus({preventScroll: true});field.select();}
 });
}

export function openCreate(): void {
 state.open = 'create';
 state.createStep = 1;
 state.createKind = 'directory';
 state.createSource = '';
 state.createName = '';
 state.createNameEdited = false;
 state.pendingNew = null;
 state.newEntryValid = false;
 state.createError = '';
 void nextTick(() => document.getElementById('create-source')?.focus({preventScroll: true}));
}

export function openSettings(): void {
 const ctx = context();
 state.open = 'settings';
 state.settingsInfo = null;
 state.settingsAutohide = ctx.autohide();
 state.settingsThemeDark = ctx.theme() === 'dark';
 void ctx.settings().then(info => {
  if (state.open === 'settings') {
   state.settingsInfo = info;
   state.settingsAutohide = ctx.autohide();
  }
 }).catch(console.error);
}

export function handleKey(event: KeyboardEvent): boolean {
 if (!state.open) return false;
 if (event.key === 'Escape') {
  if (event.isComposing) return false;
  event.preventDefault();
  void requestClose();
  return true;
 }
 if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
  if (state.open === 'edit') {event.preventDefault();void save();return true;}
  if (state.open === 'create' && state.createStep === 2) {event.preventDefault();void submitCreate();return true;}
 }
 return false;
}

export async function requestClose(): Promise<boolean> {
 const ctx = context();
 if (!state.open) return true;
 if (ctx.isBusy()) return false;
 if (state.open === 'edit' && dirty()) {
  if (!await API.Confirm('当前详情有未保存的更改，确定放弃吗？')) return false;
 }
 if (state.open === 'create' && state.pendingNew) {
  if (!await API.Confirm('新建应用还没有保存，确定放弃吗？')) return false;
  const token = state.pendingNew.token;
  state.pendingNew = null;
  await API.DiscardNewApp(token).catch(console.error);
 }
 resetSheet();
 ctx.closed();
 return true;
}

function scheduleEntryCheck(): void {
 clearTimeout(entryTimer);
 entryTimer = window.setTimeout(() => void checkEntry(), 200);
}

export async function checkEntry(): Promise<void> {
 clearTimeout(entryTimer);
 if (!state.draft) return;
 const request = ++entryRequest;
 const value = state.draft.entry;
 if (!value.trim()) {
  state.entryInvalid = false;
  state.entryHint = '入口相对于 content 目录；支持 re: 正则匹配文件名';
  return;
 }
 const valid = await API.CheckEntry(state.editingId, value).catch(() => false);
 if (request !== entryRequest) return;
 state.entryInvalid = !valid;
 state.entryHint = valid ? '入口有效：相对于 content 目录，支持 re: 正则匹配文件名' : '未找到入口文件。可以先保存配置，文件就绪后按 F5 刷新。';
}

let newEntryTimer = 0;
function scheduleNewEntryCheck(): void {
 clearTimeout(newEntryTimer);
 newEntryTimer = window.setTimeout(() => void checkNewEntry(), 200);
}
export function entryChanged(): void {scheduleEntryCheck();}
export function newEntryChanged(): void {scheduleNewEntryCheck();}

export async function checkNewEntry(): Promise<void> {
 clearTimeout(newEntryTimer);
 if (!state.draft || !state.pendingNew) return;
 const request = ++entryRequest;
 const value = state.draft.entry;
 if (!value.trim()) {
  state.entryInvalid = false;
  state.entryHint = '从来源中选择入口；支持 re: 正则匹配文件名';
  return;
 }
 const valid = await API.CheckNewEntry(state.pendingNew.token, value).catch(() => false);
 if (request !== entryRequest || state.pendingNew === null) return;
 state.newEntryValid = valid;
 state.entryInvalid = !valid;
 state.entryHint = valid ? '入口有效' : '来源中没有匹配的入口';
}

export async function save(): Promise<void> {
 const ctx = context();
 if (ctx.isBusy() || state.saving || !state.draft || !dirty() || validation(state.draft)) return;
 state.draft.relatedLinks = state.draft.relatedLinks.map(link => ({title: link.title.trim(), url: link.url.trim()})).filter(link => link.url);
 ctx.busy(true, '正在保存配置…');
 state.saving = true;
 state.saveError = '';
 try {
  const saved = await API.Save(state.editingId, state.draft) as App;
  state.open = null;
  resetSheet();
  ctx.busy(false);
  ctx.changed(saved);
  ctx.closed();
  ctx.toast('已保存 ' + saved.title);
 } catch (error) {
  state.saveError = error instanceof Error ? error.message : String(error);
 } finally {
  state.saving = false;
  ctx.busy(false);
 }
}

export async function prepare(): Promise<void> {
 const ctx = context();
 if (ctx.isBusy() || state.preparing) return;
 if (!state.createSource.trim() || !state.createName.trim()) {
  state.createError = '请选择来源并确认应用目录名';
  return;
 }
 ctx.busy(true, '正在准备应用文件…');
 state.preparing = true;
 state.createError = '';
 let prepared: NewAppInfo | null = null;
 try {
  prepared = await API.PrepareNewApp(state.createSource.trim(), state.createName.trim()) as NewAppInfo;
  if (state.pendingNew) await API.DiscardNewApp(state.pendingNew.token);
  state.pendingNew = prepared;
  const name = prepared.name || state.createName.trim();
  state.draft = {uuid: prepared.token, title: name, type: 'GUI', entry: (prepared.entries || [])[0] || '', category: '', description: '', runAsAdmin: false, relatedLinks: [], updater: [], lastUpdateAt: ''};
  state.newEntryValid = false;
  state.createStep = 2;
  ctx.busy(false);
  void nextTick(() => {void checkNewEntry();});
 } catch (error) {
  if (prepared) await API.DiscardNewApp(prepared.token).catch(console.error);
  state.createError = error instanceof Error ? error.message : String(error);
 } finally {
  ctx.busy(false);
  state.preparing = false;
 }
}

export async function submitCreate(): Promise<void> {
 const ctx = context();
 if (ctx.isBusy() || state.saving || !state.draft || !state.pendingNew) return;
 const error = createValidation();
 if (error) {state.createError = error;return;}
 const token = state.pendingNew.token;
 ctx.busy(true, '正在部署新应用…');
 state.saving = true;
 state.createError = '';
 try {
  const saved = await API.CommitNewApp(token, state.draft) as App;
  state.pendingNew = null;
  state.open = null;
  resetSheet();
  ctx.busy(false);
  ctx.created(saved);
  ctx.toast('已创建 ' + saved.title);
 } catch (failure) {
  state.createError = failure instanceof Error ? failure.message : String(failure);
 } finally {
  state.saving = false;
  ctx.busy(false);
 }
}

export async function chooseCategory(value: string): Promise<void> {
 if (!state.draft) return;
 const previous = state.draft.category;
 if (value !== newCategoryOption) {state.draft.category = value;return;}
 const name = await context().newCategory();
 state.draft.category = name || previous;
}

export function addLink(): void {
 if (!state.draft) return;
 state.draft.relatedLinks.push({title: '', url: ''});
 void nextTick(() => {
  const input = document.querySelector<HTMLInputElement>('#sheet-body [data-link="' + (state.draft!.relatedLinks.length - 1) + '"][data-key="title"]');
  input?.focus();
 });
}

export function normalizeUrl(link: Link): void {
 const value = link.url.trim();
 const normalized = value && !value.includes('://') ? 'https://' + value : value;
 if (normalized !== link.url) link.url = normalized;
}

export function addUpdater(): void {
 if (!state.draft) return;
 state.draft.updater.push('RMC');
}

export function actionOf(command: string): string {
 const colon = command.indexOf(':');
 return (colon < 0 ? command : command.slice(0, colon)).trim();
}
export function parameterOf(command: string): string {
 const colon = command.indexOf(':');
 return colon < 0 ? '' : command.slice(colon + 1);
}
export function setUpdaterAction(index: number, action: string): void {
 if (!state.draft) return;
 state.draft.updater[index] = action + (action === 'PIN' ? ':' : '');
}
export function setUpdaterParam(index: number, value: string): void {
 if (!state.draft) return;
 state.draft.updater[index] = 'PIN:' + value;
}
export function removeUpdater(index: number): void {
 state.draft?.updater.splice(index, 1);
}

export async function pickSource(): Promise<void> {
 try {
  const picked = await API.PickSource(state.createKind);
  if (!picked) return;
  state.createSource = picked;
  if (!state.createNameEdited) state.createName = suggestedName(picked);
 } catch (error) {
  state.createError = error instanceof Error ? error.message : String(error);
 }
}
export function suggestedName(source: string): string {
 const base = source.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || '';
 const withoutExtension = base.replace(/\.(zip|7z)$/i, '');
 return withoutExtension.trim() || 'app';
}
export function createNameChanged(): void {state.createNameEdited = true;}
export function createSourceChanged(): void {
 if (!state.createNameEdited) state.createName = suggestedName(state.createSource);
}

export async function toggleAutohide(): Promise<void> {
 const ctx = context();
 state.settingsAutohide = !state.settingsAutohide;
 ctx.setAutohide(state.settingsAutohide);
}
export function toggleTheme(): void {
 const ctx = context();
 state.settingsThemeDark = !state.settingsThemeDark;
 ctx.setTheme(state.settingsThemeDark ? 'dark' : 'light');
}
export async function toggleAutoStart(): Promise<void> {
 if (!state.settingsInfo) return;
 try {
  await API.SetAutoStart(!state.settingsInfo.autoStart);
  state.settingsInfo.autoStart = !state.settingsInfo.autoStart;
  context().toast(state.settingsInfo.autoStart ? '已开启开机自启' : '已关闭开机自启');
 } catch (error) {
  context().toast(error instanceof Error ? error.message : String(error));
 }
}
export function openLibrary(): void {void API.OpenLibrary().catch(error => context().toast(error instanceof Error ? error.message : String(error)));}
export function openEntryFolder(): void {void API.OpenDirectory(state.editingId).catch(error => context().toast(error instanceof Error ? error.message : String(error)));}
export function editProfile(): void {void API.EditProfile(state.editingId).catch(error => context().toast(error instanceof Error ? error.message : String(error)));}
export function openLink(url: string): void {void API.OpenLink(url).catch(error => context().toast(error instanceof Error ? error.message : String(error)));}
export async function quit(): Promise<void> {
 const ctx = context();
 if (!await API.Confirm('确定退出 App Runner 吗？退出后需要重新启动程序。')) return;
 await API.QuitApp().catch(error => ctx.toast(error instanceof Error ? error.message : String(error)));
}
export async function openUpdate(): Promise<void> {
 const ctx = context();
 const id = state.editingId;
 const closed = await requestClose();
 if (closed) ctx.openUpdate(id);
}
