export interface Link {title: string; url: string}
export interface Profile {
 uuid: string; category: string; description: string; title: string; type: string; entry: string; runAsAdmin: boolean;
 relatedLinks: Link[]; updater: string[];
}
export interface App extends Profile {
 directory: string; modified: string | null; lastUsed: string | null;
 iconURL?: string; version: string; entryValid: boolean; profileError: string;
}
export type SortMode = 'name' | 'modified' | 'recent';
const collator = new Intl.Collator('zh-CN', {numeric: true, sensitivity: 'base'});
export function visibleApps(apps: App[], query: string, mode: SortMode): App[] {
 return apps.filter(a => [a.title, a.category || '', a.description || ''].join(' ').toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())).sort((a,b) => {
  if (mode !== 'name') {
   const key = mode === 'modified' ? 'modified' : 'lastUsed';
   const delta = timestamp(b[key]) - timestamp(a[key]);
   if (delta) return delta;
  }
  return collator.compare(a.title,b.title) || collator.compare(a.directory,b.directory);
 });
}
function timestamp(value: string | null): number {return value ? Date.parse(value) || 0 : 0}
export function profileOf(app: App): Profile {
 return structuredClone({uuid: app.uuid, category: app.category || '', description: app.description || '', title: app.title, type: app.type, entry: app.entry,
  runAsAdmin: app.runAsAdmin, relatedLinks: app.relatedLinks || [], updater: app.updater || []});
}
export function validation(profile: Profile): string {
 if (/[\r\n\0]/.test(profile.category)) return '分类名称不能包含换行或空字符';
 if (!profile.title.trim()) return '请填写应用标题';
 if (!['GUI','CLI'].includes(profile.type)) return '请选择应用类型';
 if (!profile.entry.trim()) return '请填写启动入口';
 for (const command of profile.updater) {
  const index = command.indexOf(':');
  const action = (index < 0 ? command : command.slice(0,index)).trim();
  const parameter = index < 0 ? '' : command.slice(index+1).trim();
  if (!['PIN','RMC','UNPKG'].includes(action)) return '未知更新操作：' + action;
  if (action === 'PIN' && !parameter) return '固定文件/目录需要填写参数';
  if (action === 'PIN' && (parameter.replaceAll('\\','/').split('/').includes('..') || parameter.includes(':') || /^[\\/]/.test(parameter))) return '固定路径必须位于 content 目录内';
  if (action !== 'PIN' && index >= 0) return action + ' 不接受参数';
 }
 return '';
}
export function age(value: string | null, now = Date.now()): string {
 if (!value) return 'old';
 const days = (now - timestamp(value)) / 86400000;
 return days < 30 ? 'fresh' : days < 90 ? 'warm' : 'old';
}

export interface CategoryGroup {category: string; apps: App[]}
export function categoryOf(app: App): string {
 const value = (app.category || '').trim();
 return value === '未分类' ? '' : value;
}
export function groupApps(apps: App[], query: string, mode: SortMode, extra: Iterable<string> = []): CategoryGroup[] {
 const groups = new Map<string, App[]>();
 if (!query.trim()) {
  groups.set('', []);
  for (const category of extra) if (category.trim()) groups.set(category.trim(), []);
 }
 for (const app of visibleApps(apps, query, mode)) {
  const category = categoryOf(app);
  if (!groups.has(category)) groups.set(category, []);
  groups.get(category)!.push(app);
 }
 return Array.from(groups, ([category, apps]) => ({category, apps}))
  .sort((a,b)=>a.category === '' ? 1 : b.category === '' ? -1 : collator.compare(a.category,b.category));
}

export interface NewAppInfo {token: string; name: string; source: string; target: string; entries: string[]}
export function suggestedAppName(source: string): string {
 return source.trim().replace(/[\\/]+$/,'').split(/[\\/]/).at(-1)!.replace(/\.(zip|7z)$/i,'').toLocaleLowerCase().replace(/\s+/g,'-');
}

/** 应用在列表里显示的状态；'broken' 表示原配置读不出来，'unset' 表示类型还没选。 */
export type AppStatus = 'ok' | 'missing' | 'broken' | 'unset';
export function statusOf(app: App): AppStatus {
 if (app.profileError) return 'broken';
 if (!['GUI','CLI'].includes(app.type)) return 'unset';
 return app.entryValid ? 'ok' : 'missing';
}
export function relativeTime(value: string | null, now = Date.now()): string {
 const time = timestamp(value);
 if (!time) return '从未使用';
 const minutes = Math.floor(Math.max(0, now - time) / 60000);
 if (minutes < 1) return '刚刚';
 if (minutes < 60) return minutes + ' 分钟前';
 const hours = Math.floor(minutes / 60);
 if (hours < 24) return hours + ' 小时前';
 const days = Math.floor(hours / 24);
 if (days < 7) return days + ' 天前';
 if (days < 30) return Math.floor(days / 7) + ' 周前';
 return Math.floor(days / 30) + ' 个月前';
}
export function recentApps(apps: App[], limit: number): App[] {
 return apps.filter(app => timestamp(app.lastUsed) > 0)
  .sort((a,b) => timestamp(b.lastUsed) - timestamp(a.lastUsed) || collator.compare(a.title,b.title))
  .slice(0, limit);
}
function score(app: App, needle: string): number {
 const title = app.title.toLocaleLowerCase();
 if (title === needle) return 0;
 if (title.startsWith(needle)) return 1;
 if (title.includes(needle)) return 2;
 if ((app.category || '').toLocaleLowerCase().includes(needle)) return 3;
 if ([app.description, app.entry].join(' ').toLocaleLowerCase().includes(needle)) return 4;
 return 9;
}
/** 命令中心的关键词排序：标题匹配优先，其次分类，最后备注与入口。 */
export function rankApps(apps: App[], query: string, limit = 0): App[] {
 const needle = query.trim().toLocaleLowerCase();
 if (!needle) return recentApps(apps, limit > 0 ? limit : apps.length);
 const ranked = apps.map(app => ({app, score: score(app, needle)}))
  .filter(entry => entry.score < 9)
  .sort((a,b) => a.score - b.score || timestamp(b.app.lastUsed) - timestamp(a.app.lastUsed) || collator.compare(a.app.title, b.app.title))
  .map(entry => entry.app);
 return limit > 0 ? ranked.slice(0, limit) : ranked;
}

export type LibraryFilter = {kind: 'all'} | {kind: 'recent'} | {kind: 'updatable'} | {kind: 'category'; name: string};
export function filterKey(filter: LibraryFilter): string {
 return filter.kind === 'category' ? 'category:' + filter.name : filter.kind;
}
export function parseFilter(value: string): LibraryFilter {
 if (value === 'recent') return {kind: 'recent'};
 if (value === 'updatable') return {kind: 'updatable'};
 if (value.startsWith('category:')) return {kind: 'category', name: value.slice('category:'.length)};
 return {kind: 'all'};
}
const RECENT_WINDOW = 7 * 86400000;
export function filterLibrary(apps: App[], filter: LibraryFilter, query: string, mode: SortMode, now = Date.now()): App[] {
 const scoped = apps.filter(app => {
  switch (filter.kind) {
   case 'recent': {const time = timestamp(app.lastUsed);return time > 0 && now - time <= RECENT_WINDOW;}
   case 'updatable': return (app.updater || []).length > 0;
   case 'category': return categoryOf(app) === filter.name;
   default: return true;
  }
 });
 return visibleApps(scoped, query, mode);
}
