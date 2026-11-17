import * as API from '../../bindings/app-runner/launcherservice';
import { categoryOf } from '../state';
import type { App, SortMode } from '../state';
import { currentApp, loadLibraryPreferences, replaceApp, saveCategories, setBusy, shell } from './shell';

async function perform<T>(action: () => PromiseLike<T>): Promise<T | undefined> {
    try {
        return await action();
    } catch (error) {
        await API.ShowError(error instanceof Error ? error.message : String(error)).catch(console.error);
        return undefined;
    }
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

export function openDirectory(id: string): void {
    void perform(() => API.OpenDirectory(id));
}

export function openLinkUrl(url: string): void {
    void perform(() => API.OpenLink(url));
}
