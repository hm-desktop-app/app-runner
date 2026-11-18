<script setup lang="ts">
import Icon from '../Icon.vue';
import Tile from '../Tile.vue';
import { computed } from 'vue';
import { categoryOf, statusOf } from '../state';
import type { App } from '../state';
import { statusName } from '../ui';
import {
    actionOf,
    addLink,
    addUpdater,
    categoryNames,
    currentApp,
    draftCategoryChanged,
    draftTypeChanged,
    editProfile,
    entryChanged,
    focusField,
    knownActions,
    newCategoryOption,
    normalizeUrl,
    openEntryFolder,
    openLink,
    openUpdate,
    parameterOf,
    registerField,
    removeUpdater,
    setUpdaterAction,
    setUpdaterParam,
    state,
    tabList
} from './state';

const app = computed<App | undefined>(() => currentApp());
const appStatus = computed(() => (app.value ? statusName(statusOf(app.value)) : null));
const appType = computed(() => (!app.value ? '' : app.value.type === 'GUI' ? '图形应用' : app.value.type === 'CLI' ? '命令行应用' : '待配置'));
const categories = computed(() => categoryNames());
const editTitleRef = (element: unknown): void => registerField('edit-title', element as HTMLElement | null);
const linkTitleRef =
    (index: number) =>
    (element: unknown): void =>
        registerField('link-title-' + index, element as HTMLElement | null);

function onEntryInput(): void {
    entryChanged();
}
function focusLinkTitle(event: Event, index: number): void {
    const target = event.target as HTMLElement;
    if (target.closest('input,button')) return;
    focusField('link-title-' + index);
}
function onParamInput(index: number, event: Event): void {
    setUpdaterParam(index, (event.target as HTMLInputElement).value);
}
function onActionChange(index: number, event: Event): void {
    setUpdaterAction(index, (event.target as HTMLSelectElement).value);
}
function onUpdate(): void {
    void openUpdate();
}
</script>

<template>
    <div v-if="app" class="sheet-app">
        <Tile :app="app" />
        <div>
            <strong>{{ app.title }}</strong><span
                >{{ categoryOf(app) || '未分类' }} · {{ appType }}<template v-if="app.version"> · {{ app.version }}</template></span
            >
        </div>
        <span class="chip" :class="appStatus?.tone">{{ appStatus?.label }}</span>
    </div>
    <div class="tabs" role="tablist">
        <button
            v-for="item in tabList"
            :key="item.id"
            type="button"
            class="tab"
            :class="{ active: state.tab === item.id }"
            :data-tab="item.id"
            role="tab"
            :aria-selected="state.tab === item.id"
            @click="state.tab = item.id">
            {{ item.label }}
        </button>
    </div>
    <div id="sheet-tab">
        <template v-if="state.tab === 'basic'">
            <div class="field">
                <label for="edit-title">应用名称</label><input id="edit-title" :ref="editTitleRef" v-model="state.draft.title" autocomplete="off" />
            </div>
            <div class="field">
                <label for="edit-category">分类</label>
                <select id="edit-category" :value="state.draft.category" @change="draftCategoryChanged">
                    <option value="">未分类</option>
                    <option v-for="name in categories" :key="name" :value="name">{{ name }}</option>
                    <option :value="newCategoryOption">＋ 新建分类…</option>
                </select>
            </div>
            <div class="field">
                <label for="edit-type">应用类型</label
                ><select id="edit-type" :value="state.draft.type" @change="draftTypeChanged">
                    <option value="">请选择类型</option>
                    <option value="GUI">图形应用</option>
                    <option value="CLI">命令行应用</option>
                </select>
            </div>
            <div class="field">
                <label for="edit-entry">启动入口</label
                ><input
                    id="edit-entry"
                    v-model="state.draft.entry"
                    :class="{ invalid: state.entryInvalid }"
                    placeholder="app.exe 或 re:正则表达式"
                    autocomplete="off"
                    spellcheck="false"
                    @input="onEntryInput" /><small id="edit-entry-hint">{{ state.entryHint }}</small>
            </div>
            <div class="field">
                <label for="edit-description">备注</label
                ><textarea id="edit-description" v-model="state.draft.description" rows="3" placeholder="自由备注、使用提示或待办事项"></textarea>
            </div>
            <label class="check"><input id="edit-admin" v-model="state.draft.runAsAdmin" type="checkbox" /><span>以管理员身份运行</span></label>
            <div class="sheet-links">
                <button id="edit-open-dir" type="button" class="btn small" @click="openEntryFolder">
                    <span><Icon name="folder" /></span>打开应用目录</button
                ><button id="edit-profile" type="button" class="btn small" @click="editProfile">
                    <span><Icon name="external" /></span>编辑 app.toml
                </button>
            </div>
        </template>

        <template v-else-if="state.tab === 'links'">
            <template v-if="!state.draft.relatedLinks.length">
                <button id="add-link" type="button" class="list-add wide" @click="addLink">
                    <span><Icon name="plus" /></span><b>添加相关链接</b><span>官网、下载页或使用文档等地址</span>
                </button>
            </template>
            <template v-else>
                <p class="link-hint">官网、下载页或使用文档等外部地址。</p>
                <div class="link-list">
                    <div v-for="(link, index) in state.draft.relatedLinks" :key="index" class="link-card" @click="focusLinkTitle($event, index)">
                        <div class="link-fields">
                            <input
                                :ref="linkTitleRef(index)"
                                v-model="link.title"
                                class="link-title"
                                :data-link="index"
                                data-key="title"
                                placeholder="链接名称"
                                :aria-label="'链接名称 ' + (index + 1)"
                                autocomplete="off"
                                spellcheck="false" />
                            <input
                                v-model="link.url"
                                class="link-url"
                                :data-link="index"
                                data-key="url"
                                placeholder="https://"
                                :aria-label="'链接地址 ' + (index + 1)"
                                autocomplete="off"
                                spellcheck="false"
                                @change="normalizeUrl(link)" />
                        </div>
                        <div class="link-actions">
                            <button
                                type="button"
                                class="link-act"
                                :data-open-link="index"
                                title="打开链接"
                                :aria-label="'打开链接 ' + (index + 1)"
                                :disabled="!link.url.trim()"
                                @click="openLink(link.url)">
                                <span><Icon name="external" /></span>
                            </button>
                            <button
                                type="button"
                                class="link-act delete"
                                :data-remove-link="index"
                                title="删除链接"
                                :aria-label="'删除链接 ' + (index + 1)"
                                @click="state.draft.relatedLinks.splice(index, 1)">
                                <span><Icon name="close" /></span>
                            </button>
                        </div>
                    </div>
                </div>
                <button id="add-link" type="button" class="list-add" @click="addLink">
                    <span><Icon name="plus" /></span>添加链接
                </button>
            </template>
        </template>

        <template v-else>
            <template v-if="!state.draft.updater.length">
                <button id="add-updater" type="button" class="list-add wide" @click="addUpdater">
                    <span><Icon name="plus" /></span><b>添加更新操作</b><span>安装更新包时按顺序执行</span>
                </button>
            </template>
            <template v-else>
                <div class="rows">
                    <div v-for="(command, index) in state.draft.updater" :key="index" class="row-item" :data-row="index">
                        <span class="step">{{ index + 1 }}</span>
                        <select :aria-label="'更新操作 ' + (index + 1)" :data-action="index" :value="actionOf(command)" @change="onActionChange(index, $event)">
                            <option v-if="!knownActions.includes(actionOf(command))" :value="actionOf(command)">
                                {{ actionOf(command) || '请选择' }}
                            </option>
                            <option value="PIN">固定文件/目录</option>
                            <option value="RMC">清空目录</option>
                            <option value="UNPKG">解包安装包</option>
                        </select>
                        <input
                            :aria-label="'更新参数 ' + (index + 1)"
                            :data-param="index"
                            :value="parameterOf(command)"
                            :placeholder="actionOf(command) === 'PIN' ? '相对于 content 的路径' : '无需参数'"
                            :disabled="actionOf(command) !== 'PIN'"
                            @input="onParamInput(index, $event)" />
                        <button
                            type="button"
                            class="row-btn delete"
                            :data-remove-action="index"
                            title="删除操作"
                            :aria-label="'删除操作 ' + (index + 1)"
                            @click="removeUpdater(index)">
                            <span><Icon name="close" /></span>
                        </button>
                    </div>
                </div>
                <button id="add-updater" type="button" class="list-add" @click="addUpdater">
                    <span><Icon name="plus" /></span>添加操作
                </button>
            </template>
            <p class="hint">
                操作按顺序执行；清空目录时原有内容移入 content.bak，固定路径保持不动；覆盖安装只备份将被覆盖的文件。<br />固定文件/目录：清空时保留指定相对路径；清空目录：删除未固定的内容；解包安装包：把
                ZIP / 7z 或目录内容安装到 content。
            </p>
            <div class="sheet-footer-note">
                <button id="sheet-update" type="button" class="btn" :disabled="!app?.updater.length" @click="onUpdate">安装更新…</button>
            </div>
        </template>
    </div>
</template>

<style scoped>
.sheet-app {
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 11px;
    border: 1px solid var(--line);
    background: var(--surface-2);
    border-radius: 9px;
    margin-bottom: 14px;
}
.sheet-app :deep(.tile) {
    width: 38px;
    height: 38px;
    border-radius: 9px;
    font-size: 12.5px;
}
.sheet-app :deep(strong) {
    display: block;
    font-size: 12px;
    font-weight: 600;
}
.sheet-app :deep(span) {
    display: block;
    font-size: 10px;
    color: var(--muted);
    margin-top: 2px;
}
.sheet-app :deep(.chip) {
    margin-left: auto;
    flex-shrink: 0;
}
.sheet-footer-note {
    margin-top: 14px;
    display: flex;
    justify-content: flex-end;
}
.sheet-links {
    display: flex;
    gap: 6px;
    margin-top: 2px;
}
.tabs {
    display: flex;
    gap: 2px;
    border-bottom: 1px solid var(--line);
    margin-bottom: 14px;
}
.tab {
    border: 0;
    background: none;
    color: var(--muted);
    font-size: 11px;
    padding: 8px 10px;
    position: relative;
    border-radius: 5px 5px 0 0;
}
.tab:hover {
    color: var(--ink);
    background: var(--surface-2);
}
.tab.active {
    color: var(--accent-ink);
    font-weight: 600;
}
.tab.active::after {
    content: '';
    position: absolute;
    left: 8px;
    right: 8px;
    bottom: -1px;
    height: 2px;
    border-radius: 2px;
    background: var(--accent);
}
.rows {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 12px;
}
.row-item {
    display: flex;
    align-items: center;
    gap: 6px;
}
.row-item input,
.row-item select {
    height: 31px;
    border: 1px solid var(--line-strong);
    background: var(--surface);
    border-radius: 6px;
    padding: 0 9px;
    font-size: 11.5px;
    outline: 0;
    min-width: 0;
    flex: 1;
}
.row-item input:focus,
.row-item select:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
}
.row-item select {
    flex: 0 0 112px;
}
.row-item input:disabled {
    opacity: 0.5;
}
.row-item .step {
    width: 17px;
    flex: 0 0 17px;
    text-align: center;
    font-size: 10px;
    color: var(--faint);
    font-variant-numeric: tabular-nums;
}
.row-btn {
    width: 28px;
    height: 28px;
    flex: 0 0 28px;
    border: 1px solid var(--line);
    background: var(--surface);
    border-radius: 6px;
    display: grid;
    place-items: center;
    color: var(--muted);
}
.row-btn:hover {
    color: var(--accent);
    border-color: var(--accent);
}
.row-btn.delete:hover {
    color: var(--red);
    border-color: var(--red-line);
}
.row-btn :deep(svg) {
    width: 13px;
    height: 13px;
}
.link-hint {
    margin: 0 0 9px;
    font-size: 10px;
    line-height: 1.6;
    color: var(--faint);
}
.link-list {
    display: flex;
    flex-direction: column;
    gap: 7px;
    margin-bottom: 9px;
}
.link-card {
    display: flex;
    align-items: center;
    gap: 4px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--surface-2);
    padding: 6px 8px 7px;
    transition:
        border-color 0.14s,
        box-shadow 0.14s;
}
.link-card:hover {
    border-color: var(--line-strong);
}
.link-card:focus-within {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
}
.link-fields {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
}
.link-actions {
    display: flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
}
.link-title {
    height: 20px;
    border: 0;
    background: none;
    padding: 0 2px;
    font-size: 12px;
    font-weight: 600;
    outline: 0;
}
.link-title::placeholder {
    font-weight: 500;
    color: var(--faint);
}
.link-url {
    width: 100%;
    height: 18px;
    border: 0;
    background: none;
    padding: 0 2px;
    outline: 0;
    font-family: 'Cascadia Mono', Consolas, monospace;
    font-size: 10.5px;
    color: var(--muted);
}
.link-url::placeholder {
    color: var(--faint);
}
.link-act {
    width: 24px;
    height: 24px;
    flex: 0 0 24px;
    border: 0;
    background: none;
    border-radius: 6px;
    display: grid;
    place-items: center;
    color: var(--faint);
    transition: 0.12s;
}
.link-act :deep(svg) {
    width: 13px;
    height: 13px;
}
.link-act:hover {
    background: var(--surface-3);
    color: var(--accent);
}
.link-act.delete:hover {
    background: var(--red-soft);
    color: var(--red);
}
.link-act:disabled {
    opacity: 0.3;
    cursor: default;
}
.link-act:disabled:hover {
    background: none;
    color: var(--faint);
}
.list-add {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    width: 100%;
    min-height: 32px;
    padding: 0 10px;
    border: 1px dashed var(--line-strong);
    border-radius: 8px;
    background: none;
    color: var(--muted);
    font-size: 11px;
    transition: 0.14s;
}
.list-add:hover {
    border-color: var(--accent);
    color: var(--accent-ink);
    background: var(--accent-soft);
}
.list-add :deep(svg) {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
}
.list-add.wide {
    flex-direction: column;
    gap: 2px;
    padding: 18px 12px;
    border-radius: 9px;
    background: var(--surface-2);
}
.list-add.wide :deep(svg) {
    width: 20px;
    height: 20px;
    color: var(--faint);
    margin-bottom: 3px;
}
.list-add.wide b {
    font-size: 11.5px;
    font-weight: 600;
    color: var(--ink);
}
.list-add.wide span {
    font-size: 10px;
    color: var(--faint);
}
.list-add.wide:hover b {
    color: var(--accent-ink);
}
.hint {
    margin: 12px 0 0;
    font-size: 10px;
    line-height: 1.7;
    color: var(--faint);
}
</style>
