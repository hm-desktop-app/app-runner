<script setup lang="ts">
import Icon from '../Icon.vue';
import { computed } from 'vue';
import type { Profile } from '../state';
import { addLink, focusField, normalizeUrl, openLink, registerField, state } from './state';

const draft = computed(() => state.draft as Profile);
const linkTitleRef =
    (index: number) =>
    (element: unknown): void =>
        registerField('link-title-' + index, element as HTMLElement | null);

function focusLinkTitle(event: Event, index: number): void {
    const target = event.target as HTMLElement;
    if (target.closest('input,button')) return;
    focusField('link-title-' + index);
}
</script>

<template>
    <template v-if="!draft.relatedLinks.length">
        <button id="add-link" type="button" class="list-add wide" @click="addLink">
            <span><Icon name="plus" /></span><b>添加相关链接</b><span>官网、下载页或使用文档等地址</span>
        </button>
    </template>
    <template v-else>
        <p class="link-hint">官网、下载页或使用文档等外部地址。</p>
        <div class="link-list">
            <div v-for="(link, index) in draft.relatedLinks" :key="index" class="link-card" @click="focusLinkTitle($event, index)">
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
                        @click="draft.relatedLinks.splice(index, 1)">
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

<style scoped>
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
</style>
