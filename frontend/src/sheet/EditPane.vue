<script setup lang="ts">
import Tile from '../Tile.vue';
import { computed } from 'vue';
import { categoryOf, statusOf } from '../state';
import type { App } from '../state';
import { statusName } from '../ui';
import EditBasicTab from './EditBasicTab.vue';
import EditLinksTab from './EditLinksTab.vue';
import EditUpdateTab from './EditUpdateTab.vue';
import { currentApp, state, tabList } from './state';

const app = computed<App | undefined>(() => currentApp());
const appStatus = computed(() => (app.value ? statusName(statusOf(app.value)) : null));
const appType = computed(() => (!app.value ? '' : app.value.type === 'GUI' ? '图形应用' : app.value.type === 'CLI' ? '命令行应用' : '待配置'));
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
        <EditBasicTab v-if="state.tab === 'basic'" />
        <EditLinksTab v-else-if="state.tab === 'links'" />
        <EditUpdateTab v-else />
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
</style>
