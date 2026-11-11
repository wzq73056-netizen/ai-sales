<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { NButton, NCard, NTag, useMessage } from 'naive-ui';
import PageHeader from '@/components/business/PageHeader.vue';
import SystemAlerts from '@/components/business/SystemAlerts.vue';
import EmptyState from '@/components/business/EmptyState.vue';
import { fetchWorkbench, type Workbench } from '@/service/api/workbench';
import { fetchFollowupReminders, type ReminderBoard } from '@/service/api/wechat';
import { fetchDaemons, startDaemon, stopDaemon, type DaemonItem } from '@/service/api/daemon';

defineOptions({ name: 'Workbench' });

const router = useRouter();
const message = useMessage();
const data = ref<Workbench | null>(null);
const todos = ref<ReminderBoard | null>(null);
const loading = ref(false);
const daemons = ref<DaemonItem[]>([]);
const daemonLoading = ref(false);
const daemonBusy = ref('');

/** 拉取三个监听进程的状态 */
async function loadDaemons() {
  daemonLoading.value = true;

  const { data: payload } = await fetchDaemons();

  if (payload?.list) daemons.value = payload.list;

  daemonLoading.value = false;
}

/** 启动 / 停止某个监听进程 */
async function toggleDaemon(item: DaemonItem) {
  daemonBusy.value = item.Key;

  const { data: payload, error } = item.Alive ? await stopDaemon(item.Key) : await startDaemon(item.Key);

  if (error) message.error(error.message || '操作失败');
  else if (payload) {
    if (payload.list) daemons.value = payload.list;

    message.success(payload.msg || '已操作');
  }

  daemonBusy.value = '';
}

async function load() {
  loading.value = true;

  const [{ data: payload }, { data: reminderData }] = await Promise.all([fetchWorkbench(), fetchFollowupReminders()]);
  if (reminderData) todos.value = reminderData;

  if (payload) data.value = payload;

  loading.value = false;
}

function go(kind: string) {
  if (kind === 'lead') router.push('/lead');
  else if (kind === 'douyin') router.push('/douyin');
  else router.push('/wechat');
}

function goHandoff() {
  router.push('/wechat');
}

function goLead() {
  router.push('/lead');
}

function formatTime(value: string | null) {
  if (!value) return '-';

  return String(value).replace('T', ' ').slice(0, 16);
}

function gradeType(grade: string) {
  if (grade === 'S') return 'error' as const;
  if (grade === 'A') return 'warning' as const;
  if (grade === 'B') return 'info' as const;
  return 'default' as const;
}

function tierType(tier: string) {
  if (tier === 'A') return 'error' as const;
  if (tier === 'B') return 'warning' as const;
  if (tier === 'C') return 'info' as const;
  return 'default' as const;
}

onMounted(() => {
  load();
  loadDaemons();
});
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="今日工作台" :desc="`${data?.today || ''} · 只看今天需要决策的信号`">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>

    <SystemAlerts />

    <div v-if="data?.judgement" class="judgement-card">
      <div class="judgement-tag">今日判断</div>
      <div class="judgement-headline">{{ data.judgement.headline }}</div>
      <div class="judgement-grid">
        <div class="judgement-cell">
          <div class="judgement-label">依据</div>
          <div class="judgement-text">{{ (data.judgement.basis || []).join('；') || '-' }}</div>
        </div>
        <div class="judgement-cell">
          <div class="judgement-label">建议动作</div>
          <div class="judgement-text">{{ data.judgement.suggestion }}</div>
        </div>
        <div class="judgement-cell">
          <div class="judgement-label">不做的影响</div>
          <div class="judgement-text">{{ data.judgement.impact }}</div>
        </div>
      </div>
    </div>

    <div class="signal-bar">
      <span class="text-12px font-medium op-60">今日待办</span>
      <span class="signal-item text-red-500" @click="router.push('/reminders')">
        <i class="dot" />逾期 <b>{{ todos?.Overdue?.length ?? 0 }}</b>
      </span>
      <span class="signal-item text-orange-500" @click="router.push('/reminders')">
        <i class="dot" />今天 <b>{{ todos?.Today?.length ?? 0 }}</b>
      </span>
      <span class="signal-item text-blue-500" @click="router.push('/reminders')">
        <i class="dot" />未来 7 天 <b>{{ todos?.Week?.length ?? 0 }}</b>
      </span>
      <span class="signal-item op-60" @click="router.push('/reminders')">
        <i class="dot" />沉睡 <b>{{ todos?.Silent?.length ?? 0 }}</b>
      </span>
      <span class="ml-auto cursor-pointer text-12px text-blue-500" @click="router.push('/reminders')">全部提醒 →</span>
    </div>

    <div class="grid grid-cols-4 gap-12px">
      <div v-for="item in data?.items || []" :key="item.kind" class="kpi-card" @click="go(item.kind)">
        <div class="kpi-label">{{ item.title }}</div>
        <div class="kpi-value" :class="item.count > 0 ? 'text-blue-600' : 'op-35'">{{ item.count }}</div>
        <div class="kpi-hint">{{ item.subtitle }}</div>
        <div class="kpi-action">{{ item.action }} →</div>
      </div>
    </div>

    <div class="grid grid-cols-2 items-stretch gap-16px">
      <NCard :bordered="false" size="small" class="h-full">
        <template #header>
          <span>待接管工单</span>
        </template>
        <template #header-extra>
          <NButton text type="primary" size="tiny" @click="goHandoff">全部</NButton>
        </template>

        <div v-if="!data?.handoffs?.length" class="py-20px text-center text-12px op-40">
          没有待接管，AI 正在自动回复
        </div>

        <div v-for="row in data?.handoffs || []" :key="row.Id" class="mb-10px border-b border-gray-100 pb-8px dark:border-gray-800">
          <div class="flex items-center gap-6px">
            <NTag size="tiny" :type="row.Channel === 'douyin' ? 'warning' : 'success'">
              {{ row.Channel === 'douyin' ? '抖音' : '微信' }}
            </NTag>
            <span class="text-13px font-medium">{{ row.NickName }}</span>
            <NTag size="tiny" type="info">{{ row.Category }}</NTag>
          </div>
          <div class="mt-4px truncate text-12px op-60">{{ row.LastMsg }}</div>
        </div>
      </NCard>

      <NCard :bordered="false" size="small" class="h-full">
        <template #header>
          <span>高价值客户</span>
        </template>
        <template #header-extra>
          <NButton text type="primary" size="tiny" @click="goHandoff">去跟进</NButton>
        </template>

        <div v-if="!data?.hotCustomers?.length" class="py-20px text-center text-12px op-40">还没有 A/B 类客户</div>

        <div v-for="row in data?.hotCustomers || []" :key="row.Id" class="mb-10px border-b border-gray-100 pb-8px dark:border-gray-800">
          <div class="flex items-center gap-6px">
            <NTag size="tiny" :type="tierType(row.Tier)">{{ row.Tier }} 类</NTag>
            <span class="text-13px font-medium">{{ row.NickName }}</span>
            <span class="text-11px op-50">{{ row.Score }} 分</span>
            <NTag v-if="row.Stage" size="tiny">{{ row.Stage }}</NTag>
          </div>
          <div class="mt-4px truncate text-12px op-60" :title="row.NextAction">
            {{ row.NextAction || '暂无建议' }}
          </div>
        </div>
      </NCard>
    </div>

    <NCard :bordered="false" size="small">
      <template #header>
        <span>监听进程</span>
      </template>
      <template #header-extra>
        <div class="flex items-center gap-8px">
          <span class="text-11px op-40">抖音 / 小红书 / 微信</span>
          <NButton text type="primary" size="tiny" :loading="daemonLoading" @click="loadDaemons">刷新</NButton>
        </div>
      </template>

      <div class="flex flex-col gap-8px">
        <div
          v-for="item in daemons"
          :key="item.Key"
          class="flex items-center justify-between gap-12px rounded-8px border border-gray-100 px-12px py-9px dark:border-gray-800"
        >
          <div class="min-w-0 flex items-center gap-8px">
            <span class="h-7px w-7px flex-none rounded-full" :class="item.Alive ? 'bg-green-500' : 'bg-gray-300'" />
            <div class="min-w-0">
              <div class="text-13px font-medium">{{ item.Name }}</div>
              <div class="truncate text-11px op-45">
                <template v-if="item.Alive">
                  运行中 · 心跳 {{ item.Heartbeat || '-' }}
                  <template v-if="item.Pid"> · PID {{ item.Pid }}</template>
                </template>
                <template v-else-if="item.LastLog">已停止 · {{ item.LastLog }}</template>
                <template v-else>未运行</template>
              </div>
            </div>
          </div>
          <NButton
            size="tiny"
            :type="item.Alive ? 'default' : 'primary'"
            :loading="daemonBusy === item.Key"
            @click="toggleDaemon(item)"
          >
            {{ item.Alive ? '停止' : '启动' }}
          </NButton>
        </div>
      </div>
    </NCard>

    <NCard :bordered="false" size="small" class="h-full">
      <template #header>
        <span>优先线索（A1 有联系方式 · A2 需先互动）</span>
      </template>
      <template #header-extra>
        <div class="flex items-center gap-8px">
          <span class="text-11px op-50">今日新增 {{ data?.newLeadToday ?? 0 }}</span>
          <NButton text type="primary" size="tiny" @click="goLead">去线索池</NButton>
        </div>
      </template>

      <table class="data-table mobile-cards">
        <thead>
          <tr>
            <th>客户</th>
            <th>关键词</th>
            <th>地区</th>
            <th class="center">等级</th>
            <th class="center">评分</th>
            <th>来源</th>
            <th class="num">采集时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in data?.newLeads || []" :key="row.Id">
            <td class="font-medium">{{ row.Company || '-' }}</td>
            <td class="text-12px op-70">{{ row.Keyword || '-' }}</td>
            <td class="text-12px">{{ row.City || '-' }}</td>
            <td class="center">
              <NTag size="tiny" :type="gradeType(row.Grade)">
                {{ row.Grade }}{{ row.Ready ? '1' : '2' }}
              </NTag>
            </td>
            <td class="center">{{ row.Score }}</td>
            <td class="text-12px op-60">{{ row.Source || '-' }}</td>
            <td class="num text-11px op-50">{{ formatTime(row.CreatedAt) }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="!data?.newLeads?.length" class="py-30px text-center text-12px op-40">还没有新线索</div>
    </NCard>
  </div>
</template>

<style scoped>
.judgement-card {
  padding: 18px 20px;
  border: 1px solid rgba(37, 99, 235, 0.16);
  border-radius: 10px;
  background: linear-gradient(100deg, rgba(37, 99, 235, 0.06), rgba(37, 99, 235, 0.01));
}

.judgement-tag {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: #2563eb;
  text-transform: uppercase;
}

.judgement-headline {
  margin-top: 6px;
  font-size: 19px;
  font-weight: 650;
  line-height: 1.45;
  letter-spacing: -0.01em;
  color: var(--ds-text);
}

.judgement-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 18px;
  margin-top: 14px;
}

.judgement-cell {
  padding-left: 12px;
  border-left: 2px solid rgba(37, 99, 235, 0.14);
}

.judgement-label {
  margin-bottom: 4px;
  font-size: 11px;
  color: var(--ds-text-3);
}

.judgement-text {
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--ds-text-2);
}

@media (max-width: 767px) {
  .judgement-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }

  .judgement-cell {
    padding-left: 10px;
  }

  .judgement-headline {
    font-size: 17px;
  }
}
</style>