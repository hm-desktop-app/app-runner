import {test} from 'node:test';
import assert from 'node:assert/strict';
import {visibleApps,profileOf,age,validation,groupApps,suggestedAppName,statusOf,relativeTime,recentApps,rankApps,filterLibrary,filterKey,parseFilter} from './state.ts';
import type {App} from './state.ts';

const item = (title: string, extras: Partial<App> = {}): App => ({uuid:title,title,category:'',description:'',type:'GUI',entry:'app.exe',runAsAdmin:false,relatedLinks:[],updater:[],directory:title,modified:null,lastUsed:null,version:'',entryValid:true,profileError:'',...extras});
test('sorts timestamps descending, sends unknown dates last, and filters titles', () => {
 const apps = [item('Tool 10'),item('Tool 2',{modified:'2026-10-10T00:00:00Z'}),item('Other',{modified:'2026-10-11T00:00:00Z'})];
 assert.deepEqual(visibleApps(apps,' tool ','name').map(a=>a.title),['Tool 2','Tool 10']);
 assert.deepEqual(visibleApps(apps,'','modified').map(a=>a.title),['Other','Tool 2','Tool 10']);
 assert.equal(apps[0].title,'Tool 10');
});
test('age boundaries count exactly 30 and 90 days on the older side', () => {
 const now = Date.parse('2026-10-11T00:00:00Z'), days = (n: number) => new Date(now-n*86400000).toISOString();
 assert.equal(age(days(29),now),'fresh');assert.equal(age(days(30),now),'warm');assert.equal(age(days(90),now),'old');assert.equal(age(days(-1),now),'fresh');assert.equal(age(null,now),'old');
});
test('editing a draft leaves the stored row unchanged and rejects unsafe pin paths', () => {
 const app = item('App',{relatedLinks:[{title:'Home',url:'example.com'}],updater:['PIN:settings','RMC','UNPKG']});
 const draft = profileOf(app);draft.relatedLinks[0].title = 'Changed';assert.equal(app.relatedLinks[0].title,'Home');
 assert.equal(validation(draft),'');draft.updater = ['PIN:../outside'];assert.match(validation(draft),/content/);
 draft.updater = ['PIN:'];assert.match(validation(draft),/参数/);
});

test('groups categories without losing ordering and searches notes', () => {
 const apps=[item('Tool 10',{category:'开发工具',description:'诊断日志'}),item('Tool 2',{category:' 开发工具 '}),item('Paint',{category:'图形工具'}),item('Legacy')];
 const groups=groupApps(apps,'','name',['空分类']);
 assert.deepEqual(groups.find(g=>g.category==='开发工具')!.apps.map(a=>a.title),['Tool 2','Tool 10']);
 assert.equal(groups.at(-1)!.category,'');
 assert.equal(groups.find(g=>g.category==='空分类')!.apps.length,0);
 assert.deepEqual(groupApps(apps,'诊断','name').flatMap(g=>g.apps).map(a=>a.title),['Tool 10']);
 assert.deepEqual(visibleApps(apps,'开发工具','name').map(a=>a.title),['Tool 2','Tool 10']);
 const draft=profileOf(apps[0]);draft.description='unsaved';assert.equal(apps[0].description,'诊断日志');
 draft.category='bad\ncategory';assert.match(validation(draft),/分类/);
});

test('archive names suggest lower case and hyphens',()=>{
 assert.equal(suggestedAppName('C:\\Downloads\\MY App Build.ZIP'),'my-app-build');
 assert.equal(suggestedAppName('D:\\Apps\\Another   Pack.7Z'),'another-pack');
});

test('status reports broken config before missing entry',()=>{
 assert.equal(statusOf(item('A')),'ok');
 assert.equal(statusOf(item('A',{entryValid:false})),'missing');
 assert.equal(statusOf(item('A',{type:''})),'unset');
 assert.equal(statusOf(item('A',{type:'',entryValid:false,profileError:'无法解析'})),'broken');
});

test('relative time floors to the nearest unit and keeps unknown dates explicit',()=>{
 const now = Date.parse('2026-10-11T12:00:00Z'), ago = (ms: number) => new Date(now-ms).toISOString();
 assert.equal(relativeTime(null,now),'从未使用');
 assert.equal(relativeTime(ago(30*1000),now),'刚刚');
 assert.equal(relativeTime(ago(59*60000),now),'59 分钟前');
 assert.equal(relativeTime(ago(60*60000),now),'1 小时前');
 assert.equal(relativeTime(ago(23*3600000),now),'23 小时前');
 assert.equal(relativeTime(ago(6*86400000),now),'6 天前');
 assert.equal(relativeTime(ago(13*86400000),now),'1 周前');
 assert.equal(relativeTime(ago(61*86400000),now),'2 个月前');
 assert.equal(relativeTime('2027-01-01T00:00:00Z',now),'刚刚');
});

test('recent apps skip never used ones and order by time',()=>{
 const apps=[item('Old',{lastUsed:'2026-01-01T00:00:00Z'}),item('New',{lastUsed:'2026-10-10T00:00:00Z'}),item('Never')];
 assert.deepEqual(recentApps(apps,5).map(a=>a.title),['New','Old']);
 assert.deepEqual(recentApps(apps,1).map(a=>a.title),['New']);
});

test('pinyin and initials find Chinese titles and categories',()=>{
 const apps=[item('微信',{category:'社交'}),item('剪映专业版',{category:'开发工具'}),item('网易云音乐'),item('Obsidian')];
 assert.deepEqual(visibleApps(apps,'wx','name').map(a=>a.title),['微信']);
 assert.deepEqual(visibleApps(apps,'weixin','name').map(a=>a.title),['微信']);
 assert.deepEqual(visibleApps(apps,'jianying','name').map(a=>a.title),['剪映专业版']);
 assert.deepEqual(visibleApps(apps,'kfgj','name').map(a=>a.title),['剪映专业版']);
 assert.deepEqual(visibleApps(apps,'wyyyy','name').map(a=>a.title),['网易云音乐']);
 assert.deepEqual(visibleApps(apps,'obs','name').map(a=>a.title),['Obsidian']);
 assert.deepEqual(rankApps(apps,'wx').map(a=>a.title),['微信']);
});

test('command ranking prefers title matches and falls back to usage',()=>{
 const apps=[item('Code',{lastUsed:'2026-10-01T00:00:00Z'}),item('Visual Studio Code',{lastUsed:'2026-10-09T00:00:00Z'}),item('Editor',{category:'code'}),item('Other',{description:'code runner'}),item('Unrelated')];
 assert.deepEqual(rankApps(apps,'code').map(a=>a.title),['Code','Visual Studio Code','Editor','Other']);
 assert.deepEqual(rankApps(apps,'COD').map(a=>a.title),['Code','Visual Studio Code','Editor','Other']);
 assert.deepEqual(rankApps(apps,'code',2).map(a=>a.title),['Code','Visual Studio Code']);
 assert.deepEqual(rankApps(apps,'').map(a=>a.title),['Visual Studio Code','Code']);
 assert.deepEqual(rankApps(apps,'nothing-matches'),[]);
});

test('library filters scope by window, updater and category before searching',()=>{
 const now = Date.parse('2026-10-11T00:00:00Z'), ago = (days: number) => new Date(now-days*86400000).toISOString();
 const apps=[item('Fresh',{lastUsed:ago(1)}),item('Stale',{lastUsed:ago(10)}),item('Update',{updater:['RMC'],category:'开发工具'}),item('Dev',{category:'开发工具'})];
 assert.deepEqual(filterLibrary(apps,{kind:'all'},'','name').map(a=>a.title),['Dev','Fresh','Stale','Update']);
 assert.deepEqual(filterLibrary(apps,{kind:'recent'},'','name',now).map(a=>a.title),['Fresh']);
 assert.deepEqual(filterLibrary(apps,{kind:'updatable'},'','name',now).map(a=>a.title),['Update']);
 assert.deepEqual(filterLibrary(apps,{kind:'category',name:'开发工具'},'','name',now).map(a=>a.title),['Dev','Update']);
 assert.deepEqual(filterLibrary(apps,{kind:'category',name:'开发工具'},'dev','name',now).map(a=>a.title),['Dev']);
 assert.equal(filterKey({kind:'category',name:'开发工具'}),'category:开发工具');
 assert.deepEqual(parseFilter('category:开发工具'),{kind:'category',name:'开发工具'});
 assert.deepEqual(parseFilter('updatable'),{kind:'updatable'});
 assert.deepEqual(parseFilter('unknown'),{kind:'all'});
});
