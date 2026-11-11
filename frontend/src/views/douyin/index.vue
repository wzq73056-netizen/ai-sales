<script setup lang="tsx">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { NButton, NInput, NInputNumber, NPopconfirm, NSelect, NSwitch, NTag } from 'naive-ui';
import PageHeader from '@/components/business/PageHeader.vue';
import EmptyState from '@/components/business/EmptyState.vue';
import {
  deleteDouyinDevice,
  deleteDouyinWhitelist,
  fetchCreateDouyinTask,
  fetchDeleteDouyinTask,
  fetchDouyinLeadTotal,
  fetchDouyinTaskList,
  fetchGenKeywords,
  fetchRunningLog,
  fetchStartDouyinTask,
  fetchStopDouyinTask,
  fetchDouyinAccountList,
  fetchDeleteDouyinAccount,
  fetchUpdateAccountStatus,
  fetchAllDouyinAccounts,
  fetchDouyinDmList,
  fetchDmControl,
  saveDmControl,
  fetchDouyinDevices,
  fetchDouyinWhitelist,
  saveDouyinDevice,
  saveDouyinWhitelist,
  type DmControl,
  type DouyinTask,
  type DouyinAccount,
  type DouyinDm,
  type DouyinDevice,
  type DouyinWhitelist,
  type TaskForm
} from '@/service/api/douyin';

const router = useRouter();

const STATUS_TEXT: Record<number, string> = { 0: '待执行', 1: '运行中', 2: '成功', 3: '失败', 4: '已停止' };

/** 定时任务跑完后状态会被重置为待执行，展示上要区分「没跑过」和「已排期等下一轮」 */
function statusLabel(task: DouyinTask) {
  if (task.Status === 1) return '运行中';
  if (task.NextRunAt && task.FinishedAt) return '已排期';
  if (task.NextRunAt && !task.FinishedAt) return '等待首次执行';
  return STATUS_TEXT[task.Status] ?? String(task.Status);
}

function statusType(task: DouyinTask): NaiveUI.ThemeColor {
  if (task.Status === 1) return 'warning';
  if (task.NextRunAt && task.FinishedAt) return 'info';
  return STATUS_TYPE[task.Status] ?? 'default';
}
const STATUS_TYPE: Record<number, NaiveUI.ThemeColor> = { 0: 'default', 1: 'warning', 2: 'success', 3: 'error', 4: 'info' };

const tasks = ref<DouyinTask[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 10;
const loading = ref(false);
const leadTotal = ref(0);
const runningId = ref(0);
const runningLog = ref<string[]>([]);

const showCreate = ref(false);
const showLog = ref(false);
const activeTab = ref<'tasks' | 'accounts' | 'dms' | 'devices' | 'send'>('tasks');

/* ---------- 主动私信 ---------- */
const dyDmTarget = ref('');
const dyDmMessage = ref('');
const dyDmSending = ref(false);
const dyDmOutput = ref('');

async function handleSendDyDm(send: boolean) {
  if (!dyDmTarget.value) {
    window.$message?.warning('先选发送对象');

    return;
  }

  dyDmSending.value = true;
  const res = await fetch('/proxy-default/lead/sendDyDm', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-token': localStorage.getItem('token') || ''
    },
    body: JSON.stringify({ NickName: dyDmTarget.value, Message: dyDmMessage.value, Send: send })
  });
  const json = await res.json().catch(() => null);
  dyDmSending.value = false;

  if (!json || json.code !== 0) {
    window.$message?.error(json?.msg || '发送失败');

    return;
  }

  dyDmOutput.value = json.data?.output || '';
  window.$message?.success(send ? '已发送' : '预览完成（未发送）');
  if (send) dyDmMessage.value = '';
}

/* ---------- 账号管理（多浏览器） ---------- */
const devices = ref<DouyinDevice[]>([]);
const whitelist = ref<DouyinWhitelist[]>([]);
const deviceFilter = ref(0);
const showDevice = ref(false);
const deviceForm = ref<Partial<DouyinDevice>>({ Name: '', DouyinId: '', ProfileDir: '', CdpPort: 9334, Status: 1 });
const showWhitelist = ref(false);
const whitelistForm = ref<Partial<DouyinWhitelist>>({ Nickname: '', DouyinId: '', ProfileUrl: '', DeviceId: 0 });

/** 常用浏览器（不同账号可以用不同浏览器，降低关联风险） */
const browserPresets = [
  { name: 'Edge', path: 'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe' },
  { name: 'Chrome', path: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe' },
  { name: '360安全浏览器', path: 'C:\\Users\\11\\AppData\\Roaming\\360se6\\Application\\360se.exe' },
  { name: '360极速浏览器', path: 'C:\\Users\\11\\AppData\\Roaming\\360Chrome\\Chrome\\Application\\360chrome.exe' },
  { name: 'QQ浏览器', path: 'C:\\Program Files (x86)\\Tencent\\QQBrowser\\QQBrowser.exe' },
  { name: 'Brave', path: 'C:\\Program Files\\BraveSoftware\\Brave-Browser\\Application\\brave.exe' }
];

function applyBrowser(name: string) {
  const preset = browserPresets.find(item => item.name === name);

  if (!preset) return;

  deviceForm.value.BrowserName = preset.name;
  deviceForm.value.BrowserPath = preset.path;
}

async function loadDevices() {
  const { data } = await fetchDouyinDevices();

  if (data) devices.value = data.list ?? [];

  await loadWhitelist();
}

async function loadWhitelist() {
  const { data } = await fetchDouyinWhitelist(deviceFilter.value || undefined);

  if (data) whitelist.value = data.list ?? [];
}

function openDevice(row?: DouyinDevice) {
  deviceForm.value = row
    ? { ...row }
    : { Name: '', DouyinId: '', ProfileDir: 'E:\\\\ai\\\\scraper\\\\edge-profile-2', CdpPort: 9334, Status: 1 };
  showDevice.value = true;
}

async function submitDevice() {
  const { error } = await saveDouyinDevice(deviceForm.value);

  if (error) return;

  window.$message?.success('已保存');
  showDevice.value = false;
  loadDevices();
}

async function removeDevice(row: DouyinDevice) {
  if (!window.confirm('删除账号「' + row.Name + '」？')) return;

  const { error } = await deleteDouyinDevice(row.Id);

  if (error) return;

  window.$message?.success('已删除');
  loadDevices();
}

function openWhitelist(row?: DouyinWhitelist) {
  whitelistForm.value = row
    ? { ...row }
    : { Nickname: '', DouyinId: '', ProfileUrl: '', DeviceId: deviceFilter.value || devices.value[0]?.Id || 0 };
  showWhitelist.value = true;
}

async function submitWhitelist() {
  const { error } = await saveDouyinWhitelist(whitelistForm.value);

  if (error) return;

  window.$message?.success('已保存');
  showWhitelist.value = false;
  loadWhitelist();
}

async function removeWhitelist(row: DouyinWhitelist) {
  if (!window.confirm('从白名单移除「' + row.Nickname + '」？')) return;

  const { error } = await deleteDouyinWhitelist(row.Id);

  if (error) return;

  window.$message?.success('已移除');
  loadWhitelist();
}
const dms = ref<DouyinDm[]>([]);
const dmTotal = ref(0);
const dmPage = ref(1);
const dmCtl = ref<DmControl | null>(null);
const dmSaving = ref(false);
const accounts = ref<DouyinAccount[]>([]);
const accountTotal = ref(0);
const accountPage = ref(1);
const accountKeyword = ref('');
const accountFilter = ref(0);
const accountSort = ref('fans');
const showFollow = ref(false);
const followRow = ref<DouyinAccount | null>(null);
const followText = ref('');
const logTask = ref<DouyinTask | null>(null);
const genLoading = ref(false);

const form = reactive<TaskForm>({
  CrawlType: 0,
  Name: '',
  Industry: '',
  Keywords: '',
  VideosPerKeyword: 3,
  MaxDays: 180,
  LimitN: 10,
  City: '',
  ExcludeWords: '',
  ScheduleType: 'manual',
  Schedule: '',
  AutoStart: true
});

let timer: number | undefined;

const scheduleOptions = [
  { label: '手动执行', value: 'manual' },
  { label: '每天定时', value: 'daily' },
  { label: '间隔执行', value: 'interval' }
];

/** 目标视频数 = 关键词数 × 每词视频数 */
function targetVideos(task: DouyinTask) {
  const kwCount = (task.Keywords || '').split(/[,，;]/).filter(Boolean).length || 1;

  return kwCount * (task.VideosPerKeyword || 1);
}

function progressPct(task: DouyinTask) {
  const target = targetVideos(task);
  if (!target) return 0;

  return Math.min(100, Math.round((task.VideosCount / target) * 100));
}

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)));
const logText = computed(() => (logTask.value?.Log || runningLog.value.join('\n') || '暂无日志'));

async function load() {
  loading.value = true;
  const { data } = await fetchDouyinTaskList({ page: page.value, pageSize });
  loading.value = false;

  if (data) {
    tasks.value = data.records;
    total.value = data.total;
  }
}

async function loadRunning() {
  const { data } = await fetchRunningLog();

  if (data) {
    runningId.value = data.RunningTaskId;
    runningLog.value = data.Log || [];
  }
}

const failedTasks = computed(() => tasks.value.filter(t => t.Status === 3));

async function reloadFailed() {
  for (const t of failedTasks.value) {
    await fetchStartDouyinTask(t.Id);
  }
  window.$message?.success('已重新提交失败任务');
  await load();
}

async function loadDmControl() {
  const { data } = await fetchDmControl();

  if (data) dmCtl.value = data;
}

async function toggleDm(field: 'listening' | 'autoReply' | 'autoSend', value: boolean) {
  if (!dmCtl.value) return;

  const pay = {
    listen: field === 'listening' ? value : dmCtl.value.Listening,
    autoReply: field === 'autoReply' ? value : dmCtl.value.AutoReply,
    autoSend: field === 'autoSend' ? value : dmCtl.value.AutoSend,
    interval: 30
  };

  // 打开真实发送前二次确认
  if (field === 'autoSend' && value) {
    const ok = await new Promise<boolean>(resolve => {
      window.$dialog?.warning({
        title: '确认开启真实发送？',
        content: '开启后 AI 生成的回复会直接发送给对方，不再需要人工确认。',
        positiveText: '确认开启',
        negativeText: '取消',
        onPositiveClick: () => resolve(true),
        onNegativeClick: () => resolve(false),
        onClose: () => resolve(false),
        onMaskClick: () => resolve(false)
      });
    });

    if (!ok) return;
  }

  dmSaving.value = true;

  const { data } = await saveDmControl(pay);

  if (data) dmCtl.value = data;

  dmSaving.value = false;
  window.$message?.success('已保存');
}

async function loadDms() {
  const { data } = await fetchDouyinDmList({ page: dmPage.value, pageSize: 10 });

  if (data) {
    dms.value = data.records;
    dmTotal.value = data.total;
  }
}

async function loadAccounts() {
  const { data } = await fetchDouyinAccountList({
    page: accountPage.value,
    pageSize: 10,
    keyword: accountKeyword.value,
    competitor: accountFilter.value,
    sortBy: accountSort.value
  });

  if (data) {
    accounts.value = data.records;
    accountTotal.value = data.total;
  }
}

async function handleDeleteAccount(id: number) {
  const { error } = await fetchDeleteDouyinAccount(id);

  if (error) return;

  window.$message?.success('已删除');
  loadAccounts();
}

function openFollow(row: DouyinAccount) {
  followRow.value = row;
  followText.value = row.Remark || '';
  showFollow.value = true;
}

async function handleSaveFollow() {
  if (!followRow.value) return;

  const { error } = await fetchUpdateAccountStatus(followRow.value.Id, 1, followText.value);

  if (error) return;

  window.$message?.success('已记录跟进');
  showFollow.value = false;
  loadAccounts();
}

async function handleExport() {
  const { data } = await fetchAllDouyinAccounts({
    keyword: accountKeyword.value,
    competitor: accountFilter.value,
    sortBy: accountSort.value
  });

  if (!data.length) {
    window.$message?.warning('没有可导出的数据');
    return;
  }

  const header = ['昵称', '抖音号', '粉丝数', '简介', '类型', '评分', '状态', '跟进备注', '主页', '关键词'];

  const rows = data.map(r => [
    r.Nickname,
    r.DouyinId,
    String(r.Fans),
    (r.Bio || '').replace(/"/g, '""'),
    r.IsCompetitor ? '同行' : '潜在',
    String(r.Score),
    ['未处理', '已跟进', '无效'][r.Status] || '',
    (r.Remark || '').replace(/"/g, '""'),
    r.ProfileUrl,
    r.Keyword
  ]);

  const csv = [header, ...rows].map(cols => cols.map(c => `"${c}"`).join(',')).join('\r\n');
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `抖音账号线索_${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

async function handleAccountStatus(row: DouyinAccount, status: number) {
  const { error } = await fetchUpdateAccountStatus(row.Id, status);

  if (error) return;

  window.$message?.success('已更新');
  loadAccounts();
}

function formatFans(n: number) {
  if (!n) return '0';

  return n >= 10000 ? `${(n / 10000).toFixed(1)}万` : String(n);
}

async function loadLeadTotal() {
  const { data } = await fetchDouyinLeadTotal();
  leadTotal.value = data ?? 0;
}

async function openCreate() {
  await loadDevices();

  if (!form.DeviceId && devices.value.length) form.DeviceId = devices.value[0].Id;
  Object.assign(form, {
    CrawlType: 0,
    Name: '',
    Industry: '',
    Keywords: '',
    VideosPerKeyword: 3,
    MaxDays: 180,
    LimitN: 10,
    City: '',
    ExcludeWords: '',
    ScheduleType: 'manual',
    Schedule: '',
    AutoStart: true
  });
  showCreate.value = true;
}

async function handleGenKeywords() {
  if (!form.Industry.trim()) {
    window.$message?.warning('请先填写行业');
    return;
  }

  genLoading.value = true;
  const { data, error } = await fetchGenKeywords(form.Industry.trim());
  genLoading.value = false;

  if (error) return;

  form.Keywords = (data || []).join(',');
  window.$message?.success(`已生成 ${data.length} 个关键词`);
}

async function handleCreate() {
  if (!form.Keywords.trim()) {
    window.$message?.warning('请填写或生成关键词');
    return;
  }
  if (form.ScheduleType !== 'manual' && !form.Schedule) {
    window.$message?.warning('请填写调度时间');
    return;
  }

  const { error } = await fetchCreateDouyinTask({ ...form });

  if (error) return;

  window.$message?.success('任务已创建');
  showCreate.value = false;
  page.value = 1;
  load();
  loadRunning();
}

async function handleStart(task: DouyinTask) {
  const { error } = await fetchStartDouyinTask(task.Id);

  if (error) return;

  window.$message?.success('已启动');
  load();
  loadRunning();
}

async function handleStop() {
  const { error } = await fetchStopDouyinTask();

  if (error) return;

  window.$message?.success('已停止');
  load();
}

async function handleDelete(id: number) {
  const { error } = await fetchDeleteDouyinTask(id);

  if (error) return;

  window.$message?.success('已删除');
  load();
}

function openLog(task: DouyinTask) {
  logTask.value = task;
  showLog.value = true;
}

onMounted(async () => {
  await load();
  await loadRunning();
  await loadLeadTotal();
  await loadAccounts();
  await loadDms();
  await loadDmControl();

  timer = window.setInterval(async () => {
    await loadRunning();
    if (runningId.value === 0) await load();
  }, 3000);
});

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer);
});
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="抖音获客" desc="输入行业 → AI 生成搜索关键词 → 采集评论区意向客户（只读，不发送任何内容）">
      <template #actions>
        <div class="head-stat">
          <span class="value text-primary">{{ leadTotal }}</span>
          <span class="label">抖音线索</span>
        </div>
        <NButton size="small" @click="router.push('/lead')">去线索池</NButton>
        <NButton type="primary" size="small" @click="openCreate">新建任务</NButton>
      </template>
    </PageHeader>

    <NAlert v-if="failedTasks.length" type="error" :bordered="false" class="mb-2">
      <div class="flex items-center gap-12px">
        <span>有 {{ failedTasks.length }} 个任务执行失败：{{ failedTasks[0].Keywords }}</span>
        <NButton size="tiny" type="error" ghost @click="reloadFailed">全部重试</NButton>
        <NButton size="tiny" text @click="openLog(failedTasks[0])">查看日志</NButton>
      </div>
    </NAlert>

    <div class="seg-tabs">
      <button :class="{ 'is-active': activeTab === 'tasks' }" @click="activeTab = 'tasks'">采集任务</button>
      <button :class="{ 'is-active': activeTab === 'accounts' }" @click="activeTab = 'accounts'; loadAccounts()">账号线索</button>
      <button :class="{ 'is-active': activeTab === 'dms' }" @click="activeTab = 'dms'; loadDms()">私信记录</button>
      <button :class="{ 'is-active': activeTab === 'devices' }" @click="activeTab = 'devices'; loadDevices()">账号管理</button>
      <button :class="{ 'is-active': activeTab === 'send' }" @click="activeTab = 'send'; loadDevices()">主动私信</button>
    </div>

    <NCard v-if="activeTab === 'tasks'" title="任务列表" :bordered="false" size="small" class="flex-1">
      <template #header-extra>
        <NButton v-if="runningId" size="small" type="error" ghost @click="handleStop">停止当前任务</NButton>
      </template>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>任务信息</th>
            <th>行业</th>
            <th>关键词</th>
            <th class="center">状态</th>
            <th>备注</th>
            <th class="center">进度</th>
            <th class="center">评论</th>
            <th class="center">线索</th>
            <th>开始时间</th>
            <th>下次执行</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="task.Id" >
            <td>
              <NTag size="small" :type="task.CrawlType === 1 ? 'info' : 'default'">{{ task.CrawlType === 1 ? '账号' : '视频' }}</NTag>
            </td>
            <td>{{ task.Industry || '-' }}</td>
            <td class="max-w-260px truncate py-10px" :title="task.Keywords">{{ task.Keywords }}</td>
            <td class="center">
              <NTag size="small" :type="statusType(task)">
                {{ statusLabel(task) }}
              </NTag>
            </td>
            <td class="center">
              <div class="text-12px">{{ task.VideosCount }} / {{ targetVideos(task) }}</div>
              <div class="mx-auto mt-3px h-4px w-60px overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                <div class="h-full bg-primary" :style="{ width: progressPct(task) + '%' }" />
              </div>
            </td>
            <td class="center">{{ task.CommentsCount }}</td>
            <td class="py-10px text-center font-semibold text-primary">{{ task.LeadsCount }}</td>
            <td>{{ task.StartedAt?.replace('T', ' ').slice(0, 19) || '-' }}</td>
            <td>{{ task.NextRunAt?.replace('T', ' ').slice(0, 19) || '-' }}</td>
            <td class="num">
              <div class="flex justify-end gap-6px">
                <NButton text size="tiny" @click="openLog(task)">日志</NButton>
                <NButton v-if="task.Status !== 1" text type="primary" size="tiny" @click="handleStart(task)">执行</NButton>
                <NPopconfirm @positive-click="handleDelete(task.Id)">
                  <template #trigger>
                    <NButton text type="error" size="tiny">删除</NButton>
                  </template>
                  确认删除该任务？
                </NPopconfirm>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <EmptyState v-if="!tasks.length" title="还没有采集任务" desc="点右上角「新建任务」，输入行业让 AI 生成关键词" />

      <div class="mt-12px flex items-center justify-between">
        <span class="text-12px op-50">共 {{ total }} 个任务</span>
        <div class="flex items-center gap-6px">
          <NButton size="tiny" :disabled="page <= 1" @click="page--; load()">上一页</NButton>
          <span class="text-12px">{{ page }} / {{ pageCount }}</span>
          <NButton size="tiny" :disabled="page >= pageCount" @click="page++; load()">下一页</NButton>
        </div>
      </div>
    </NCard>

    <NCard v-if="activeTab === 'accounts'" title="账号线索" :bordered="false" size="small" class="flex-1">
      <template #header-extra>
        <div class="flex items-center gap-8px">
          <NInput v-model:value="accountKeyword" placeholder="关键词/昵称" clearable class="w-160px" size="small" @keyup.enter="accountPage = 1; loadAccounts()" />
          <NSelect
            v-model:value="accountFilter"
            size="small"
            class="w-130px"
            :options="[{ label: '全部账号', value: 0 }, { label: '仅看同行', value: 1 }, { label: '排除同行', value: 2 }]"
            @update:value="accountPage = 1; loadAccounts()"
          />
          <NSelect
            v-model:value="accountSort"
            size="small"
            class="w-120px"
            :options="[{ label: '按粉丝数', value: 'fans' }, { label: '按评分', value: 'score' }]"
            @update:value="accountPage = 1; loadAccounts()"
          />
          <NButton size="small" @click="loadAccounts">刷新</NButton>
          <NButton size="small" type="primary" ghost @click="handleExport">导出Excel</NButton>
        </div>
      </template>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>昵称</th>
            <th>抖音号</th>
            <th class="num">粉丝</th>
            <th>简介</th>
            <th class="center">类型</th>
            <th class="center">评分</th>
            <th class="center">状态</th>
            <th>备注</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in accounts" :key="row.Id" >
            <td class="max-w-180px truncate py-10px" :title="row.Nickname">{{ row.Nickname }}</td>
            <td class="py-10px op-70">{{ row.DouyinId || '-' }}</td>
            <td class="py-10px text-right font-medium">{{ formatFans(row.Fans) }}</td>
            <td class="max-w-280px truncate py-10px op-70" :title="row.Bio">{{ row.Bio || '-' }}</td>
            <td class="center">
              <NTag size="small" :type="row.IsCompetitor ? 'warning' : 'success'">
                {{ row.IsCompetitor ? '同行' : '潜在' }}
              </NTag>
            </td>
            <td class="center">{{ row.Score }}</td>
            <td class="center">
              <NSelect
                size="small"
                class="w-90px"
                :value="row.Status"
                :options="[{ label: '未处理', value: 0 }, { label: '已跟进', value: 1 }, { label: '无效', value: 2 }]"
                @update:value="(v: number) => handleAccountStatus(row, v)"
              />
            </td>
            <td class="max-w-160px truncate py-10px op-70" :title="row.Remark">{{ row.Remark || '-' }}</td>
            <td class="num">
              <NButton text type="primary" size="tiny" class="mr-6px" @click="openFollow(row)">跟进</NButton>
              <NPopconfirm @positive-click="handleDeleteAccount(row.Id)">
                <template #trigger>
                  <NButton text type="error" size="tiny">删除</NButton>
                </template>
                确认删除？
              </NPopconfirm>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!accounts.length" class="py-40px text-center text-12px op-50">
        还没有账号线索，建一个「账号获客」任务后自动入库
      </div>

      <div class="mt-12px flex items-center justify-between">
        <span class="text-12px op-50">共 {{ accountTotal }} 个账号</span>
        <div class="flex items-center gap-6px">
          <NButton size="tiny" :disabled="accountPage <= 1" @click="accountPage--; loadAccounts()">上一页</NButton>
          <span class="text-12px">{{ accountPage }} / {{ Math.max(1, Math.ceil(accountTotal / 10)) }}</span>
          <NButton size="tiny" :disabled="accountPage * 10 >= accountTotal" @click="accountPage++; loadAccounts()">下一页</NButton>
        </div>
      </div>
    </NCard>

    <NModal v-model:show="showFollow" preset="card" class="w-560px" title="跟进记录">
      <div class="mb-8px text-13px op-70">
        {{ followRow?.Nickname }} · 粉丝 {{ formatFans(followRow?.Fans ?? 0) }}
      </div>
      <NInput v-model:value="followText" type="textarea" :autosize="{ minRows: 4 }" placeholder="记录沟通内容，例如：已加微信，下周报价" />
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showFollow = false">取消</NButton>
          <NButton type="primary" @click="handleSaveFollow">保存并标记已跟进</NButton>
        </div>
      </template>
    </NModal>

    <NCard v-if="activeTab === 'dms'" :bordered="false" size="small" class="mb-2">
      <div class="flex flex-wrap items-center gap-20px">
        <span class="text-13px font-medium">私信自动化</span>

        <div class="flex items-center gap-8px">
          <NSwitch
            :value="dmCtl?.Listening"
            :loading="dmSaving"
            @update:value="(v: boolean) => toggleDm('listening', v)"
          />
          <span class="text-12px">监听收件箱</span>
        </div>

        <div class="flex items-center gap-8px">
          <NSwitch
            :value="dmCtl?.AutoReply"
            :disabled="!dmCtl?.Listening"
            :loading="dmSaving"
            @update:value="(v: boolean) => toggleDm('autoReply', v)"
          />
          <span class="text-12px">AI 生成回复</span>
        </div>

        <div class="flex items-center gap-8px">
          <NSwitch
            :value="dmCtl?.AutoSend"
            :disabled="!dmCtl?.AutoReply"
            :loading="dmSaving"
            @update:value="(v: boolean) => toggleDm('autoSend', v)"
          />
          <span class="text-12px" :class="dmCtl?.AutoSend ? 'text-error font-medium' : ''">真实发送</span>
        </div>

        <div class="ml-auto text-11px op-60">
          <span :class="dmCtl?.Running ? 'text-success' : 'op-50'">
            {{ dmCtl?.Running ? '守护进程运行中' : '守护进程未运行' }}
          </span>
          <span v-if="dmCtl?.Rounds"> · 第 {{ dmCtl.Rounds }} 轮 · {{ dmCtl.Conversations }} 个会话 · {{ dmCtl.LastCheckAt }}</span>
        </div>
      </div>

      <div v-if="dmCtl?.LastError" class="mt-6px text-11px text-error">最近错误：{{ dmCtl.LastError }}</div>

      <div v-if="dmCtl?.LastAutoReply" class="mt-6px whitespace-pre-wrap rounded-4px bg-gray-50 p-8px text-11px op-70 dark:bg-gray-800">
        {{ dmCtl.LastAutoReply }}
      </div>
    </NCard>

    <NCard v-if="activeTab === 'dms'" title="私信记录" :bordered="false" size="small" class="flex-1">
      <template #header-extra>
        <NButton size="small" @click="loadDms">刷新</NButton>
      </template>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>目标</th>
            <th>话术</th>
            <th>发送时间</th>
            <th class="center">回复</th>
            <th>回复内容</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in dms" :key="row.Id" >
            <td>
              <div class="font-medium">{{ row.Nickname || '-' }}</div>
              <div class="text-11px op-50">{{ row.DouyinId || '-' }}</div>
            </td>
            <td class="max-w-260px truncate py-10px op-70" :title="row.Message">{{ row.Message || '-' }}</td>
            <td class="py-10px text-12px op-60">{{ row.SentAt || '-' }}</td>
            <td class="center">
              <NTag size="small" :type="row.ReplyStatus ? 'success' : row.Status === 1 ? 'default' : 'error'">
                {{ row.ReplyStatus ? '已回复' : row.Status === 1 ? '等待回复' : '发送失败' }}
              </NTag>
            </td>
            <td class="max-w-220px truncate py-10px" :title="row.ReplyText">
              {{ row.ReplyText || '-' }}
            </td>
            <td class="num">
              <a
                :href="row.ProfileUrl"
                target="_blank"
                rel="noreferrer"
                class="text-12px text-primary hover:underline"
              >
                打开主页
              </a>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!dms.length" class="py-40px text-center text-12px op-50">
        还没有私信记录
      </div>

      <div class="mt-12px flex items-center justify-between">
        <span class="text-12px op-50">共 {{ dmTotal }} 条</span>
        <div class="flex items-center gap-6px">
          <NButton size="tiny" :disabled="dmPage <= 1" @click="dmPage--; loadDms()">上一页</NButton>
          <span class="text-12px">{{ dmPage }} / {{ Math.max(1, Math.ceil(dmTotal / 10)) }}</span>
          <NButton size="tiny" :disabled="dmPage * 10 >= dmTotal" @click="dmPage++; loadDms()">下一页</NButton>
        </div>
      </div>
    </NCard>

    <NCard v-if="activeTab === 'send'" title="主动私信" :bordered="false" size="small">
      <div class="flex flex-wrap items-end gap-12px">
        <div>
          <div class="mb-6px text-12px">发送对象</div>
          <NSelect
            v-model:value="dyDmTarget"
            :options="whitelist.map((w: any) => ({ label: `${w.Nickname}（${w.DouyinId}）`, value: w.Nickname }))"
            placeholder="选白名单里的人"
            class="w-240px"
          />
        </div>
        <div class="flex-1">
          <div class="mb-6px text-12px">消息内容</div>
          <NInput v-model:value="dyDmMessage" placeholder="留空则由 AI 生成" />
        </div>
        <NButton :loading="dyDmSending" @click="handleSendDyDm(false)">预览</NButton>
        <NButton type="primary" :loading="dyDmSending" @click="handleSendDyDm(true)">发送</NButton>
      </div>
      <div class="mt-8px text-11px op-40">
        任意会话都能发。白名单里的人会自动核对抖音号（防认错人），名单外的也能发。
      </div>
      <pre v-if="dyDmOutput" class="mt-10px max-h-160px overflow-auto whitespace-pre-wrap rounded bg-gray-50 p-10px text-11px dark:bg-gray-800">{{ dyDmOutput }}</pre>
    </NCard>

    <NCard v-if="activeTab === 'devices'" :bordered="false" size="small" class="flex-1">
      <template #header>
        <div class="flex items-center gap-12px">
          <span>采集账号（一个账号一个浏览器）</span>
        </div>
      </template>
      <template #header-extra>
        <NButton size="small" type="primary" @click="openDevice()">新增账号</NButton>
      </template>

      <table class="data-table mb-20px">
        <thead>
          <tr>
            <th>备注名</th>
            <th>抖音号</th>
            <th class="center">浏览器</th>
            <th class="center">CDP 端口</th>
            <th>用户数据目录</th>
            <th class="center">状态</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in devices" :key="row.Id" >
            <td class="font-medium">{{ row.Name }}</td>
            <td>{{ row.DouyinId || '-' }}</td>
            <td class="center">
              <NTag size="small" type="info">{{ row.BrowserName || 'Edge' }}</NTag>
            </td>
            <td class="center">{{ row.CdpPort }}</td>
            <td class="max-w-320px truncate py-10px text-11px op-60" :title="row.ProfileDir">{{ row.ProfileDir }}</td>
            <td class="center">
              <NTag size="small" :type="row.Status === 1 ? 'success' : 'default'">
                {{ row.Status === 1 ? '启用' : '停用' }}
              </NTag>
            </td>
            <td class="num">
              <NButton text type="primary" size="tiny" class="mr-6px" @click="openDevice(row)">编辑</NButton>
              <NButton text type="error" size="tiny" @click="removeDevice(row)">删除</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="mb-8px flex items-center gap-12px">
        <span class="text-13px font-medium">自动回复名单（也可当测试发送目标）</span>
        <NSelect
          v-model:value="deviceFilter"
          :options="[{ label: '全部账号', value: 0 }, ...devices.map(d => ({ label: d.Name, value: d.Id }))]"
          class="w-160px"
          size="small"
          @update:value="loadWhitelist()"
        />
        <NButton size="tiny" type="primary" ghost @click="openWhitelist()">添加白名单</NButton>
      </div>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>昵称</th>
            <th>抖音号</th>
            <th>主页</th>
            <th class="center">启用</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in whitelist" :key="row.Id" >
            <td class="font-medium">{{ row.Nickname }}</td>
            <td>{{ row.DouyinId }}</td>
            <td class="max-w-360px truncate py-10px text-11px text-primary" :title="row.ProfileUrl">{{ row.ProfileUrl }}</td>
            <td class="center">
              <NTag size="small" :type="row.Enabled === 1 ? 'success' : 'default'">
                {{ row.Enabled === 1 ? '是' : '否' }}
              </NTag>
            </td>
            <td class="num">
              <NButton text type="primary" size="tiny" class="mr-6px" @click="openWhitelist(row)">编辑</NButton>
              <NButton text type="error" size="tiny" @click="removeWhitelist(row)">移除</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!whitelist.length" class="py-30px text-center text-12px op-50">
        还没有白名单（只有名单里的人才会被自动回复）
      </div>
    </NCard>

    <NModal v-model:show="showDevice" preset="card" class="w-560px" title="抖音采集账号">
      <div class="flex-col-stretch gap-12px">
        <div>
          <div class="mb-6px text-12px">备注名</div>
          <NInput v-model:value="deviceForm.Name" placeholder="如：主账号、小号" />
        </div>
        <div>
          <div class="mb-6px text-12px">抖音号</div>
          <NInput v-model:value="deviceForm.DouyinId" placeholder="登录后看到的抖音号" />
        </div>
        <div>
          <div class="mb-6px text-12px">浏览器</div>
          <NSelect
            :value="deviceForm.BrowserName"
            :options="browserPresets.map(b => ({ label: b.name, value: b.name }))"
            @update:value="(v: string) => applyBrowser(v)"
          />
        </div>
        <div>
          <div class="mb-6px text-12px">浏览器程序路径</div>
          <NInput v-model:value="deviceForm.BrowserPath" placeholder="C:\Program Files\Google\Chrome\Application\chrome.exe" />
        </div>
        <div>
          <div class="mb-6px text-12px">用户数据目录（每个账号独立）</div>
          <NInput v-model:value="deviceForm.ProfileDir" placeholder="E:\ai\scraper\edge-profile-2" />
        </div>
        <div>
          <div class="mb-6px text-12px">CDP 调试端口</div>
          <NInputNumber v-model:value="deviceForm.CdpPort" :min="9000" :max="9999" class="w-full" />
        </div>
        <div class="flex items-center gap-8px">
          <NSwitch v-model:value="deviceForm.Status" :checked-value="1" :unchecked-value="0" />
          <span class="text-12px">启用（启用后才会启动监听）</span>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showDevice = false">取消</NButton>
          <NButton type="primary" @click="submitDevice">保存</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="showWhitelist" preset="card" class="w-560px" title="私信白名单">
      <div class="flex-col-stretch gap-12px">
        <div>
          <div class="mb-6px text-12px">归属账号</div>
          <NSelect
            v-model:value="whitelistForm.DeviceId"
            :options="devices.map(d => ({ label: `${d.Name}（${d.CdpPort}）`, value: d.Id }))"
          />
        </div>
        <div>
          <div class="mb-6px text-12px">对方昵称</div>
          <NInput v-model:value="whitelistForm.Nickname" placeholder="如：庸" />
        </div>
        <div>
          <div class="mb-6px text-12px">对方抖音号（严格校验用）</div>
          <NInput v-model:value="whitelistForm.DouyinId" placeholder="如：134527749" />
        </div>
        <div>
          <div class="mb-6px text-12px">主页地址</div>
          <NInput v-model:value="whitelistForm.ProfileUrl" placeholder="https://www.douyin.com/user/..." />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showWhitelist = false">取消</NButton>
          <NButton type="primary" @click="submitWhitelist">保存</NButton>
        </div>
      </template>
    </NModal>

    <!-- 新建任务 -->
    <NModal v-model:show="showCreate" preset="card" class="w-720px" title="新建抖音获客任务">
      <div class="flex-col-stretch gap-14px">
        <div>
          <div class="mb-6px text-13px">任务名称</div>
          <NInput v-model:value="form.Name" placeholder="留空自动生成，如：代理记账 · 小规模纳税人" :maxlength="50" />
        </div>

        <div>
          <div class="mb-6px text-13px">执行账号</div>
          <NSelect
            v-model:value="form.DeviceId"
            :options="devices.map(d => ({ label: `${d.Name}（端口 ${d.CdpPort}）`, value: d.Id }))"
            placeholder="选一个抖音账号执行"
          />
        </div>

        <div>
          <div class="mb-6px text-13px">任务类型</div>
          <NRadioGroup v-model:value="form.CrawlType">
            <NRadio :value="0">视频获客（搜视频 → 评论区）</NRadio>
            <NRadio :value="1">账号获客（搜用户 → 账号主页）</NRadio>
          </NRadioGroup>
        </div>

        <div>
          <div class="mb-6px text-13px">行业 / 产品</div>
          <div class="flex gap-8px">
            <NInput v-model:value="form.Industry" placeholder="如：代理记账、SaaS 进销存" />
            <NButton type="primary" ghost :loading="genLoading" @click="handleGenKeywords">AI 生成关键词</NButton>
          </div>
        </div>

        <div>
          <div class="mb-6px text-13px">搜索关键词（英文逗号分隔，AI 生成后可修改）</div>
          <NInput
            v-model:value="form.Keywords"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 6 }"
            placeholder="代理记账多少钱,小规模纳税人,个体户记账"
          />
        </div>

        <div class="flex flex-wrap items-center gap-20px">
          <div class="flex items-center gap-8px">
            <span class="text-12px op-60">每词视频数</span>
            <NInputNumber v-model:value="form.VideosPerKeyword" :min="1" :max="20" class="w-90px" />
          </div>
          <div class="flex items-center gap-8px">
            <span class="text-12px op-60">评论时间(天)</span>
            <NInputNumber v-model:value="form.MaxDays" :min="0" :max="3650" class="w-100px" />
          </div>
          <div class="flex items-center gap-8px">
            <span class="text-12px op-60">每视频上限</span>
            <NInputNumber v-model:value="form.LimitN" :min="1" :max="50" class="w-90px" />
          </div>
          <div class="flex items-center gap-8px">
            <span class="text-12px op-60">地区</span>
            <NInput v-model:value="form.City" placeholder="留空不限" class="w-120px" />
          </div>
        </div>

        <div>
          <div class="mb-6px text-13px">排除词</div>
          <NInput
            v-model:value="form.ExcludeWords"
            placeholder="评论含这些词直接跳过，逗号分隔，如：同行,招聘,加盟"
          />
        </div>

        <div class="flex flex-wrap items-center gap-12px">
          <span class="text-12px op-60">执行方式</span>
          <NSelect v-model:value="form.ScheduleType" :options="scheduleOptions" class="w-140px" />
          <NInput
            v-if="form.ScheduleType === 'daily'"
            v-model:value="form.Schedule"
            placeholder="09:00"
            class="w-120px"
          />
          <NInput
            v-if="form.ScheduleType === 'interval'"
            v-model:value="form.Schedule"
            placeholder="间隔分钟，如 60"
            class="w-160px"
          />
          <NCheckbox v-model:checked="form.AutoStart">创建后立即执行</NCheckbox>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showCreate = false">取消</NButton>
          <NButton type="primary" @click="handleCreate">创建</NButton>
        </div>
      </template>
    </NModal>

    <!-- 日志 -->
    <NModal v-model:show="showLog" preset="card" class="w-820px" title="运行日志">
      <pre class="max-h-500px overflow-auto whitespace-pre-wrap rounded-8px bg-gray-50 p-12px text-12px leading-relaxed dark:bg-gray-800">{{
        logText
      }}</pre>
    </NModal>
  </div>
</template>

<style scoped></style>
