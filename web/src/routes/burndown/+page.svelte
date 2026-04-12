<script>
  import { onMount, tick } from 'svelte';
  import { api, fmtDate } from '$lib/api.js';
  import { Chart, LineController, LineElement, PointElement, LinearScale, TimeScale, CategoryScale, Filler, Tooltip, Legend } from 'chart.js';

  Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Filler, Tooltip, Legend);

  let canvas;
  let chart;
  let data = [];
  let loading = true;
  let error = '';

  onMount(async () => {
    try {
      data = await api.getBurndown();
      loading = false;
      await tick(); // wait for canvas to appear in the DOM
      renderChart();
    } catch (e) {
      error = e.message;
      loading = false;
    }
  });

  function renderChart() {
    if (!canvas || !data.length) return;

    const labels    = data.map(d => d.date);
    const remaining = data.map(d => d.remaining);
    const completed = data.map(d => d.completed);

    chart = new Chart(canvas, {
      type: 'line',
      data: {
        labels,
        datasets: [
          {
            label: 'Remaining',
            data: remaining,
            borderColor: '#f43f5e',
            backgroundColor: 'rgba(244,63,94,0.08)',
            fill: true,
            tension: 0.3,
            pointRadius: 2,
            pointHoverRadius: 5,
          },
          {
            label: 'Completed',
            data: completed,
            borderColor: '#10b981',
            backgroundColor: 'rgba(16,185,129,0.08)',
            fill: true,
            tension: 0.3,
            pointRadius: 2,
            pointHoverRadius: 5,
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: 'index', intersect: false },
        plugins: {
          legend: {
            labels: { color: '#8892a4', font: { size: 12 } }
          },
          tooltip: {
            backgroundColor: '#1a1d27',
            borderColor: '#2a2d3e',
            borderWidth: 1,
            titleColor: '#e2e8f0',
            bodyColor: '#8892a4',
            callbacks: {
              title: items => fmtDate(items[0].label)
            }
          }
        },
        scales: {
          x: {
            grid:  { color: '#2a2d3e' },
            ticks: {
              color: '#8892a4',
              maxTicksLimit: 12,
              callback: (_, i) => i % Math.ceil(labels.length / 12) === 0 ? fmtDate(labels[i]) : ''
            }
          },
          y: {
            grid:  { color: '#2a2d3e' },
            ticks: { color: '#8892a4' },
            beginAtZero: true
          }
        }
      }
    });
  }

  // summary stats
  $: lastPoint  = data[data.length - 1];
  $: firstPoint = data[0];
  $: peakRemaining = data.reduce((m, d) => Math.max(m, d.remaining), 0);
</script>

<div class="p-8 max-w-5xl mx-auto">
  <h1 class="text-2xl font-semibold mb-1">Burndown</h1>
  <p class="text-dim text-sm mb-8">Tasks remaining vs completed over time</p>

  {#if loading}
    <div class="text-dim text-sm">Loading...</div>
  {:else if error}
    <div class="text-red-400 text-sm">Error: {error}</div>
  {:else if data.length === 0}
    <div class="text-dim text-sm">No data yet.</div>
  {:else}

  <!-- Summary stats -->
  <div class="grid grid-cols-3 gap-4 mb-8">
    <div class="card text-center">
      <div class="text-3xl font-bold text-rose-400">{lastPoint?.remaining ?? 0}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Still Remaining</div>
    </div>
    <div class="card text-center">
      <div class="text-3xl font-bold text-emerald-400">{lastPoint?.completed ?? 0}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Completed</div>
    </div>
    <div class="card text-center">
      <div class="text-3xl font-bold text-text">{peakRemaining}</div>
      <div class="text-xs text-dim mt-1 uppercase tracking-wide">Peak Backlog</div>
    </div>
  </div>

  <!-- Chart -->
  <div class="card" style="height: 380px; padding: 1.5rem;">
    <canvas bind:this={canvas}></canvas>
  </div>

  {/if}
</div>
