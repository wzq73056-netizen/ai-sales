<script setup lang="ts">
defineOptions({ name: 'EmptyState' });

interface Props {
  title?: string;
  desc?: string;
}

withDefaults(defineProps<Props>(), {
  title: '暂无数据',
  desc: ''
});
</script>

<template>
  <div class="ds-empty">
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" class="op-20">
      <rect x="3" y="5" width="18" height="14" rx="2.5" stroke="currentColor" stroke-width="1.4" />
      <path d="M3 10h18" stroke="currentColor" stroke-width="1.4" />
      <path d="M8 15h4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
    </svg>
    <div class="ds-empty-title">{{ title }}</div>
    <div v-if="desc" class="ds-empty-desc">{{ desc }}</div>
    <div v-if="$slots.default" class="mt-8px">
      <slot />
    </div>
  </div>
</template>
