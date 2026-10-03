<script lang="ts">
  import { login, getCurrentParent } from '../lib/api';
  import type { Parent } from '../lib/types';

  let {
    onSuccess,
    onGotoRegister,
    errorMessage = '',
  }: {
    onSuccess: (p: Parent) => void;
    onGotoRegister: () => void;
    errorMessage?: string;
  } = $props();

  let email = $state('');
  let password = $state('');
  let submitting = $state(false);
  let error = $state('');

  async function handleLogin(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    submitting = true;
    try {
      await login(email, password);
      const parent = await getCurrentParent();
      onSuccess(parent);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Login failed';
    } finally {
      submitting = false;
    }
  }
</script>

<section>
  <h2>Parent Login</h2>
  {#if errorMessage}<p class="err">{errorMessage}</p>{/if}
  {#if error}<p class="err">{error}</p>{/if}

  <form onsubmit={handleLogin}>
    <label>
      Email
      <input type="email" bind:value={email} required />
    </label>
    <label>
      Password
      <input type="password" bind:value={password} required />
    </label>
    <button type="submit" disabled={submitting}>
      {submitting ? 'Logging in…' : 'Log in'}
    </button>
  </form>

  <p>
    No account?
    <button type="button" class="link" onclick={() => onGotoRegister()}>
      Register
    </button>
  </p>
</section>

<style>
  label { display: block; margin-bottom: 0.75rem; }
  input { width: 100%; padding: 0.5rem; margin-top: 0.25rem; }
  button { padding: 0.5rem 1.25rem; cursor: pointer; }
  .link { background: none; border: none; color: #4a6cf7; text-decoration: underline; padding: 0; }
  .err { color: #c0392b; }
</style>