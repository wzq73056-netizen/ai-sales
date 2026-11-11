<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';

defineOptions({ name: 'OfflineBanner' });

const offline = ref(!navigator.onLine);
const back = ref(false);

function handleOffline() {
  offline.value = true;
  back.value = false;
}

function handleOnline() {
  offline.value = false;
  back.value = true;

  window.setTimeout(() => {
    back.value = false;
  }, 2500);
}

onMounted(() => {
  window.addEventListener('offline', handleOffline);
  window.addEventListener('online', handleOnline);
});

onUnmounted(() => {
  window.removeEventListener('offline', handleOffline);
  window.removeEventListener('online', handleOnline);
});
</script>

<template>
  <div v-if="offline" class="offline-banner">
    <SvgIcon icon="mdi:wifi-off" class="text-14px" />
    <span>当前离线，展示的是上次缓存的数据</span>
  </div>
  <div v-else-if="back" class="offline-banner is-online">
    <SvgIcon icon="mdi:wifi" class="text-14px" />
    <span>网络已恢复</span>
  </div>
</template>

<style scoped>
.offline-banner {
  position: fixed;
  left: 50%;
  top: 12px;
  z-index: 200;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 999px;
  font-size: 12px;
  color: #fff;
  background: rgba(15, 23, 42, 0.86);
  transform: translateX(-50%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.22);
}

.offline-banner.is-online {
  background: rgba(5, 150, 105, 0.92);
}
</style>
