<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { NButton, NPopconfirm, NPopselect, NSelect, NTag } from 'naive-ui';
import { fetchTableViewList, saveTableView } from '@/service/api/lead';
import PageHeader from '@/components/business/PageHeader.vue';
import { loadColumnPrefs as loadColumnPrefsRemote, saveColumnPrefs as saveColumnPrefsRemote } from '@/utils/table-prefs';
import ErrorState from '@/components/business/ErrorState.vue';
import TableSkeleton from '@/components/business/TableSkeleton.vue';
import {
  convertLead,
  addWxFriend,
  exportLeads,
  fetchDeleteLead,
  fetchGetLeadList,
  fetchUpdateLead,
  genLeadDmScript,
  genLeadDmScripts,
  type LeadRow
} from '@/service/api/lead';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { useAppStore } from '@/store/modules/app';

const appStore = useAppStore();

/** 移动端少显示几列：提取信息 / 来源 / 私信话术 / 评分 */
/* ---------- 列设置（显隐持久化） ---------- */
const COLUMN_KEY = 'table:lead:v1';

const ALL_COLUMNS = [
  { key: 'ExtractInfo', label: '提取信息' },
  { key: 'Side', label: '方向' },
  { key: 'Source', label: '来源' },
  { key: 'Status', label: '线索检验' },
  { key: 'Grade', label: '等级' },
  { key: 'DmScript', label: '私信话术' },
  { key: 'Score', label: '评分' }
];

/** 移动端少显示几列 */
const MOBILE_HIDE_COLUMNS = ['ExtractInfo', 'Source', 'DmScript', 'Score'];

const hiddenColumns = ref<string[]>([]);

async function loadColumnPrefs() {
  const saved = await loadColumnPrefsRemote('lead', COLUMN_KEY);

  if (saved?.length) hiddenColumns.value = saved;
}

function saveHiddenColumns(next: string[]) {
  hiddenColumns.value = next;
  saveColumnPrefsRemote('lead', COLUMN_KEY, next);
}
const VIEWS_KEY = 'views:lead:v1';

interface LeadView {
  name: string;
  side?: string;
  source?: string;
  grade?: string;
  status?: number;
  hidden: string[];
}

const views = ref<LeadView[]>([]);
const activeView = ref('');

async function loadViews() {
  try {
    const raw = localStorage.getItem(VIEWS_KEY);

    if (raw) views.value = JSON.parse(raw);
  } catch {
    /* 忽略 */
  }

  const { data } = await fetchTableViewList('lead');

  if (data?.length) {
    views.value = data
      .filter(row => row.Name !== "__columns")
      .map(row => {
        try {
          return JSON.parse(row.Payload) as LeadView;
        } catch {
          return null;
        }
      })
      .filter(Boolean) as LeadView[];
  }
}

function persistViews(next: LeadView[]) {
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

  const view: LeadView = {
    name,
    side: searchParams.value.side,
    source: searchParams.value.source,
    grade: searchParams.value.grade,
    status: searchParams.value.status,
    hidden: [...hiddenColumns.value]
  };

  persistViews([...views.value.filter(v => v.name !== name), view]);
  activeView.value = name;

  saveTableView({ tableKey: 'lead', name, payload: JSON.stringify(view) }).catch(() => {});
}

function applyView(name: string) {
  const view = views.value.find(v => v.name === name);

  if (!view) return;

  activeView.value = name;
  searchParams.value.side = view.side;
  searchParams.value.source = view.source;
  searchParams.value.grade = view.grade;
  searchParams.value.status = view.status;
  searchParams.value.current = 1;
  saveHiddenColumns([...view.hidden]);
  syncQuery();
  getData();
}

/* ---------- URL 同步 ---------- */
const route = useRoute();
const router = useRouter();

function syncQuery() {
  const query: Record<string, string> = {};

  if (searchParams.value.side) query.side = searchParams.value.side;
  if (searchParams.value.source) query.source = searchParams.value.source;
  if (searchParams.value.grade) query.grade = searchParams.value.grade;
  if (searchParams.value.current > 1) query.page = String(searchParams.value.current);
  if (searchParams.value.sort && searchParams.value.sort !== 'created_at') query.sort = searchParams.value.sort;
  if (searchParams.value.order) query.order = searchParams.value.order === 'ascending' ? 'asc' : 'desc';

  router.replace({ query });
}

function restoreQuery() {
  const q = route.query;

  searchParams.value.side = (q.side as string) || undefined;
  searchParams.value.source = (q.source as string) || undefined;
  searchParams.value.grade = (q.grade as string) || undefined;
  searchParams.value.current = Number(q.page || 1) || 1;

  if (q.sort) searchParams.value.sort = String(q.sort);

  if (q.order) searchParams.value.order = q.order === 'asc' ? 'ascending' : 'descending';
}

function mobilesafe(columns: any[]) {
  return columns.filter(col => {
    if (hiddenColumns.value.includes(col.key)) return false;
    if (appStore.isMobile && MOBILE_HIDE_COLUMNS.includes(col.key)) return false;

    return true;
  });
}
function formatTime(value: string) {
  if (!value) return '-';

  return String(value).replace('T', ' ').replace(/\.\d+Z?$/, '').replace(/[+-]\d{2}:\d{2}$/, '');
}

const STATUS_TEXT: Record<number, string> = {
  0: '新线索',
  1: '已联系',
  2: '有意向',
  3: '已成交',
  4: '无效'
};

const STATUS_TYPE: Record<number, NaiveUI.ThemeColor> = {
  0: 'default',
  1: 'info',
  2: 'warning',
  3: 'success',
  4: 'error'
};

const searchParams = ref({ current: 1, size: 10, status: undefined as number | undefined, source: undefined as string | undefined, grade: undefined as string | undefined, side: undefined as string | undefined, sort: 'created_at' as string | undefined, order: 'descending' as string | undefined });

const scriptLoading = ref<number | null>(null);

/** 生成单条线索的抖音私信开场白 */
async function handleGenScript(row: LeadRow) {
  scriptLoading.value = row.Id;
  const { data, error } = await genLeadDmScript(row.Id);
  scriptLoading.value = null;

  if (error) return;

  row.DmScript = data?.script ?? '';
  window.$message?.success('话术已生成');
}

/** 批量给 A 类线索生成话术 */
async function handleGenAll() {
  const { data, error } = await genLeadDmScripts('A', 10);

  if (error) return;

  window.$message?.success(`已生成 ${data?.generated ?? 0} 条`);
  getData();
}

const totalCount = ref(0);
const loadError = ref('');

/* ---------- 本地排序（表格 remote，排序在页内做） ---------- */
const sortState = ref<{ key: string; dir: 'asc' | 'desc' }>({ key: 'CreatedAt', dir: 'desc' });

/** 表头排序：改服务端排序参数并重新取数（翻页顺序也正确） */
function applySorter(sorter: any) {
  const fieldMap: Record<string, string> = { Score: 'score', CreatedAt: 'created_at', Id: 'id' };

  if (!sorter || !sorter.order) {
    searchParams.value.sort = 'created_at';
    searchParams.value.order = 'descending';
  } else {
    searchParams.value.sort = fieldMap[String(sorter.columnKey)] || 'created_at';
    searchParams.value.order = sorter.order === 'ascend' ? 'ascending' : 'descending';
  }

  searchParams.value.current = 1;
  syncQuery();
  getData();
}
const sortedData = computed(() => {
  const rows = [...data.value];
  const { key, dir } = sortState.value;
  const factor = dir === 'asc' ? 1 : -1;

  rows.sort((a: any, b: any) => {
    const va = a?.[key];
    const vb = b?.[key];

    if (va === vb) return 0;
    if (va === null || va === undefined || va === '') return 1;
    if (vb === null || vb === undefined || vb === '') return -1;

    const numA = Number(va);
    const numB = Number(vb);

    if (!Number.isNaN(numA) && !Number.isNaN(numB)) return (numA - numB) * factor;

    return String(va).localeCompare(String(vb)) * factor;
  });

  return rows;
});

const { columns, data, loading, getData, mobilePagination } = useNaivePaginatedTable({
  api: async () => {
    loadError.value = '';

    const result = await fetchGetLeadList({
      page: searchParams.value.current,
      pageSize: searchParams.value.size,
      status: searchParams.value.status,
      source: searchParams.value.source,
      side: searchParams.value.side,
      grade: searchParams.value.grade,
      sort: searchParams.value.sort,
      order: searchParams.value.order
    });

    if (result.error) {
      loadError.value = (result.error as any)?.message || '请求失败，请稍后重试';
    }

    return result;
  },
  transform: response => {
    const result = defaultTransform(response);

    totalCount.value = (result as any).total ?? (result as any).records?.length ?? 0;

    return result;
  },
  onPaginationParamsChange: params => {
    syncQuery();
    searchParams.value.current = params.page;
    searchParams.value.size = params.pageSize;
  },
  columns: () => mobilesafe([
    {
      type: 'selection',
      width: 44
    },
    {
      key: 'Company',
      title: '用户',
      minWidth: 150,
      render: row => (
        <div class="flex items-center gap-8px">
          {row.Avatar ? (
            <img src={row.Avatar} class="h-28px w-28px flex-shrink-0 rounded-full object-cover" referrerpolicy="no-referrer" />
          ) : (
            <div class="h-28px w-28px flex-center flex-shrink-0 rounded-full bg-primary/10 text-12px text-primary font-semibold">
              {(row.Company || 'U').slice(0, 1)}
            </div>
          )}
          <div class="min-w-0">
            <div class="truncate text-13px font-medium">{row.Company || '-'}</div>
            <div class="flex items-center gap-4px text-11px op-45">
              <span>{row.Keyword || '-'}</span>
              {row.City ? <span>· {row.City}</span> : null}
            </div>
          </div>
        </div>
      )
    },
    {
      key: 'Remark',
      title: '线索内容',
      minWidth: 300,
      render: row => (
        <div class="truncate text-13px" title={row.Remark || ''}>
          {row.Remark || '-'}
        </div>
      )
    },
    {
      key: 'ExtractInfo',
      title: '提取信息',
      width: 150,
      ellipsis: { tooltip: true },
      render: row => {
        const info = row.ExtractInfo || '';

        if (!info) return <span class="op-40">-</span>;

        // 抖音用户 ID（MS4wL 开头）渲染成可点开的主页链接
        if (info.startsWith('MS4wL')) {
          return (
            <a
              href={`https://www.douyin.com/user/${info}`}
              target="_blank"
              rel="noreferrer"
              class="text-12px text-primary hover:underline"
              title={info}
            >
              打开主页
            </a>
          );
        }

        return <span class="text-12px text-primary font-medium">{info}</span>;
      }
    },
    {
      key: 'Side',
      title: '方向',
      width: 92,
      render: row => {
        const supplier = row.Side === 'supplier';

        return (
          <NTag size="small" type={supplier ? 'warning' : 'info'}>
            {supplier ? '供我方' : '我方客户'}
          </NTag>
        );
      }
    },
    {
      key: 'Source',
      title: '来源',
      width: 90,
      render: row => (
        <NTag size="small" type={row.Source === 'douyin' ? 'success' : row.Source === 'xhs' ? 'error' : row.Source === 'douyin_account' ? 'info' : 'default'}>
          {row.Source === 'douyin' ? '抖音' : row.Source === 'xhs' ? '小红书' : row.Source === 'douyin_account' ? '账号' : row.Source === 'scrape' ? '采集' : '手工'}
        </NTag>
      )
    },
    {
      key: 'Status',
      title: '线索检验',
      width: 110,
      render: row => (
        <NSelect
          size="small"
          value={row.Status}
          options={[0, 1, 2, 3, 4].map(v => ({ label: STATUS_TEXT[v], value: v }))}
          onUpdateValue={(value: number) => handleStatusChange(row, value)}
        />
      )
    },
    {
      key: 'Grade',
      title: '等级',
      width: 80,
      align: 'center',
      render: row => (
        <NTag
          size="small"
          type={row.Grade === 'S' ? 'error' : row.Grade === 'A' ? 'warning' : row.Grade === 'B' ? 'info' : 'default'}
        >
          {row.Grade ? `${row.Grade}${row.ContactReady ? '1' : '2'}` : '-'}
        </NTag>
      )
    },
    {
      key: 'DmScript',
      title: '私信话术',
      width: 300,
      render: row => (
        <div class="flex items-center gap-6px">
          <span class="truncate text-12px" title={row.DmScript}>
            {row.DmScript || <span class="op-40">未生成</span>}
          </span>
          <NButton
            size="tiny"
            text
            type="primary"
            loading={scriptLoading.value === row.Id}
            onClick={() => handleGenScript(row)}
          >
            {row.DmScript ? '重写' : '生成'}
          </NButton>
        </div>
      )
    },
    {
      key: 'Score',
      sorter: true,
      title: '评分',
      width: 70,
      align: 'center',
      render: row => (
        <span class={row.Score >= 70 ? 'text-success font-semibold' : row.Score >= 50 ? 'text-primary' : 'op-50'}>
          {row.Score}
        </span>
      )
    },
    {
      key: 'CreatedAt',
      sorter: true,
      title: '执行时间',
      width: 160,
      render: row => <span class="text-12px op-60">{formatTime(row.CreatedAt)}</span>
    },
    {
      key: 'operate',
      title: '操作',
      width: 150,
      align: 'center',
      render: row => (
        <div class="flex-center gap-6px">
          <NButton type="primary" ghost size="small" onClick={() => handleConvert(row)}>
            转客户
          </NButton>
          <NButton size="small" onClick={() => handleAddWx(row)}>
            加好友
          </NButton>
          <NPopconfirm onPositiveClick={() => handleDelete(row.Id)}>
          {{
            default: () => '确认删除？',
            trigger: () => (
              <NButton type="error" ghost size="small">
                删除
              </NButton>
            )
          }}
          </NPopconfirm>
        </div>
      )
    }
  ]),
});

async function handleStatusChange(row: LeadRow, status: number) {
  const { error } = await fetchUpdateLead({ Id: row.Id, Status: status });

  if (error) return;

  window.$message?.success(`已标记为「${STATUS_TEXT[status]}」`);
  getData();
}

async function handleAddWx(row: any) {
  const ok = window.confirm(
    `给「${row.ContactName || ''}」发微信好友申请？\n\n` +
      '会先搜索号码、确认是可添加状态，再发送申请。'
  );

  if (!ok) return;

  const { data, error } = await addWxFriend(row.Id, '你好，我这边做鸡蛋批发');

  if (error) {
    window.$message?.error(error.message || '执行失败');

    return;
  }

  window.$message?.info(data?.output || '已执行，看控制台输出');
}

const checkedRowKeys = ref<number[]>([]);
const importing = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

function triggerImport() {
  fileInput.value?.click();
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];

  if (!file) return;

  importing.value = true;
  const form = new FormData();
  form.append('file', file);

  try {
    const res = await fetch('/proxy-default/lead/importLeads', {
      method: 'POST',
      headers: { 'x-token': localStorage.getItem('token') || '' },
      body: form
    });
    const json = await res.json();
    importing.value = false;

    if (json.code !== 0) {
      window.$message?.error(json.msg || '导入失败');

      return;
    }

    const data = json.data || {};
    window.$message?.success(`导入 ${data.inserted ?? 0} 条，跳过 ${data.skipped ?? 0} 条，失败 ${data.failed ?? 0} 条`);
    getData();
  } catch (e) {
    importing.value = false;
    window.$message?.error('导入请求失败');
  } finally {
    input.value = '';
  }
}

async function handleBatchConvert() {
  if (!checkedRowKeys.value.length) return;

  const { data: result, error } = await convertLeads(checkedRowKeys.value);

  if (error) return;

  window.$message?.success(`转客户 ${result?.converted ?? 0} 条，重复 ${result?.duplicated ?? 0} 条，失败 ${result?.failed ?? 0} 条`);
  checkedRowKeys.value = [];
  getData();
}

async function handleConvert(row: any) {
  const { error } = await convertLead(row.Id);

  if (error) return;

  window.$message?.success(`${row.ContactName} 已转入客户池`);
  getData();
}

async function handleExport() {
  await exportLeads({
    source: searchParams.value.source,
    side: searchParams.value.side,
    grade: searchParams.value.grade
  });
  window.$message?.success('已导出 Excel');
}

async function handleDelete(id: number) {
  const { error } = await fetchDeleteLead(id);

  if (error) return;

  window.$message?.success('删除成功');
  getData();
}

function handleGradeFilter(value: string | undefined) {
  searchParams.value.grade = value;
  searchParams.value.current = 1;
  getData();
}

/** 方向筛选：切换后回到第一页重新查 */
function handleSideFilter() {
  searchParams.value.current = 1;
  getData();
}
function handleSourceFilter(value: string | undefined) {
  searchParams.value.source = value;
  searchParams.value.current = 1;
  getData();
}

function handleFilter(value: number | undefined) {
  searchParams.value.status = value;
  searchParams.value.current = 1;
  getData();
}
onMounted(() => {
  loadColumnPrefs();
  loadViews();
  restoreQuery();
  getData();
});
</script>
<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="线索池" desc="采集来的线索都在这里，按来源和等级筛选，勾选后可批量转客户">
      <template #actions>
        <NSelect
          v-model:value="searchParams.side"
          size="small"
          clearable
          placeholder="方向"
          style="width: 118px"
          :options="[
            { label: '我方客户', value: 'buyer' },
            { label: '供我方', value: 'supplier' }
          ]"
          @update:value="handleSideFilter"
        />
        <NPopselect
          :options="views.map(v => ({ label: v.name, value: v.name }))"
          :value="activeView || null"
          placeholder="视图"
          @update:value="applyView"
        >
          <NButton size="small">{{ activeView || '视图' }}</NButton>
        </NPopselect>
        <NButton size="small" @click="saveCurrentView">存为视图</NButton>
        <NPopselect
          :value="hiddenColumns"
          multiple
          :options="ALL_COLUMNS.map(c => ({ label: c.label, value: c.key }))"
          @update:value="saveHiddenColumns"
        >
          <NButton size="small">列设置</NButton>
        </NPopselect>
        <NButton size="small" :loading="importing" @click="triggerImport">导入 Excel</NButton>
        <NButton size="small" @click="handleExport">导出 Excel</NButton>
        <NButton size="small" type="primary" ghost @click="handleGenAll">批量生成 A 类话术</NButton>
      </template>
    </PageHeader>

    <NCard :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <NSpace>
          <NSelect
            :value="searchParams.source"
            style="width: 130px"
            placeholder="全部来源"
            clearable
            :options="[
              { label: '抖音', value: 'douyin' },
              { label: '小红书', value: 'xhs' },
              { label: '采集', value: 'scrape' },
              { label: '手工', value: 'manual' }
            ]"
            @update:value="handleSourceFilter"
          />
          <NSelect
            :value="searchParams.grade"
            style="width: 120px"
            placeholder="全部等级"
            clearable
            :options="[
              { label: 'S 级', value: 'S' },
              { label: 'A 级', value: 'A' },
              { label: 'B 级', value: 'B' },
              { label: 'C 级', value: 'C' },
              { label: 'D 级', value: 'D' }
            ]"
            @update:value="handleGradeFilter"
          />
          <input ref="fileInput" type="file" accept=".xlsx,.xls" class="hidden" @change="handleFileChange" />
        </NSpace>
      </template>

      <ErrorState
        v-if="loadError"
        title="线索加载失败"
        :desc="loadError"
        :loading="loading"
        @retry="getData"
      />

      <TableSkeleton v-else-if="loading && !data.length" :rows="8" :columns="6" />

      <NDataTable
        v-else
        v-model:checked-row-keys="checkedRowKeys"
        :columns="columns"
        :data="sortedData"
        :loading="loading"
        :pagination="mobilePagination"
        :row-key="row => row.Id"
        :flex-height="!appStore.isMobile"
        :scroll-x="appStore.isMobile ? 760 : 1500"
        size="small"
        @update:sorter="applySorter"
        remote
        class="lead-table sm:h-full"
      />
    </NCard>

    <div v-if="checkedRowKeys.length" class="bulk-bar">
      <span class="bulk-count">已选 {{ checkedRowKeys.length }} 条（本页 {{ data.length }} 条，共 {{ totalCount }} 条）</span>
      <NButton size="small" type="primary" @click="handleBatchConvert">转为客户</NButton>
      <NButton size="small" @click="handleExport">导出所选项</NButton>
      <NButton size="small" quaternary @click="checkedRowKeys = []">取消选择</NButton>
    </div>
  </div>
</template>

<style scoped>
.bulk-bar {
  position: fixed;
  left: 50%;
  bottom: 56px;
  z-index: 300;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border: 1px solid var(--ds-border);
  border-radius: 10px;
  background: var(--ds-surface);
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.16);
  transform: translateX(-50%);
}

.bulk-count {
  font-size: 12px;
  color: var(--ds-text-2);
}

@media (max-width: 767px) {
  .bulk-bar {
    left: 12px;
    right: 12px;
    bottom: 72px;
    justify-content: center;
    transform: none;
  }
}
</style>