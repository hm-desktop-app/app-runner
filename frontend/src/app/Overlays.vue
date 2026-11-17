<script setup lang="ts">
import { ref, watch } from 'vue';
import * as store from './store';

const categoryDialog = ref<HTMLDialogElement | null>(null);
const groupDialog = ref<HTMLDialogElement | null>(null);
const updateDialog = ref<HTMLDialogElement | null>(null);
const activity = ref<HTMLDivElement | null>(null);

function sync(dialog: HTMLDialogElement | null, open: boolean, focusField = true): void {
    if (!dialog) return;
    if (open) {
        if (!dialog.open) dialog.showModal();
        if (focusField) {
            const field = dialog.querySelector<HTMLInputElement>('form input');
            field?.focus();
            field?.select();
        }
    } else if (dialog.open) {
        dialog.close();
    }
}
watch(
    () => store.overlays.category.open,
    (open) => sync(categoryDialog.value, open)
);
watch(
    () => store.overlays.group.open,
    (open) => sync(groupDialog.value, open)
);
watch(
    () => store.overlays.update.open,
    (open) => sync(updateDialog.value, open, false)
);

function cancel(event: Event): void {
    if (store.shell.busy) event.preventDefault();
}

watch(
    () => store.shell.busy,
    (open) => {
        const node = activity.value;
        if (!node) return;
        if (open && !node.matches(':popover-open')) node.showPopover();
        else if (!open && node.matches(':popover-open')) node.hidePopover();
    },
    { flush: 'post' }
);
</script>

<template>
    <div id="toast" class="toast" :class="{ show: store.overlays.toast.visible }" role="status" aria-live="polite">
        <i v-html="store.icons.check"></i><span>{{ store.overlays.toast.message }}</span>
    </div>
    <div id="activity" ref="activity" class="activity" role="status" popover="manual">
        <span class="spinner" aria-hidden="true"></span><span id="activity-text">{{ store.overlays.activity }}</span>
    </div>

    <dialog id="category-dialog" ref="categoryDialog" @cancel="cancel" @close="store.categoryDialogClosed()">
        <form id="category-form" @submit.prevent="store.submitCategory()">
            <h2>新建分类</h2>
            <p id="category-dialog-message">{{ store.overlays.category.message }}</p>
            <label for="category-name">分类名称</label>
            <input id="category-name" v-model="store.overlays.category.name" autocomplete="off" placeholder="如：开发工具" />
            <p id="category-dialog-error" role="alert">{{ store.overlays.category.error }}</p>
            <div class="dialog-actions">
                <button id="category-cancel" type="button" @click="store.closeCategoryDialog()">取消</button>
                <button id="category-create" type="submit" class="primary" :disabled="store.overlays.category.saving">创建分类</button>
            </div>
        </form>
    </dialog>

    <dialog id="group-edit-dialog" ref="groupDialog" @cancel="cancel" @close="store.closeGroupDialog()">
        <form id="group-edit-form" @submit.prevent="store.submitGroup(false)">
            <h2>编辑分类</h2>
            <p id="group-edit-summary">{{ store.overlays.group.summary }}</p>
            <label for="group-edit-name">分类名称</label>
            <input id="group-edit-name" v-model="store.overlays.group.name" autocomplete="off" />
            <p id="group-edit-error" role="alert">{{ store.overlays.group.error }}</p>
            <p class="dialog-hint">删除分类后，组内应用会移至“未分类”。</p>
            <div class="dialog-actions">
                <button id="group-delete" type="button" class="danger-button" :disabled="store.shell.busy" @click="store.submitGroup(true)">删除分类</button>
                <button id="group-edit-cancel" type="button" @click="store.closeGroupDialog()">取消</button>
                <button id="group-rename" type="submit" class="primary" :disabled="store.shell.busy">重命名</button>
            </div>
        </form>
    </dialog>

    <dialog id="update-source-dialog" ref="updateDialog" class="source-dialog" @cancel="cancel" @close="store.closeUpdateDialog()">
        <form id="update-source-form" @submit.prevent="store.submitUpdate()">
            <h2>安装更新</h2>
            <p id="update-source-title">{{ store.overlays.update.title }}</p>
            <div v-show="!store.overlays.update.installed" id="update-source-fields">
                <label
                    >来源类型
                    <select id="update-kind" v-model="store.overlays.update.kind" :disabled="store.shell.busy">
                        <option value="archive">压缩包（ZIP / 7z）</option>
                        <option value="directory">目录</option>
                    </select>
                </label>
                <label
                    >更新来源
                    <div class="source-path">
                        <input
                            id="update-source"
                            v-model="store.overlays.update.source"
                            placeholder="选择来源，或粘贴完整路径"
                            autocomplete="off"
                            :disabled="store.shell.busy" />
                        <button
                            id="update-browse"
                            type="button"
                            :disabled="store.shell.busy || store.overlays.update.picking"
                            @click="store.browseUpdateSource()">
                            选择…
                        </button>
                    </div>
                </label>
                <small>按已保存的更新步骤安装，开始前把原有内容移入 content.bak。</small>
            </div>
            <p id="update-source-error" role="alert">{{ store.overlays.update.error }}</p>
            <div v-show="!store.overlays.update.installed" id="update-source-actions" class="dialog-actions">
                <button id="update-source-cancel" type="button" :disabled="store.shell.busy" @click="store.closeUpdateDialog()">取消</button>
                <button id="update-install" type="submit" class="primary" :disabled="store.shell.busy">安装更新</button>
            </div>
            <div v-show="store.overlays.update.installed" id="update-result-actions" class="dialog-actions">
                <button id="update-delete-backup" type="button" :disabled="store.shell.busy" @click="store.deleteUpdateBackup()">删除备份</button>
                <button id="update-keep-backup" type="button" class="primary" @click="store.closeUpdateDialog()">保留备份并关闭</button>
            </div>
        </form>
    </dialog>
</template>
