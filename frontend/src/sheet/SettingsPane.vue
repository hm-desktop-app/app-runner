<script setup lang="ts">
import { computed } from 'vue';
import { context, openLibrary, quit, state, toggleAutoStart, toggleAutohide, toggleTheme } from './state';

const hotkey = computed(() => (state.settingsInfo?.hotkey || 'Ctrl+Alt+Space').split('+').join(' '));
const libraryPath = computed(() => context().library() || state.settingsInfo?.library || '未知');
</script>

<template>
    <div class="setting-row">
        <div class="setting-copy"><b>全局热键</b><span>窗口隐藏后，随时唤回命令中心</span></div>
        <span id="hotkey-pill" class="hotkey-pill">{{ hotkey }}</span>
    </div>
    <div class="setting-row">
        <div class="setting-copy"><b>关闭时隐藏到托盘</b><span>关闭按钮只收起窗口，右键托盘图标可退出</span></div>
        <span class="chip ok">始终启用</span>
    </div>
    <div class="setting-row">
        <div class="setting-copy"><b>运行后隐藏窗口</b><span>启动应用后自动收起，按热键再次唤起</span></div>
        <button
            id="toggle-autohide"
            type="button"
            class="switch"
            :class="{ on: state.settingsAutohide }"
            role="switch"
            :aria-checked="state.settingsAutohide"
            aria-label="运行后隐藏窗口"
            @click="toggleAutohide"></button>
    </div>
    <div class="setting-row">
        <div class="setting-copy"><b>开机自启</b><span>登录后在托盘常驻，热键随时可用</span></div>
        <button
            id="toggle-autostart"
            type="button"
            class="switch"
            :class="{ on: !!state.settingsInfo?.autoStart }"
            role="switch"
            :aria-checked="!!state.settingsInfo?.autoStart"
            aria-label="开机自启"
            @click="toggleAutoStart"></button>
    </div>
    <div class="setting-row">
        <div class="setting-copy"><b>深色主题</b><span>切换界面配色，立即生效</span></div>
        <button
            id="toggle-theme"
            type="button"
            class="switch"
            :class="{ on: state.settingsThemeDark }"
            role="switch"
            :aria-checked="state.settingsThemeDark"
            aria-label="深色主题"
            @click="toggleTheme"></button>
    </div>
    <div class="setting-row">
        <div class="setting-copy">
            <b>应用库</b><span class="mono">{{ libraryPath }}</span>
        </div>
        <button id="settings-open-library" type="button" class="btn small" @click="openLibrary">打开目录</button>
    </div>
    <p v-if="state.settingsInfo?.hotkeyError" class="sheet-error">全局热键注册失败：{{ state.settingsInfo.hotkeyError }}（可能已被其他程序占用）</p>
    <div class="setting-row">
        <div class="setting-copy"><b>退出 App Runner</b><span>退出后热键与托盘图标都会消失</span></div>
        <button id="settings-quit" type="button" class="btn small danger" @click="quit">退出</button>
    </div>
</template>

<style scoped>
.setting-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    padding: 11px 0;
    border-bottom: 1px solid var(--line);
}
.setting-row:last-child {
    border-bottom: 0;
}
.setting-copy {
    min-width: 0;
}
.setting-copy b {
    display: block;
    font-size: 11.5px;
    font-weight: 600;
    margin-bottom: 2px;
}
.setting-copy span {
    display: block;
    font-size: 10px;
    color: var(--muted);
    line-height: 1.5;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
.switch {
    width: 34px;
    height: 19px;
    border-radius: 99px;
    background: var(--line-strong);
    position: relative;
    border: 0;
    flex-shrink: 0;
    transition: 0.15s;
}
.switch::after {
    content: '';
    position: absolute;
    width: 15px;
    height: 15px;
    border-radius: 50%;
    background: #fff;
    top: 2px;
    left: 2px;
    transition: 0.15s;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}
.switch.on {
    background: var(--accent);
}
.switch.on::after {
    left: 17px;
}
.hotkey-pill {
    font-family: 'Cascadia Mono', Consolas, monospace;
    font-size: 10.5px;
    border: 1px solid var(--line-strong);
    border-radius: 5px;
    padding: 4px 8px;
    background: var(--surface-2);
    color: var(--muted);
    white-space: nowrap;
}
</style>
