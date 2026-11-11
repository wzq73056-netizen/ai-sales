<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAppStore } from '@/store/modules/app';

defineOptions({ name: 'MobileTabBar' });

const appStore = useAppStore();
const route = useRoute();
const router = useRouter();

const items = [
  { key: 'workbench', label: '工作台', icon: 'mdi:monitor-dashboard', path: '/workbench' },
  { key: 'lead', label: '线索', icon: 'mdi:magnify-scan', path: '/lead' },
  { key: 'customers', label: '客户', icon: 'mdi:account-group-outline', path: '/customers' },
  { key: 'reminders', label: '提醒', icon: 'mdi:bell-ring-outline', path: '/reminders' },
  { key: 'inbox', label: '收件箱', icon: 'mdi:inbox-arrow-down-outline', path: '/inbox' }
];

const current = computed(() => route.path);

/** 登录、异常等独立页不显示底部导航 */
const hiddenPaths = ['/login', '/403', '/404', '/500'];

const visible = computed(() => appStore.isMobile && !hiddenPaths.includes(route.path));

function go(path: string) {
  if (current.value !== path) router.push(path);
}
</script>

<template>
  <nav v-if="visible" class="mobile-tabbar">
    <button
      v-for="item in items"
      :key="item.key"
      type="button"
      class="mobile-tabbar__item"
      :class="{ 'is-active': current === item.path }"
      @click="go(item.path)"
    >
      <SvgIcon :icon="item.icon" class="text-18px" />
      <span>{{ item.label }}</span>
    </button>
  </nav>
</template>

<style scoped>
.mobile-tabbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 60;
  display: flex;
  align-items: stretch;
  height: 54px;
  padding-bottom: env(safe-area-inset-bottom);
  border-top: 1px solid var(--ds-border);
  background: var(--ds-surface);
}

.mobile-tabbar__item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  font-size: 10.5px;
  color: var(--ds-text-3);
  background: transparent;
  border: none;
  transition: color 150ms ease;
}

.mobile-tabbar__item.is-active {
  color: var(--ds-primary);
}
</style>
