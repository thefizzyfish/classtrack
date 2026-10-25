<script>
  import { onMount } from 'svelte';
  import { api, fmtDate } from '$lib/api.js';

  let classes = [];
  let selectedClass = '';
  let pasteText = '';
  let preview = null;
  let loading = false;
  let importing = false;
  let error = '';
  let success = '';

  // 'add' inserts new tasks; 'reschedule' moves due dates of existing tasks by name
  let mode = 'add';
  let reschedule = null;

  // new class form
  let showNewClass = false;
  let newClassName = '';
  let newClassSemester = '';
  let creatingClass = false;

  onMount(async () => {
    try {
      classes = await api.getClasses();
      if (classes.length) selectedClass = classes[0].id;
    } catch (e) {
      error = e.message;
    }
  });

  async function createClass() {
    if (!newClassName.trim()) return;
    creatingClass = true;
    error = '';
    try {
      const res = await fetch('/api/classes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newClassName.trim(), semester: newClassSemester.trim() })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error);
      // reload class list and select the new one
      classes = await api.getClasses();
      selectedClass = data.id;
      showNewClass = false;
      newClassName = '';
      newClassSemester = '';
    } catch (e) {
      error = e.message;
    } finally {
      creatingClass = false;
    }
  }

  function setMode(m) {
    mode = m;
    preview = null;
    reschedule = null;
    success = '';
  }

  async function parsePreview() {
    if (!pasteText.trim()) return;
    loading = true;
    error = '';
    success = '';
    try {
      if (mode === 'reschedule') {
        reschedule = await api.reschedule(selectedClass, pasteText, false);
        return;
      }
      const cls = classes.find(c => c.id === selectedClass);
      const year = cls?.created ? parseInt(cls.created.slice(0, 4)) : new Date().getFullYear();
      preview = await api.importPreview(pasteText, year);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function doImport() {
    if (!preview?.length || !selectedClass) return;
    importing = true;
    error = '';
    success = '';
    try {
      const res = await api.importConfirm(selectedClass, pasteText);
      success = `Imported ${res.imported} task${res.imported !== 1 ? 's' : ''} into ${selectedClass}.`;
      pasteText = '';
      preview = null;
    } catch (e) {
      error = e.message;
    } finally {
      importing = false;
    }
  }

  async function doReschedule() {
    if (!reschedule?.moved.length || !selectedClass) return;
    importing = true;
    error = '';
    try {
      const res = await api.reschedule(selectedClass, pasteText, true);
      success = `Moved ${res.moved.length} task${res.moved.length !== 1 ? 's' : ''} in ${selectedClass}. Completion progress kept.`;
      pasteText = '';
      reschedule = null;
    } catch (e) {
      error = e.message;
    } finally {
      importing = false;
    }
  }

  $: grouped = groupByDate(preview ?? []);

  function groupByDate(items) {
    const map = new Map();
    for (const item of items) {
      if (!map.has(item.date)) map.set(item.date, []);
      map.get(item.date).push(item);
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
  }
</script>

<div class="p-8 max-w-3xl mx-auto">
  <h1 class="text-2xl font-semibold mb-1">Import from Canvas</h1>
  <p class="text-dim text-sm mb-8">
    Paste the Canvas calendar export. Dates carry forward, type prefixes are stripped,
    and administrative rows are skipped.
  </p>

  {#if error}
    <div class="mb-4 px-4 py-3 rounded-lg bg-red-900/20 border border-red-800/40 text-red-300 text-sm">{error}</div>
  {/if}
  {#if success}
    <div class="mb-4 px-4 py-3 rounded-lg bg-emerald-900/20 border border-emerald-800/40 text-emerald-300 text-sm">{success}</div>
  {/if}

  <div class="card space-y-5 mb-6">

    <!-- Class selector -->
    <div>
      <label class="block text-xs text-dim uppercase tracking-wide mb-2">Target Class</label>

      {#if !showNewClass}
        <div class="flex gap-2">
          <select
            bind:value={selectedClass}
            class="flex-1 bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text
                   focus:outline-none focus:border-indigo-500"
          >
            {#each classes as cls}
              <option value={cls.id}>{cls.name} ({cls.id})</option>
            {/each}
          </select>
          <button
            class="btn-ghost text-sm whitespace-nowrap"
            on:click={() => showNewClass = true}
          >
            + New class
          </button>
        </div>

      {:else}
        <!-- Inline new class form -->
        <div class="bg-bg border border-indigo-500/40 rounded-lg p-4 space-y-3">
          <div class="text-xs text-indigo-400 font-medium uppercase tracking-wide mb-1">New Class</div>
          <div class="flex gap-2">
            <input
              bind:value={newClassName}
              placeholder="e.g. CS 101"
              class="flex-1 bg-surface border border-border rounded-lg px-3 py-2 text-sm text-text
                     focus:outline-none focus:border-indigo-500"
              on:keydown={e => e.key === 'Enter' && createClass()}
            />
            <input
              bind:value={newClassSemester}
              placeholder="Semester (optional)"
              class="flex-1 bg-surface border border-border rounded-lg px-3 py-2 text-sm text-text
                     focus:outline-none focus:border-indigo-500"
            />
          </div>
          <div class="text-xs text-dim">
            Creates a new class in the database with ID
            <code class="text-indigo-300">{(newClassName || 'class-name').toLowerCase().replace(/\s+/g, '-')}</code>
          </div>
          <div class="flex gap-2">
            <button class="btn-primary text-sm" on:click={createClass} disabled={creatingClass || !newClassName.trim()}>
              {creatingClass ? 'Creating…' : 'Create'}
            </button>
            <button class="btn-ghost text-sm" on:click={() => { showNewClass = false; error = ''; }}>
              Cancel
            </button>
          </div>
        </div>
      {/if}
    </div>

    <!-- Mode -->
    <div>
      <label class="block text-xs text-dim uppercase tracking-wide mb-2">Mode</label>
      <div class="flex gap-2">
        <button class={mode === 'add' ? 'btn-primary text-sm' : 'btn-ghost text-sm'} on:click={() => setMode('add')}>
          Add new tasks
        </button>
        <button class={mode === 'reschedule' ? 'btn-primary text-sm' : 'btn-ghost text-sm'} on:click={() => setMode('reschedule')}>
          Update due dates
        </button>
      </div>
      {#if mode === 'reschedule'}
        <p class="text-xs text-dim mt-2">
          Matches the paste to this class's existing tasks by name and moves their due dates.
          Completion progress is kept. Nothing is added or deleted. Run "Add new tasks" afterwards to pick up anything new.
        </p>
      {/if}
    </div>

    <!-- Paste area -->
    <div>
      <label class="block text-xs text-dim uppercase tracking-wide mb-2">Canvas Calendar Paste</label>
      <textarea
        bind:value={pasteText}
        rows="10"
        placeholder="Paste the Canvas calendar table here..."
        class="w-full bg-bg border border-border rounded-lg px-3 py-2 text-sm text-text font-mono
               focus:outline-none focus:border-indigo-500 resize-y"
      ></textarea>
    </div>

    <div class="flex gap-3">
      <button
        class="btn-primary"
        on:click={parsePreview}
        disabled={loading || !pasteText.trim() || !selectedClass}
      >
        {loading ? 'Parsing…' : 'Preview'}
      </button>
      {#if preview?.length}
        <button
          class="btn-primary bg-emerald-700 hover:bg-emerald-600"
          on:click={doImport}
          disabled={importing}
        >
          {importing ? 'Importing…' : `Import ${preview.length} task${preview.length !== 1 ? 's' : ''}`}
        </button>
      {/if}
      {#if reschedule?.moved.length}
        <button
          class="btn-primary bg-emerald-700 hover:bg-emerald-600"
          on:click={doReschedule}
          disabled={importing}
        >
          {importing ? 'Updating…' : `Move ${reschedule.moved.length} task${reschedule.moved.length !== 1 ? 's' : ''}`}
        </button>
      {/if}
      {#if preview !== null || reschedule !== null}
        <button class="btn-ghost" on:click={() => { preview = null; reschedule = null; }}>Clear</button>
      {/if}
    </div>
  </div>

  <!-- Reschedule preview -->
  {#if reschedule !== null}
    <div class="text-sm text-dim mb-3">
      {reschedule.moved.length} to move · {reschedule.unchanged} already on the right date
    </div>
    {#if reschedule.moved.length}
      <div class="space-y-1 mb-6">
        {#each reschedule.moved as m}
          <div class="flex items-center gap-3 px-3 py-1.5 bg-surface border border-border rounded-lg text-sm">
            <span class="text-dim whitespace-nowrap">{fmtDate(m.from)} → <span class="text-indigo-300">{fmtDate(m.to)}</span></span>
            <span class="text-text flex-1">{m.text}</span>
            {#if m.completed}<span class="text-xs text-emerald-400">done</span>{/if}
          </div>
        {/each}
      </div>
    {:else}
      <div class="text-dim text-sm italic mb-6">No due dates to change.</div>
    {/if}

    {#each [
      ['Ambiguous (skipped)', reschedule.ambiguous.map(a => ({ text: a }))],
      ['In paste but not in class (not added)', reschedule.new],
      ['In class but not in paste (left as-is)', reschedule.missing],
    ] as [title, items]}
      {#if items.length}
        <details class="mb-3">
          <summary class="text-xs text-dim uppercase tracking-wide cursor-pointer">{title}: {items.length}</summary>
          <div class="space-y-1 mt-2">
            {#each items as it}
              <div class="px-3 py-1 text-sm text-dim">
                {#if it.dueDate}<span class="mr-2">{fmtDate(it.dueDate)}</span>{/if}{it.text}
              </div>
            {/each}
          </div>
        </details>
      {/if}
    {/each}
  {/if}

  <!-- Preview -->
  {#if preview !== null}
    {#if preview.length === 0}
      <div class="text-dim text-sm italic">No tasks found — check the paste format.</div>
    {:else}
      <div class="text-sm text-dim mb-3">{preview.length} task{preview.length !== 1 ? 's' : ''} parsed</div>
      <div class="space-y-4">
        {#each grouped as [date, tasks]}
          <div>
            <div class="text-xs text-indigo-400 font-medium uppercase tracking-wide mb-2">
              {fmtDate(date)} — {tasks.length} task{tasks.length !== 1 ? 's' : ''}
            </div>
            <div class="space-y-1">
              {#each tasks as task}
                <div class="flex items-center gap-2 px-3 py-1.5 bg-surface border border-border rounded-lg">
                  <span class="w-1.5 h-1.5 rounded-full bg-indigo-400 shrink-0"></span>
                  <span class="text-sm text-text">{task.text}</span>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>
