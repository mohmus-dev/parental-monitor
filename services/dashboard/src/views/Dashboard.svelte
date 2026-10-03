<script lang="ts">
  import { onMount } from 'svelte';
  import {
    getChildren,
    getDashboardSummary,
    registerChild,
    createPairingInvite,
    deleteChildren,
    deleteAccount,
  } from '../lib/api';
  import type { Parent, Child, PairingInvite } from '../lib/types';

  let {
    parent,
    onLogout,
  }: {
    parent: Parent;
    onLogout: () => void;
  } = $props();

  let children: Child[] = $state([]);
  let selected: Set<string> = $state(new Set());
  let loading = $state(true);
  let error = $state('');
  let confirmDeleteAccount = $state(false);
  let pairingInvites: Record<string, PairingInvite> = $state({});
  let generatingInviteFor = $state('');
  let acknowledgementChecked = $state(false);

  let newName = $state('');
  let newAge: number | undefined = $state(undefined);
  let adding = $state(false);

  let dashboardStats = $derived(() => {
    const total = children.length;
    const paired = children.filter((child) => child.device_id).length;
    const pending = children.filter((child) => !child.device_id).length;
    return {
      total,
      paired,
      pending,
      alerts: Math.max(1, Math.min(9, paired + total)),
    };
  });

  let alertFeed = $state<any[]>([]);

  let recentSearches = $state<any[]>([]);

  onMount(async () => {
    await loadChildren();
  });

  async function loadChildren() {
    loading = true;
    error = '';
    try {
      const [childrenResponse, dashboardResponse] = await Promise.all([
        getChildren(),
        getDashboardSummary(),
      ]);
      children = childrenResponse;

      alertFeed = (dashboardResponse.alerts ?? []).map((alert) => ({
        id: alert.id,
        childName: children.find((child) => child.id === alert.child_id)?.name ?? 'Child',
        category: alert.category,
        query: alert.query,
        time: new Date(alert.timestamp).toLocaleString(),
        risk: alert.category.toLowerCase().includes('adult') || alert.keyword.toLowerCase().includes('porn') ? 'High' : 'Medium',
      }));

      recentSearches = (dashboardResponse.recent_activity ?? []).map((item) => ({
        id: item.id,
        childName: item.child_name,
        query: item.query,
        time: new Date(item.timestamp).toLocaleString(),
      }));
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load dashboard data';
    } finally {
      loading = false;
    }
  }

  async function handleAddChild(e: SubmitEvent) {
    e.preventDefault();
    adding = true;
    error = '';
    try {
      await registerChild({
        name: newName,
        age: newAge,
        app_name: 'Parental Monitor CLI',
      });
      newName = '';
      newAge = undefined;
      await loadChildren();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to register child';
    } finally {
      adding = false;
    }
  }

  async function handleCreatePairingInvite(childId: string) {
    if (!acknowledgementChecked) {
      error = 'Please acknowledge consent before generating a pairing code.';
      return;
    }

    generatingInviteFor = childId;
    error = '';
    try {
      pairingInvites[childId] = await createPairingInvite(childId);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to create pairing code';
    } finally {
      generatingInviteFor = '';
    }
  }

  function toggle(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selected = next;
  }

  async function handleDeleteSelected() {
    if (selected.size === 0) return;
    error = '';
    try {
      await deleteChildren([...selected]);
      selected = new Set();
      await loadChildren();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Delete failed';
    }
  }

  async function handleDeleteAccount() {
    error = '';
    try {
      await deleteAccount();
      onLogout();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Account deletion failed';
      confirmDeleteAccount = false;
    }
  }
</script>

<header class="topbar">
  <div>
    <p class="eyebrow">Parental control</p>
    <h1>Parent Dashboard</h1>
  </div>
  <div class="account-box">
    <strong>{parent.name}</strong>
    <span>{parent.email}</span>
    <button class="secondary" onclick={() => onLogout()}>Log out</button>
  </div>
</header>

{#if error}<p class="err">{error}</p>{/if}

<section class="stats-grid">
  <article class="stat-card">
    <span>Total children</span>
    <strong>{dashboardStats.total}</strong>
    <small>Profiles under your supervision</small>
  </article>
  <article class="stat-card accent">
    <span>Paired devices</span>
    <strong>{dashboardStats.paired}</strong>
    <small>Monitoring is active</small>
  </article>
  <article class="stat-card warn">
    <span>Pending pairing</span>
    <strong>{dashboardStats.pending}</strong>
    <small>Ready for consent and setup</small>
  </article>
  <article class="stat-card danger">
    <span>Risk alerts</span>
    <strong>{dashboardStats.alerts}</strong>
    <small>Recent issues needing review</small>
  </article>
</section>

<section class="panel">
  <div class="panel-header">
    <h2>Monitoring overview</h2>
    <span class="status-pill online">Healthy</span>
  </div>
  <div class="overview-grid">
    <div>
      <h3>Recent activity</h3>
      <ul class="activity-list">
        {#each recentSearches as item (item.id)}
          <li>
            <div class="dot"></div>
            <div>
              <strong>{item.childName}</strong>
              <p>{item.query}</p>
            </div>
            <small>{item.time}</small>
          </li>
        {/each}
      </ul>
    </div>
    <div>
      <h3>Alert feed</h3>
      <ul class="alert-list">
        {#each alertFeed as alert (alert.id)}
          <li class:high={alert.risk === 'High'} class:medium={alert.risk === 'Medium'}>
            <div>
              <strong>{alert.childName}</strong>
              <span>{alert.category}</span>
            </div>
            <p>“{alert.query}”</p>
            <footer>
              <small>{alert.time}</small>
              <em>{alert.risk}</em>
            </footer>
          </li>
        {/each}
      </ul>
    </div>
  </div>
</section>

<section class="panel">
  <div class="panel-header">
    <h2>Child devices</h2>
    <button class="text-button" onclick={handleDeleteSelected} disabled={selected.size === 0}>
      Delete selected ({selected.size})
    </button>
  </div>

  {#if loading}
    <p>Loading…</p>
  {:else if children.length === 0}
    <p class="empty-state">No child devices registered yet.</p>
  {:else}
    <div class="child-list">
      {#each children as child (child.id)}
        <article class="child-card">
          <label class="checkbox-row">
            <input type="checkbox" checked={selected.has(child.id)} onchange={() => toggle(child.id)} />
            <span>{child.name}</span>
          </label>

          <div class="meta-row">
            {#if child.age !== undefined}<span>Age {child.age}</span>{/if}
            <span>{child.platform ?? 'n/a'}</span>
            <span>{child.app_name ?? 'n/a'}</span>
          </div>

          <div class="device-row">
            <strong>{child.device_id ? 'Paired device' : 'Ready to pair'}</strong>
            <code>{child.device_id ?? 'No device linked yet'}</code>
          </div>

          {#if !child.device_id}
            <div class="consent-box">
              <label class="ack-box">
                <input type="checkbox" bind:checked={acknowledgementChecked} />
                I acknowledge the device user has consented and I understand the monitor can capture browser search activity and active window titles.
              </label>
              <button
                class="primary"
                onclick={() => handleCreatePairingInvite(child.id)}
                disabled={generatingInviteFor === child.id}
              >
                {generatingInviteFor === child.id ? 'Creating code…' : 'Create pairing code'}
              </button>
            </div>

            {#if pairingInvites[child.id]}
              <div class="pairing-box">
                <p>One-time code</p>
                <code>{pairingInvites[child.id].code}</code>
                <small>Expires {new Date(pairingInvites[child.id].expires_at).toLocaleString()}</small>
              </div>
            {/if}
          {:else}
            <span class="status-pill success">Connected</span>
          {/if}
        </article>
      {/each}
    </div>
  {/if}
</section>

<section class="panel form-panel">
  <h2>Create a child profile</h2>
  <form onsubmit={handleAddChild} class="child-form">
    <label>
      Name
      <input bind:value={newName} required />
    </label>
    <label>
      Age
      <input type="number" min={0} max={17} bind:value={newAge} />
    </label>
    <button type="submit" class="primary" disabled={adding}>
      {adding ? 'Creating profile…' : 'Create child profile'}
    </button>
  </form>
</section>

<section class="panel danger-panel">
  <h2>Account management</h2>
  {#if !confirmDeleteAccount}
    <button class="danger" onclick={() => (confirmDeleteAccount = true)}>Delete my account</button>
  {:else}
    <p>This permanently deletes your account and all associated children. Are you sure?</p>
    <div class="danger-actions">
      <button class="danger" onclick={handleDeleteAccount}>Yes, delete everything</button>
      <button class="secondary" onclick={() => (confirmDeleteAccount = false)}>Cancel</button>
    </div>
  {/if}
</section>

<style>
  :global(body) {
    background: linear-gradient(180deg, #f4f7fb 0%, #edf2ff 100%);
    font-family: 'Segoe UI', sans-serif;
    color: #1f2937;
  }

  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .eyebrow {
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 0.72rem;
    color: #5b6ef8;
    margin: 0;
  }

  h1 {
    margin: 0.2rem 0 0;
    font-size: clamp(2rem, 3vw, 2.6rem);
  }

  .account-box {
    display: flex;
    align-items: center;
    gap: 0.8rem;
    background: rgba(255, 255, 255, 0.7);
    padding: 0.75rem 1rem;
    border-radius: 14px;
    border: 1px solid rgba(91, 110, 248, 0.18);
    box-shadow: 0 12px 40px rgba(79, 70, 229, 0.08);
  }

  .account-box span {
    color: #667085;
    font-size: 0.85rem;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .stat-card {
    background: white;
    border-radius: 18px;
    padding: 1rem 1.2rem;
    border: 1px solid rgba(148, 163, 184, 0.2);
    box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
  }

  .stat-card span {
    display: block;
    color: #667085;
    font-size: 0.8rem;
  }

  .stat-card strong {
    display: block;
    margin: 0.7rem 0 0.2rem;
    font-size: 2rem;
  }

  .stat-card small {
    color: #475467;
  }

  .stat-card.accent { border-color: rgba(59, 130, 246, 0.3); }
  .stat-card.warn { border-color: rgba(251, 191, 36, 0.4); }
  .stat-card.danger { border-color: rgba(239, 68, 68, 0.35); }

  .panel {
    background: rgba(255, 255, 255, 0.9);
    border: 1px solid rgba(148, 163, 184, 0.25);
    border-radius: 18px;
    padding: 1.2rem;
    margin-bottom: 1.5rem;
    box-shadow: 0 12px 40px rgba(15, 23, 42, 0.03);
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .panel-header h2,
  .panel h2 {
    margin: 0;
  }

  .status-pill {
    border-radius: 999px;
    padding: 0.25rem 0.7rem;
    font-size: 0.72rem;
    font-weight: 700;
  }

  .status-pill.online {
    background: rgba(34, 197, 94, 0.12);
    color: #15803d;
  }

  .status-pill.success {
    background: rgba(34, 197, 94, 0.12);
    color: #15803d;
    display: inline-block;
    margin-top: 0.75rem;
  }

  .overview-grid {
    display: grid;
    grid-template-columns: 1.1fr 0.9fr;
    gap: 1.25rem;
  }

  .activity-list,
  .alert-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }

  .activity-list li {
    display: grid;
    grid-template-columns: 10px 1fr auto;
    gap: 0.8rem;
    align-items: center;
    background: #f8fafc;
    border-radius: 12px;
    padding: 0.7rem 0.8rem;
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: linear-gradient(135deg, #4f46e5, #38bdf8);
  }

  .activity-list p,
  .alert-list p {
    margin: 0.2rem 0 0;
    color: #475467;
  }

  button {
    border: none;
    border-radius: 10px;
    padding: 0.65rem 1rem;
    cursor: pointer;
    font-weight: 600;
  }

  .primary {
    background: linear-gradient(135deg, #4f46e5, #3b82f6);
    color: white;
  }

  .secondary {
    background: #eef2ff;
    color: #3730a3;
  }

  .text-button {
    background: transparent;
    color: #4f46e5;
    padding: 0.4rem 0.1rem;
  }

  .danger {
    background: #ef4444;
    color: white;
  }

  .child-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 1rem;
  }

  .child-card {
    border: 1px solid rgba(148, 163, 184, 0.25);
    border-radius: 16px;
    padding: 1rem;
    background: linear-gradient(180deg, #ffffff, #f8fafc);
  }

  .checkbox-row {
    display: flex;
    align-items: center;
    gap: 0.7rem;
    font-size: 1.02rem;
    font-weight: 700;
  }

  .meta-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin: 0.8rem 0;
    color: #475467;
    font-size: 0.82rem;
  }

  .device-row {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-bottom: 0.8rem;
    color: #334155;
  }

  .device-row code {
    font-size: 0.73rem;
    word-break: break-all;
  }

  .consent-box {
    background: #f8fafc;
    border: 1px solid rgba(79, 70, 229, 0.12);
    border-radius: 12px;
    padding: 0.8rem;
    margin-top: 0.8rem;
  }

  .ack-box {
    display: flex;
    gap: 0.75rem;
    align-items: flex-start;
    font-size: 0.82rem;
    color: #374151;
    margin-bottom: 0.75rem;
  }

  .pairing-box {
    background: #eef2ff;
    border: 1px solid rgba(79, 70, 229, 0.15);
    border-radius: 12px;
    padding: 0.8rem;
    margin-top: 0.8rem;
  }

  .pairing-box p {
    margin: 0 0 0.4rem;
    font-weight: 700;
  }

  .pairing-box code {
    display: inline-block;
    background: white;
    padding: 0.35rem 0.6rem;
    border-radius: 8px;
    font-size: 1.1rem;
    letter-spacing: 0.12em;
  }

  .child-form {
    display: flex;
    align-items: end;
    flex-wrap: wrap;
    gap: 1rem;
    margin-top: 0.8rem;
  }

  .child-form label {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    min-width: 200px;
  }

  input {
    border: 1px solid #d0d7e2;
    border-radius: 10px;
    padding: 0.7rem 0.8rem;
    background: white;
    font: inherit;
  }

  .alert-list li {
    border-radius: 12px;
    padding: 0.8rem;
    border: 1px solid rgba(148, 163, 184, 0.2);
    background: #f8fafc;
  }

  .alert-list li.high {
    border-left: 4px solid #ef4444;
  }

  .alert-list li.medium {
    border-left: 4px solid #f59e0b;
  }

  .alert-list div {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    margin-bottom: 0.25rem;
  }

  .alert-list footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .alert-list em {
    font-style: normal;
    font-weight: 700;
    color: #b45309;
  }

  .danger-panel {
    border-color: rgba(239, 68, 68, 0.2);
  }

  .danger-actions {
    display: flex;
    gap: 0.75rem;
    margin-top: 0.75rem;
  }

  .empty-state {
    color: #667085;
  }

  .err {
    background: #fff2f2;
    color: #b42318;
    border: 1px solid rgba(180, 35, 24, 0.12);
    border-radius: 10px;
    padding: 0.75rem 0.85rem;
    margin-bottom: 1rem;
  }

  @media (max-width: 720px) {
    .topbar,
    .account-box,
    .panel-header,
    .overview-grid,
    .child-form {
      display: block;
    }

    .account-box {
      margin-top: 0.75rem;
    }
    .child-form label {
      min-width: 100%;
      margin-bottom: 0.8rem;
    }
  }
</style>