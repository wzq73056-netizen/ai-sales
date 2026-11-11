<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import PageHeader from '@/components/business/PageHeader.vue';
import { loadColumnPrefs as loadColumnPrefsRemote, saveColumnPrefs as saveColumnPrefsRemote } from '@/utils/table-prefs';
import TableSkeleton from '@/components/business/TableSkeleton.vue';
import EmptyState from '@/components/business/EmptyState.vue';
import ErrorState from '@/components/business/ErrorState.vue';
import { NButton, NCard, NPopselect, NTag } from 'naive-ui';
import { fetchInbox, type Inbox, type InboxItem } from '@/service/api/inbox';
import { fetchTableViewList, saveTableView } from '@/service/api/lead';

defineOptions({ name: 'Inbox' });

const router = useRouter();

const data = ref<Inbox | null>(null);
const loading = ref(false);
const loadError = ref('');
const route = useRoute();

const channel = ref<'' | 'wechat' | 'douyin' | 'xhs'>('');

/** 渠道筛选写入 URL，刷新/分享后保留 */
function syncQuery() {
  const query: Record<string, string> = {};

  if (channel.value) query.channel = channel.value;

  router.replace({ query });
}

function restoreQuery() {
  const c = route.query.channel as string;

  if (c === 'wechat' || c === 'douyin' || c === 'xhs') channel.value = c;
}

const channelTabs: { label: string; value: '' | 'wechat' | 'douyin' | 'xhs' }[] = [
  { label: '全部', value: '' },
  { label: '微信', value: 'wechat' },
  { label: '抖音', value: 'douyin' },
  { label: '小红书', value: 'xhs' }
];

const items = computed(() => data.value?.items ?? []);

/* ---------- 列设置（显隐持久化） ---------- */
const COLUMN_KEY = 'table:inbox:v1';

const ALL_COLUMNS = [
  { key: 'Channel', label: '平台' },
  { key: 'Name', label: '客户' },
  { key: 'Text', label: '内容' },
  { key: 'Kind', label: '类型' },
  { key: 'Time', label: '时间' }
];

const visibleColumns = ref<string[]>(ALL_COLUMNS.map(c => c.key));

async function loadColumnPrefs() {
  const saved = await loadColumnPrefsRemote('inbox', COLUMN_KEY);

  if (saved?.length) visibleColumns.value = saved;
}

function saveColumnPrefs(next: string[]) {
  visibleColumns.value = next;
  saveColumnPrefsRemote('inbox', COLUMN_KEY, next);
}
function isVisible(key: string) {
  return visibleColumns.value.includes(key);
}

/* ---------- 视图：渠道 + 列，本地兜底 + 服务端同步 ---------- */
const VIEWS_KEY = 'views:inbox:v1';

interface InboxView {
  name: string;
  channel: '' | 'wechat' | 'douyin' | 'xhs';
  columns: string[];
}

const views = ref<InboxView[]>([]);
const activeView = ref('');

function persistViews(next: InboxView[]) {
  views.value = next;

  try {
    localStorage.setItem(VIEWS_KEY, JSON.stringify(next));
  } catch {
    /* 忽略 */
  }
}

async function loadViews() {
  try {
    const raw = localStorage.getItem(VIEWS_KEY);

    if (raw) views.value = JSON.parse(raw);
  } catch {
    /* 忽略 */
  }

  try {
    const { data } = await fetchTableViewList('inbox');

    if (data?.length) {
      views.value = data
        .filter(row => row.Name !== '__columns')
        .map(row => {
          try {
            return JSON.parse(row.Payload) as InboxView;
          } catch {
            return null;
          }
        })
        .filter(Boolean) as InboxView[];
    }
  } catch {
    /* 服务端不可用时用本地 */
  }
}

function saveCurrentView() {
  const name = window.prompt('给这个视图起个名字', activeView.value || '我的视图');

  if (!name) return;

  if (name.trim().startsWith('__')) {
    window.$message?.error('视图名称不能以 __ 开头');

    return;
  }

  const view: InboxView = { name, channel: channel.value, columns: [...visibleColumns.value] };

  persistViews([...views.value.filter(v => v.name !== name), view]);
  activeView.value = name;

  saveTableView({ tableKey: 'inbox', name, payload: JSON.stringify(view) }).catch(() => {});
}

function applyView(name: string) {
  const view = views.value.find(v => v.name === name);

  if (!view) return;

  activeView.value = name;
  channel.value = view.channel;
  saveColumnPrefs([...view.columns]);
  syncQuery();
  load();
}

async function load() {
  loading.value = true;
  loadError.value = '';

  const { data: payload, error } = await fetchInbox(channel.value || undefined);

  if (error) {
    loadError.value = (error as any)?.message || '请求失败，请稍后重试';
    loading.value = false;

    return;
  }

  if (payload) data.value = payload;

  loading.value = false;
}

function channelLabel(value: string) {
  if (value === 'wechat') return '微信';
  if (value === 'douyin') return '抖音';

  return '小红书';
}

function channelType(value: string) {
  if (value === 'wechat') return 'success' as const;
  if (value === 'douyin') return 'warning' as const;

  return 'error' as const;
}

function open(row: InboxItem) {
  if (row.Channel === 'wechat' || row.Kind === 'handoff') {
    router.push('/wechat');

    return;
  }

  router.push(row.Channel === 'douyin' ? '/douyin' : '/xhs');
}

onMounted(() => {
  loadColumnPrefs();
  loadViews();
  restoreQuery();
  load();
});
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="收件箱" desc="三个平台的待接管和最近交流，按平台筛选">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>



    <div class="grid grid-cols-5 gap-12px">
      <div class="kpi-card" @click="channel = ''; load()">
        <div class="kpi-label">待接管</div>
        <div class="kpi-value" :class="(data?.stats.handoffCount ?? 0) > 0 ? 'text-red-500' : 'op-35'">
          {{ data?.stats.handoffCount ?? 0 }}
        </div>
        <div class="kpi-hint">需人工处理</div>
      </div>
      <div class="kpi-card" @click="channel = 'wechat'; load()">
        <div class="kpi-label">微信</div>
        <div class="kpi-value">{{ data?.stats.wechatCount ?? 0 }}</div>
        <div class="kpi-hint">最近对话</div>
      </div>
      <div class="kpi-card" @click="channel = 'douyin'; load()">
        <div class="kpi-label">抖音</div>
        <div class="kpi-value">{{ data?.stats.douyinCount ?? 0 }}</div>
        <div class="kpi-hint">私信记录</div>
      </div>
      <div class="kpi-card" @click="channel = 'xhs'; load()">
        <div class="kpi-label">小红书</div>
        <div class="kpi-value">{{ data?.stats.xhsCount ?? 0 }}</div>
        <div class="kpi-hint">线索动态</div>
      </div>
      <div class="kpi-card">
        <div class="kpi-label">合计</div>
        <div class="kpi-value">{{ (data?.stats.handoffCount ?? 0) + (data?.stats.messageCount ?? 0) }}</div>
        <div class="kpi-hint">待处理 + 最近</div>
      </div>
    </div>

    <NCard :bordered="false" size="small" class="flex-1">
      <template #header>
        <div class="flex items-center justify-between gap-12px">
          <div class="seg-tabs">
            <button
              v-for="tab in channelTabs"
              :key="tab.value"
              :class="{ 'is-active': channel === tab.value }"
              @click="channel = tab.value; syncQuery(); load()"
            >
              {{ tab.label }}
            </button>
          </div>

          <div class="flex items-center gap-6px">
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
      </template>

      <ErrorState
        v-if="loadError"
        title="收件箱加载失败"
        :desc="loadError"
        :loading="loading"
        @retry="load"
      />

      <TableSkeleton v-else-if="loading && !items.length" :rows="5" :columns="5" />

      <table v-else class="data-table mobile-cards sticky-header">
        <thead>
          <tr>
            <th v-if="isVisible('Channel')" style="width: 80px">平台</th>
            <th v-if="isVisible('Name')" style="width: 160px">客户</th>
            <th v-if="isVisible('Text')">内容</th>
            <th v-if="isVisible('Kind')" style="width: 90px" class="center">类型</th>
            <th v-if="isVisible('Time')" style="width: 140px" class="num">时间</th>
            <th style="width: 80px" class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in items" :key="`${row.Channel}-${row.Id}-${index}`">
            <td v-if="isVisible('Channel')" data-label="渠道">
              <NTag size="small" :type="channelType(row.Channel)">{{ channelLabel(row.Channel) }}</NTag>
            </td>
            <td v-if="isVisible('Name')" class="font-medium" data-label="客户">{{ row.Name || '-' }}</td>
            <td v-if="isVisible('Text')" class="truncate-cell" :title="row.Text" data-label="内容">{{ row.Text || '-' }}</td>
            <td v-if="isVisible('Kind')" class="center" data-label="状态">
              <NTag size="tiny" :type="row.Kind === 'handoff' ? 'error' : 'default'">
                {{ row.Kind === 'handoff' ? row.Tag || '转人工' : row.Tag || '消息' }}
              </NTag>
            </td>
            <td v-if="isVisible('Time')" class="num text-12px op-50" data-label="时间">{{ row.Time || '-' }}</td>
            <td class="num" data-label="操作">
              <NButton text type="primary" size="tiny" @click="open(row)">去处理</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <EmptyState v-if="!loadError && !loading && !items.length" title="收件箱是空的" desc="三个平台有新消息时会自动汇总到这里" />
    </NCard>
  </div>
</template>