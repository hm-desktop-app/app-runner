<script setup lang="ts">
import Icon from '../Icon.vue';
import { computed } from 'vue';
import { categoryOf, relativeTime, statusOf } from '../state';
import type { App, SortMode } from '../state';
import { openCreate, openEdit } from '../sheet/state';
import { statusName } from '../ui';
import Tile from '../Tile.vue';
import { launch, openDirectory, setSort } from './actions';
import { openCategoryDialog, openGroupEditor, openUpdateDialog } from './dialogs';
import {
    categoryTone,
    drag,
    dragLeaveCategory,
    dragOverCategory,
    dropOnCategory,
    dropOnNewCategory,
    endDrag,
    filterRows,
    goBack,
    libraryRows,
    selectApp,
    selectFilter,
    startDrag
} from './library';
import { hideWindow, registerSearchInput, shell, suggestedCount } from './shell';

const filters = computed(() => filterRows());
const rows = computed(() => libraryRows());
const selected = computed<App | undefined>(() => (shell.librarySelected ? shell.apps.find((app) => app.uuid === shell.librarySelected) : undefined));
const selectedStatus = computed(() => (selected.value ? statusName(statusOf(selected.value)) : null));
const selectedRunnable = computed(() => !!selected.value && selected.value.entryValid && ['GUI', 'CLI'].includes(selected.value.type));
const countText = computed(() => rows.value.length + ' 个应用' + (shell.libraryQuery.trim() ? ' · “' + shell.libraryQuery.trim() + '”' : ''));

function bindSearch(element: unknown): void {
    registerSearchInput('library', element as HTMLInputElement | null);
}
function typeText(app: App): string {
    return (app.type === 'GUI' ? '图形应用' : app.type === 'CLI' ? '命令行应用' : '待配置') + (app.version ? ' · ' + app.version : '');
}
function rowMouseDown(event: MouseEvent, id: string): void {
    const target = event.target as HTMLElement;
    if (target.closest('.row-actions')) return;
    startDrag(id);
}
function dragStart(event: DragEvent, id: string): void {
    startDrag(id);
    event.dataTransfer?.setData('text/plain', id);
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
}
function dragOver(event: DragEvent, key: string): void {
    dragOverCategory(event, key);
}
function dragLeave(key: string): void {
    dragLeaveCategory(key);
}
function dropCategory(event: DragEvent, name: string): void {
    void dropOnCategory(event, name);
}
function dropNewCategory(event: DragEvent): void {
    void dropOnNewCategory(event);
}
function onSortChange(event: Event): void {
    setSort((event.target as HTMLSelectElement).value as SortMode);
}
function onRowClick(id: string): void {
    selectApp(id);
}
function rowDoubleClick(id: string): void {
    selectApp(id);
    void launch(id);
}
function openGroup(event: Event, name: string): void {
    event.preventDefault();
    openGroupEditor(name);
}
</script>

<template>
    <section id="lib-view" class="view lib-view" :class="{ 'is-active': shell.mode === 'library' }" aria-label="应用库">
        <div class="lib-toolbar">
            <button id="lib-back" type="button" class="lib-back" title="返回命令中心 · Esc" aria-label="返回命令中心" @click="goBack">
                <span><Icon name="back" /></span>
            </button>
            <div class="lib-title">
                <h2>应用库</h2>
                <p id="lib-count">{{ countText }}</p>
            </div>
            <label class="lib-search"
                ><span aria-hidden="true"><Icon name="search" /></span
                ><input
                    id="lib-search"
                    :ref="bindSearch"
                    v-model="shell.libraryQuery"
                    placeholder="搜索名称、分类、备注或入口；支持拼音…"
                    autocomplete="off"
                    spellcheck="false"
                    aria-label="搜索应用"
                    @keydown.enter.prevent="shell.librarySelected && launch(shell.librarySelected)"
            /></label>
            <div class="lib-tools">
                <select id="lib-sort" class="sort" aria-label="排序方式" :value="shell.sort" @change="onSortChange">
                    <option value="recent">最近使用</option>
                    <option value="name">名称</option>
                    <option value="modified">更新时间</option>
                </select>
                <button id="new-app" type="button" class="btn primary" :disabled="shell.busy || shell.loading" @click="openCreate">
                    <span><Icon name="plus" /></span>新建应用
                </button>
                <button id="lib-close" type="button" class="win-close" title="收起窗口" aria-label="收起窗口" @click="hideWindow">
                    <span><Icon name="close" /></span>
                </button>
            </div>
        </div>
        <div class="lib-body">
            <aside id="lib-filters" class="lib-filters" aria-label="筛选">
                <div class="filter-group">
                    <div class="filter-title">浏览</div>
                    <button
                        v-for="row in filters.browsing"
                        :key="row.key"
                        type="button"
                        class="filter-item"
                        :class="{ active: row.tone === 'active' }"
                        :data-filter="row.key"
                        :aria-pressed="row.tone === 'active'"
                        @click="selectFilter(row.key)">
                        <span v-if="row.icon"><Icon :name="row.icon" /></span><span>{{ row.label }}</span
                        ><span class="count">{{ row.count }}</span>
                    </button>
                </div>
                <div class="filter-group">
                    <div class="filter-title">分类</div>
                    <button
                        v-for="row in filters.categories"
                        :key="row.key"
                        type="button"
                        class="filter-item"
                        :class="{ active: row.tone === 'active', 'drop-target': drag.over === row.key }"
                        :data-filter="row.key"
                        :aria-pressed="row.tone === 'active'"
                        draggable="false"
                        title="右键编辑或删除分类"
                        @click="selectFilter(row.key)"
                        @contextmenu="openGroup($event, row.label)"
                        @dragover="dragOver($event, row.key)"
                        @dragleave="dragLeave(row.key)"
                        @drop="dropCategory($event, row.label)">
                        <i class="fdot" :style="'--c:' + categoryTone(row.label)"></i><span>{{ row.label }}</span
                        ><span class="count">{{ row.count }}</span>
                    </button>
                    <button
                        id="new-category"
                        type="button"
                        class="filter-add"
                        :class="{ 'drop-target': drag.over === 'new' }"
                        @click="openCategoryDialog()"
                        @dragover="dragOver($event, 'new')"
                        @dragleave="dragLeave('new')"
                        @drop="dropNewCategory">
                        <span><Icon name="plus" /></span>新建分类
                    </button>
                </div>
            </aside>
            <div class="lib-table-pane">
                <div class="table-scroll">
                    <table class="lib-table">
                        <thead>
                            <tr>
                                <th>应用</th>
                                <th>分类</th>
                                <th>上次使用</th>
                                <th>状态</th>
                                <th class="action-head"></th>
                            </tr>
                        </thead>
                        <tbody id="lib-table">
                            <tr v-if="!rows.length" class="table-empty">
                                <td colspan="5">没有匹配的应用<br />试试其他关键词或筛选条件。</td>
                            </tr>
                            <template v-else-if="shell.libraryVisited">
                                <tr
                                    v-for="app in rows"
                                    :key="app.uuid"
                                    :data-id="app.uuid"
                                    draggable="true"
                                    :class="{ selected: app.uuid === shell.librarySelected }"
                                    @mousedown="rowMouseDown($event, app.uuid)"
                                    @dragstart="dragStart($event, app.uuid)"
                                    @dragend="endDrag"
                                    @click="onRowClick(app.uuid)"
                                    @dblclick="rowDoubleClick(app.uuid)">
                                    <td>
                                        <div class="cell-app">
                                            <span class="cell-tile"><Tile :app="app" lazy /></span>
                                            <div class="cell-main">
                                                <span class="cell-name">{{ app.title }}</span><span class="cell-type">{{ typeText(app) }}</span>
                                            </div>
                                        </div>
                                    </td>
                                    <td>
                                        <span class="cell-cat">{{ categoryOf(app) || '未分类' }}</span>
                                    </td>
                                    <td>
                                        <span class="cell-time">{{ relativeTime(app.lastUsed) }}</span>
                                    </td>
                                    <td>
                                        <span class="status"
                                            ><i class="dot" :class="statusName(statusOf(app)).tone"></i>{{ statusName(statusOf(app)).label }}</span
                                        >
                                    </td>
                                    <td>
                                        <div class="row-actions">
                                            <button
                                                type="button"
                                                data-act="run"
                                                :data-id="app.uuid"
                                                title="运行"
                                                :aria-label="'运行 ' + app.title"
                                                @click.stop="launch(app.uuid)">
                                                <span><Icon name="play" /></span></button
                                            ><button
                                                type="button"
                                                data-act="edit"
                                                :data-id="app.uuid"
                                                title="编辑"
                                                :aria-label="'编辑 ' + app.title"
                                                @click.stop="openEdit(app.uuid)">
                                                <span><Icon name="edit" /></span>
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            </template>
                        </tbody>
                    </table>
                </div>
                <div id="lib-selection" class="lib-selection" :class="{ has: !!selected }">
                    <template v-if="selected">
                        <div class="sel-app">
                            <span class="sel-tile"><Tile :app="selected" /></span>
                            <div class="sel-main">
                                <span class="sel-kicker">已选择</span><span class="sel-name">{{ selected.title }}</span
                                ><span class="sel-meta"
                                    >{{ categoryOf(selected) || '未分类' }} · {{ selectedStatus?.label }} · {{ relativeTime(selected.lastUsed) }}</span
                                >
                            </div>
                        </div>
                        <div class="sel-actions">
                            <button v-if="selected.updater.length" type="button" class="btn update" data-sel="update" @click="openUpdateDialog(selected.uuid)">
                                <span><Icon name="update" /></span>更新
                            </button>
                            <button type="button" class="btn primary" data-sel="run" :disabled="!selectedRunnable" @click="launch(selected.uuid)">
                                <span><Icon name="play" /></span>运行
                            </button>
                            <button type="button" class="btn" data-sel="edit" @click="openEdit(selected.uuid)">
                                <span><Icon name="edit" /></span>编辑
                            </button>
                            <button type="button" class="btn" data-sel="dir" @click="openDirectory(selected.uuid)">
                                <span><Icon name="folder" /></span>目录
                            </button>
                        </div>
                    </template>
                    <template v-else>
                        <div class="sel-summary">
                            <b>{{ shell.apps.length }} 个应用</b><span>·</span><span>{{ suggestedCount() }} 个建议更新</span>
                        </div>
                        <div class="sel-hints">
                            <span><kbd>↑</kbd><kbd>↓</kbd> 选择</span><span><kbd>↵</kbd> 运行</span><span><kbd>E</kbd> 编辑</span
                            ><span><kbd>Esc</kbd> 返回</span>
                        </div>
                    </template>
                </div>
            </div>
        </div>
    </section>
</template>

<style scoped>
.lib-toolbar {
    --wails-draggable: drag;
}
.lib-toolbar input,
.lib-toolbar select,
.lib-toolbar button,
.lib-toolbar label {
    --wails-draggable: no-drag;
}
.lib-view {
    flex-direction: column;
    height: 100%;
}
.lib-toolbar {
    height: 58px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 0 16px;
    border-bottom: 1px solid var(--line);
    background: var(--surface);
}
.lib-back {
    width: 28px;
    height: 28px;
    flex-shrink: 0;
    display: grid;
    place-items: center;
    border: 1px solid var(--line);
    background: var(--surface);
    border-radius: 7px;
    color: var(--muted);
}
.lib-back:hover {
    color: var(--accent);
    border-color: var(--accent);
}
.lib-back :deep(svg) {
    width: 14px;
    height: 14px;
}
.lib-title {
    min-width: 0;
}
.lib-title h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 650;
    letter-spacing: -0.015em;
    line-height: 1.2;
}
.lib-title p {
    margin: 3px 0 0;
    font-size: 10.5px;
    color: var(--muted);
}
.lib-search {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 7px;
    height: 33px;
    width: 232px;
    padding: 0 10px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--surface-2);
    transition: 0.14s;
}
.lib-search:focus-within {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
    background: var(--surface);
}
.lib-search span {
    display: grid;
    place-items: center;
    color: var(--faint);
    flex-shrink: 0;
}
.lib-search :deep(svg) {
    width: 14px;
    height: 14px;
}
.lib-search input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: 0;
    background: none;
    font-size: 11.5px;
}
.lib-search input::placeholder {
    color: var(--faint);
}
.lib-tools {
    display: flex;
    align-items: center;
    gap: 7px;
    flex-shrink: 0;
}
.sort {
    height: 33px;
    border: 1px solid var(--line);
    background: var(--surface);
    border-radius: 7px;
    padding: 0 26px 0 9px;
    font-size: 11px;
    color: var(--muted);
    appearance: none;
    background-repeat: no-repeat;
    background-position: right 8px center;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='9' height='9' viewBox='0 0 24 24' fill='none' stroke='%23888f86' stroke-width='2.5' stroke-linecap='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
}
.lib-tools .btn {
    height: 33px;
    font-size: 11px;
    padding: 0 11px;
}
.lib-body {
    flex: 1;
    display: grid;
    grid-template-columns: 186px minmax(0, 1fr);
    min-height: 0;
}
.lib-filters {
    border-right: 1px solid var(--line);
    background: var(--surface-2);
    padding: 12px 8px;
    overflow: auto;
    scrollbar-gutter: stable;
}
.filter-group {
    margin-bottom: 16px;
}
.filter-title {
    font-size: 9.5px;
    font-weight: 650;
    letter-spacing: 0.09em;
    color: var(--faint);
    padding: 0 9px;
    margin-bottom: 6px;
}
.filter-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 7px 9px;
    margin-bottom: 1px;
    border: 0;
    background: none;
    border-radius: 7px;
    color: var(--ink);
    font-size: 11.5px;
    text-align: left;
    position: relative;
}
.filter-item:hover {
    background: var(--surface);
}
.filter-item.active {
    background: var(--accent-soft);
    color: var(--accent-ink);
    font-weight: 600;
}
.filter-item.active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 8px;
    bottom: 8px;
    width: 2px;
    border-radius: 2px;
    background: var(--accent);
}
.filter-item :deep(svg) {
    width: 13px;
    height: 13px;
    color: var(--faint);
    flex-shrink: 0;
}
.filter-item.active :deep(svg) {
    color: var(--accent);
}
.filter-item .count {
    margin-left: auto;
    font-size: 10px;
    color: var(--faint);
    font-variant-numeric: tabular-nums;
}
.filter-item.active .count {
    color: var(--accent-ink);
    opacity: 0.7;
}
.filter-item.drop-target,
.filter-add.drop-target {
    background: var(--accent-soft);
    box-shadow: inset 0 0 0 1px var(--accent);
}
.filter-add {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 7px 9px;
    border: 0;
    background: none;
    border-radius: 7px;
    color: var(--muted);
    font-size: 11.5px;
    text-align: left;
}
.filter-add:hover {
    background: var(--surface);
    color: var(--ink);
}
.filter-add :deep(svg) {
    width: 12px;
    height: 12px;
}
.fdot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--c, var(--faint));
    margin: 0 3.5px;
    flex-shrink: 0;
}
.lib-table-pane {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    background: var(--surface);
}
.table-scroll {
    flex: 1;
    overflow: auto;
    scrollbar-gutter: stable;
}
.lib-table {
    width: 100%;
    border-collapse: collapse;
    table-layout: fixed;
}
.lib-table th {
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--surface-2);
    border-bottom: 1px solid var(--line);
    padding: 9px 12px;
    text-align: left;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.06em;
    color: var(--faint);
    white-space: nowrap;
}
.lib-table th:first-child {
    padding-left: 16px;
    width: 34%;
}
.lib-table th:nth-child(2) {
    width: 19%;
}
.lib-table th:nth-child(3) {
    width: 17%;
}
.lib-table th:nth-child(4) {
    width: 15%;
}
.lib-table th.action-head {
    width: 15%;
}
.lib-table td {
    border-bottom: 1px solid var(--line);
    padding: 9px 12px;
    font-size: 11.5px;
    vertical-align: middle;
}
.lib-table td:first-child {
    padding-left: 16px;
}
.lib-table tbody tr {
    cursor: default;
    transition: background 0.1s;
}
.lib-table tbody tr:hover {
    background: var(--surface-2);
}
.lib-table tbody tr.selected {
    background: var(--accent-soft);
}
.lib-table tbody tr.selected td:first-child {
    box-shadow: inset 2px 0 0 var(--accent);
}
.lib-table tbody tr.selected .cell-name {
    font-weight: 650;
}
.cell-app {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
}
.cell-app :deep(.tile) {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    font-size: 11px;
}
.cell-tile {
    display: flex;
    flex-shrink: 0;
}
.cell-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
}
.cell-name {
    font-size: 13px;
    line-height: 1.3;
    font-weight: 560;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.cell-type {
    font-size: 10px;
    line-height: 1.3;
    color: var(--faint);
}
.cell-cat {
    font-size: 11px;
    color: var(--muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: block;
}
.cell-time {
    font-size: 11px;
    color: var(--faint);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
}
.status {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 10.5px;
    color: var(--muted);
    white-space: nowrap;
}
.row-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 2px;
    opacity: 0;
    transition: 0.12s;
}
.lib-table tbody tr:hover .row-actions,
.lib-table tbody tr.selected .row-actions {
    opacity: 1;
}
.row-actions button {
    width: 26px;
    height: 26px;
    border: 1px solid var(--line);
    background: var(--surface);
    border-radius: 6px;
    display: grid;
    place-items: center;
    color: var(--muted);
}
.row-actions button:hover {
    color: var(--accent);
    border-color: var(--accent);
}
.row-actions :deep(svg) {
    width: 13px;
    height: 13px;
}
.table-empty td {
    padding: 54px 20px;
    text-align: center;
    color: var(--muted);
    font-size: 12px;
    line-height: 1.7;
    border: 0;
}
.dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    display: inline-block;
    background: var(--faint);
}
.dot.ok {
    background: var(--faint);
}
.dot.warn {
    background: #c25a4e;
}
.dot.muted {
    background: var(--faint);
}
:global([data-theme='dark']) .dot.warn {
    background: #e08a7e;
}
.lib-selection {
    height: 56px;
    flex-shrink: 0;
    border-top: 1px solid var(--line);
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 0 16px;
    background: var(--surface-2);
}
.lib-selection.has {
    box-shadow: inset 2px 0 0 var(--accent);
}
.sel-app {
    display: flex;
    align-items: center;
    gap: 9px;
    min-width: 0;
}
.sel-app :deep(.tile) {
    width: 28px;
    height: 28px;
    border-radius: 7px;
    font-size: 10px;
}
.sel-tile {
    display: flex;
    flex-shrink: 0;
}
.sel-main {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
}
.sel-kicker {
    font-size: 8.5px;
    line-height: 1;
    letter-spacing: 0.08em;
    color: var(--faint);
}
.sel-name {
    font-size: 11.5px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.sel-meta {
    font-size: 10px;
    color: var(--muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.sel-actions {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
}
.sel-actions .btn {
    height: 30px;
    font-size: 11px;
    padding: 0 11px;
}
.sel-summary {
    display: flex;
    align-items: center;
    gap: 9px;
    font-size: 10.5px;
    color: var(--muted);
}
.sel-summary b {
    color: var(--ink);
    font-weight: 600;
}
.sel-hints {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 9px;
    font-size: 9.5px;
    color: var(--faint);
}
.sel-hints kbd {
    font-size: 9px;
    padding: 1px 4px;
}
@media (max-width: 900px) {
    .lib-body {
        grid-template-columns: 160px minmax(0, 1fr);
    }
    .lib-search {
        width: 168px;
    }
}
@media (max-width: 790px) {
    .lib-filters {
        display: none;
    }
    .lib-body {
        grid-template-columns: 1fr;
    }
}
</style>
