<script setup lang="ts">
defineOptions({ name: 'ErrorState' });

interface Props {
  title?: string;
  desc?: string;
  loading?: boolean;
}

withDefaults(defineProps<Props>(), {
  title: '加载失败',
  desc: '网络或服务出了问题，重试一下',
  loading: false
});

const emit = defineEmits<{ retry: [] }>();
</script>

<template>
  <div class="error-state">
    <svg width="36" height="36" viewBox="0 0 24 24" fill="none" class="op-25">
      <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="1.4" />
      <path d="M12 8v5" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
      <circle cx="12" cy="16" r="0.9" fill="currentColor" />
    </svg>
    <div class="error-title">{{ title }}</div>
    <div class="error-desc">{{ desc }}</div>
    <NButton size="small" :loading="loading" @click="emit('retry')">重试</NButton>
  </div>
</template>

<style scoped>
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 46px 16px;
  color: #b91c1c;
}

.error-title {
  font-size: 13px;
  font-weight: 550;
}

.error-desc {
  font-size: 12px;
  color: var(--ds-text-3);
}

.error-state :deep(.n-button) {
  margin-top: 6px;
}
</style>
