import { rankApps, recentApps, visibleApps } from '../state';
import type { App } from '../state';
import { openCreate, openSettings } from '../sheet/state';
import { icons } from '../ui';
import { launch, refresh } from './actions';
import { setMode, shell } from './shell';

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
