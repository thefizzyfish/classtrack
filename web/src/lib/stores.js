import { writable } from 'svelte/store';

// Shared nav badge counts — populated by layout, readable anywhere
export const summary = writable({ overdue: 0, dueToday: 0, dueThisWeek: 0 });
