<script lang="ts">
  import Login from './views/Login.svelte';
  import Register from './views/Register.svelte';
  import Dashboard from './views/Dashboard.svelte';
  import { logout } from './lib/api';
  import type { Parent } from './lib/types';

  type View = 'login' | 'register' | 'dashboard';

  let view: View = $state('login');
  let parent: Parent | null = $state(null);
  let errorMessage = $state('');

  function handleExpired() {
    parent = null;
    view = 'login';
    errorMessage = 'Your session has expired. Please log in again.';
  }

  function onLoginSuccess(p: Parent) {
    parent = p;
    view = 'dashboard';
    errorMessage = '';
  }

  function onRegisterSuccess() {
    view = 'login';
  }

  async function handleLogout() {
    try {
      await logout();
    } finally {
      parent = null;
      view = 'login';
    }
  }
</script>

<svelte:window on:session:expired={handleExpired} />

<main>
  {#if view === 'login'}
    <Login
      onSuccess={onLoginSuccess}
      onGotoRegister={() => (view = 'register')}
      {errorMessage}
    />
  {:else if view === 'register'}
    <Register
      onSuccess={onRegisterSuccess}
      onGotoLogin={() => (view = 'login')}
    />
  {:else if view === 'dashboard' && parent}
    <Dashboard {parent} onLogout={handleLogout} />
  {/if}
</main>