<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { fetchSystemAlerts, type SystemAlert } from '@/service/api/alert';

defineOptions({ name: 'SystemAlerts' });

const alerts = ref<SystemAlert[]>([]);
const wxLastOk = ref('');
const dismissed = ref<Record<string, boolean>>({});

async function load() {
  const { data } = await fetchSystemAlerts();

  if (data) {
    alerts.value = data.list || [];
    wxLastOk.value = data.wx?.LastSuccess || '';
  }
}

function dismiss(title: string) {
  dismissed.value = { ...dismissed.value, [title]: true };
}

onMounted(load);
</script>

<template>
  <div v-if="alerts.length" class="flex flex-col gap-8px">
    <div
      v-for="item in alerts"
      v-show="!dismissed[item.Title]"
      :key="item.Title"
      class="system-alert"
      :class="item.Level === 'error' ? 'is-error' : 'is-warning'"
    >
      <SvgIcon :icon="item.Level === 'error' ? 'mdi:alert-circle' : 'mdi:alert'" class="text-16px flex-none" />
      <div class="min-w-0 flex-1">
        <div class="text-13px font-medium">{{ item.Title }}</div>
        <div class="mt-2px text-12px op-75">{{ item.Detail }}</div>
        <div v-if="wxLastOk" class="mt-2px text-11px op-50">微信最后一次正常时间：{{ wxLastOk }}</div>
      </div>
      <NButton size="tiny" quaternary @click="dismiss(item.Title)">知道了</NButton>
    </div>
  </div>
</template>

<style scoped>
.system-alert {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid rgba(220, 38, 38, 0.25);
  background: rgba(220, 38, 38, 0.06);
  color: #b91c1c;
}

.system-alert.is-warning {
  border-color: rgba(217, 119, 6, 0.28);
  background: rgba(217, 119, 6, 0.07);
  color: #b45309;
}
</style>
