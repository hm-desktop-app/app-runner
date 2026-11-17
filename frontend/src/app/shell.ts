import { reactive } from 'vue';
import * as API from '../../bindings/app-runner/launcherservice';
import { groupApps, suggestsUpdate } from '../state';
import type { App, LibraryFilter, SortMode } from '../state';
import * as sheet from '../sheet/state';
import type { Theme } from '../sheet/state';

export type Mode = 'command' | 'library';
export type SourceKind = 'archive' | 'directory';

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

export const overlays = reactive({
    toast: { message: '', visible: false },
    activity: '',
    category: { open: false, appID: '', message: '', name: '', error: '', saving: false },
    group: { open: false, original: '', summary: '', name: '', error: '', saving: false },
    update: { open: false, appID: '', title: '', kind: 'archive' as SourceKind, source: '', error: '', installed: false, picking: false }
});

let libraryKey = '';
let toastTimer = 0;
const searchInputs: Partial<Record<Mode, HTMLInputElement>> = {};

/* ---------- preferences ---------- */
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
export function loadPreferences(): void {
    try {
        if (localStorage.getItem('app-runner-autohide') === '1') shell.autohide = true;
        if (localStorage.getItem('app-runner-theme') === 'dark') shell.theme = 'dark';
    } catch {}
    document.documentElement.dataset.theme = shell.theme;
}
export function loadLibraryPreferences(path: string): void {
    if (path === libraryKey) return;
    libraryKey = path;
    shell.extraCategories = [];
    try {
        const saved = JSON.parse(localStorage.getItem('app-runner-categories:' + path) || '{}');
        for (const name of saved.categories || [])
            if (typeof name === 'string' && name.trim() && name.trim() !== '未分类') shell.extraCategories.push(name.trim());
    } catch {}
}
export function saveCategories(): void {
    try {
        localStorage.setItem('app-runner-categories:' + libraryKey, JSON.stringify({ categories: shell.extraCategories }));
    } catch {}
}

/* ---------- derived state ---------- */
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
export function replaceApp(saved: App): void {
    const index = shell.apps.findIndex((app) => app.uuid === saved.uuid);
    if (index >= 0) shell.apps[index] = saved;
    else shell.apps.push(saved);
}

/* ---------- chrome ---------- */
export function setAutohide(value: boolean): void {
    shell.autohide = value;
    saveAutohide(value);
}
export function setTheme(value: Theme): void {
    shell.theme = value;
    document.documentElement.dataset.theme = value;
    saveTheme(value);
    void API.SetTheme(value === 'dark').catch(console.error);
}
export function setMode(next: Mode): void {
    shell.mode = next;
    if (next === 'library') shell.libraryVisited = true;
    setTimeout(focusSearch, 30);
}
export function registerSearchInput(mode: Mode, element: HTMLInputElement | null): void {
    if (element) searchInputs[mode] = element;
    else delete searchInputs[mode];
}
export function focusSearch(select = true): void {
    if (sheet.state.open !== null) return;
    const input = searchInputs[shell.mode];
    if (!input) return;
    input.focus();
    if (select) input.select();
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
export function hideWindow(): void {
    void API.HideWindow().catch(console.error);
}
/** True while a modal overlay dialog is open, so global shortcuts and autohide stay out of the way. */
export function dialogOpen(): boolean {
    return overlays.category.open || overlays.group.open || overlays.update.open;
}
