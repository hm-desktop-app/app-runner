import type {App, AppStatus} from './state';

export const $ = <T extends HTMLElement = HTMLElement>(selector: string): T => document.querySelector(selector)!;
export const $$ = <T extends HTMLElement = HTMLElement>(selector: string): T[] => Array.from(document.querySelectorAll<T>(selector));
const entities: Record<string,string> = {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'};
export const escape = (value: string): string => String(value).replace(/[&<>"']/g, ch => entities[ch]!);

const svg = (body: string, attributes = 'fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"'): string => '<svg viewBox="0 0 24 24" ' + attributes + '>' + body + '</svg>';
export const icons = {
 play: svg('<path d="M8 5.5a1 1 0 0 1 1.5-.87l9 6.5a1 1 0 0 1 0 1.74l-9 6.5A1 1 0 0 1 8 18.5z" fill="currentColor" stroke="none"/>'),
 search: svg('<circle cx="11" cy="11" r="7"/><path d="m20 20-3.2-3.2"/>', 'fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"'),
 plus: svg('<path d="M12 5v14M5 12h14"/>', 'fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"'),
 grid: svg('<rect x="4" y="4" width="6" height="6" rx="1"/><rect x="14" y="4" width="6" height="6" rx="1"/><rect x="4" y="14" width="6" height="6" rx="1"/><rect x="14" y="14" width="6" height="6" rx="1"/>'),
 clock: svg('<circle cx="12" cy="12" r="8"/><path d="M12 8v4.4l3 1.8"/>'),
 update: svg('<path d="M20 11a8 8 0 1 0-2.3 6.3"/><path d="M20 5v6h-6"/>'),
 settings: svg('<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>'),
 sun: svg('<circle cx="12" cy="12" r="4.2"/><path d="M12 2.6v2.2M12 19.2v2.2M4.4 4.4l1.6 1.6M18 18l1.6 1.6M2.6 12h2.2M19.2 12h2.2M4.4 19.6 6 18M18 6l1.6-1.6"/>'),
 moon: svg('<path d="M20 14.4A8.4 8.4 0 0 1 9.6 4a8.4 8.4 0 1 0 10.4 10.4z"/>'),
 edit: svg('<path d="m4 20 4.2-1 10.4-10.4a2 2 0 0 0 0-2.8l-.4-.4a2 2 0 0 0-2.8 0L5 15.8z"/><path d="m13.5 7.5 3 3"/>'),
 folder: svg('<path d="M3.5 7.5A1.5 1.5 0 0 1 5 6h4l2 2h8a1.5 1.5 0 0 1 1.5 1.5v8A1.5 1.5 0 0 1 19 19H5a1.5 1.5 0 0 1-1.5-1.5z"/>'),
 check: svg('<path d="m5 12 4.2 4.2L19 6.5"/>', 'fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"'),
 close: svg('<path d="m6 6 12 12M18 6 6 18"/>', 'fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"'),
 external: svg('<path d="M14 4h6v6M20 4l-9 9"/><path d="M19 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h5"/>'),
 refresh: svg('<path d="M20 11a8 8 0 1 0-2.3 6.3"/><path d="M20 5v6h-6"/>'),
};

const statusNames: Record<AppStatus,{label: string; tone: string}> = {
 ok: {label: '正常', tone: 'ok'},
 missing: {label: '入口缺失', tone: 'warn'},
 broken: {label: '配置异常', tone: 'warn'},
 unset: {label: '未配置', tone: 'muted'},
};
export function statusName(status: AppStatus): {label: string; tone: string} { return statusNames[status]; }

/** 无图标时用标题哈希得到稳定的低饱和配色，避免整屏灰块。 */
export function tileTone(title: string): string {
 let hash = 0;
 for (const character of title) hash = (hash * 31 + character.codePointAt(0)!) % 360;
 return '--a:hsl(' + hash + ' 30% 44%);--b:hsl(' + (hash + 28) % 360 + ' 34% 58%)';
}
const failedIcons = new Set<string>();
/** 记住解码失败的图标，避免每次重绘都重新请求同一个坏地址。 */
export function iconFailed(url: string): void {failedIcons.add(url);}
export function tile(app: App, extraClass = '', lazy = false): string {
 const mark = escape(Array.from(app.title || '?')[0] || '?');
 const image = app.iconURL && !failedIcons.has(app.iconURL) ? '<img class="tile-icon" src="' + escape(app.iconURL) + '" alt="" decoding="async"' + (lazy ? ' loading="lazy"' : '') + '>' : '';
 return '<span class="tile ' + extraClass + '" style="' + tileTone(app.title) + '" aria-hidden="true"><span class="tile-mark">' + mark + '</span>' + image + '</span>';
}
