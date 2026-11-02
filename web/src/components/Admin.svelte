<script>
  import AdminLogin from './AdminLogin.svelte';
  import AdminDashboard from './AdminDashboard.svelte';
  import { migrateKey } from '../storage.js';

  migrateKey(localStorage, 'jfshare_admin_key', 'jellyshare_admin_key');

  let apiKey = localStorage.getItem('jellyshare_admin_key') || '';
  let isLoggedIn = !!apiKey;

  function handleLogin(event) {
    apiKey = event.detail.apiKey;
    localStorage.setItem('jellyshare_admin_key', apiKey);
    isLoggedIn = true;
  }

  function handleLogout() {
    apiKey = '';
    localStorage.removeItem('jellyshare_admin_key');
    isLoggedIn = false;
  }
</script>

{#if isLoggedIn}
  <AdminDashboard {apiKey} on:logout={handleLogout} />
{:else}
  <AdminLogin on:login={handleLogin} />
{/if}
