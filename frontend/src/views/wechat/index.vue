<script setup lang="ts">
import { onMounted, ref } from 'vue';
import PageHeader from '@/components/business/PageHeader.vue';
import { NButton, NCard, NDrawer, NDrawerContent, NInput, NInputNumber, NSelect, NSwitch, NTag, useMessage } from 'naive-ui';
import {
  addWxFollowup,
  claimWxHandoff,
  fetchFunnelStats,
  fetchWxAddConfig,
  fetchWxAddQueue,
  fetchWxCustomerDetail,
  fetchWxCustomerGrouped,
  fetchWxCustomerList,
  fetchWxHandoffList,
  resumeAi,
  resolveWxHandoff,
  runWxAddOnce,
  saveWxAddConfig,
  updateWxCustomer,
  updateWxHandoffResult,
  type FunnelStats,
  type WxAddConfig,
  type WxAddQueueRow,
  type WxAddStats,
  type WxCustomer,
  type WxCustomerDetail,
  type WxFollowup,
  type WxHandoff
} from '@/service/api/wechat';

defineOptions({ name: 'WechatGrowth' });

const message = useMessage();

const activeTab = ref<'handoff' | 'customer' | 'addwx'>('handoff');

/* ---------- 转化漏斗 ---------- */
const funnel = ref<FunnelStats | null>(null);

async function loadFunnel() {
  const { data } = await fetchFunnelStats();

  if (data) funnel.value = data;
}

/* ---------- 待接管 ---------- */
const list = ref<WxHandoff[]>([]);
const total = ref(0);
const page = ref(1);
const status = ref(0);
const loading = ref(false);

const statusOptions = [
  { label: '待接管', value: 0 },
  { label: '已接管', value: 1 },
  { label: '已处理', value: 2 },
  { label: '已忽略', value: 3 }
];

async function load() {
  loading.value = true;

  const { data } = await fetchWxHandoffList({ page: page.value, pageSize: 20, status: status.value });

  if (data) {
    list.value = data.records;
    total.value = data.total;
  }

  loading.value = false;
}

async function handleClaim(row: WxHandoff) {
  const { error } = await claimWxHandoff(row.Id, 'admin');

  if (error) return;

  message.success('已认领，AI 不再自动回复该会话');
  load();
}

async function handleResolve(row: WxHandoff, next: number) {
  const { error } = await resolveWxHandoff(row.Id, next);

  if (error) return;

  message.success(next === 3 ? '已忽略' : '已处理');
  load();
}

async function handleWechatAdded(row: WxHandoff, next: number) {
  const { error } = await updateWxHandoffResult(row.Id, next, next >= 2 ? '已通过验证' : '已加好友');

  if (error) return;

  message.success(next >= 2 ? '已标记通过' : '已标记加好友');
  load();
}

async function handleResumeAi(row: WxHandoff) {
  const { error } = await resumeAi(row.Id);

  if (error) return;

  message.success('AI 已恢复该会话的自动回复');
  load();
}

function wechatLabel(added: number) {
  if (added >= 2) return '已通过';
  if (added === 1) return '已加好友';

  return '未加';
}

function urgencyTag(level: number) {
  if (level >= 3) return { type: 'error' as const, text: '紧急' };
  if (level === 2) return { type: 'warning' as const, text: '较急' };
  return { type: 'default' as const, text: '一般' };
}

function formatTime(value: string | null) {
  if (!value) return '-';

  return String(value).replace('T', ' ').slice(0, 19);
}

/* ---------- 客户（按昵称合并） ---------- */
const customers = ref<WxCustomer[]>([]);
const minScore = ref(0);

async function loadCustomers() {
  const { data } = await fetchWxCustomerGrouped({ minScore: minScore.value });

  if (data) customers.value = data.list ?? [];
}

async function handleFollow(row: WxCustomer, next: number) {
  const { error } = await markWxCustomerFollow(row.Id, next);

  if (error) return;

  message.success('已更新');
  loadCustomers();
}

function tierType(tier: string) {
  if (tier === 'A') return 'error' as const;
  if (tier === 'B') return 'warning' as const;
  if (tier === 'C') return 'info' as const;
  return 'default' as const;
}

function stageType(stage: string) {
  if (stage === '待成交' || stage === '已成交') return 'success' as const;
  if (stage === '议价') return 'warning' as const;
  if (stage === '比价') return 'info' as const;
  if (stage === '无效') return 'error' as const;
  return 'default' as const;
}

function kindType(kind: string) {
  if (kind === '转人工') return 'error' as const;
  if (kind === '成交') return 'success' as const;
  if (kind === '自动回复') return 'success' as const;
  if (kind === '备注') return 'info' as const;
  return 'default' as const;
}

/* ---------- 客户详情抽屉 ---------- */
const showDetail = ref(false);
const detail = ref<WxCustomer | null>(null);
const followups = ref<WxFollowup[]>([]);
const noteText = ref('');
const noteSaving = ref(false);

async function loadDetail(nick: string) {
  const { data, error } = await fetchWxCustomerDetail(nick);

  if (error) {
    message.error('加载详情失败');

    return;
  }

  detail.value = data.Customer;
  followups.value = data.Followups ?? [];
}

function openDetail(row: WxCustomer) {
  showDetail.value = true;
  detail.value = row;
  followups.value = [];
  loadDetail(row.NickName);
}

async function submitNote() {
  const content = noteText.value.trim();

  if (!content || !detail.value) return;

  noteSaving.value = true;

  const { error } = await addWxFollowup({
    NickName: detail.value.NickName,
    ChatId: detail.value.ChatId,
    Kind: '备注',
    Content: content,
    Operator: 'admin'
  });

  noteSaving.value = false;

  if (error) return;

  noteText.value = '';
  message.success('已记录');
  loadDetail(detail.value.NickName);
}

function switchTab(tab: 'handoff' | 'customer' | 'addwx') {
  activeTab.value = tab;

  if (tab === 'handoff') load();
  else if (tab === 'customer') loadCustomers();
  else loadWxAdd();
}

/* ---------- 自动加好友 ---------- */
const wxAddConfig = ref<WxAddConfig | null>(null);
const wxAddQueue = ref<WxAddQueueRow[]>([]);
const wxAddStats = ref<WxAddStats | null>(null);
const wxAddSaving = ref(false);
const wxAddRunning = ref(false);

async function loadWxAdd() {
  const [{ data: cfg }, { data: queue }] = await Promise.all([fetchWxAddConfig(), fetchWxAddQueue()]);

  if (cfg) {
    wxAddConfig.value = cfg.config;
    wxAddStats.value = cfg.stats;
  }

  if (queue) {
    wxAddQueue.value = queue.list ?? [];
    wxAddStats.value = queue.stats ?? wxAddStats.value;
  }
}

async function saveWxAdd() {
  if (!wxAddConfig.value) return;

  wxAddSaving.value = true;
  const { error } = await saveWxAddConfig(wxAddConfig.value);
  wxAddSaving.value = false;

  if (error) return;

  window.$message?.success('已保存');
  loadWxAdd();
}

async function handleRunWxAdd() {
  wxAddRunning.value = true;
  const { data, error } = await runWxAddOnce();
  wxAddRunning.value = false;

  if (error) {
    window.$message?.error(error.message || '执行失败');

    return;
  }

  window.$message?.info(data?.note || '已放行');
  setTimeout(loadWxAdd, 5000);
}

onMounted(() => {
  load();
  loadCustomers();
  loadFunnel();
});
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="微信接待" desc="人工接管工单、客户跟进价值">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>

    <div class="seg-tabs">
      <button :class="{ 'is-active': activeTab === 'handoff' }" @click="switchTab('handoff')">待接管</button>
      <button :class="{ 'is-active': activeTab === 'customer' }" @click="switchTab('customer')">客户跟进</button>
      <button :class="{ 'is-active': activeTab === 'addwx' }" @click="switchTab('addwx')">自动加好友</button>
    </div>
    <NCard v-if="activeTab === 'addwx'" :bordered="false" size="small" class="flex-1">
      <div v-if="wxAddConfig" class="mb-14px flex flex-wrap items-end gap-12px border-b border-gray-100 pb-14px dark:border-gray-800">
        <div>
          <div class="mb-6px text-12px">开关</div>
          <NSwitch v-model:value="wxAddConfig.Enabled" :checked-value="1" :unchecked-value="0" />
        </div>
        <div>
          <div class="mb-6px text-12px">每日上限</div>
          <NInputNumber v-model:value="wxAddConfig.DailyLimit" :min="1" :max="50" class="w-100px" />
        </div>
        <div>
          <div class="mb-6px text-12px">间隔（分钟）</div>
          <div class="flex items-center gap-6px">
            <NInputNumber v-model:value="wxAddConfig.IntervalMin" :min="1" class="w-90px" />
            <span class="op-40">-</span>
            <NInputNumber v-model:value="wxAddConfig.IntervalMax" :min="1" class="w-90px" />
          </div>
        </div>
        <div>
          <div class="mb-6px text-12px">风控暂停（分钟）</div>
          <NInputNumber v-model:value="wxAddConfig.PauseMinutes" :min="10" :max="480" class="w-110px" />
        </div>
        <div>
          <div class="mb-6px text-12px">执行时段</div>
          <div class="flex items-center gap-6px">
            <NInput v-model:value="wxAddConfig.WindowStart" class="w-80px" />
            <span class="op-40">-</span>
            <NInput v-model:value="wxAddConfig.WindowEnd" class="w-80px" />
          </div>
        </div>
        <div class="flex-1">
          <div class="mb-6px text-12px">默认验证语</div>
          <NInput v-model:value="wxAddConfig.Remark" />
        </div>
        <NButton type="primary" :loading="wxAddSaving" @click="saveWxAdd">保存</NButton>
        <NButton :loading="wxAddRunning" @click="handleRunWxAdd">立即执行一次</NButton>
      </div>

      <div v-if="wxAddStats" class="mb-12px flex gap-20px text-12px">
        <span>今日：<b>{{ wxAddStats.todayTotal }}</b> / {{ wxAddStats.dailyLimit }}</span>
        <span class="text-green-600">成功 <b>{{ wxAddStats.todayOk }}</b></span>
        <span class="text-red-500">失败 <b>{{ wxAddStats.todayFail }}</b></span>
        <span>排队 <b>{{ wxAddStats.pending }}</b></span>
        <span v-if="wxAddConfig?.PausedUntil" class="text-orange-500">风控暂停至 {{ wxAddConfig.PausedUntil }}</span>
      </div>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>客户</th>
            <th>号码</th>
            <th>验证语</th>
            <th style="width: 90px" class="center">状态</th>
            <th>原因</th>
            <th style="width: 140px" class="num">执行时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in wxAddQueue" :key="row.Id">
            <td class="font-medium">{{ row.ContactName || '-' }}</td>
            <td>{{ row.Contact }}</td>
            <td class="truncate-cell text-12px op-60">{{ row.Remark || '-' }}</td>
            <td class="center">
              <NTag size="small" :type="row.Status === 1 ? 'success' : row.Status === 0 ? 'info' : 'error'">
                {{ row.Status === 0 ? '待执行' : row.Status === 1 ? '已发送' : row.Status === 3 ? '号码无效' : row.Status === 4 ? '风控' : '失败' }}
              </NTag>
            </td>
            <td class="truncate-cell text-12px op-60">{{ row.Reason || '-' }}</td>
            <td class="num text-12px op-50">{{ row.ExecutedAt || '-' }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="!wxAddQueue.length" class="py-30px text-center text-12px op-40">
        队列为空：到线索池点「加好友」，或在批量里加入队列
      </div>
    </NCard>

    <NCard v-if="activeTab === 'handoff'" :bordered="false" size="small" class="flex-1">
      <template #header>
        <div class="flex items-center gap-12px">
          <span>转人工 / 待跟进</span>
          <NSelect
            v-model:value="status"
            :options="statusOptions"
            class="w-120px"
            size="small"
            @update:value="page = 1; load()"
          />
        </div>
      </template>
      <template #header-extra>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>客户</th>
            <th class="center">渠道</th>
            <th class="center">类别</th>
            <th>联系方式</th>
            <th class="center">紧急</th>
            <th>客户最后一句</th>
            <th class="center">微信跟进</th>
            <th>时间</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in list" :key="row.Id" >
            <td class="font-medium">{{ row.NickName || '-' }}</td>
            <td class="center">
              <NTag size="small" :type="row.Channel === 'douyin' ? 'warning' : 'success'">
                {{ row.Channel === 'douyin' ? '抖音' : '微信' }}
              </NTag>
            </td>
            <td class="center">
              <NTag size="small" type="info">{{ row.Category || '-' }}</NTag>
            </td>
            <td class="text-12px text-primary">{{ row.Contact || '-' }}</td>
            <td class="center">
              <NTag size="small" :type="urgencyTag(row.Urgency).type">{{ urgencyTag(row.Urgency).text }}</NTag>
            </td>
            <td class="truncate-cell" :title="row.LastMsg">{{ row.LastMsg || '-' }}</td>
            <td class="center">
              <template v-if="row.Category === '转微信'">
                <NTag v-if="row.WechatAdded" size="small" type="success">{{ wechatLabel(row.WechatAdded) }}</NTag>
                <NButton v-else text type="primary" size="tiny" @click="handleWechatAdded(row, 1)">已加</NButton>
                <NButton
                  v-if="row.WechatAdded === 1"
                  text
                  type="primary"
                  size="tiny"
                  class="ml-6px"
                  @click="handleWechatAdded(row, 2)"
                >
                  已通过
                </NButton>
              </template>
              <span v-else class="op-40">-</span>
            </td>
            <td class="text-12px op-50">{{ formatTime(row.CreatedAt) }}</td>
            <td class="num">
              <NButton v-if="row.Status === 0" text type="primary" size="tiny" class="mr-6px" @click="handleClaim(row)">
                接管
              </NButton>
              <NButton v-if="row.Status < 2" text size="tiny" class="mr-6px" @click="handleResolve(row, 2)">完成</NButton>
              <NButton v-if="row.Status < 2" text size="tiny" @click="handleResolve(row, 3)">忽略</NButton>
              <span v-if="row.Status === 1" class="text-11px op-50">{{ row.ClaimUser }} 接管中</span>
              <NButton v-if="row.AiPaused" text type="warning" size="tiny" class="ml-6px" @click="handleResumeAi(row)">
                恢复AI
              </NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!list.length" class="py-40px text-center text-12px op-50">
        暂无待接管（AI 遇到打款 / 合同 / 看现场 / 对方留号码会自动建单）
      </div>

      <div class="mt-12px flex items-center justify-between">
        <span class="text-12px op-50">共 {{ total }} 条</span>
        <div class="flex items-center gap-6px">
          <NButton size="tiny" :disabled="page <= 1" @click="page--; load()">上一页</NButton>
          <span class="text-12px">{{ page }}</span>
          <NButton size="tiny" :disabled="page * 20 >= total" @click="page++; load()">下一页</NButton>
        </div>
      </div>
    </NCard>

    <NCard v-if="activeTab === 'customer'" :bordered="false" size="small" class="flex-1">
      <template #header>
        <div class="flex items-center gap-12px">
          <span>客户跟进价值</span>
          <NSelect
            v-model:value="minScore"
            :options="[
              { label: '全部', value: 0 },
              { label: 'A 类（70+）', value: 70 },
              { label: 'B 类（50+）', value: 50 }
            ]"
            class="w-140px"
            size="small"
            @update:value="loadCustomers()"
          />
        </div>
      </template>
      <template #header-extra>
        <NButton size="small" @click="loadCustomers">刷新</NButton>
      </template>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>客户</th>
            <th class="center">分层</th>
            <th class="center">阶段</th>
            <th class="center">分数</th>
            <th>模型判断</th>
            <th>下一步</th>
            <th class="center">沉默</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in customers"
            :key="row.NickName"
            class="cursor-pointer border-b border-gray-100 hover:bg-gray-50 dark:border-gray-800 dark:hover:bg-gray-800"
            @click="openDetail(row)"
          >
            <td class="font-medium">
              <div>{{ row.NickName || '-' }}</div>
              <div class="truncate-cell text-11px op-50" :title="row.Reason">{{ row.Reason }}</div>
            </td>
            <td class="center">
              <NTag size="small" :type="tierType(row.Tier)">{{ row.Tier }} 类</NTag>
            </td>
            <td class="center">
              <NTag size="small" :type="stageType(row.Stage)">{{ row.Stage || '-' }}</NTag>
            </td>
            <td class="center">
              <div class="font-semibold">{{ row.Score }}</div>
              <div v-if="row.LlmScore >= 0" class="text-10px op-50">模型 {{ row.LlmScore }}</div>
            </td>
            <td class="truncate-cell text-11px op-70" :title="row.LlmReason">{{ row.LlmReason || '-' }}</td>
            <td class="truncate-cell text-11px" :title="row.NextAction">{{ row.NextAction || '-' }}</td>
            <td class="center">{{ row.SilentDays }} 天</td>
            <td class="py-10px text-right" @click.stop>
              <NButton text type="primary" size="tiny" class="mr-6px" @click="handleFollow(row, 1)">跟进中</NButton>
              <NButton text size="tiny" class="mr-6px" @click="handleFollow(row, 2)">已成交</NButton>
              <NButton text size="tiny" @click="handleFollow(row, 3)">无效</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!customers.length" class="py-40px text-center text-12px op-50">
        暂无客户数据（监听器跑一轮后会自动同步）
      </div>
    </NCard>

    <NCard v-if="activeTab === 'funnel'" :bordered="false" size="small" class="flex-1">
      <template #header>
        <span>转化漏斗</span>
      </template>
      <template #header-extra>
        <NButton size="small" @click="loadFunnel">刷新</NButton>
      </template>

      <template v-if="funnel">
        <div class="mb-20px grid grid-cols-4 gap-12px">
          <div class="rounded bg-gray-50 p-14px dark:bg-gray-800">
            <div class="text-11px op-50">客户总数</div>
            <div class="mt-4px text-22px font-semibold">{{ funnel.total }}</div>
          </div>
          <div class="rounded bg-gray-50 p-14px dark:bg-gray-800">
            <div class="text-11px op-50">成交</div>
            <div class="mt-4px text-22px font-semibold text-green-600">{{ funnel.dealCount }}</div>
          </div>
          <div class="rounded bg-gray-50 p-14px dark:bg-gray-800">
            <div class="text-11px op-50">成交率</div>
            <div class="mt-4px text-22px font-semibold">{{ funnel.dealRate.toFixed(1) }}%</div>
          </div>
          <div class="rounded bg-gray-50 p-14px dark:bg-gray-800">
            <div class="text-11px op-50">平均分 / 工单</div>
            <div class="mt-4px text-22px font-semibold">
              {{ funnel.avgScore }} <span class="text-14px op-50">/ {{ funnel.handoffTotal }}</span>
            </div>
          </div>
        </div>

        <div class="mb-20px">
          <div class="mb-10px text-13px font-medium">阶段转化</div>
          <div v-for="(stage, index) in funnel.stages" :key="stage.name" class="mb-10px">
            <div class="mb-4px flex items-center justify-between text-12px">
              <span>{{ index + 1 }}. {{ stage.name }}</span>
              <span class="op-60">{{ stage.count }} 人 · {{ stage.rate.toFixed(0) }}%</span>
            </div>
            <div class="h-22px w-full overflow-hidden rounded bg-gray-100 dark:bg-gray-800">
              <div
                class="flex h-full items-center justify-end rounded bg-blue-500 pr-8px text-11px text-white transition-all"
                :style="{ width: `${Math.max(stage.rate, 2)}%` }"
              />
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-20px">
          <div>
            <div class="mb-10px text-13px font-medium">工单类别</div>
            <div v-if="!(funnel.categories || []).length" class="text-12px op-50">暂无工单</div>
            <div v-for="item in funnel.categories" :key="item.name" class="mb-8px flex items-center gap-8px">
              <NTag size="small" type="info">{{ item.name }}</NTag>
              <div class="h-8px flex-1 rounded bg-gray-100 dark:bg-gray-800">
                <div
                  class="h-full rounded bg-orange-400"
                  :style="{
                    width: `${(item.count / Math.max(...(funnel.categories || []).map(c => c.count), 1)) * 100}%`
                  }"
                />
              </div>
              <span class="w-24px text-right text-12px">{{ item.count }}</span>
            </div>
          </div>

          <div>
            <div class="mb-10px text-13px font-medium">最近 7 天</div>
            <table class="w-full text-12px">
              <thead>
                <tr class="border-b border-gray-200 text-11px op-50 dark:border-gray-700">
                  <th class="py-6px text-left font-normal">日期</th>
                  <th class="py-6px text-center font-normal">新客户</th>
                  <th class="py-6px text-center font-normal">新工单</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="day in funnel.trend" :key="day.date" >
                  <td class="py-6px">{{ day.date }}</td>
                  <td class="py-6px text-center">{{ day.customers }}</td>
                  <td class="py-6px text-center">{{ day.handoffs }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>

      <div v-else class="py-40px text-center text-12px op-50">加载中…</div>
    </NCard>

    <NDrawer v-model:show="showDetail" :width="520" placement="right">
      <NDrawerContent :title="detail?.NickName || '客户详情'" closable>
        <template v-if="detail">
          <div class="mb-16px flex items-center gap-8px">
            <NTag :type="tierType(detail.Tier)" size="small">{{ detail.Tier }} 类</NTag>
            <NTag :type="stageType(detail.Stage)" size="small">{{ detail.Stage || '未评估' }}</NTag>
            <span class="text-12px op-60">总分 {{ detail.Score }}</span>
            <span v-if="detail.LlmScore >= 0" class="text-12px op-60">/ 模型 {{ detail.LlmScore }}</span>
          </div>

          <div class="mb-16px rounded bg-gray-50 p-12px text-12px leading-relaxed dark:bg-gray-800">
            <div class="mb-6px"><span class="op-50">模型判断：</span>{{ detail.LlmReason || '-' }}</div>
            <div><span class="op-50">下一步：</span>{{ detail.NextAction || '-' }}</div>
          </div>

          <div class="mb-8px text-13px font-medium">跟进记录（{{ followups.length }}）</div>

          <div class="mb-16px flex gap-8px">
            <NInput v-model:value="noteText" placeholder="写一条跟进备注…" size="small" @keyup.enter="submitNote" />
            <NButton size="small" type="primary" :loading="noteSaving" @click="submitNote">记录</NButton>
          </div>

          <div v-if="!followups.length" class="py-20px text-center text-12px op-50">还没有跟进记录</div>

          <div v-for="item in followups" :key="item.Id" class="mb-10px border-l-2 border-gray-200 pl-10px dark:border-gray-700">
            <div class="mb-2px flex items-center gap-6px">
              <NTag size="tiny" :type="kindType(item.Kind)">{{ item.Kind }}</NTag>
              <span class="text-11px op-50">{{ item.Operator }}</span>
              <span class="text-11px op-40">{{ formatTime(item.CreatedAt) }}</span>
            </div>
            <div class="text-12px leading-relaxed">{{ item.Content }}</div>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>