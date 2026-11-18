<script setup lang="ts">
import Icon from '../Icon.vue';
import CreatePane from './CreatePane.vue';
import EditPane from './EditPane.vue';
import SettingsPane from './SettingsPane.vue';
import { computed } from 'vue';
import { canSave, createValidation, currentApp, dirty, editError, prepare, requestClose, save, state, submitCreate } from './state';

const title = computed(() => (state.open === 'edit' ? '编辑应用' : state.open === 'create' ? '新建应用' : state.open === 'settings' ? '设置' : ''));
const subtitle = computed(() => {
    if (state.open === 'edit') return currentAppTitle();
    if (state.open === 'create') return state.createStep === 1 ? '第 1 步 · 选择来源' : '第 2 步 · 填写详情';
    if (state.open === 'settings') return '热键、启动与外观';
    return '';
});
const editStatus = computed(() => {
    const error = state.saveError || editError();
    const changed = dirty();
    return { text: error || (changed ? '有未保存的更改' : '所有更改已保存'), dirty: !error && changed, error: !!error };
});
const createStatus = computed(() => {
    const error = state.createError || createValidation();
    return { text: error || '确认后会写入应用配置', error: !!error };
});

function currentAppTitle(): string {
    return currentApp()?.title || '';
}
function onScrim(): void {
    void requestClose();
}
function onClose(): void {
    void requestClose();
}
function onCancel(): void {
    void requestClose();
}
function onSettingsDone(): void {
    void requestClose();
}
</script>

<template>
    <div id="sheet-layer" class="sheet-layer" :class="{ open: state.open !== null }" :aria-hidden="state.open === null ? 'true' : 'false'">
        <div id="sheet-scrim" class="sheet-scrim" @click="onScrim"></div>
        <aside class="sheet" :class="{ wide: state.open === 'create' }" role="dialog" aria-modal="true" aria-labelledby="sheet-title">
            <header class="sheet-head">
                <div>
                    <h2 id="sheet-title">{{ title }}</h2>
                    <p id="sheet-sub">{{ subtitle }}</p>
                </div>
                <button id="sheet-close" type="button" class="sheet-close" aria-label="关闭面板" @click="onClose">
                    <span><Icon name="close" /></span>
                </button>
            </header>

            <div id="sheet-body" class="sheet-body">
                <EditPane v-if="state.open === 'edit' && state.draft" />
                <CreatePane v-else-if="state.open === 'create'" />
                <SettingsPane v-else-if="state.open === 'settings'" />
            </div>

            <footer id="sheet-foot" class="sheet-foot">
                <template v-if="state.open === 'edit'">
                    <span id="sheet-state" class="sheet-state" :class="{ dirty: editStatus.dirty, error: editStatus.error }">{{ editStatus.text }}</span>
                    <div class="sheet-actions">
                        <button id="sheet-cancel" type="button" class="btn" @click="onCancel">取消</button
                        ><button id="sheet-save" type="button" class="btn primary" :disabled="!canSave()" @click="save">保存</button>
                    </div>
                </template>
                <template v-else-if="state.open === 'create' && state.createStep === 1">
                    <span class="sheet-state">先确认来源，再填写应用信息</span>
                    <div class="sheet-actions">
                        <button id="create-cancel" type="button" class="btn" @click="onCancel">取消</button
                        ><button id="create-next" type="button" class="btn primary" :disabled="state.preparing" @click="prepare">识别并继续</button>
                    </div>
                </template>
                <template v-else-if="state.open === 'create'">
                    <span id="sheet-state" class="sheet-state" :class="{ error: createStatus.error }">{{ createStatus.text }}</span>
                    <div class="sheet-actions">
                        <button id="create-back" type="button" class="btn" @click="state.createStep = 1">上一步</button
                        ><button id="create-submit" type="button" class="btn primary" :disabled="createStatus.error || state.saving" @click="submitCreate">
                            创建应用
                        </button>
                    </div>
                </template>
                <template v-else-if="state.open === 'settings'">
                    <span class="sheet-state">设置会立即生效</span>
                    <div class="sheet-actions"><button id="settings-done" type="button" class="btn primary" @click="onSettingsDone">完成</button></div>
                </template>
            </footer>
        </aside>
    </div>
</template>

<style scoped>
.sheet-layer {
    position: absolute;
    inset: 0;
    z-index: 30;
    visibility: hidden;
    pointer-events: none;
    transition: visibility 0s linear 0.22s;
}
.sheet-layer.open {
    visibility: visible;
    pointer-events: auto;
    transition-delay: 0s;
}
.sheet-scrim {
    position: absolute;
    inset: 0;
    background: rgba(30, 28, 22, 0.24);
    backdrop-filter: blur(2px);
    opacity: 0;
    transition: opacity 0.16s ease;
}
.sheet-layer.open .sheet-scrim {
    opacity: 1;
}
.sheet {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: 412px;
    max-width: calc(100% - 30px);
    background: var(--surface);
    border-left: 1px solid var(--line);
    box-shadow: -24px 0 60px rgba(0, 0, 0, 0.14);
    display: flex;
    flex-direction: column;
    transform: translateX(100%);
    transition: transform 0.22s cubic-bezier(0.22, 1, 0.36, 1);
}
.sheet-layer.open .sheet {
    transform: none;
}
.sheet-layer.open .sheet-head,
.sheet-layer.open .sheet-body,
.sheet-layer.open .sheet-foot {
    animation: sheet-part 0.19s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes sheet-part {
    from {
        opacity: 0;
        transform: translateX(10px);
    }
    to {
        opacity: 1;
        transform: none;
    }
}
.sheet.wide {
    width: 460px;
}
.sheet-head {
    height: 58px;
    flex-shrink: 0;
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 14px 18px 0;
    border-bottom: 1px solid var(--line);
}
.sheet-head h2 {
    margin: 0;
    font-size: 14.5px;
    font-weight: 650;
    letter-spacing: -0.01em;
}
.sheet-head p {
    margin: 3px 0 0;
    font-size: 10px;
    color: var(--muted);
}
.sheet-close {
    margin-left: auto;
    width: 28px;
    height: 28px;
    border: 0;
    background: none;
    color: var(--muted);
    display: grid;
    place-items: center;
    border-radius: 6px;
}
.sheet-close:hover {
    background: var(--surface-2);
    color: var(--ink);
}
.sheet-close :deep(svg) {
    width: 14px;
    height: 14px;
}
.sheet-body {
    flex: 1;
    overflow: auto;
    scrollbar-gutter: stable;
    padding: 16px 18px 18px;
}
.sheet-foot {
    min-height: 56px;
    flex-shrink: 0;
    border-top: 1px solid var(--line);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 10px 18px;
    background: var(--surface);
}
.sheet-state {
    font-size: 10px;
    color: var(--faint);
}
.sheet-state.dirty {
    color: var(--amber);
}
.sheet-state.error {
    color: var(--red);
}
.sheet-actions {
    display: flex;
    gap: 7px;
}
.sheet-actions .btn {
    height: 32px;
    font-size: 11.5px;
    padding: 0 12px;
}

@media (max-width: 790px) {
    .sheet {
        width: 100%;
        max-width: 100%;
    }
}
</style>
