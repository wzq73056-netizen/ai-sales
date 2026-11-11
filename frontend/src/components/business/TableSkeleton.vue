<script setup lang="ts">
import { computed } from 'vue';

defineOptions({ name: 'TableSkeleton' });

interface Props {
  rows?: number;
  columns?: number;
}

const props = withDefaults(defineProps<Props>(), {
  rows: 6,
  columns: 5
});

const widths = ['22%', '30%', '14%', '10%', '16%', '12%', '8%'];

const colWidths = computed(() => Array.from({ length: props.columns }, (_, i) => widths[i % widths.length]));
</script>

<template>
  <div class="skeleton-table">
    <div v-for="row in rows" :key="row" class="skeleton-row">
      <div v-for="(w, i) in colWidths" :key="i" class="skeleton-bar" :style="{ width: w }" />
    </div>
  </div>
</template>
