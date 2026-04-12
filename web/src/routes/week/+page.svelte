<script>
  import { onMount } from 'svelte';
  import { api, fmtDate, addDays } from '$lib/api.js';

  const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

  let weekData = null;
  let loading = true;
  let error = '';
  let anchorDate = new Date().toISOString().slice(0, 10);

  async function load() {
    loading = true;
    error = '';
    try {
      weekData = await api.getWeek(anchorDate);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function prevWeek() { anchorDate = addDays(weekData.weekStart, -7); load(); }
  function nextWeek() { anchorDate = addDays(weekData.weekStart,  7); load(); }

  const today = new Date().toISOString().slice(0, 10);

  const classColors = {
    default: 'bg-indigo-500/10 border-indigo-500/20 text-indigo-300',
    1:       'bg-cyan-500/10   border-cyan-500/20   text-cyan-300',
  };

  const typeColors = {
    exam:       'text-red-400',
    quiz:       'text-yellow-400',
    discussion: 'text-blue-400',
    assignment: 'text-emerald-400',
    other:      'text-dim',
  };

  // map classID → color index
  let classIdx = {};
  $: if (weekData) {
    const seen = {};
    let i = 0;
    Object.values(weekData.byDay).flat().forEach(t => {
      if (!(t.classID in seen)) seen[t.classID] = i++;
    });
    classIdx = seen;
  }

  const palette = [
    'border-l-indigo-500 bg-indigo-500/5',
    'border-l-cyan-500   bg-cyan-500/5',
    'border-l-amber-500  bg-amber-500/5',
    'border-l-rose-500   bg-rose-500/5',
  ];

  async function toggle(task) {
    const next = !task.completed;
    task.completed = next;
    weekData = { ...weekData };
    try {
      await api.toggleTask(task.id, next);
      weekData = await api.getWeek(anchorDate);
    } catch (e) {
      task.completed = !next;
      weekData = { ...weekData };
      alert('Error: ' + e.message);
    }
  }
</script>

<div class="p-8 max-w-7xl mx-auto">
  <div class="flex items-center gap-4 mb-8">
    <h1 class="text-2xl font-semibold">Weekly View</h1>
    {#if weekData}
      <div class="flex items-center gap-2 ml-auto">
        <button class="btn-ghost px-3 py-1.5 text-sm" on:click={prevWeek}>← Prev</button>
        <span class="text-sm text-dim px-4">
          {fmtDate(weekData.weekStart)} – {fmtDate(weekData.weekEnd)}
        </span>
        <button class="btn-ghost px-3 py-1.5 text-sm" on:click={nextWeek}>Next →</button>
      </div>
    {/if}
  </div>

  {#if loading}
    <div class="text-dim text-sm">Loading...</div>
  {:else if error}
    <div class="text-red-400 text-sm">Error: {error}</div>
  {:else if weekData}

  <div class="grid grid-cols-7 gap-3">
    {#each DAYS as day, i}
      {@const dateKey = addDays(weekData.weekStart, i)}
      {@const tasks = weekData.byDay[dateKey] ?? []}
      {@const isToday = dateKey === today}

      <div class="min-h-48 rounded-xl border {isToday ? 'border-indigo-500/50 bg-indigo-500/5' : 'border-border bg-surface'} p-3">
        <!-- Day header -->
        <div class="mb-3">
          <div class="text-xs font-medium {isToday ? 'text-indigo-400' : 'text-dim'} uppercase tracking-wide">{day}</div>
          <div class="text-lg font-semibold {isToday ? 'text-indigo-300' : 'text-text'}">
            {new Date(dateKey + 'T12:00:00').getDate()}
          </div>
        </div>

        <!-- Tasks -->
        <div class="space-y-1.5">
          {#each tasks as task}
            {@const ci = classIdx[task.classID] ?? 0}
            <div class="border-l-2 pl-2 py-0.5 rounded-r {palette[ci % palette.length]} {task.completed ? 'opacity-40' : ''}">
              <button
                class="flex items-start gap-1.5 w-full text-left group"
                on:click={() => toggle(task)}
              >
                <span class="mt-0.5 w-3.5 h-3.5 shrink-0 rounded-sm border
                             {task.completed ? 'bg-indigo-600 border-indigo-600' : 'border-border group-hover:border-indigo-400'}
                             flex items-center justify-center transition-colors">
                  {#if task.completed}<span class="text-white text-[8px] leading-none">✓</span>{/if}
                </span>
                <span class="text-xs leading-snug {task.completed ? 'line-through text-dim' : 'text-text'}">
                  {task.text}
                </span>
              </button>
              <div class="flex items-center gap-1 mt-0.5 ml-5">
                <span class="text-[10px] {typeColors[task.type] ?? 'text-dim'}">{task.type}</span>
                <span class="text-[10px] text-muted">·</span>
                <span class="text-[10px] text-dim">{task.className}</span>
              </div>
            </div>
          {/each}

          {#if tasks.length === 0}
            <div class="text-xs text-muted italic py-2">—</div>
          {/if}
        </div>
      </div>
    {/each}
  </div>

  <!-- Legend -->
  <div class="mt-6 flex flex-wrap gap-4 text-xs text-dim">
    {#each Object.entries(classIdx) as [id, i]}
      <span class="flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-sm {palette[i % palette.length].split(' ')[0].replace('border-l-', 'bg-')}"></span>
        {id}
      </span>
    {/each}
  </div>

  {/if}
</div>
