<script setup lang="ts">
import Icon from '../Icon.vue';
import { computed } from 'vue';
import type { App, Profile } from '../state';
import { actionOf, addUpdater, currentApp, knownActions, openUpdate, parameterOf, removeUpdater, setUpdaterAction, setUpdaterParam, state } from './state';

const draft = computed(() => state.draft as Profile);
const app = computed<App | undefined>(() => currentApp());

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
    <template v-if="!draft.updater.length">
        <button id="add-updater" type="button" class="list-add wide" @click="addUpdater">
            <span><Icon name="plus" /></span><b>添加更新操作</b><span>安装更新包时按顺序执行</span>
        </button>
    </template>
    <template v-else>
        <div class="rows">
            <div v-for="(command, index) in draft.updater" :key="index" class="row-item" :data-row="index">
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

<style scoped>
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
.hint {
    margin: 12px 0 0;
    font-size: 10px;
    line-height: 1.7;
    color: var(--faint);
}
.sheet-footer-note {
    margin-top: 14px;
    display: flex;
    justify-content: flex-end;
}
</style>
