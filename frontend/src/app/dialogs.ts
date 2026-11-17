import * as API from '../../bindings/app-runner/launcherservice';
import { categoryOf } from '../state';
import type { App } from '../state';
import type { SettingsInfo, SheetContext } from '../sheet/state';
import { changeCategory } from './actions';
import {
    categoryNames,
    currentApp,
    focusSearch,
    overlays,
    replaceApp,
    saveCategories,
    setAutohide,
    setBusy,
    setMode,
    setTheme,
    shell,
    showToast
} from './shell';

let categoryCreated: ((name?: string) => void) | null = null;
const errorText = (error: unknown): string => (error instanceof Error ? error.message : String(error));

/* ---------- new category ---------- */
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

/* ---------- edit category ---------- */
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

/* ---------- install update ---------- */
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

/* ---------- sheet host ---------- */
export function sheetContext(): SheetContext {
    return {
        apps: () => shell.apps,
        categories: () => categoryNames(),
        newCategory: () => new Promise<string | undefined>((resolve) => openCategoryDialog('', resolve)),
        library: () => shell.libraryPath,
        settings: async () => (await API.Settings()) as unknown as SettingsInfo,
        theme: () => shell.theme,
        setTheme: (value) => setTheme(value),
        autohide: () => shell.autohide,
        setAutohide: (value) => setAutohide(value),
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
}
