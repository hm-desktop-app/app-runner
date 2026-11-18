<script setup lang="ts">
import Icon from '../Icon.vue';
import { computed } from 'vue';
import {
    categoryNames,
    createNameChanged,
    createSourceChanged,
    draftCategoryChanged,
    draftTypeChanged,
    newCategoryOption,
    newEntryChanged,
    pickSource,
    registerField,
    state
} from './state';

const categories = computed(() => categoryNames());
const createSourceRef = (element: unknown): void => registerField('create-source', element as HTMLElement | null);

function onKindChange(event: Event): void {
    state.createKind = (event.target as HTMLSelectElement).value;
}
function onSourceInput(event: Event): void {
    state.createSource = (event.target as HTMLInputElement).value;
    createSourceChanged();
}
function onNameInput(event: Event): void {
    state.createName = (event.target as HTMLInputElement).value;
    createNameChanged();
}
function onEntryInput(): void {
    newEntryChanged();
}
</script>

<template>
    <div v-if="state.createStep === 1">
        <div class="step-line"><span class="step-dot">1</span><b>来源</b><span>→</span><span>2 填写详情</span></div>
        <div class="dropzone">
            <span><Icon name="folder" /></span><strong>选择软件目录或压缩包</strong><span>支持目录、ZIP、7z；也可以直接粘贴完整路径</span>
        </div>
        <div class="field">
            <label for="create-kind">来源类型</label
            ><select id="create-kind" :value="state.createKind" @change="onKindChange">
                <option value="directory">目录</option>
                <option value="archive">压缩包（ZIP / 7z）</option>
            </select>
        </div>
        <div class="field">
            <label for="create-source">来源路径</label>
            <div class="input-row">
                <input
                    id="create-source"
                    :ref="createSourceRef"
                    :value="state.createSource"
                    placeholder="例如 D:\Downloads\my-app.zip"
                    autocomplete="off"
                    spellcheck="false"
                    @input="onSourceInput" /><button id="create-browse" type="button" class="btn" @click="pickSource">选择…</button>
            </div>
        </div>
        <div class="field">
            <label for="create-name">应用目录名</label
            ><input id="create-name" :value="state.createName" placeholder="如：my-app" autocomplete="off" @input="onNameInput" /><small
                >将部署到应用库中的同名目录；已有目录不会被覆盖。</small
            >
        </div>
        <p id="create-error" class="sheet-error" role="alert">{{ state.createError }}</p>
    </div>
    <div v-else-if="state.pendingNew && state.draft">
        <div class="step-line"><span class="step-dot ok">1</span><span>来源</span><span>→</span><span class="step-dot">2</span><b>填写详情</b></div>
        <div class="notice">
            文件已准备到 <strong>{{ state.pendingNew.target }}</strong>，填写应用信息后才会写入配置。
        </div>
        <div class="field"><label for="create-title">应用名称</label><input id="create-title" v-model="state.draft.title" autocomplete="off" /></div>
        <div class="field">
            <label for="create-category">分类</label>
            <select id="create-category" :value="state.draft.category" @change="draftCategoryChanged">
                <option value="">未分类</option>
                <option v-for="name in categories" :key="name" :value="name">{{ name }}</option>
                <option :value="newCategoryOption">＋ 新建分类…</option>
            </select>
        </div>
        <div class="field">
            <label for="create-type">应用类型</label
            ><select id="create-type" :value="state.draft.type" @change="draftTypeChanged">
                <option value="">请选择类型</option>
                <option value="GUI">图形应用</option>
                <option value="CLI">命令行应用</option>
            </select>
        </div>
        <div class="field">
            <label for="create-entry">启动入口</label
            ><input
                id="create-entry"
                v-model="state.draft.entry"
                list="create-entries"
                :class="{ invalid: state.entryInvalid }"
                placeholder="app.exe 或 re:正则表达式"
                autocomplete="off"
                spellcheck="false"
                @input="onEntryInput" /><datalist id="create-entries">
                <option v-for="entry in state.pendingNew.entries || []" :key="entry" :value="entry"></option></datalist
            ><small id="create-entry-hint">{{ state.entryHint }}</small>
        </div>
        <div class="field">
            <label for="create-description">备注</label
            ><textarea id="create-description" v-model="state.draft.description" rows="3" placeholder="自由备注、使用提示或待办事项"></textarea>
        </div>
        <label class="check"><input id="create-admin" v-model="state.draft.runAsAdmin" type="checkbox" /><span>以管理员身份运行</span></label>
        <p id="create-error" class="sheet-error" role="alert">{{ state.createError }}</p>
    </div>
</template>

<style scoped>
.input-row {
    display: flex;
    gap: 6px;
}
.input-row input {
    flex: 1;
    min-width: 0;
}
.input-row .btn {
    height: 33px;
}
.dropzone {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 24px 14px;
    border: 1.5px dashed var(--line-strong);
    border-radius: 9px;
    background: var(--surface-2);
    text-align: center;
    color: var(--muted);
    margin-bottom: 13px;
}
.dropzone :deep(svg) {
    width: 22px;
    height: 22px;
    color: var(--faint);
    margin-bottom: 2px;
}
.dropzone strong {
    font-size: 12px;
    font-weight: 600;
    color: var(--ink);
}
.dropzone span {
    font-size: 10px;
    color: var(--faint);
}
.step-line {
    display: flex;
    align-items: center;
    gap: 7px;
    margin-bottom: 14px;
    font-size: 10px;
    color: var(--faint);
}
.step-line b {
    color: var(--accent-ink);
}
.step-dot {
    width: 17px;
    height: 17px;
    border-radius: 50%;
    display: grid;
    place-items: center;
    background: var(--accent-soft);
    color: var(--accent-ink);
    font-size: 9px;
}
.step-dot.ok {
    background: var(--accent);
    color: var(--on-accent);
}
.notice {
    border: 1px solid var(--accent-soft);
    background: var(--accent-soft);
    color: var(--accent-ink);
    border-radius: 7px;
    padding: 10px 12px;
    font-size: 11px;
    line-height: 1.6;
    margin-bottom: 13px;
}
.notice strong {
    font-weight: 600;
}
</style>
