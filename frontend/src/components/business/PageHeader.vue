<script setup lang="ts">
defineOptions({ name: 'PageHeader' });

interface Props {
  title: string;
  desc?: string;
}

defineProps<Props>();
</script>

<template>
  <div class="app-page-header">
    <div class="min-w-0">
      <div class="app-page-title">{{ title }}</div>
      <div v-if="desc" class="app-page-desc">{{ desc }}</div>
    </div>
    <div v-if="$slots.actions" class="page-actions">
      <slot name="actions" />
    </div>
  </div>
</template>
