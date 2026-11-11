<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import PageHeader from '@/components/business/PageHeader.vue';
import EmptyState from '@/components/business/EmptyState.vue';
import { useRouter } from 'vue-router';
import { NButton, NCard, NTag } from 'naive-ui';
import { fetchFollowupReminders, type ReminderBoard, type ReminderRow } from '@/service/api/wechat';

defineOptions({ name: 'Reminders' });

const router = useRouter();

const data = ref<ReminderBoard | null>(null);
const loading = ref(false);
const activeBucket = ref<'Overdue' | 'Today' | 'Week' | 'Silent'>('Today');

const buckets = [
  { key: 'Overdue' as const, label: '逾期', color: 'error' as const },
  { key: 'Today' as const, label: '今天', color: 'warning' as const },
  { key: 'Week' as const, label: '未来 7 天', color: 'info' as const },
  { key: 'Silent' as const, label: '沉睡客户', color: 'default' as const }
];

const rows = computed<ReminderRow[]>(() => (data.value?.[activeBucket.value] as ReminderRow[]) ?? []);

function countOf(key: 'Overdue' | 'Today' | 'Week' | 'Silent') {
  return (data.value?.[key] as ReminderRow[] | undefined)?.length ?? 0;
}

async function load() {
  loading.value = true;
  const { data: payload } = await fetchFollowupReminders();

  if (payload) {
    data.value = payload;

    // 默认落在第一个有内容的桶
    const order: ('Overdue' | 'Today' | 'Week' | 'Silent')[] = ['Overdue', 'Today', 'Week', 'Silent'];
    const first = order.find(key => ((payload[key] as ReminderRow[]) || []).length > 0);

    if (first) activeBucket.value = first;
  }

  loading.value = false;
}

function openCustomer() {
  router.push('/wechat');
}

onMounted(load);
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="跟进提醒" desc="该跟谁、什么时候跟、为什么该跟">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>



    <div class="grid grid-cols-4 gap-12px">
      <div
        v-for="item in buckets"
        :key="item.key"
        class="kpi-card"
        :class="activeBucket === item.key ? 'ring-2 ring-blue-400' : ''"
        @click="activeBucket = item.key"
      >
        <div class="kpi-label">{{ item.label }}</div>
        <div class="kpi-value" :class="countOf(item.key) > 0 ? 'text-blue-600' : 'op-35'">
          {{ countOf(item.key) }}
        </div>
        <div class="kpi-hint">
          {{ item.key === 'Overdue' ? '已过期未跟进' : item.key === 'Today' ? '今天到期' : item.key === 'Week' ? '7 天内到期' : '超过 7 天没说话' }}
        </div>
      </div>
    </div>

    <NCard :bordered="false" size="small" class="flex-1">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="text-13px">{{ buckets.find(b => b.key === activeBucket)?.label }}　共 {{ rows.length }} 条</span>
          <NButton text type="primary" size="tiny" @click="openCustomer">去客户跟进</NButton>
        </div>
      </template>

      <table class="data-table mobile-cards">
        <thead>
          <tr>
            <th style="width: 160px">客户</th>
            <th style="width: 70px" class="center">等级</th>
            <th style="width: 100px" class="center">阶段</th>
            <th style="width: 60px" class="num">评分</th>
            <th>下一步动作</th>
            <th style="width: 150px" class="num">计划跟进</th>
            <th style="width: 90px" class="num">沉默天数</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.Id">
            <td class="font-medium" data-label="客户">{{ row.NickName || '-' }}</td>
            <td class="center" data-label="等级">
              <NTag size="small" :type="row.Tier === 'A' ? 'success' : row.Tier === 'B' ? 'warning' : 'default'">
                {{ row.Tier || '-' }}
              </NTag>
            </td>
            <td class="center text-12px" data-label="阶段">{{ row.Stage || '-' }}</td>
            <td class="num" data-label="评分">{{ row.Score }}</td>
            <td class="truncate-cell text-12px op-70" data-label="下一步">{{ row.NextAction || row.Reason || '-' }}</td>
            <td class="num text-12px op-50" data-label="计划跟进">{{ row.NextFollow || '-' }}</td>
            <td class="num text-12px" data-label="沉默天数" :class="row.SilentDays >= 7 ? 'text-orange-500' : 'op-50'">
              {{ row.SilentDays }}
            </td>
          </tr>
        </tbody>
      </table>

      <EmptyState v-if="!rows.length" title="这一组没有要跟的客户" desc="有逾期、今天到期或沉默超过 7 天的客户时会出现在这里" />
    </NCard>
  </div>
</template>
