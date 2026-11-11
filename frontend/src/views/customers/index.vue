<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import PageHeader from '@/components/business/PageHeader.vue';
import TableSkeleton from '@/components/business/TableSkeleton.vue';
import ErrorState from '@/components/business/ErrorState.vue';
import EmptyState from '@/components/business/EmptyState.vue';
import MediaContent from '@/components/business/MediaContent.vue';
import { NButton, NCard, NDatePicker, NDrawer, NDrawerContent, NInput, NPopselect, NSelect, NTag, useMessage } from 'naive-ui';
import {
  addWxFollowup,
  fetchFollowupReminders,
  fetchWxCustomerDetail,
  fetchWxCustomerList,
  updateWxCustomer,
  type WxCustomer,
  type WxFollowup
} from '@/service/api/wechat';
import { deleteTableView, fetchTableViewList, saveTableView } from '@/service/api/lead';
import { loadColumnPrefs as loadColumnPrefsRemote, saveColumnPrefs as saveColumnPrefsRemote } from '@/utils/table-prefs';

defineOptions({ name: 'Customers' });

const message = useMessage();
const route = useRoute();
const router = useRouter();

/** 筛选条件同步到 URL，方便分享和刷新后保留 */
function syncQuery() {
  const query: Record<string, string> = {};

  if (tier.value) query.tier = tier.value;
  if (stage.value) query.stage = stage.value;
  if (side.value) query.side = side.value;
  if (keyword.value) query.q = keyword.value;
  if (page.value > 1) query.page = String(page.value);
  if (sortState.value.key !== 'SilentDays') query.sort = sortState.value.key;
  if (sortState.value.dir !== 'desc') query.order = sortState.value.dir;

  router.replace({ query });
}

function restoreQuery() {
  const q = route.query;

  tier.value = (q.tier as string) || null;
  stage.value = (q.stage as string) || null;
  side.value = (q.side as string) || null;
  keyword.value = (q.q as string) || '';
  page.value = Number(q.page || 1) || 1;

  const sortKey = (q.sort as string) || 'SilentDays';
  const sortDir = (q.order as string) === 'asc' ? 'asc' : 'desc';

  sortState.value = { key: sortKey, dir: sortDir };
}

const list = ref<WxCustomer[]>([]);
const queue = ref<{ Id: number; NickName: string; Tier: string; Stage: string; Score: number; NextAction: string; Reason: string; NextFollow: string }[]>([]);
const total = ref(0);
const loading = ref(false);
const loadError = ref('');

/* ---------- 列设置（显隐持久化） ---------- */
const COLUMN_KEY = 'table:customers:v1';

const ALL_COLUMNS = [
  { key: 'Side', label: '方向' },
  { key: 'Tier', label: '等级' },
  { key: 'Stage', label: '阶段' },
  { key: 'Score', label: '评分' },
  { key: 'NextAction', label: '下一步动作' },
  { key: 'NextFollowAt', label: '计划跟进' },
  { key: 'LastMsgAt', label: '最近消息' },
  { key: 'SilentDays', label: '沉默' }
];

const visibleColumns = ref<string[]>(ALL_COLUMNS.map(c => c.key));

async function loadColumnPrefs() {
  const columns = await loadColumnPrefsRemote('customers', COLUMN_KEY);

  if (columns?.length) visibleColumns.value = columns;
}

function saveColumnPrefs(next: string[]) {
  visibleColumns.value = next;
  saveColumnPrefsRemote('customers', COLUMN_KEY, next);
}
function isVisible(key: string) {
  return visibleColumns.value.includes(key);
}

/* ---------- 排序（点表头切换，持久化） ---------- */
const SORT_KEY = 'sort:customers:v1';

const sortState = ref<{ key: string; dir: 'asc' | 'desc' }>({ key: 'SilentDays', dir: 'desc' });

const SORTABLE: Record<string, 'num' | 'text'> = {
  NickName: 'text',
  Score: 'num',
  NextFollowAt: 'num',
  LastMsgAt: 'num',
  SilentDays: 'num'
};

function loadSort() {
  try {
    const raw = localStorage.getItem(SORT_KEY);
    if (raw) sortState.value = JSON.parse(raw);
  } catch {
    /* 忽略 */
  }
}

function toggleSort(key: string) {
  if (!SORTABLE[key]) return;

  sortState.value =
    sortState.value.key === key
      ? { key, dir: sortState.value.dir === "asc" ? "desc" : "asc" }
      : { key, dir: SORTABLE[key] === "num" ? "desc" : "asc" };

  try {
    localStorage.setItem(SORT_KEY, JSON.stringify(sortState.value));
  } catch {
    /* 忽略 */
  }

  // 服务端排序：重取第一页
  page.value = 1;
  load();
}
function sortMark(key: string) {
  if (sortState.value.key !== key) return "";

  return sortState.value.dir === 'asc' ? ' ↑' : ' ↓';
}

const sortedList = computed(() => {
  const rows = [...list.value];
  const { key, dir } = sortState.value;
  const factor = dir === 'asc' ? 1 : -1;

  rows.sort((a: any, b: any) => {
    const va = a?.[key];
    const vb = b?.[key];

    if (va === vb) return 0;
    if (va === null || va === undefined || va === '') return 1;
    if (vb === null || vb === undefined || vb === '') return -1;

    return SORTABLE[key] === 'num'
      ? (Number(va) - Number(vb)) * factor
      : String(va).localeCompare(String(vb)) * factor;
  });

  return rows;
});

/* ---------- 保存视图（筛选 + 列 + 排序） ---------- */
const VIEWS_KEY = 'views:customers:v1';

interface SavedView {
  name: string;
  tier: string | null;
  stage: string | null;
  side: string | null;
  keyword: string;
  columns: string[];
  sort: { key: string; dir: 'asc' | 'desc' };
}

const views = ref<SavedView[]>([]);
const activeView = ref('');

async function loadViews() {
  // 本地兜底
  try {
    const raw = localStorage.getItem(VIEWS_KEY);
    if (raw) views.value = JSON.parse(raw);
  } catch {
    /* 忽略 */
  }

  const { data } = await fetchTableViewList('customers');

  if (data?.length) {
    views.value = data
      .map(row => {
        try {
          return JSON.parse(row.Payload) as SavedView;
        } catch {
          return null;
        }
      })
      .filter(Boolean) as SavedView[];
  }
}

function persistViews(next: SavedView[]) {
  views.value = next;

  try {
    localStorage.setItem(VIEWS_KEY, JSON.stringify(next));
  } catch {
    /* 忽略 */
  }
}

function saveCurrentView() {
  const name = window.prompt('给这个视图起个名字', activeView.value || '我的视图');

  if (!name) return;

  // "__" 开头是系统保留名（列偏好占用），避免冲突
  if (name.trim().startsWith('__')) {
    window.$message?.error('视图名称不能以 __ 开头');

    return;
  }

  const view: SavedView = {
    name,
    tier: tier.value,
    stage: stage.value,
    side: side.value,
    keyword: keyword.value,
    columns: [...visibleColumns.value],
    sort: { ...sortState.value }
  };

  persistViews([...views.value.filter(v => v.name !== name), view]);
  activeView.value = name;

  // 同步到服务端（跨设备），失败不影响本地
  saveTableView({ tableKey: 'customers', name, payload: JSON.stringify(view) }).catch(() => {});
}

async function removeActiveView() {
  if (!activeView.value) return;

  const name = activeView.value;

  persistViews(views.value.filter(v => v.name !== name));
  activeView.value = '';

  try {
    await deleteTableView('customers', name);
  } catch {
    /* 忽略 */
  }
}

function applyView(name: string) {
  const view = views.value.find(v => v.name === name);

  if (!view) return;

  activeView.value = name;
  tier.value = view.tier;
  stage.value = view.stage;
  side.value = view.side;
  keyword.value = view.keyword;
  sortState.value = { ...view.sort };
  page.value = 1;

  saveColumnPrefs([...view.columns]);
  load();
}

const page = ref(1);
const pageSize = 20;

const tier = ref<string | null>(null);
const stage = ref<string | null>(null);
const side = ref<string | null>(null);
const keyword = ref('');

const tierOptions = ['S', 'A', 'B', 'C', 'D'].map(v => ({ label: `${v} 类`, value: v }));
const stageOptions = ['了解', '比价', '议价', '待成交', '已成交', '无效'].map(v => ({ label: v, value: v }));

/* 详情抽屉 */
const showDetail = ref(false);
const detail = ref<{ customer: WxCustomer; followups: WxFollowup[] } | null>(null);
const detailLoading = ref(false);

const editStage = ref<string | null>(null);
const editNextAt = ref<number | null>(null);
const editNextAction = ref('');
const saving = ref(false);

const followKind = ref('follow');
const followContent = ref('');
const followSaving = ref(false);

async function loadQueue() {
  const { data } = await fetchFollowupReminders();

  if (!data) return;

  const merged = [
    ...(data.Overdue || []).map(r => ({ ...r, Reason: r.Reason || '逾期未跟进' })),
    ...(data.Today || []),
    ...(data.Silent || [])
  ];

  queue.value = merged.slice(0, 8);
}

async function load() {
  loading.value = true;
  loadError.value = '';

  const { data, error } = await fetchWxCustomerList({
    page: page.value,
    pageSize,
    tier: tier.value || undefined,
    stage: stage.value || undefined,
    side: side.value || undefined,
    keyword: keyword.value || undefined,
    sort: sortState.value.key,
    order: sortState.value.dir === 'asc' ? 'ascending' : 'descending'
  });
  loading.value = false;
  syncQuery();


  if (error) {
    loadError.value = (error as any)?.message || '请求失败，请稍后重试';

    return;
  }
  if (data) {
    list.value = (data as any).records ?? [];
    total.value = data.total ?? 0;
  }
}

function resetFilters() {
  tier.value = null;
  stage.value = null;
  keyword.value = '';
  page.value = 1;
  load();
}

function changePage(next: number) {
  page.value = next;
  load();
}

async function openDetail(row: WxCustomer) {
  showDetail.value = true;
  detailLoading.value = true;
  const { data } = await fetchWxCustomerDetail(row.NickName);
  detailLoading.value = false;

  if (data) {
    const raw = data as any;

    detail.value = { customer: raw.Customer ?? raw.customer, followups: raw.Followups ?? raw.followups ?? [] };
    editStage.value = detail.value.customer?.Stage || null;
    editNextAt.value = detail.value.customer?.NextFollowAt ? new Date(detail.value.customer.NextFollowAt).getTime() : null;
    editNextAction.value = detail.value.customer?.NextAction || '';
  }
}

async function saveDetail() {
  if (!detail.value?.customer) return;

  saving.value = true;
  const { error } = await updateWxCustomer({
    Id: detail.value.customer.Id,
    Stage: editStage.value || '',
    NextFollowAt: editNextAt.value ? formatDateTime(editNextAt.value) : '',
    NextAction: editNextAction.value
  });
  saving.value = false;

  if (error) return;

  message.success('已保存');
  load();
  openDetail(detail.value.customer);
}

async function submitFollowup() {
  if (!detail.value?.customer || !followContent.value.trim()) {
    message.warning('先写点内容');

    return;
  }

  followSaving.value = true;
  const { error } = await addWxFollowup({
    NickName: detail.value.customer.NickName,
    ChatId: detail.value.customer.ChatId || '',
    Kind: followKind.value,
    Content: followContent.value,
    Operator: 'admin'
  });
  followSaving.value = false;

  if (error) return;

  followContent.value = '';
  message.success('已记录');
  openDetail(detail.value.customer);
}

function formatDateTime(ts: number) {
  const d = new Date(ts);
  const pad = (n: number) => String(n).padStart(2, '0');

  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:00`;
}

/** ISO 时间 → 2026-11-10 12:21 */
function shortTime(value?: string | null) {
  if (!value) return '-';

  return String(value).replace('T', ' ').replace(/\.\d+/, '').replace(/[+-]\d{2}:\d{2}$/, '').slice(0, 16);
}

function tierType(value?: string) {
  if (value === 'S' || value === 'A') return 'success' as const;
  if (value === 'B') return 'warning' as const;

  return 'default' as const;
}

onMounted(() => {
  loadColumnPrefs();
  loadViews();
  restoreQuery();
  load();
  loadQueue();
});
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="客户" desc="已进入跟进池的客户，按等级、阶段筛选，点行看时间线">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>



    <div class="filter-bar">
      <NInput v-model:value="keyword" placeholder="搜索昵称" clearable class="w-180px" @keyup.enter="page = 1; load()" />
      <NSelect v-model:value="tier" :options="tierOptions" placeholder="全部等级" clearable class="w-120px" @update:value="page = 1; load()" />
      <NSelect v-model:value="stage" :options="stageOptions" placeholder="全部阶段" clearable class="w-120px" @update:value="page = 1; load()" />
      <NSelect
        v-model:value="side"
        :options="[
          { label: '我方客户', value: 'buyer' },
          { label: '供我方', value: 'supplier' }
        ]"
        placeholder="全部方向"
        clearable
        class="w-120px"
        @update:value="page = 1; load()"
      />
      <NButton size="small" @click="page = 1; load()">筛选</NButton>
      <NButton size="small" quaternary @click="resetFilters">重置</NButton>
      <div class="ml-auto flex items-center gap-8px">
        <span class="text-12px op-50">共 {{ total }} 位客户</span>
        <NPopselect
          :options="views.map(v => ({ label: v.name, value: v.name }))"
          :value="activeView || null"
          placeholder="视图"
          @update:value="applyView"
        >
          <NButton size="tiny" quaternary>{{ activeView || '视图' }}</NButton>
        </NPopselect>
        <NButton size="tiny" quaternary @click="saveCurrentView">存为视图</NButton>
        <NPopselect
          :value="visibleColumns"
          multiple
          :options="ALL_COLUMNS.map(c => ({ label: c.label, value: c.key }))"
          @update:value="saveColumnPrefs"
        >
          <NButton size="tiny" quaternary>列设置</NButton>
        </NPopselect>
      </div>
    </div>

    <div class="customer-layout">
    <NCard :bordered="false" size="small">
            <ErrorState
        v-if="loadError"
        title="客户列表加载失败"
        :desc="loadError"
        :loading="loading"
        @retry="load"
      />

      <TableSkeleton v-else-if="loading && !list.length" :rows="6" :columns="6" />

      <table v-else class="data-table mobile-cards sticky-header">
        <thead>
          <tr>
            <th style="width: 150px">客户</th>
            <th v-if="isVisible('Side')" style="width: 92px" class="center">方向</th>
            <th v-if="isVisible('Tier')" style="width: 70px" class="center">等级</th>
            <th v-if="isVisible('Stage')" style="width: 90px" class="center">阶段</th>
            <th v-if="isVisible('Score')" style="width: 60px" class="num sortable" @click="toggleSort('Score')">评分{{ sortMark('Score') }}</th>
            <th v-if="isVisible('NextAction')">下一步动作</th>
            <th v-if="isVisible('NextFollowAt')" style="width: 140px" class="num sortable" @click="toggleSort('NextFollowAt')">计划跟进{{ sortMark('NextFollowAt') }}</th>
            <th v-if="isVisible('LastMsgAt')" style="width: 140px" class="num sortable" @click="toggleSort('LastMsgAt')">最近消息{{ sortMark('LastMsgAt') }}</th>
            <th v-if="isVisible('SilentDays')" style="width: 80px" class="num sortable" @click="toggleSort('SilentDays')">沉默{{ sortMark('SilentDays') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in sortedList" :key="row.Id" class="cursor-pointer" @click="openDetail(row)">
            <td class="font-medium" data-label="客户">{{ row.NickName }}</td>
            <td v-if="isVisible('Side')" class="center" data-label="方向">
              <NTag size="small" :type="row.Side === 'supplier' ? 'warning' : 'info'">
                {{ row.Side === 'supplier' ? '供我方' : '我方客户' }}
              </NTag>
            </td>
            <td v-if="isVisible('Tier')" class="center" data-label="等级">
              <NTag size="small" :type="tierType(row.Tier)">{{ row.Tier || '-' }}</NTag>
            </td>
            <td v-if="isVisible('Stage')" class="center text-12px" data-label="阶段">{{ row.Stage || '-' }}</td>
            <td v-if="isVisible('Score')" class="num" data-label="评分">{{ row.Score }}</td>
            <td v-if="isVisible('NextAction')" class="truncate-cell text-12px op-70" data-label="下一步">{{ row.NextAction || '-' }}</td>
            <td v-if="isVisible('NextFollowAt')" class="num text-12px op-50" data-label="计划跟进">{{ shortTime(row.NextFollowAt) }}</td>
            <td v-if="isVisible('LastMsgAt')" class="num text-12px op-50" data-label="最近消息">{{ shortTime(row.LastMsgAt) }}</td>
            <td v-if="isVisible('SilentDays')" class="num text-12px" data-label="沉默" :class="(row.SilentDays ?? 0) >= 7 ? 'text-orange-500' : 'op-50'">
              {{ row.SilentDays ?? 0 }}
            </td>
          </tr>
        </tbody>
      </table>

      <EmptyState v-if="!loadError && !loading && !list.length" title="没有符合条件的客户" desc="换个筛选条件，或先把线索转成客户" />

      <div class="table-footer">
        <span>共 {{ total }} 条</span>
        <div class="flex items-center gap-8px">
          <NButton size="tiny" :disabled="page <= 1" @click="changePage(page - 1)">上一页</NButton>
          <span>第 {{ page }} 页</span>
          <NButton size="tiny" :disabled="page * pageSize >= total" @click="changePage(page + 1)">下一页</NButton>
        </div>
      </div>
    </NCard>

    <NCard title="跟进队列" :bordered="false" size="small">
      <div class="flex flex-col gap-10px">
        <div
          v-for="item in queue"
          :key="item.Id"
          class="cursor-pointer rounded-lg border border-gray-100 p-10px transition-colors hover:border-blue-300 dark:border-gray-800"
          @click="openDetail({ ...item, NickName: item.NickName } as any)"
        >
          <div class="flex items-center justify-between">
            <span class="text-13px font-medium">{{ item.NickName }}</span>
            <NTag size="tiny" :type="item.Tier === 'A' ? 'success' : 'default'">{{ item.Tier || '-' }}</NTag>
          </div>
          <div class="mt-2px text-11px op-50">{{ item.Reason || item.Stage }}</div>
          <div class="mt-4px truncate text-12px op-70">{{ item.NextAction || '-' }}</div>
        </div>
        <EmptyState v-if="!queue.length" title="今天没有要跟的" desc="有逾期或沉默客户时会出现在这里" />
      </div>
    </NCard>
    </div>

    <NDrawer v-model:show="showDetail" :width="520">
      <NDrawerContent :title="detail?.customer?.NickName || '客户详情'" closable>
        <div v-if="detailLoading" class="py-40px text-center text-12px op-40">加载中…</div>

        <template v-else-if="detail">
          <div class="drawer-facts">
            <div class="fact">
              <div class="fact-label">方向</div>
              <div class="fact-value">
                <NTag size="small" :type="detail.customer.Side === 'supplier' ? 'warning' : 'info'">
                  {{ detail.customer.Side === 'supplier' ? '供我方' : '我方客户' }}
                </NTag>
              </div>
            </div>
            <div class="fact">
              <div class="fact-label">等级 / 评分</div>
              <div class="fact-value">
                <NTag size="small" :type="detail.customer.Tier === 'A' ? 'success' : detail.customer.Tier === 'B' ? 'warning' : 'default'">
                  {{ detail.customer.Tier || '-' }}
                </NTag>
                <span class="ml-6px">{{ detail.customer.Score }}</span>
              </div>
            </div>
            <div class="fact">
              <div class="fact-label">阶段</div>
              <div class="fact-value">{{ detail.customer.Stage || '-' }}</div>
            </div>
            <div class="fact">
              <div class="fact-label">沉默天数</div>
              <div class="fact-value" :class="(detail.customer.SilentDays ?? 0) >= 7 ? 'text-orange-500' : ''">
                {{ detail.customer.SilentDays ?? 0 }} 天
              </div>
            </div>
            <div class="fact">
              <div class="fact-label">最近消息</div>
              <div class="fact-value">{{ shortTime(detail.customer.LastMsgAt) }}</div>
            </div>
            <div class="fact">
              <div class="fact-label">状态</div>
              <div class="fact-value">{{ detail.customer.FollowStatus === 1 ? '已成交' : '跟进中' }}</div>
            </div>
          </div>

          <div class="mt-16px border-t border-gray-100 pt-14px dark:border-gray-800">
            <div class="mb-8px text-13px font-medium">跟进计划</div>
            <div class="flex flex-col gap-10px">
              <div class="form-row">
                <div class="form-label">阶段</div>
                <NSelect v-model:value="editStage" :options="stageOptions" placeholder="选择阶段" clearable />
              </div>
              <div class="form-row">
                <div class="form-label">下次跟进</div>
                <NDatePicker v-model:value="editNextAt" type="datetime" placeholder="选择时间" clearable />
              </div>
              <div class="form-row">
                <div class="form-label">下一步动作</div>
                <NInput v-model:value="editNextAction" type="textarea" :rows="2" placeholder="要做什么，例如：报价并约看货" />
              </div>
              <NButton size="small" type="primary" :block="true" :loading="saving" @click="saveDetail">保存计划</NButton>
            </div>
          </div>

          <div class="mt-16px border-t border-gray-100 pt-14px dark:border-gray-800">
            <div class="mb-8px text-13px font-medium">跟进记录</div>
            <div class="flex flex-col gap-8px">
              <NSelect
                v-model:value="followKind"
                :options="[
                  { label: '一般跟进', value: 'follow' },
                  { label: '报价', value: 'quote' },
                  { label: '成交', value: 'deal' }
                ]"
              />
              <NInput v-model:value="followContent" type="textarea" :rows="2" placeholder="记录一次沟通" />
              <NButton size="small" :loading="followSaving" @click="submitFollowup">添加记录</NButton>
            </div>

            <div class="mt-12px flex flex-col gap-10px">
              <div v-for="item in detail.followups || []" :key="item.Id" class="rounded border border-gray-100 p-10px dark:border-gray-800">
                <div class="flex items-center justify-between text-12px op-50">
                  <span>{{ item.Kind || '跟进' }} · {{ item.Operator || 'admin' }}</span>
                  <span>{{ shortTime(item.CreatedAt) }}</span>
                </div>
                <div class="mt-4px">
                  <MediaContent :content="item.Content" :media-url="item.MediaUrl" />
                </div>
              </div>
              <div v-if="!(detail.followups || []).length" class="py-16px text-center text-12px op-40">还没有跟进记录</div>
            </div>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>
<style scoped>
.sortable {
  cursor: pointer;
  user-select: none;
}

.sortable:hover {
  color: var(--ds-text);
}

.drawer-facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px 16px;
  padding: 12px 14px;
  border: 1px solid var(--ds-border);
  border-radius: 10px;
  background: rgba(148, 163, 184, 0.05);
}

.fact-label {
  font-size: 11px;
  color: var(--ds-text-3);
}

.fact-value {
  margin-top: 3px;
  font-size: 13px;
  color: var(--ds-text);
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-label {
  font-size: 12px;
  color: var(--ds-text-3);
}

@media (max-width: 767px) {
  .drawer-facts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>