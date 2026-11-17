import { reactive } from 'vue';
import { filterKey, filterLibrary, parseFilter } from '../state';
import type { App, LibraryFilter } from '../state';
import { icons } from '../ui';
import { openCategoryDialog } from './dialogs';
import { changeCategory } from './actions';
import { categoryNames, setMode, shell } from './shell';

export interface FilterRow {
    key: string;
    label: string;
    icon: string;
    count: number;
    category: boolean;
    tone?: string;
}

/** Drag state: which app is being dragged and which category drop target is hovered. */
export const drag = reactive({ appID: '', over: '' });

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
export function goBack(): void {
    shell.commandQuery = '';
    setMode('command');
}

/* ---------- drag and drop ---------- */
export function startDrag(id: string): void {
    drag.appID = id;
}
export function endDrag(): void {
    drag.appID = '';
    drag.over = '';
}
function dragKey(event: DragEvent): string {
    return drag.appID || event.dataTransfer?.getData('text/plain') || '';
}
export function dragOverCategory(event: DragEvent, key: string): void {
    if (!drag.appID) return;
    event.preventDefault();
    drag.over = key;
}
export function dragLeaveCategory(key: string): void {
    if (drag.over === key) drag.over = '';
}
export async function dropOnCategory(event: DragEvent, name: string): Promise<void> {
    event.preventDefault();
    drag.over = '';
    const id = dragKey(event);
    if (id) await changeCategory(id, name);
}
export async function dropOnNewCategory(event: DragEvent): Promise<void> {
    event.preventDefault();
    drag.over = '';
    const id = dragKey(event);
    if (id) openCategoryDialog(id);
}
