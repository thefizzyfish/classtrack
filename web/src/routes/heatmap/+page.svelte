<script>
  import { onMount } from 'svelte';
  import { api } from '$lib/api.js';

  let data = [];
  let loading = true;
  let error = '';
  let tooltip = null; // { x, y, date, count, points }

  onMount(async () => {
    try {
      data = await api.getHeatmap();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

  // Build a 52-week grid ending on the next Saturday from today
  $: grid = buildGrid(data);

  function buildGrid(points) {
    const map = {};
    let maxCount = 0;
    for (const p of points) {
      map[p.date] = p;
      if (p.count > maxCount) maxCount = p.count;
    }

    const today = new Date();
    // Find end: next Saturday on or after today
    const endDay = new Date(today);
    endDay.setDate(endDay.getDate() + (6 - endDay.getDay()));
    // Start: 51 weeks before that Sunday
    const startDay = new Date(endDay);
    startDay.setDate(startDay.getDate() - 7 * 51 - 6); // 52 weeks of Sun–Sat
    // Snap start to nearest Sunday
    startDay.setDate(startDay.getDate() - startDay.getDay());

    const weeks = [];
    let cur = new Date(startDay);

    while (cur <= endDay) {
      const week = [];
      for (let d = 0; d < 7; d++) {
        const iso = cur.toISOString().slice(0, 10);
        const p = map[iso];
        week.push({
          date: iso,
          count: p?.count ?? 0,
          points: p?.points ?? 0,
          future: cur > today,
          maxCount,
        });
        cur.setDate(cur.getDate() + 1);
      }
      weeks.push(week);
    }
    return weeks;
  }

  function cellColor(count, maxCount, future) {
    if (future) return '#1a1d27';
    if (count === 0) return '#2a2d3e';
    const intensity = Math.min(count / Math.max(maxCount, 1), 1);
    // Purple gradient
    const colors = ['#3730a3', '#4338ca', '#4f46e5', '#6366f1', '#818cf8'];
    const idx = Math.floor(intensity * (colors.length - 1));
    return colors[idx];
  }

  function fmtDate(iso) {
    const [y, m, d] = iso.split('-').map(Number);
    return new Date(y, m - 1, d).toLocaleDateString('en-US', {
      weekday: 'short', month: 'short', day: 'numeric'
    });
  }

  // month labels: find weeks where month changes
  $: monthLabels = buildMonthLabels(grid);

  function buildMonthLabels(weeks) {
    const labels = [];
    let lastMonth = -1;
    weeks.forEach((week, i) => {
      const m = new Date(week[0].date + 'T12:00:00').getMonth();
      if (m !== lastMonth) {
        labels.push({ weekIdx: i, label: new Date(week[0].date + 'T12:00:00').toLocaleDateString('en-US', { month: 'short' }) });
        lastMonth = m;
      }
    });
    return labels;
  }

  // stats
  $: totalCompletions = data.reduce((s, d) => s + d.count, 0);
  $: totalPoints      = data.reduce((s, d) => s + d.points, 0);
  $: activeDays       = data.filter(d => d.count > 0).length;
  $: bestDay          = data.reduce((best, d) => d.count > (best?.count ?? 0) ? d : best, null);
</script>

<div class="p-8 max-w-5xl mx-auto">
  <h1 class="text-2xl font-semibold mb-1">Activity Heatmap</h1>
  <p class="text-dim text-sm mb-8">Task completions per day</p>

  {#if loading}
    <div class="text-dim text-sm">Loading...</div>
  {:else if error}
    <div class="text-red-400 text-sm">Error: {error}</div>
  {:else}

  <!-- Stats row -->
  <div class="grid grid-cols-4 gap-4 mb-8">
    <div class="card text-center">
      <div class="text-2xl font-bold text-text">{totalCompletions}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Total Completions</div>
    </div>
    <div class="card text-center">
      <div class="text-2xl font-bold text-amber-400">{totalPoints.toLocaleString()}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Points Earned</div>
    </div>
    <div class="card text-center">
      <div class="text-2xl font-bold text-text">{activeDays}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Active Days</div>
    </div>
    <div class="card text-center">
      <div class="text-xl font-bold text-indigo-400">{bestDay ? bestDay.count : 0}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Best Day</div>
      {#if bestDay}<div class="text-xs text-dim mt-0.5">{fmtDate(bestDay.date)}</div>{/if}
    </div>
  </div>

  <!-- Heatmap grid -->
  <div class="card overflow-x-auto relative">
    <!-- Month labels -->
    <div class="flex mb-2" style="padding-left: 2rem;">
      {#each monthLabels as { weekIdx, label }}
        <div class="text-xs text-dim absolute" style="left: calc(2rem + {weekIdx * 14}px)">{label}</div>
      {/each}
    </div>
    <div style="height: 1.2rem;"></div>

    <div class="flex gap-0.5" style="padding-left: 2rem;">
      {#each grid as week}
        <div class="flex flex-col gap-0.5">
          {#each week as cell}
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <div
              class="w-3 h-3 rounded-sm cursor-default transition-transform hover:scale-125"
              style="background-color: {cellColor(cell.count, cell.maxCount, cell.future)}"
              on:mouseenter={(e) => {
                if (!cell.future) tooltip = { x: e.clientX, y: e.clientY, ...cell };
              }}
              on:mouseleave={() => tooltip = null}
            ></div>
          {/each}
        </div>
      {/each}
    </div>

    <!-- Day labels -->
    <div class="absolute left-0 top-8 flex flex-col gap-0.5">
      {#each ['', 'M', '', 'W', '', 'F', ''] as label}
        <div class="w-6 h-3 text-[9px] text-dim flex items-center justify-end pr-1">{label}</div>
      {/each}
    </div>

    <!-- Color legend -->
    <div class="flex items-center gap-2 mt-4 text-xs text-dim">
      <span>Less</span>
      {#each ['#2a2d3e', '#3730a3', '#4f46e5', '#6366f1', '#818cf8'] as c}
        <div class="w-3 h-3 rounded-sm" style="background-color: {c}"></div>
      {/each}
      <span>More</span>
    </div>
  </div>

  {/if}
</div>

<!-- Tooltip -->
{#if tooltip}
  <div
    class="fixed z-50 pointer-events-none bg-surface border border-border rounded-lg px-3 py-2 text-xs shadow-xl"
    style="left: {tooltip.x + 12}px; top: {tooltip.y - 10}px"
  >
    <div class="font-medium text-text">{fmtDate(tooltip.date)}</div>
    <div class="text-dim mt-0.5">{tooltip.count} completion{tooltip.count !== 1 ? 's' : ''}</div>
    {#if tooltip.points > 0}
      <div class="text-amber-400">+{tooltip.points} pts</div>
    {/if}
  </div>
{/if}
