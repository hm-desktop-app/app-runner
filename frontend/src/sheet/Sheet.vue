<script setup lang="ts">
import {computed} from 'vue';
import * as API from '../../bindings/app-runner/launcherservice';
import type {App} from '../state';
import {icons} from '../ui';
import {
 actionOf, addLink, addUpdater, canSave, categoryNames, chooseCategory, context,
 createNameChanged, createSourceChanged, createValidation, currentApp, dirty, editError, editProfile,
 entryChanged, knownActions, newCategoryOption, newEntryChanged, normalizeUrl, openEntryFolder, openLink,
 openUpdate, openLibrary, parameterOf, pickSource, prepare, quit, removeUpdater, requestClose, save,
 setUpdaterAction, setUpdaterParam, state, submitCreate, summaryHtml, tabList, toggleAutoStart, toggleAutohide,
 toggleTheme,
} from './state';

const app = computed<App | undefined>(() => currentApp());
const summary = computed(() => summaryHtml());
const categories = computed(() => categoryNames());
const title = computed(() => state.open === 'edit' ? '编辑应用' : state.open === 'create' ? '新建应用' : state.open === 'settings' ? '设置' : '');
const subtitle = computed(() => {
 if (state.open === 'edit') return app.value?.title || '';
 if (state.open === 'create') return state.createStep === 1 ? '第 1 步 · 选择来源' : '第 2 步 · 填写详情';
 if (state.open === 'settings') return '热键、启动与外观';
 return '';
});
const editStatus = computed(() => {
 const error = state.saveError || editError();
 const changed = dirty();
 return {text: error || (changed ? '有未保存的更改' : '所有更改已保存'), dirty: !error && changed, error: !!error};
});
const createStatus = computed(() => {
 const error = state.createError || createValidation();
 return {text: error || '确认后会写入应用配置', error: !!error};
});
const hotkey = computed(() => (state.settingsInfo?.hotkey || 'Ctrl+Alt+Space').split('+').join(' '));
const libraryPath = computed(() => context().library() || state.settingsInfo?.library || '未知');

function onCategoryChange(event: Event): void {
 const select = event.target as HTMLSelectElement;
 const value = select.value;
 if (!state.draft) return;
 if (value !== newCategoryOption) {state.draft.category = value;return;}
 void chooseCategory(value).then(() => {select.value = state.draft?.category || '';});
}
function onEntryInput(): void {entryChanged();}
function onTypeChange(event: Event): void {if (state.draft) state.draft.type = (event.target as HTMLSelectElement).value;}
function onCreateTypeChange(event: Event): void {if (state.draft) state.draft.type = (event.target as HTMLSelectElement).value;}
function onKindChange(event: Event): void {state.createKind = (event.target as HTMLSelectElement).value;}
function onSourceInput(event: Event): void {state.createSource = (event.target as HTMLInputElement).value;createSourceChanged();}
function onNameInput(event: Event): void {state.createName = (event.target as HTMLInputElement).value;createNameChanged();}
function onEntryInputCreate(): void {newEntryChanged();}
function focusLinkTitle(event: Event): void {
 const target = event.target as HTMLElement;
 if (target.closest('input,button')) return;
 (event.currentTarget as HTMLElement).querySelector<HTMLInputElement>('.link-title')?.focus();
}
function onParamInput(index: number, event: Event): void {setUpdaterParam(index, (event.target as HTMLInputElement).value);}
function onActionChange(index: number, event: Event): void {setUpdaterAction(index, (event.target as HTMLSelectElement).value);}
function onUpdate(): void {void openUpdate();}
function onSettingsDone(): void {void requestClose();}
function onScrim(): void {void requestClose();}
function onClose(): void {void requestClose();}
function onCancel(): void {void requestClose();}
</script>

<template>
 <div id="sheet-layer" class="sheet-layer" :class="{open: state.open !== null}" :aria-hidden="state.open === null ? 'true' : 'false'">
  <div class="sheet-scrim" id="sheet-scrim" @click="onScrim"></div>
  <aside class="sheet" :class="{wide: state.open === 'create'}" role="dialog" aria-modal="true" aria-labelledby="sheet-title">
   <header class="sheet-head">
    <div><h2 id="sheet-title">{{ title }}</h2><p id="sheet-sub">{{ subtitle }}</p></div>
    <button type="button" class="sheet-close" id="sheet-close" aria-label="关闭面板" @click="onClose"><span v-html="icons.close"></span></button>
   </header>

   <div class="sheet-body" id="sheet-body">
    <template v-if="state.open === 'edit' && state.draft">
     <div class="sheet-app" v-html="summary"></div>
     <div class="tabs" role="tablist">
      <button v-for="item in tabList" :key="item.id" type="button" class="tab" :class="{active: state.tab === item.id}" :data-tab="item.id" role="tab" :aria-selected="state.tab === item.id" @click="state.tab = item.id">{{ item.label }}</button>
     </div>
     <div id="sheet-tab">
      <template v-if="state.tab === 'basic'">
       <div class="field"><label for="edit-title">应用名称</label><input id="edit-title" v-model="state.draft.title" autocomplete="off"></div>
       <div class="field"><label for="edit-category">分类</label>
        <select id="edit-category" :value="state.draft.category" @change="onCategoryChange">
         <option value="">未分类</option>
         <option v-for="name in categories" :key="name" :value="name">{{ name }}</option>
         <option :value="newCategoryOption">＋ 新建分类…</option>
        </select>
       </div>
       <div class="field"><label for="edit-type">应用类型</label><select id="edit-type" :value="state.draft.type" @change="onTypeChange"><option value="">请选择类型</option><option value="GUI">图形应用</option><option value="CLI">命令行应用</option></select></div>
       <div class="field"><label for="edit-entry">启动入口</label><input id="edit-entry" v-model="state.draft.entry" :class="{invalid: state.entryInvalid}" placeholder="app.exe 或 re:正则表达式" autocomplete="off" spellcheck="false" @input="onEntryInput"><small id="edit-entry-hint">{{ state.entryHint }}</small></div>
       <div class="field"><label for="edit-description">备注</label><textarea id="edit-description" v-model="state.draft.description" rows="3" placeholder="自由备注、使用提示或待办事项"></textarea></div>
       <label class="check"><input type="checkbox" id="edit-admin" v-model="state.draft.runAsAdmin"><span>以管理员身份运行</span></label>
       <div class="sheet-links"><button type="button" class="btn small" id="edit-open-dir" @click="openEntryFolder"><span v-html="icons.folder"></span>打开应用目录</button><button type="button" class="btn small" id="edit-profile" @click="editProfile"><span v-html="icons.external"></span>编辑 app.toml</button></div>
      </template>

      <template v-else-if="state.tab === 'links'">
       <template v-if="!state.draft.relatedLinks.length">
        <button type="button" class="list-add wide" id="add-link" @click="addLink"><span v-html="icons.plus"></span><b>添加相关链接</b><span>官网、下载页或使用文档等地址</span></button>
       </template>
       <template v-else>
        <p class="link-hint">官网、下载页或使用文档等外部地址。</p>
        <div class="link-list">
         <div v-for="(link, index) in state.draft.relatedLinks" :key="index" class="link-card" @click="focusLinkTitle">
          <div class="link-fields">
           <input class="link-title" v-model="link.title" :data-link="index" data-key="title" placeholder="链接名称" :aria-label="'链接名称 ' + (index + 1)" autocomplete="off" spellcheck="false">
           <input class="link-url" v-model="link.url" :data-link="index" data-key="url" placeholder="https://" :aria-label="'链接地址 ' + (index + 1)" autocomplete="off" spellcheck="false" @change="normalizeUrl(link)">
          </div>
          <div class="link-actions">
           <button type="button" class="link-act" :data-open-link="index" title="打开链接" :aria-label="'打开链接 ' + (index + 1)" :disabled="!link.url.trim()" @click="openLink(link.url)"><span v-html="icons.external"></span></button>
           <button type="button" class="link-act delete" :data-remove-link="index" title="删除链接" :aria-label="'删除链接 ' + (index + 1)" @click="state.draft.relatedLinks.splice(index, 1)"><span v-html="icons.close"></span></button>
          </div>
         </div>
        </div>
        <button type="button" class="list-add" id="add-link" @click="addLink"><span v-html="icons.plus"></span>添加链接</button>
       </template>
      </template>

      <template v-else>
       <template v-if="!state.draft.updater.length">
        <button type="button" class="list-add wide" id="add-updater" @click="addUpdater"><span v-html="icons.plus"></span><b>添加更新操作</b><span>安装更新包时按顺序执行</span></button>
       </template>
       <template v-else>
        <div class="rows">
         <div v-for="(command, index) in state.draft.updater" :key="index" class="row-item" :data-row="index">
          <span class="step">{{ index + 1 }}</span>
          <select :aria-label="'更新操作 ' + (index + 1)" :data-action="index" :value="actionOf(command)" @change="onActionChange(index, $event)">
           <option v-if="!knownActions.includes(actionOf(command))" :value="actionOf(command)">{{ actionOf(command) || '请选择' }}</option>
           <option value="PIN">固定文件/目录</option>
           <option value="RMC">清空目录</option>
           <option value="UNPKG">解包安装包</option>
          </select>
          <input :aria-label="'更新参数 ' + (index + 1)" :data-param="index" :value="parameterOf(command)" :placeholder="actionOf(command) === 'PIN' ? '相对于 content 的路径' : '无需参数'" :disabled="actionOf(command) !== 'PIN'" @input="onParamInput(index, $event)">
          <button type="button" class="row-btn delete" :data-remove-action="index" title="删除操作" :aria-label="'删除操作 ' + (index + 1)" @click="removeUpdater(index)"><span v-html="icons.close"></span></button>
         </div>
        </div>
        <button type="button" class="list-add" id="add-updater" @click="addUpdater"><span v-html="icons.plus"></span>添加操作</button>
       </template>
       <p class="hint">操作按顺序执行；清空目录时原有内容移入 content.bak，固定路径保持不动；覆盖安装只备份将被覆盖的文件。<br>固定文件/目录：清空时保留指定相对路径；清空目录：删除未固定的内容；解包安装包：把 ZIP / 7z 或目录内容安装到 content。</p>
       <div class="sheet-footer-note"><button type="button" class="btn" id="sheet-update" :disabled="!app?.updater.length" @click="onUpdate">安装更新…</button></div>
      </template>
     </div>
    </template>

    <template v-else-if="state.open === 'create'">
     <div v-if="state.createStep === 1">
      <div class="step-line"><span class="step-dot">1</span><b>来源</b><span>→</span><span>2 填写详情</span></div>
      <div class="dropzone"><span v-html="icons.folder"></span><strong>选择软件目录或压缩包</strong><span>支持目录、ZIP、7z；也可以直接粘贴完整路径</span></div>
      <div class="field"><label for="create-kind">来源类型</label><select id="create-kind" :value="state.createKind" @change="onKindChange"><option value="directory">目录</option><option value="archive">压缩包（ZIP / 7z）</option></select></div>
      <div class="field"><label for="create-source">来源路径</label><div class="input-row"><input id="create-source" :value="state.createSource" placeholder="例如 D:\Downloads\my-app.zip" autocomplete="off" spellcheck="false" @input="onSourceInput"><button type="button" class="btn" id="create-browse" @click="pickSource">选择…</button></div></div>
      <div class="field"><label for="create-name">应用目录名</label><input id="create-name" :value="state.createName" placeholder="如：my-app" autocomplete="off" @input="onNameInput"><small>将部署到应用库中的同名目录；已有目录不会被覆盖。</small></div>
      <p class="sheet-error" id="create-error" role="alert">{{ state.createError }}</p>
     </div>
     <div v-else-if="state.pendingNew && state.draft">
      <div class="step-line"><span class="step-dot ok">1</span><span>来源</span><span>→</span><span class="step-dot">2</span><b>填写详情</b></div>
      <div class="notice">文件已准备到 <strong>{{ state.pendingNew.target }}</strong>，填写应用信息后才会写入配置。</div>
      <div class="field"><label for="create-title">应用名称</label><input id="create-title" v-model="state.draft.title" autocomplete="off"></div>
      <div class="field"><label for="create-category">分类</label>
       <select id="create-category" :value="state.draft.category" @change="onCategoryChange">
        <option value="">未分类</option>
        <option v-for="name in categories" :key="name" :value="name">{{ name }}</option>
        <option :value="newCategoryOption">＋ 新建分类…</option>
       </select>
      </div>
      <div class="field"><label for="create-type">应用类型</label><select id="create-type" :value="state.draft.type" @change="onCreateTypeChange"><option value="">请选择类型</option><option value="GUI">图形应用</option><option value="CLI">命令行应用</option></select></div>
      <div class="field"><label for="create-entry">启动入口</label><input id="create-entry" list="create-entries" v-model="state.draft.entry" :class="{invalid: state.entryInvalid}" placeholder="app.exe 或 re:正则表达式" autocomplete="off" spellcheck="false" @input="onEntryInputCreate"><datalist id="create-entries"><option v-for="entry in state.pendingNew.entries || []" :key="entry" :value="entry"></option></datalist><small id="create-entry-hint">{{ state.entryHint }}</small></div>
      <div class="field"><label for="create-description">备注</label><textarea id="create-description" v-model="state.draft.description" rows="3" placeholder="自由备注、使用提示或待办事项"></textarea></div>
      <label class="check"><input type="checkbox" id="create-admin" v-model="state.draft.runAsAdmin"><span>以管理员身份运行</span></label>
      <p class="sheet-error" id="create-error" role="alert">{{ state.createError }}</p>
     </div>
    </template>

    <template v-else-if="state.open === 'settings'">
     <div class="setting-row"><div class="setting-copy"><b>全局热键</b><span>窗口隐藏后，随时唤回命令中心</span></div><span class="hotkey-pill" id="hotkey-pill">{{ hotkey }}</span></div>
     <div class="setting-row"><div class="setting-copy"><b>关闭时隐藏到托盘</b><span>关闭按钮只收起窗口，右键托盘图标可退出</span></div><span class="chip ok">始终启用</span></div>
     <div class="setting-row"><div class="setting-copy"><b>运行后隐藏窗口</b><span>启动应用后自动收起，按热键再次唤起</span></div><button type="button" class="switch" :class="{on: state.settingsAutohide}" id="toggle-autohide" role="switch" :aria-checked="state.settingsAutohide" aria-label="运行后隐藏窗口" @click="toggleAutohide"></button></div>
     <div class="setting-row"><div class="setting-copy"><b>开机自启</b><span>登录后在托盘常驻，热键随时可用</span></div><button type="button" class="switch" :class="{on: !!state.settingsInfo?.autoStart}" id="toggle-autostart" role="switch" :aria-checked="!!state.settingsInfo?.autoStart" aria-label="开机自启" @click="toggleAutoStart"></button></div>
     <div class="setting-row"><div class="setting-copy"><b>深色主题</b><span>切换界面配色，立即生效</span></div><button type="button" class="switch" :class="{on: state.settingsThemeDark}" id="toggle-theme" role="switch" :aria-checked="state.settingsThemeDark" aria-label="深色主题" @click="toggleTheme"></button></div>
     <div class="setting-row"><div class="setting-copy"><b>应用库</b><span class="mono">{{ libraryPath }}</span></div><button type="button" class="btn small" id="settings-open-library" @click="openLibrary">打开目录</button></div>
     <p v-if="state.settingsInfo?.hotkeyError" class="sheet-error">全局热键注册失败：{{ state.settingsInfo.hotkeyError }}（可能已被其他程序占用）</p>
     <div class="setting-row"><div class="setting-copy"><b>退出 App Runner</b><span>退出后热键与托盘图标都会消失</span></div><button type="button" class="btn small danger" id="settings-quit" @click="quit">退出</button></div>
    </template>
   </div>

   <footer class="sheet-foot" id="sheet-foot">
    <template v-if="state.open === 'edit'">
     <span class="sheet-state" id="sheet-state" :class="{dirty: editStatus.dirty, error: editStatus.error}">{{ editStatus.text }}</span>
     <div class="sheet-actions"><button type="button" class="btn" id="sheet-cancel" @click="onCancel">取消</button><button type="button" class="btn primary" id="sheet-save" :disabled="!canSave()" @click="save">保存</button></div>
    </template>
    <template v-else-if="state.open === 'create' && state.createStep === 1">
     <span class="sheet-state">先确认来源，再填写应用信息</span>
     <div class="sheet-actions"><button type="button" class="btn" id="create-cancel" @click="onCancel">取消</button><button type="button" class="btn primary" id="create-next" :disabled="state.preparing" @click="prepare">识别并继续</button></div>
    </template>
    <template v-else-if="state.open === 'create'">
     <span class="sheet-state" id="sheet-state" :class="{error: createStatus.error}">{{ createStatus.text }}</span>
     <div class="sheet-actions"><button type="button" class="btn" id="create-back" @click="state.createStep = 1">上一步</button><button type="button" class="btn primary" id="create-submit" :disabled="createStatus.error || state.saving" @click="submitCreate">创建应用</button></div>
    </template>
    <template v-else-if="state.open === 'settings'">
     <span class="sheet-state">设置会立即生效</span>
     <div class="sheet-actions"><button type="button" class="btn primary" id="settings-done" @click="onSettingsDone">完成</button></div>
    </template>
   </footer>
  </aside>
 </div>
</template>
