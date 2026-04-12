const BASE = '/api';

async function req(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' }
  };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(BASE + path, opts);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  return res.json();
}

export const api = {
  getClasses:      ()             => req('GET',   '/classes'),
  getClass:        (id)           => req('GET',   `/classes/${id}`),
  updateClass:     (id, data)     => req('PATCH',  `/classes/${id}`, data),
  deleteClass:     (id)           => req('DELETE', `/classes/${id}`),
  toggleTask:      (id, completed)=> req('PATCH', `/tasks/${id}/toggle`, { completed }),
  getWeek:         (date)         => req('GET',   `/week${date ? `?date=${date}` : ''}`),
  getBurndown:     ()             => req('GET',   '/stats/burndown'),
  getHeatmap:      ()             => req('GET',   '/stats/heatmap'),
  getSummary:      ()             => req('GET',   '/stats/summary'),
  importPreview:   (text, year)   => req('POST',  '/import/preview', { text, baseYear: year }),
  importConfirm:   (classId, text)=> req('POST',  '/import', { classId, text }),
};

// Class color palette — indexed by position in classes array
export const CLASS_COLORS = [
  { bg: 'bg-indigo-500/10', border: 'border-indigo-500/30', text: 'text-indigo-400', dot: '#6366f1' },
  { bg: 'bg-cyan-500/10',   border: 'border-cyan-500/30',   text: 'text-cyan-400',   dot: '#06b6d4' },
  { bg: 'bg-amber-500/10',  border: 'border-amber-500/30',  text: 'text-amber-400',  dot: '#f59e0b' },
  { bg: 'bg-rose-500/10',   border: 'border-rose-500/30',   text: 'text-rose-400',   dot: '#f43f5e' },
];

export function classColor(idx) {
  return CLASS_COLORS[idx % CLASS_COLORS.length];
}

export function fmtDate(iso) {
  if (!iso) return '';
  const [y, m, d] = iso.split('-').map(Number);
  return new Date(y, m - 1, d).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric'
  });
}

export function fmtWeekRange(start, end) {
  return `${fmtDate(start)} – ${fmtDate(end)}`;
}

export function todayISO() {
  return new Date().toISOString().slice(0, 10);
}

export function addDays(iso, n) {
  const [y, m, d] = iso.split('-').map(Number);
  const dt = new Date(y, m - 1, d + n);
  return dt.toISOString().slice(0, 10);
}
