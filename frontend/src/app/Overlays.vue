<script setup lang="ts">
import Icon from '../Icon.vue';
import { ref, watch } from 'vue';
import * as dialogs from './dialogs';
import { overlays, shell } from './shell';

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
    () => overlays.category.open,
    (open) => sync(categoryDialog.value, open)
);
watch(
    () => overlays.group.open,
    (open) => sync(groupDialog.value, open)
);
watch(
    () => overlays.update.open,
    (open) => sync(updateDialog.value, open, false)
);

function cancel(event: Event): void {
    if (shell.busy) event.preventDefault();
}

watch(
    () => shell.busy,
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
    <div id="toast" class="toast" :class="{ show: overlays.toast.visible }" role="status" aria-live="polite">
        <i><Icon name="check" /></i><span>{{ overlays.toast.message }}</span>
    </div>
    <div id="activity" ref="activity" class="activity" role="status" popover="manual">
        <span class="spinner" aria-hidden="true"></span><span id="activity-text">{{ overlays.activity }}</span>
    </div>

    <dialog id="category-dialog" ref="categoryDialog" @cancel="cancel" @close="dialogs.categoryDialogClosed()">
        <form id="category-form" @submit.prevent="dialogs.submitCategory()">
            <h2>新建分类</h2>
            <p id="category-dialog-message">{{ overlays.category.message }}</p>
            <label for="category-name">分类名称</label>
            <input id="category-name" v-model="overlays.category.name" autocomplete="off" placeholder="如：开发工具" />
            <p id="category-dialog-error" role="alert">{{ overlays.category.error }}</p>
            <div class="dialog-actions">
                <button id="category-cancel" type="button" @click="dialogs.closeCategoryDialog()">取消</button>
                <button id="category-create" type="submit" class="primary" :disabled="overlays.category.saving">创建分类</button>
            </div>
        </form>
    </dialog>

    <dialog id="group-edit-dialog" ref="groupDialog" @cancel="cancel" @close="dialogs.closeGroupDialog()">
        <form id="group-edit-form" @submit.prevent="dialogs.submitGroup(false)">
            <h2>编辑分类</h2>
            <p id="group-edit-summary">{{ overlays.group.summary }}</p>
            <label for="group-edit-name">分类名称</label>
            <input id="group-edit-name" v-model="overlays.group.name" autocomplete="off" />
            <p id="group-edit-error" role="alert">{{ overlays.group.error }}</p>
            <p class="dialog-hint">删除分类后，组内应用会移至“未分类”。</p>
            <div class="dialog-actions">
                <button id="group-delete" type="button" class="danger-button" :disabled="shell.busy" @click="dialogs.submitGroup(true)">删除分类</button>
                <button id="group-edit-cancel" type="button" @click="dialogs.closeGroupDialog()">取消</button>
                <button id="group-rename" type="submit" class="primary" :disabled="shell.busy">重命名</button>
            </div>
        </form>
    </dialog>

    <dialog id="update-source-dialog" ref="updateDialog" class="source-dialog" @cancel="cancel" @close="dialogs.closeUpdateDialog()">
        <form id="update-source-form" @submit.prevent="dialogs.submitUpdate()">
            <h2>安装更新</h2>
            <p id="update-source-title">{{ overlays.update.title }}</p>
            <div v-show="!overlays.update.installed" id="update-source-fields">
                <label
                    >来源类型
                    <select id="update-kind" v-model="overlays.update.kind" :disabled="shell.busy">
                        <option value="archive">压缩包（ZIP / 7z）</option>
                        <option value="directory">目录</option>
                    </select>
                </label>
                <label
                    >更新来源
                    <div class="source-path">
                        <input
                            id="update-source"
                            v-model="overlays.update.source"
                            placeholder="选择来源，或粘贴完整路径"
                            autocomplete="off"
                            :disabled="shell.busy" />
                        <button id="update-browse" type="button" :disabled="shell.busy || overlays.update.picking" @click="dialogs.browseUpdateSource()">
                            选择…
                        </button>
                    </div>
                </label>
                <small>按已保存的更新步骤安装，开始前把原有内容移入 content.bak。</small>
            </div>
            <p id="update-source-error" role="alert">{{ overlays.update.error }}</p>
            <div v-show="!overlays.update.installed" id="update-source-actions" class="dialog-actions">
                <button id="update-source-cancel" type="button" :disabled="shell.busy" @click="dialogs.closeUpdateDialog()">取消</button>
                <button id="update-install" type="submit" class="primary" :disabled="shell.busy">安装更新</button>
            </div>
            <div v-show="overlays.update.installed" id="update-result-actions" class="dialog-actions">
                <button id="update-delete-backup" type="button" :disabled="shell.busy" @click="dialogs.deleteUpdateBackup()">删除备份</button>
                <button id="update-keep-backup" type="button" class="primary" @click="dialogs.closeUpdateDialog()">保留备份并关闭</button>
            </div>
        </form>
    </dialog>
</template>
