<script setup lang="ts">
import Icon from '../Icon.vue';
import { computed } from 'vue';
import type { Profile } from '../state';
import {
    categoryNames,
    draftCategoryChanged,
    draftTypeChanged,
    editProfile,
    entryChanged,
    newCategoryOption,
    openEntryFolder,
    registerField,
    state
} from './state';

const draft = computed(() => state.draft as Profile);
const categories = computed(() => categoryNames());
const editTitleRef = (element: unknown): void => registerField('edit-title', element as HTMLElement | null);

function onEntryInput(): void {
    entryChanged();
}
</script>

<template>
    <div class="field"><label for="edit-title">应用名称</label><input id="edit-title" :ref="editTitleRef" v-model="draft.title" autocomplete="off" /></div>
    <div class="field">
        <label for="edit-category">分类</label>
        <select id="edit-category" :value="draft.category" @change="draftCategoryChanged">
            <option value="">未分类</option>
            <option v-for="name in categories" :key="name" :value="name">{{ name }}</option>
            <option :value="newCategoryOption">＋ 新建分类…</option>
        </select>
    </div>
    <div class="field">
        <label for="edit-type">应用类型</label
        ><select id="edit-type" :value="draft.type" @change="draftTypeChanged">
            <option value="">请选择类型</option>
            <option value="GUI">图形应用</option>
            <option value="CLI">命令行应用</option>
        </select>
    </div>
    <div class="field">
        <label for="edit-entry">启动入口</label
        ><input
            id="edit-entry"
            v-model="draft.entry"
            :class="{ invalid: state.entryInvalid }"
            placeholder="app.exe 或 re:正则表达式"
            autocomplete="off"
            spellcheck="false"
            @input="onEntryInput" /><small id="edit-entry-hint">{{ state.entryHint }}</small>
    </div>
    <div class="field">
        <label for="edit-description">备注</label
        ><textarea id="edit-description" v-model="draft.description" rows="3" placeholder="自由备注、使用提示或待办事项"></textarea>
    </div>
    <label class="check"><input id="edit-admin" v-model="draft.runAsAdmin" type="checkbox" /><span>以管理员身份运行</span></label>
    <div class="sheet-links">
        <button id="edit-open-dir" type="button" class="btn small" @click="openEntryFolder">
            <span><Icon name="folder" /></span>打开应用目录</button
        ><button id="edit-profile" type="button" class="btn small" @click="editProfile">
            <span><Icon name="external" /></span>编辑 app.toml
        </button>
    </div>
</template>

<style scoped>
.sheet-links {
    display: flex;
    gap: 6px;
    margin-top: 2px;
}
</style>
