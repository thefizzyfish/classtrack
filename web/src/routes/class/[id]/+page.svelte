<script>
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api, classColor, fmtDate } from '$lib/api.js';

  let cls = null;
  let loading = true;
  let error = '';

  // Class settings
  let settingsOpen = false;
  let settingSemester = '';
  let settingArchived = false;
  let settingSaving = false;
  let settingsError = '';

  // Deletion (only available when archived)
  let deleteConfirmText = '';
  let deleting = false;

  const today = new Date().toISOString().slice(0, 10);

  $: id = $page.params.id;
  $: colorIdx = 0; // will be set when classes load

  onMount(async () => {
    try {
      const all = await api.getClasses();
      colorIdx = all.findIndex(c => c.id === id);
      cls = all.find(c => c.id === id) ?? await api.getClass(id);
      settingSemester = cls?.semester ?? '';
      settingArchived = cls?.archived ?? false;
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

  async function saveSettings() {
    settingSaving = true;
    settingsError = '';
    try {
      await api.updateClass(id, { semester: settingSemester, archived: settingArchived });
      cls = await api.getClass(id);
      settingsOpen = false;
    } catch (e) {
      settingsError = e.message;
    } finally {
      settingSaving = false;
    }
  }

  async function deleteClass() {
    if (deleteConfirmText !== cls.name || deleting) return;
    deleting = true;
    settingsError = '';
    try {
      await api.deleteClass(id);
      window.location.href = '/';
    } catch (e) {
      settingsError = e.message;
      deleting = false;
    }
  }

  async function toggle(task) {
    const next = !task.completed;
    task.completed = next;
    task.completedAt = next ? today : null;
    cls = { ...cls };
    try {
      await api.toggleTask(task.id, next);
      cls = await api.getClass(id);
    } catch (e) {
      task.completed = !next;
      cls = { ...cls };
      alert('Error: ' + e.message);
    }
  }

  // Group tasks by dueDate, overdue first then future
  $: { overdueTasks; dueSections; } // reactive block trigger
  $: overdueTasks = (cls?.tasks ?? []).filter(t => !t.completed && t.dueDate < today);
  $: dueSections  = buildSections(cls?.tasks ?? []);

  function buildSections(tasks) {
    const map = new Map();
    for (const t of tasks) {
      if (!t.completed && t.dueDate < today) continue; // shown in overdue
      const key = t.dueDate;
      if (!map.has(key)) map.set(key, []);
      map.get(key).push(t);
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
  }

  function paceLabel(score) {
    if (score >=  5) return { text: `+${score.toFixed(0)}% ahead`, cls: 'text-emerald-400' };
    if (score <= -5) return { text: `${score.toFixed(0)}% behind`, cls: 'text-red-400' };
    return { text: 'On track', cls: 'text-yellow-400' };
  }

  const typeColors = {
    exam:       'text-red-400',
    quiz:       'text-yellow-400',
    discussion: 'text-blue-400',
    assignment: 'text-emerald-400',
    other:      'text-dim',
  };
</script>

<div class="p-8 max-w-3xl mx-auto">
  <!-- Back -->
  <a href="/" class="inline-flex items-center gap-1.5 text-sm text-dim hover:text-text mb-6 transition-colors">
    ← Dashboard
  </a>

  {#if loading}
    <div class="text-dim text-sm">Loading...</div>
  {:else if error}
    <div class="text-red-400 text-sm">Error: {error}</div>
  {:else if cls}

  {@const color = classColor(colorIdx)}
  {@const pl = paceLabel(cls.stats.onPaceScore)}

  <!-- Header -->
  <div class="flex items-start justify-between mb-6">
    <div>
      <span class="text-xs {color.text} font-mono uppercase tracking-wider">{cls.id}</span>
      <h1 class="text-3xl font-semibold text-text mt-1">{cls.name}</h1>
      {#if cls.semester}
        <span class="text-sm text-dim mt-0.5 block">{cls.semester}</span>
      {/if}
    </div>
    <button
      class="mt-1 p-1.5 rounded-lg text-dim hover:text-text hover:bg-surface transition-colors"
      on:click={() => { settingsOpen = !settingsOpen; settingsError = ''; deleteConfirmText = ''; }}
      title="Class settings"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none"
           stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06
                 a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09
                 A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83
                 l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09
                 A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83
                 l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09
                 a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83
                 l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09
                 a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
    </button>
  </div>

  <!-- Settings panel -->
  {#if settingsOpen}
    <div class="mb-6 rounded-xl border border-border bg-surface p-4 space-y-4">
      <div class="text-xs font-medium text-dim uppercase tracking-wide">Class Settings</div>

      <div class="flex gap-3 items-end">
        <div class="flex-1">
          <label class="block text-xs text-dim mb-1.5">Semester</label>
          <input
            bind:value={settingSemester}
            placeholder="e.g. Spring 2026"
            class="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text
                   focus:outline-none focus:border-indigo-500"
            on:keydown={e => e.key === 'Enter' && saveSettings()}
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-xs text-dim">Archived</label>
          <button
            class="flex items-center gap-2 px-3 py-2 rounded-lg border text-sm transition-colors
                   {settingArchived
                     ? 'border-amber-600/50 bg-amber-900/20 text-amber-400'
                     : 'border-border bg-bg text-dim hover:border-indigo-500'}"
            on:click={() => settingArchived = !settingArchived}
          >
            <span class="w-3 h-3 rounded-sm border-2 flex items-center justify-center shrink-0
                         {settingArchived ? 'border-amber-400 bg-amber-400' : 'border-current'}">
              {#if settingArchived}<span class="text-bg text-[8px] leading-none font-bold">✓</span>{/if}
            </span>
            {settingArchived ? 'Archived' : 'Active'}
          </button>
        </div>
      </div>

      {#if settingArchived}
        <p class="text-xs text-amber-400/80">
          Archived classes are hidden from the dashboard and weekly view but included in heatmap history.
        </p>
      {/if}

      {#if settingsError}
        <p class="text-xs text-red-400">{settingsError}</p>
      {/if}

      <div class="flex gap-2">
        <button
          class="btn-primary text-sm"
          on:click={saveSettings}
          disabled={settingSaving}
        >{settingSaving ? 'Saving…' : 'Save'}</button>
        <button
          class="btn-ghost text-sm"
          on:click={() => { settingsOpen = false; settingSemester = cls.semester ?? ''; settingArchived = cls.archived ?? false; deleteConfirmText = ''; }}
        >Cancel</button>
      </div>

      {#if cls.archived}
        <div class="border-t border-red-900/40 pt-4 space-y-3">
          <div class="text-xs font-medium text-red-400 uppercase tracking-wide">Danger Zone</div>
          <p class="text-xs text-dim">
            Permanently deletes this class and all its tasks and completion history. Cannot be undone.
          </p>
          <div>
            <label class="block text-xs text-dim mb-1.5">
              Type <span class="font-mono text-text">{cls.name}</span> to confirm
            </label>
            <input
              bind:value={deleteConfirmText}
              placeholder={cls.name}
              class="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text
                     focus:outline-none focus:border-red-500"
            />
          </div>
          <button
            class="px-3 py-2 rounded-lg text-sm font-medium transition-colors
                   {deleteConfirmText === cls.name && !deleting
                     ? 'bg-red-700 hover:bg-red-600 text-white cursor-pointer'
                     : 'bg-bg border border-border text-dim cursor-not-allowed opacity-50'}"
            disabled={deleteConfirmText !== cls.name || deleting}
            on:click={deleteClass}
          >{deleting ? 'Deleting…' : 'Delete class permanently'}</button>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Stats strip -->
  <div class="grid grid-cols-4 gap-3 mb-6">
    <div class="card text-center py-3">
      <div class="text-xl font-bold text-text">{cls.stats.completed}/{cls.stats.total}</div>
      <div class="text-[10px] text-dim uppercase tracking-wide mt-0.5">Tasks</div>
    </div>
    <div class="card text-center py-3">
      <div class="text-xl font-bold text-amber-400">{cls.stats.earnedPoints}</div>
      <div class="text-[10px] text-dim uppercase tracking-wide mt-0.5">Points</div>
    </div>
    <div class="card text-center py-3">
      <div class="text-xl font-bold {pl.cls}">{cls.stats.onPaceScore >= 0 ? '+' : ''}{cls.stats.onPaceScore.toFixed(0)}%</div>
      <div class="text-[10px] text-dim uppercase tracking-wide mt-0.5">Pace</div>
    </div>
    <div class="card text-center py-3">
      <div class="text-xl font-bold text-text">{cls.stats.streak}</div>
      <div class="text-[10px] text-dim uppercase tracking-wide mt-0.5">Streak</div>
    </div>
  </div>

  <!-- Progress bar -->
  <div class="flex items-center gap-3 mb-8">
    <div class="flex-1 h-2 bg-border rounded-full overflow-hidden">
      <div class="h-full rounded-full transition-all duration-500"
           style="width: {cls.stats.percentage}%; background-color: {color.dot}"></div>
    </div>
    <span class="text-sm font-mono text-dim">{Math.round(cls.stats.percentage)}%</span>
  </div>

  <!-- Overdue section -->
  {#if overdueTasks.length > 0}
    <div class="mb-6 rounded-xl bg-red-900/15 border border-red-800/30 p-4">
      <div class="text-xs font-medium text-red-400 uppercase tracking-wide mb-3">
        Overdue — {overdueTasks.length} task{overdueTasks.length !== 1 ? 's' : ''}
      </div>
      <ul class="space-y-2">
        {#each overdueTasks as task}
          <li class="flex items-start gap-3">
            <button
              class="mt-0.5 w-4 h-4 shrink-0 rounded border border-red-700
                     hover:border-red-400 hover:bg-red-900/30 transition-colors"
              on:click={() => toggle(task)}
            ></button>
            <div class="flex-1 min-w-0">
              <div class="text-sm text-red-200/80">{task.text}</div>
              {#if task.subtasks?.length}
                <div class="mt-1 space-y-1 ml-3">
                  {#each task.subtasks as sub}
                    <div class="flex items-center gap-2">
                      <button
                        class="w-3 h-3 shrink-0 rounded-sm border
                               {sub.completed ? 'bg-indigo-600 border-indigo-600' : 'border-red-700 hover:border-red-400'}
                               transition-colors"
                        on:click={() => toggle(sub)}
                      ></button>
                      <span class="text-xs {sub.completed ? 'line-through text-dim' : 'text-red-200/60'}">{sub.text}</span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
            <span class="text-xs text-red-400 font-medium shrink-0">{fmtDate(task.dueDate)}</span>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  <!-- Tasks by due date -->
  <div class="space-y-6">
    {#each dueSections as [dueDate, tasks]}
      {@const isPast = dueDate < today}
      {@const isToday = dueDate === today}
      {@const allDone = tasks.every(t => t.completed)}

      <div class="{allDone ? 'opacity-50' : ''}">
        <!-- Date header -->
        <div class="flex items-center gap-3 mb-3">
          <div class="text-sm font-medium {isToday ? 'text-indigo-300' : isPast ? 'text-dim' : 'text-text'}">
            {fmtDate(dueDate)}
            {#if isToday}<span class="text-xs text-indigo-400 ml-1">today</span>{/if}
          </div>
          <div class="flex-1 h-px bg-border"></div>
          <div class="text-xs text-dim">
            {tasks.filter(t => t.completed).length}/{tasks.length}
          </div>
        </div>

        <!-- Tasks -->
        <ul class="space-y-2">
          {#each tasks as task}
            <li class="flex items-start gap-3 group">
              <button
                class="mt-0.5 w-4 h-4 shrink-0 rounded border
                       {task.completed
                         ? 'bg-indigo-600 border-indigo-600'
                         : 'border-border group-hover:border-indigo-400'}
                       flex items-center justify-center transition-colors"
                on:click={() => toggle(task)}
              >
                {#if task.completed}
                  <span class="text-white text-[9px] leading-none">✓</span>
                {/if}
              </button>

              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-sm {task.completed ? 'line-through text-dim' : 'text-text'} leading-snug">
                    {task.text}
                  </span>
                  <span class="{typeColors[task.type] ?? 'text-dim'} text-[10px] shrink-0">{task.type}</span>
                  <span class="text-[10px] text-muted shrink-0">{task.points}pt</span>
                </div>

                <!-- Subtasks -->
                {#if task.subtasks?.length}
                  <div class="mt-1.5 space-y-1 ml-1">
                    {#each task.subtasks as sub}
                      <div class="flex items-center gap-2">
                        <button
                          class="w-3 h-3 shrink-0 rounded-sm border
                                 {sub.completed ? 'bg-indigo-600 border-indigo-600' : 'border-border hover:border-indigo-400'}
                                 flex items-center justify-center transition-colors"
                          on:click={() => toggle(sub)}
                        >
                          {#if sub.completed}<span class="text-white text-[7px]">✓</span>{/if}
                        </button>
                        <span class="text-xs {sub.completed ? 'line-through text-dim' : 'text-dim'}">{sub.text}</span>
                        {#if sub.completedAt}
                          <span class="text-[10px] text-muted">{fmtDate(sub.completedAt)}</span>
                        {/if}
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>

              {#if task.completedAt}
                <span class="text-[10px] text-dim shrink-0">{fmtDate(task.completedAt)}</span>
              {/if}
            </li>
          {/each}
        </ul>
      </div>
    {/each}
  </div>

  {/if}
</div>
