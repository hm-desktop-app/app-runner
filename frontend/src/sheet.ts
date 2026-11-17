import {createApp} from 'vue';
import Sheet from './sheet/Sheet.vue';
import * as store from './sheet/state';

export type {EditTab, SettingsInfo, SheetContext, SheetName, Theme} from './sheet/state';

createApp(Sheet).mount('#sheet-root');

const bind = (ctx: store.SheetContext): void => store.setContext(ctx);

export const isOpen = (): boolean => store.state.open !== null;
export function openEdit(ctx: store.SheetContext, id: string): void {bind(ctx);store.openEdit(id);}
export function openCreate(ctx: store.SheetContext): void {bind(ctx);store.openCreate();}
export function openSettings(ctx: store.SheetContext): void {bind(ctx);store.openSettings();}
export function requestClose(ctx: store.SheetContext): Promise<boolean> {bind(ctx);return store.requestClose();}
export function handleKey(ctx: store.SheetContext, event: KeyboardEvent): boolean {bind(ctx);return store.handleKey(event);}
