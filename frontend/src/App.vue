<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue';
import zhCN from '@arco-design/web-vue/es/locale/lang/zh-cn';
import { useAuthStore } from './stores/auth';

const auth = useAuthStore();

async function refreshLoginState() {
  auth.syncFromStorage();
  if (!auth.accessToken) return;
  try {
    await auth.loadMe();
  } catch {
    auth.logout();
  }
}

function handleStorage(event: StorageEvent) {
  if (['bitapi.access_token', 'bitapi.refresh_token', 'bitapi.user'].includes(event.key || '')) {
    void refreshLoginState();
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    void refreshLoginState();
  }
}

onMounted(() => {
  void refreshLoginState();
  window.addEventListener('storage', handleStorage);
  window.addEventListener('focus', refreshLoginState);
  document.addEventListener('visibilitychange', handleVisibilityChange);
});

onBeforeUnmount(() => {
  window.removeEventListener('storage', handleStorage);
  window.removeEventListener('focus', refreshLoginState);
  document.removeEventListener('visibilitychange', handleVisibilityChange);
});
</script>

<template>
  <a-config-provider :locale="zhCN">
    <router-view />
  </a-config-provider>
</template>
