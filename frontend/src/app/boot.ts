import { Events } from '@wailsio/runtime';
import * as API from '../../bindings/app-runner/launcherservice';
import * as sheet from '../sheet/state';
import { launch, refresh } from './actions';
import { commandItems, execute, moveCommandSelection } from './commands';
import { openCategoryDialog, openUpdateDialog, sheetContext } from './dialogs';
import { goBack, moveLibrarySelection } from './library';
import { clearSearch, dialogOpen, focusSearch, hideWindow, loadPreferences, overlays, setTheme, shell } from './shell';

let blurTimer = 0;

function scheduleAutoHide(): void {
    window.clearTimeout(blurTimer);
    blurTimer = window.setTimeout(() => {
        if (shell.busy || sheet.state.open !== null || shell.mode !== 'command' || dialogOpen()) return;
        hideWindow();
    }, 180);
}

const updateStages: Record<string, string> = { cleanup: '正在清理旧备份', backup: '正在准备备份', install: '正在安装文件' };

function onUpdateProgress(event: { data?: unknown }): void {
    if (!shell.busy) return;
    const data = (event.data || {}) as { stage?: string; done?: number; total?: number };
    const label = updateStages[data.stage || ''] || '正在安装更新';
    const done = typeof data.done === 'number' ? data.done : 0,
        total = typeof data.total === 'number' ? data.total : 0;
    overlays.activity = total > 0 ? label + ' ' + done.toLocaleString('zh-CN') + ' / ' + total.toLocaleString('zh-CN') + '…' : label + '…';
}

function onKeyDown(event: KeyboardEvent): void {
    const target = event.target as HTMLElement;
    if (dialogOpen()) return;
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
                sheet.openEdit(item.app.uuid);
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
            sheet.openEdit(shell.librarySelected);
            return;
        }
    }
    if (!typing && !modifier && !event.altKey && event.key.length === 1) {
        if (shell.mode === 'command') shell.commandQuery += event.key;
        else shell.libraryQuery += event.key;
        focusSearch(false);
        event.preventDefault();
    }
}

export function start(): void {
    loadPreferences();
    setTheme(shell.theme);
    sheet.setContext(sheetContext());
    commandItems();

    window.addEventListener('blur', scheduleAutoHide);
    window.addEventListener('focus', () => window.clearTimeout(blurTimer));
    Events.On('app-runner:window-lost-focus', scheduleAutoHide);
    Events.On('app-runner:update-progress', onUpdateProgress);
    document.addEventListener('keydown', onKeyDown);
    window.addEventListener('focus', () => {
        if (sheet.state.open !== null || dialogOpen()) return;
        clearSearch();
        focusSearch();
    });
    void refresh().then(() => focusSearch());
}

export { openCategoryDialog, openUpdateDialog };
