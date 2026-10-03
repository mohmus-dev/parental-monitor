<script lang="ts">
  import { registerParent } from '../lib/api';

  let {
    onSuccess,
    onGotoLogin,
  }: {
    onSuccess: () => void;
    onGotoLogin: () => void;
  } = $props();

  let name = $state('');
  let email = $state('');
  let password = $state('');
  let submitting = $state(false);
  let error = $state('');

  async function handleRegister(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    submitting = true;
    try {
      await registerParent({
        name,
        email,
        password,
        app_name: 'Parent Dashboard',
        platform: 'windows',
        device_id: `parent-${crypto.randomUUID()}`,
      });
      onSuccess(); // registration doesn't create a session → go to login
    } catch (err) {
      error = err instanceof Error ? err.message : 'Registration failed';
    } finally {
      submitting = false;
    }
  }
</script>

<section>
  <h2>Register Parent Account</h2>
  {#if error}<p class="err">{error}</p>{/if}

  <form onsubmit={handleRegister}>
    <label>Name <input bind:value={name} required /></label>
    <label>Email <input type="email" bind:value={email} required /></label>
    <label>Password <input type="password" bind:value={password} required minlength={8} /></label>
    <button type="submit" disabled={submitting}>
      {submitting ? 'Registering…' : 'Register'}
    </button>
  </form>

  <p>
    Already have an account?
    <button type="button" class="link" onclick={() => onGotoLogin()}>
      Log in
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