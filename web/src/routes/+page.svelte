<script>
  import { onMount } from 'svelte';
  import { api, classColor, fmtDate } from '$lib/api.js';
  import { summary } from '$lib/stores.js';

  let classes = [];
  let loading = true;
  let error = '';

  const today = new Date().toISOString().slice(0, 10);

  onMount(async () => {
    try {
      classes = await api.getClasses();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

  async function toggle(task) {
    const next = !task.completed;
    task.completed = next;
    task.completedAt = next ? today : null;
    classes = [...classes];
    try {
      await api.toggleTask(task.id, next);
      classes = await api.getClasses();
    } catch (e) {
      task.completed = !next;
      classes = [...classes];
      alert('Error: ' + e.message);
    }
  }

  $: globalTotal     = classes.reduce((s, c) => s + c.stats.total, 0);
  $: globalCompleted = classes.reduce((s, c) => s + c.stats.completed, 0);
  $: globalPct       = globalTotal ? Math.round(globalCompleted / globalTotal * 100) : 0;
  $: globalPoints    = classes.reduce((s, c) => s + c.stats.earnedPoints, 0);
  $: globalMaxPts    = classes.reduce((s, c) => s + c.stats.totalPoints, 0);
  $: globalStreak    = classes.reduce((s, c) => Math.max(s, c.stats.streak), 0);
  $: avgPace         = classes.length ? classes.reduce((s, c) => s + c.stats.onPaceScore, 0) / classes.length : 0;
  $: globalPaceLabel = paceLabel(avgPace);

  function paceLabel(score) {
    if (score >=  5) return { text: `+${score.toFixed(0)}% ahead`, cls: 'text-emerald-400' };
    if (score <= -5) return { text: `${score.toFixed(0)}% behind`, cls: 'text-red-400' };
    return { text: 'On track', cls: 'text-yellow-400' };
  }

  function overdueTasks(cls) {
    return (cls.tasks ?? [])
      .filter(t => !t.completed && t.dueDate < today)
      .sort((a, b) => a.dueDate.localeCompare(b.dueDate));
  }

  function upcomingTasks(cls) {
    return (cls.tasks ?? [])
      .filter(t => !t.completed && t.dueDate >= today)
      .sort((a, b) => a.dueDate.localeCompare(b.dueDate))
      .slice(0, 4);
  }
</script>

<div class="p-8 max-w-6xl mx-auto">
  <h1 class="text-2xl font-semibold mb-1">Dashboard</h1>
  <p class="text-dim text-sm mb-8">
    {new Date().toLocaleDateString('en-US', { weekday:'long', month:'long', day:'numeric', year:'numeric' })}
  </p>

  {#if loading}
    <div class="text-dim text-sm">Loading...</div>
  {:else if error}
    <div class="text-red-400 text-sm">Error: {error}</div>
  {:else}

  <!-- Overdue banner -->
  {#if $summary.overdue > 0}
    <div class="mb-6 px-5 py-3 rounded-xl bg-red-900/20 border border-red-800/40 flex items-center gap-3">
      <span class="text-red-400 font-medium">{$summary.overdue} overdue task{$summary.overdue !== 1 ? 's' : ''}</span>
      <span class="text-red-500/60 text-sm">— scroll down to see which classes</span>
    </div>
  {/if}

  <!-- Global stats bar -->
  <div class="grid grid-cols-4 gap-4 mb-8">
    <div class="card text-center">
      <div class="text-3xl font-bold text-text">{globalCompleted}<span class="text-dim text-lg">/{globalTotal}</span></div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Tasks Done</div>
      <div class="mt-3 h-1.5 bg-border rounded-full overflow-hidden">
        <div class="h-full bg-indigo-500 rounded-full transition-all" style="width: {globalPct}%"></div>
      </div>
      <div class="text-xs text-dim mt-1">{globalPct}%</div>
    </div>

    <div class="card text-center">
      <div class="text-3xl font-bold text-text">{globalPoints.toLocaleString()}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Points Earned</div>
      <div class="mt-3 h-1.5 bg-border rounded-full overflow-hidden">
        <div class="h-full bg-amber-500 rounded-full transition-all"
             style="width: {globalMaxPts ? Math.round(globalPoints/globalMaxPts*100) : 0}%"></div>
      </div>
      <div class="text-xs text-dim mt-1">of {globalMaxPts.toLocaleString()} total</div>
    </div>

    <div class="card text-center">
      <div class="text-3xl font-bold text-text">{globalStreak}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Day Streak</div>
      <div class="mt-4 text-xl">{globalStreak >= 7 ? '🔥' : globalStreak >= 3 ? '⚡' : '—'}</div>
    </div>

    <div class="card text-center">
      <div class="text-3xl font-bold {globalPaceLabel.cls}">{avgPace >= 0 ? '+' : ''}{avgPace.toFixed(0)}%</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Overall Pace</div>
      <div class="mt-3 text-sm {globalPaceLabel.cls}">{globalPaceLabel.text}</div>
    </div>
  </div>

  <!-- Class cards -->
  <div class="grid grid-cols-2 gap-6">
    {#each classes as cls, i}
      {@const color = classColor(i)}
      {@const pl = paceLabel(cls.stats.onPaceScore)}
      {@const overdue = overdueTasks(cls)}
      {@const upcoming = upcomingTasks(cls)}

      <div class="card {color.border} border flex flex-col gap-0">
        <!-- Header — click to class detail -->
        <a href="/class/{cls.id}" class="flex items-start justify-between mb-4 group">
          <div>
            <span class="text-xs {color.text} font-mono uppercase tracking-wider">{cls.id}</span>
            <h2 class="text-lg font-semibold text-text mt-0.5 group-hover:text-indigo-300 transition-colors">
              {cls.name}
              <span class="text-dim text-sm font-normal ml-1">↗</span>
            </h2>
          </div>
          <div class="text-right">
            <div class="text-sm font-medium {pl.cls}">{pl.text}</div>
            <div class="text-xs text-dim">{cls.stats.earnedPoints}/{cls.stats.totalPoints} pts</div>
          </div>
        </a>

        <!-- Progress bar -->
        <div class="flex items-center gap-3 mb-1">
          <div class="flex-1 h-2 bg-border rounded-full overflow-hidden">
            <div class="h-full rounded-full transition-all duration-500"
                 style="width: {cls.stats.percentage}%; background-color: {color.dot}"></div>
          </div>
          <span class="text-sm font-mono text-dim w-12 text-right">
            {Math.round(cls.stats.percentage)}%
          </span>
        </div>
        <div class="text-xs text-dim mb-4">
          {cls.stats.completed}/{cls.stats.total} tasks
          {#if cls.stats.streak > 0}· {cls.stats.streak}d streak{/if}
        </div>

        <!-- Overdue tasks -->
        {#if overdue.length > 0}
          <div class="mb-4 rounded-lg bg-red-900/15 border border-red-800/30 p-3">
            <div class="text-xs font-medium text-red-400 uppercase tracking-wide mb-2">
              Overdue ({overdue.length})
            </div>
            <ul class="space-y-1.5">
              {#each overdue as task}
                <li class="flex items-start gap-2">
                  <button
                    class="mt-0.5 w-4 h-4 shrink-0 rounded border border-red-700
                           hover:border-red-400 hover:bg-red-900/30 transition-colors"
                    on:click={() => toggle(task)}
                    aria-label="Complete task"
                  ></button>
                  <span class="text-sm text-red-200/80 flex-1 leading-snug line-clamp-2">{task.text}</span>
                  <span class="text-xs text-red-400 shrink-0 font-medium">{fmtDate(task.dueDate)}</span>
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Upcoming tasks -->
        <div class="text-xs text-dim uppercase tracking-wide mb-2">Upcoming</div>
        {#if upcoming.length === 0 && overdue.length === 0}
          <div class="text-sm text-dim italic">All caught up</div>
        {:else if upcoming.length === 0}
          <div class="text-sm text-dim italic">No upcoming tasks</div>
        {:else}
          <ul class="space-y-1.5">
            {#each upcoming as task}
              <li class="flex items-start gap-2">
                <button
                  class="mt-0.5 w-4 h-4 shrink-0 rounded border border-border
                         hover:border-indigo-400 transition-colors"
                  on:click={() => toggle(task)}
                  aria-label="Complete task"
                ></button>
                <span class="text-sm text-text flex-1 leading-snug">{task.text}</span>
                <span class="text-xs shrink-0
                             {task.dueDate === today ? 'text-red-400 font-medium' : 'text-dim'}">
                  {fmtDate(task.dueDate)}
                </span>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    {/each}
  </div>

  {/if}
</div>
