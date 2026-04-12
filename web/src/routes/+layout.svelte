<script>
  import '../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api.js';
  import { summary } from '$lib/stores.js';

  const nav = [
    { href: '/',         label: 'Dashboard', icon: '▣' },
    { href: '/week',     label: 'This Week', icon: '◫', badge: 'week' },
    { href: '/burndown', label: 'Burndown',  icon: '↘' },
    { href: '/heatmap',  label: 'Heatmap',   icon: '⊞' },
    { href: '/import',   label: 'Import',    icon: '⊕' },
  ];

  async function refreshSummary() {
    try {
      summary.set(await api.getSummary());
    } catch (_) {}
  }

  onMount(refreshSummary);

  // Refresh badge counts on every navigation
  $: $page && refreshSummary();
</script>

<div class="flex h-screen overflow-hidden">
  <!-- Sidebar -->
  <nav class="w-52 shrink-0 bg-surface border-r border-border flex flex-col py-6 px-3 gap-1">
    <div class="px-3 mb-6">
      <span class="text-lg font-semibold text-text tracking-tight">classtrack</span>
      <div class="text-xs text-dim mt-0.5">NVCC</div>
    </div>

    {#each nav as item}
      {@const active = $page.url.pathname === item.href}
      <a
        href={item.href}
        class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors duration-100
               {active
                 ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30'
                 : 'text-dim hover:text-text hover:bg-border/50'}"
      >
        <span class="text-base leading-none">{item.icon}</span>
        <span class="flex-1">{item.label}</span>

        {#if item.badge === 'week' && $summary.dueThisWeek > 0}
          <span class="text-[10px] font-medium px-1.5 py-0.5 rounded-full
                       {$summary.overdue > 0 ? 'bg-red-900/60 text-red-300' : 'bg-indigo-900/60 text-indigo-300'}">
            {$summary.dueThisWeek}
          </span>
        {/if}
      </a>
    {/each}

    <!-- Overdue warning -->
    {#if $summary.overdue > 0}
      <div class="mt-auto mx-1 px-3 py-2 rounded-lg bg-red-900/20 border border-red-800/40">
        <div class="text-xs font-medium text-red-400">
          {$summary.overdue} overdue
        </div>
        <div class="text-[10px] text-red-500/70 mt-0.5">needs attention</div>
      </div>
    {/if}
  </nav>

  <!-- Main content -->
  <main class="flex-1 overflow-y-auto">
    <slot />
  </main>
</div>
