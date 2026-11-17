<script setup lang="ts">
import {computed} from 'vue';
import {commandGroups, categoryOf, execute, hideWindow, hoverCommand, icons, launch, openDirectory, openEdit, openLinkUrl, openUpdateDialog, peekItem, relativeTime, shell, statusName, statusOf, suggestedCount, tile} from './store';
import {suggestsUpdate} from '../state';
import type {AppItem, Item} from './store';

function subText(app: AppItem['app']): string {
 const status = statusName(statusOf(app));
 let text = (categoryOf(app) || '未分类') + ' · ' + (app.type === 'GUI' ? '图形应用' : app.type === 'CLI' ? '命令行应用' : '待配置');
 if (status.tone !== 'ok') text += ' · ' + status.label;
 return text;
}
function rowClick(item: Item): void {
 if (item.kind === 'command') execute(item.key);
 else {hoverCommand(item.key);document.getElementById('cmd-search')?.focus();}
}
function rowDoubleClick(item: Item): void {if (item.kind === 'app') execute(item.key);}
const peek = computed(() => peekItem());
const peekStatus = computed(() => peek.value?.kind === 'app' ? statusName(statusOf(peek.value.app)) : null);
const peekRunnable = computed(() => peek.value?.kind === 'app' && peek.value.app.entryValid && ['GUI','CLI'].includes(peek.value.app.type));
const peekLinks = computed(() => peek.value?.kind === 'app' ? peek.value.app.relatedLinks.filter(link => link.url.trim()) : []);
const peekModified = computed(() => {
 if (peek.value?.kind !== 'app' || !peek.value.app.modified) return '未知';
 return new Date(peek.value.app.modified).toLocaleDateString('zh-CN');
});
</script>

<template>
 <section class="view cmd-view" :class="{'is-active': shell.mode === 'command'}" id="cmd-view" aria-label="命令中心">
  <div class="cmd-search">
   <span class="cmd-search-icon" aria-hidden="true" v-html="icons.search"></span>
   <input id="cmd-search" v-model="shell.commandQuery" placeholder="搜索名称、分类、备注或入口；支持拼音；输入 > 只看命令" autocomplete="off" spellcheck="false" role="combobox" aria-expanded="true" aria-controls="cmd-results" aria-label="搜索应用或命令" @keydown.enter.prevent="shell.commandSelected && execute(shell.commandSelected)">
   <kbd class="esc">Esc</kbd>
   <button type="button" class="win-close" id="cmd-close" title="收起窗口" aria-label="收起窗口" @click="hideWindow"><span v-html="icons.close"></span></button>
  </div>
  <div class="cmd-body">
   <div class="cmd-results" id="cmd-results" role="listbox" aria-label="搜索结果">
    <template v-for="group in commandGroups()" :key="group.kind">
     <div class="cmd-group">{{ group.label }}</div>
     <button v-for="item in group.items" :key="item.key" type="button" class="result" :class="{selected: item.key === shell.commandSelected}" :data-key="item.key" role="option" :aria-selected="item.key === shell.commandSelected" @mouseenter="hoverCommand(item.key)" @click="rowClick(item)" @dblclick="rowDoubleClick(item)">
      <span v-if="item.kind === 'command'" class="cmd-icon" v-html="item.command.icon"></span>
      <span v-else class="result-tile" v-html="tile(item.app)"></span>
      <span class="result-main">
       <span class="result-name">{{ item.kind === 'command' ? item.command.name : item.app.title }}</span>
       <span class="result-sub">{{ item.kind === 'command' ? item.command.sub : subText(item.app) }}</span>
      </span>
      <span v-if="item.kind === 'command'" class="result-key">{{ item.command.key }}</span>
      <span v-else class="result-time">{{ relativeTime(item.app.lastUsed) }}</span>
      <span v-if="item.kind === 'app'" class="result-enter">↵</span>
     </button>
    </template>
    <div v-if="!commandGroups().length" class="cmd-empty">没有匹配“{{ (shell.commandQuery.trim().startsWith('>') ? shell.commandQuery.trim().slice(1).trim() : shell.commandQuery.trim()) }}”<br>试试应用名称、分类、备注、入口，或用 &gt; 前缀只搜命令。</div>
   </div>
   <aside class="cmd-peek" id="cmd-peek" aria-label="预览">
    <template v-if="peek?.kind === 'command'">
     <div class="peek-command">
      <span class="cmd-icon" v-html="peek.command.icon"></span>
      <h2>{{ peek.command.name }}</h2>
      <p>{{ peek.command.sub }}</p>
      <div class="peek-note"><span class="peek-note-title">说明</span>{{ peek.command.note }}</div>
     </div>
     <div class="peek-actions"><button type="button" class="btn primary" :data-exec="peek.key" @click="execute(peek.key)">执行</button></div>
    </template>
    <template v-else-if="peek?.kind === 'app'">
     <div class="peek-head"><span class="result-tile large" v-html="tile(peek.app, 'large')"></span><div class="peek-title"><h2>{{ peek.app.title }}</h2><div class="chips"><span class="chip">{{ categoryOf(peek.app) || '未分类' }}</span><span v-if="peekStatus" class="chip" :class="peekStatus.tone">{{ peekStatus.label }}</span><span v-if="suggestsUpdate(peek.app)" class="chip update">建议更新</span><span v-if="peek.app.runAsAdmin" class="chip">管理员</span></div></div></div>
     <p class="peek-desc"><template v-if="peek.app.description">{{ peek.app.description }}</template><span v-else class="faint">还没有备注</span></p>
     <div v-if="peekLinks.length" class="peek-links"><span class="peek-note-title">相关链接</span><button v-for="link in peekLinks" :key="link.url" type="button" class="peek-link" :data-open-url="link.url" @click="openLinkUrl(link.url)"><span>{{ link.title || link.url }}</span><i v-html="icons.external"></i></button></div>
     <button v-if="peek.app.updater.length" type="button" class="update-box" :data-update="peek.app.uuid" @click="openUpdateDialog(peek.app.uuid)"><span v-html="icons.update"></span><b>已配置更新方式</b><span>安装更新 ›</span></button>
     <div class="peek-meta">
      <div class="peek-row"><span>启动入口</span><span class="mono">{{ peek.app.entry || '未填写' }}</span></div>
      <div class="peek-row"><span>版本</span><span class="mono">{{ peek.app.version || '未知' }}</span></div>
      <div class="peek-row"><span>上次使用</span><span>{{ relativeTime(peek.app.lastUsed) }}</span></div>
      <div class="peek-row"><span>更新时间</span><span>{{ peekModified }}</span></div>
      <div class="peek-row"><span>所在目录</span><span class="mono" :title="peek.app.directory">{{ peek.app.directory.split(/[\\/]/).pop() }}</span></div>
     </div>
     <div class="peek-actions">
      <button type="button" class="btn primary" :data-run="peek.app.uuid" :disabled="!peekRunnable" @click="launch(peek.app.uuid)"><span v-html="icons.play"></span>运行</button>
      <div class="btn-row">
       <button type="button" class="btn" :data-edit="peek.app.uuid" @click="openEdit(peek.app.uuid)"><span v-html="icons.edit"></span>编辑</button>
       <button type="button" class="btn" :data-dir="peek.app.uuid" @click="openDirectory(peek.app.uuid)"><span v-html="icons.folder"></span>目录</button>
      </div>
     </div>
     <div class="peek-hint">↵ 运行 · E 编辑 · Esc 清空搜索</div>
    </template>
    <div v-else class="cmd-empty">输入关键词开始搜索</div>
   </aside>
  </div>
  <footer class="cmd-foot">
   <div class="cmd-stats" id="cmd-stats"><b>{{ shell.apps.length }} 个应用</b><span>·</span><span>{{ suggestedCount() }} 个建议更新</span></div>
   <div class="cmd-keys"><span><kbd>↑</kbd><kbd>↓</kbd> 选择</span><span><kbd>↵</kbd> 运行</span><span><kbd>E</kbd> 编辑</span><span><kbd>Ctrl K</kbd> 聚焦</span></div>
  </footer>
 </section>
</template>

<style scoped>
.cmd-search{--wails-draggable:drag}
.cmd-search input,.cmd-search kbd,.cmd-search button{--wails-draggable:no-drag}
.cmd-view{flex-direction:column;height:100%}
.cmd-search{height:62px;flex-shrink:0;display:flex;align-items:center;gap:11px;padding:0 18px;border-bottom:1px solid var(--line);background:var(--surface)}
.cmd-search-icon{display:grid;place-items:center;color:var(--faint);flex-shrink:0}
.cmd-search-icon :deep(svg){width:20px;height:20px}
.cmd-search input{flex:1;min-width:0;border:0;outline:0;background:none;font-size:16px;letter-spacing:-.01em}
.cmd-search input::placeholder{color:var(--faint)}
.cmd-search .esc{opacity:.85}
.cmd-body{flex:1;display:grid;grid-template-columns:minmax(0,1fr) 308px;min-height:0}
.cmd-results{overflow:auto;scrollbar-gutter:stable;padding:8px 8px 14px}
.cmd-group{font-size:10px;font-weight:650;letter-spacing:.1em;color:var(--faint);padding:10px 10px 6px}
.result{display:flex;align-items:center;gap:11px;width:100%;padding:8px 10px;border:0;background:none;border-radius:8px;text-align:left;position:relative;color:var(--ink)}
.result:hover{background:var(--surface-2)}
.result.selected{background:var(--accent-soft)}
.result.selected::before{content:"";position:absolute;left:0;top:9px;bottom:9px;width:2px;border-radius:2px;background:var(--accent)}
.result :deep(.tile){width:34px;height:34px;border-radius:8px;font-size:11.5px}
.result-tile{display:flex;flex-shrink:0}
.result-main{min-width:0;flex:1;display:flex;flex-direction:column;gap:2px}
.result-name{font-size:13.5px;line-height:1.3;font-weight:560;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.result-sub{font-size:11px;line-height:1.3;color:var(--muted);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.result-time{font-size:10px;color:var(--faint);white-space:nowrap;font-variant-numeric:tabular-nums}
.result-enter{width:16px;text-align:center;font-size:10px;color:var(--accent);opacity:0;transition:.12s}
.result.selected .result-enter{opacity:1}
.cmd-icon{width:34px;height:34px;border-radius:8px;display:grid;place-items:center;background:var(--surface-2);border:1px solid var(--line);color:var(--muted);flex-shrink:0}
.cmd-icon :deep(svg){width:15px;height:15px}
.result.selected .cmd-icon{background:var(--accent-soft);border-color:transparent;color:var(--accent)}
.result-key{font-size:10px;color:var(--faint)}
.result.selected .result-key{color:var(--muted)}
.result.selected .result-name{font-weight:650}
.cmd-empty{padding:48px 20px;text-align:center;color:var(--muted);font-size:12px;line-height:1.7}
.cmd-peek{border-left:1px solid var(--line);background:var(--surface-2);padding:20px 18px;display:flex;flex-direction:column;min-height:0;overflow:auto;scrollbar-gutter:stable}
.peek-head{display:flex;align-items:flex-start;gap:12px}
.peek-head :deep(.tile){width:52px;height:52px;border-radius:13px;font-size:18px}
.peek-title{min-width:0;padding-top:1px}
.peek-title h2{margin:0;font-size:16.5px;line-height:1.25;font-weight:650;letter-spacing:-.02em;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.chips{display:flex;flex-wrap:wrap;gap:5px;margin-top:7px}
.peek-desc{margin:13px 0 0;font-size:11.5px;line-height:1.65;color:var(--muted);white-space:pre-wrap}
.update-box{display:flex;align-items:center;gap:8px;width:100%;margin-top:12px;padding:9px 10px;border:1px solid var(--amber-line);border-radius:7px;background:var(--amber-soft);color:var(--amber);font-size:10px;text-align:left}
.update-box :deep(svg){width:13px;height:13px;flex-shrink:0}
.update-box b{font-weight:600}
.update-box span{margin-left:auto;font-size:10px;opacity:.85;white-space:nowrap}
.peek-links{margin-top:13px;display:flex;flex-direction:column;gap:5px}
.peek-link{display:flex;align-items:center;gap:7px;width:100%;height:29px;padding:0 9px;border:1px solid var(--line);border-radius:6px;background:var(--surface);color:var(--ink);font-size:11px;text-align:left;transition:.14s}
.peek-link span{flex:1;min-width:0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.peek-link :deep(svg){width:12px;height:12px;color:var(--faint);flex-shrink:0}
.peek-link:hover{border-color:var(--accent);color:var(--accent-ink)}
.peek-link:hover :deep(svg){color:var(--accent)}
.peek-meta{margin-top:13px;border-top:1px solid var(--line);border-bottom:1px solid var(--line);padding:3px 0}
.peek-row{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:7px 0;font-size:11px}
.peek-row span:first-child{color:var(--muted);flex-shrink:0}
.peek-row span:last-child{text-align:right;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;font-variant-numeric:tabular-nums}
.peek-actions{margin-top:auto;padding-top:15px;display:flex;flex-direction:column;gap:7px}
.peek-hint{margin-top:9px;text-align:center;font-size:9.5px;color:var(--faint)}
.peek-command{flex:1;display:flex;flex-direction:column;justify-content:center}
.peek-note{margin-top:14px;padding-top:12px;border-top:1px solid var(--line);font-size:11px;line-height:1.65;color:var(--muted)}
.peek-note-title{display:block;font-size:9.5px;font-weight:650;letter-spacing:.08em;color:var(--faint);margin-bottom:5px}
.peek-command .cmd-icon{width:52px;height:52px;border-radius:13px;margin-bottom:13px}
.peek-command .cmd-icon :deep(svg){width:21px;height:21px}
.peek-command h2{margin:0;font-size:16.5px;font-weight:650}
.peek-command p{margin:7px 0 0;font-size:11.5px;line-height:1.6;color:var(--muted)}
.cmd-foot{height:38px;flex-shrink:0;border-top:1px solid var(--line);display:flex;align-items:center;gap:11px;padding:0 16px;font-size:10px;color:var(--faint);background:var(--surface)}
.cmd-stats{display:flex;align-items:center;gap:8px}
.cmd-stats b{color:var(--muted);font-weight:500}
.cmd-keys{margin-left:auto;display:flex;align-items:center;gap:10px}
.cmd-foot kbd{font-size:9px;padding:1px 4px}
@media (max-width:900px){.cmd-body{grid-template-columns:1fr}.cmd-peek{display:none}}
</style>
