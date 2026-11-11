<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { NButton, NCard, NInput, NInputNumber, NModal, NSelect, NTag, useMessage } from 'naive-ui';
import PageHeader from '@/components/business/PageHeader.vue';
import EmptyState from '@/components/business/EmptyState.vue';
import {
  deleteXhsWhitelist,
  fetchXhsCollectStatus,
  fetchXhsDevices,
  fetchXhsLeadStats,
  fetchXhsRuns,
  genXhsScripts,
  fetchXhsSchedule,
  fetchXhsLeads,
  previewXhsLeadDm,
  sendXhsLeadDm,
  genLeadDmScript as genLeadDmScriptApi,
  saveXhsSchedule,
  type XhsSchedule,
  fetchXhsWhitelist,
  saveXhsDevice,
  sendXhsDm,
  saveXhsWhitelist,
  startXhsCollect,
  fetchKeywords,
  saveKeyword,
  deleteKeyword,
  fetchBlacklist,
  saveBlacklist,
  deleteBlacklist,
  type ScrapeKeyword,
  type ScrapeBlacklist,
  type XhsRun,
  type XhsCollectStatus,
  type XhsDevice,
  type XhsLeadStats,
  type XhsWhitelist
} from '@/service/api/xhs';

defineOptions({ name: 'XhsGrowth' });

const message = useMessage();

const devices = ref<XhsDevice[]>([]);
const whitelist = ref<XhsWhitelist[]>([]);
const stats = ref<XhsLeadStats | null>(null);
const status = ref<XhsCollectStatus | null>(null);

const keywords = ref('鸡蛋批发');
const notes = ref(3);
const limit = ref(30);

const runs = ref<XhsRun[]>([]);
const dmTarget = ref('');
const dmMessage = ref('');
const dmSending = ref(false);
const scriptLoading = ref(false);
const schedule = ref<XhsSchedule | null>(null);
const leads = ref<{ Id: number; ContactName: string; Remark: string; Grade: string; Score: number }[]>([]);
const leadTarget = ref<number | null>(null);
const leadMessage = ref('');
const leadPreview = ref('');
const leadLoading = ref(false);
const kwList = ref<ScrapeKeyword[]>([]);
const keywordInput = ref('');
const blackList = ref<ScrapeBlacklist[]>([]);
const blackValue = ref('');
const blackReason = ref('');
const scheduleSaving = ref(false);
const showLog = ref(false);
const logText = ref('');

const showDevice = ref(false);
const deviceForm = ref<Partial<XhsDevice>>({ Name: '', DouyinId: '', ProfileDir: '', BrowserPath: '', BrowserName: 'Edge', CdpPort: 9334, Status: 1 });

const showWhitelist = ref(false);
const whitelistForm = ref<Partial<XhsWhitelist>>({ Nickname: '', DouyinId: '', ProfileUrl: '', DeviceId: 0 });

let timer: ReturnType<typeof setInterval> | null = null;

async function loadAll() {
  const [d, w, s, c, r] = await Promise.all([
    fetchXhsDevices(),
    fetchXhsWhitelist(),
    fetchXhsLeadStats(),
    fetchXhsCollectStatus(),
    fetchXhsRuns()
  ]);

  if (r.data) runs.value = r.data.list ?? [];

  if (d.data) devices.value = d.data.list ?? [];
  if (w.data) whitelist.value = w.data.list ?? [];
  if (s.data) stats.value = s.data;
  if (c.data) status.value = c.data;
}

async function loadKeywords() {
  const { data } = await fetchKeywords('xhs');

  if (data) kwList.value = data.list ?? [];
}

async function addKeyword() {
  const value = keywordInput.value.trim();

  if (!value) return;

  const { error } = await saveKeyword({ Platform: 'xhs', Keyword: value, Enabled: 1 });

  if (error) return;

  keywordInput.value = '';
  loadKeywords();
}

async function removeKeyword(id: number) {
  const { error } = await deleteKeyword(id);

  if (error) return;

  loadKeywords();
}

async function loadBlacklist() {
  const { data } = await fetchBlacklist();

  if (data) blackList.value = data.list ?? [];
}

async function addBlack() {
  const value = blackValue.value.trim();

  if (!value) return;

  const { error } = await saveBlacklist({ Kind: 'nickname', Value: value, Reason: blackReason.value });

  if (error) return;

  blackValue.value = '';
  blackReason.value = '';
  loadBlacklist();
}

async function removeBlack(id: number) {
  const { error } = await deleteBlacklist(id);

  if (error) return;

  loadBlacklist();
}

async function handleCollectWithLibrary() {
  const all = kwList.value.map(item => item.Keyword).join(',');

  if (!all) {
    message.warning('词库是空的，先加关键词');

    return;
  }

  keywords.value = all;
  await handleCollect();
}

async function loadLeads() {
  const { data } = await fetchXhsLeads();

  if (!data) return;

  // 只保留有价值的线索（S/A/B），按分数从高到低
  leads.value = (data.list ?? [])
    .filter(l => ['S', 'A', 'B'].includes(l.Grade))
    .sort((a, b) => b.Score - a.Score);
}

/** 给选中的线索生成一条开场话术，填进输入框 */
async function handleGenLeadScript() {
  if (!leadTarget.value) {
    message.warning('先选一条线索');

    return;
  }

  leadLoading.value = true;
  const { data, error } = await genLeadDmScriptApi(leadTarget.value);
  leadLoading.value = false;

  if (error) return;

  leadMessage.value = data?.script ?? '';
  message.success('话术已生成，可点预览看效果');
}

async function handleSendLeadDm() {
  if (!leadTarget.value) {
    message.warning('先选一条线索');

    return;
  }

  const target = leads.value.find(l => l.Id === leadTarget.value);
  const ok = window.confirm(
    `确认给「${target?.ContactName || ''}」真发私信？\n\n` +
      '互相关注的号不限条数；如果对方还没关注你，只能发 1 条。'
  );

  if (!ok) return;

  leadLoading.value = true;
  const { data, error } = await sendXhsLeadDm({
    LeadId: leadTarget.value,
    Message: leadMessage.value
  });
  leadLoading.value = false;

  if (error) {
    leadPreview.value = error.message || '发送失败';

    return;
  }

  leadPreview.value = data?.output || '（没有输出）';
  message.success('已执行发送');
}

async function handlePreviewLeadDm() {
  if (!leadTarget.value) {
    message.warning('先选一条线索');

    return;
  }

  leadLoading.value = true;
  const { data, error } = await previewXhsLeadDm({
    LeadId: leadTarget.value,
    Message: leadMessage.value
  });
  leadLoading.value = false;

  if (error) {
    leadPreview.value = error.message || '预览失败';

    return;
  }

  leadPreview.value = data?.output || '（没有输出）';
}

async function loadSchedule() {
  const { data } = await fetchXhsSchedule();

  if (data) schedule.value = data;
}

async function saveScheduleCfg() {
  if (!schedule.value) return;

  scheduleSaving.value = true;
  const { error } = await saveXhsSchedule(schedule.value);
  scheduleSaving.value = false;

  if (error) return;

  message.success('定时配置已保存');
  loadSchedule();
}

async function handleGenScripts() {
  scriptLoading.value = true;
  const { data, error } = await genXhsScripts();
  scriptLoading.value = false;

  if (error) return;

  message.success(`已生成 ${data?.generated ?? 0} 条话术`);
}

function openLog(row: XhsRun) {
  logText.value = row.LogTail || '（没有日志）';
  showLog.value = true;
}

async function handleSendDm(send: boolean) {
  if (!dmTarget.value) {
    message.warning('先选一个白名单对象');

    return;
  }

  dmSending.value = true;
  const { data, error } = await sendXhsDm({
    NickName: dmTarget.value,
    Message: dmMessage.value,
    Send: send
  });
  dmSending.value = false;

  if (error) {
    message.error(error.message || '发送失败');

    return;
  }

  if (send) {
    message.success(data?.ok ? '已发送' : '发送失败，看输出');
  } else {
    message.success('预览完成（未发送）');
  }

  dmMessage.value = send ? '' : dmMessage.value;
  loadAll();
}

async function handleCollect() {
  const { data, error } = await startXhsCollect({
    Keywords: keywords.value,
    Notes: notes.value,
    Limit: limit.value
  });

  if (error) {
    message.warning(error.message || '启动失败');

    return;
  }

  status.value = data ?? null;
  message.success('已开始采集，稍后自动刷新');
}

function openDevice() {
  deviceForm.value = {
    Name: '',
    DouyinId: '',
    ProfileDir: 'E:\\ai\\scraper\\edge-profile-xhs2',
    BrowserPath: 'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    BrowserName: 'Edge',
    CdpPort: 9334,
    Status: 1
  };
  showDevice.value = true;
}

async function submitDevice() {
  const { error } = await saveXhsDevice(deviceForm.value);

  if (error) return;

  message.success('已保存');
  showDevice.value = false;
  loadAll();
}

function openWhitelist(row?: XhsWhitelist) {
  whitelistForm.value = row
    ? { ...row }
    : { Nickname: '', DouyinId: '', ProfileUrl: '', DeviceId: devices.value[0]?.Id ?? 0 };
  showWhitelist.value = true;
}

async function submitWhitelist() {
  const { error } = await saveXhsWhitelist(whitelistForm.value);

  if (error) return;

  message.success('已保存');
  showWhitelist.value = false;
  loadAll();
}

async function removeWhitelist(row: XhsWhitelist) {
  if (!window.confirm('移除白名单「' + row.Nickname + '」？')) return;

  const { error } = await deleteXhsWhitelist(row.Id);

  if (error) return;

  message.success('已移除');
  loadAll();
}

async function toggleDevice(row: XhsDevice) {
  const { error } = await saveXhsDevice({ ...row, Status: row.Status === 1 ? 0 : 1 });

  if (error) return;

  loadAll();
}

onMounted(() => {
  loadAll();
  loadSchedule();
  loadLeads();
  loadKeywords();
  loadBlacklist();
  timer = setInterval(loadAll, 15000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="小红书获客" desc="按关键词采笔记和评论，评论里问价的人入线索池">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>

    <div class="grid grid-cols-4 gap-12px">
      <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
        <div class="text-12px op-50">小红书线索</div>
        <div class="mt-6px text-28px font-semibold">{{ stats?.total ?? 0 }}</div>
      </div>
      <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
        <div class="text-12px op-50">今日新增</div>
        <div class="mt-6px text-28px font-semibold text-blue-600">{{ stats?.today ?? 0 }}</div>
      </div>
      <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
        <div class="text-12px op-50">私信白名单</div>
        <div class="mt-6px text-28px font-semibold">{{ whitelist.length }}</div>
      </div>
      <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
        <div class="text-12px op-50">采集状态</div>
        <div class="mt-6px text-18px font-semibold" :class="status?.running ? 'text-orange-500' : 'op-60'">
          {{ status?.running ? '采集中…' : '空闲' }}
        </div>
        <div class="mt-4px text-11px op-40">{{ status?.finished || status?.started || '-' }}</div>
      </div>
    </div>

    <NCard title="采集" :bordered="false" size="small">
      <div class="flex flex-wrap items-end gap-12px">
        <div>
          <div class="mb-6px text-12px">关键词（逗号分隔）</div>
          <NInput v-model:value="keywords" placeholder="鸡蛋批发,鸡蛋价格" class="w-300px" />
        </div>
        <div>
          <div class="mb-6px text-12px">每个关键词笔记数</div>
          <NInputNumber v-model:value="notes" :min="1" :max="10" class="w-140px" />
        </div>
        <div>
          <div class="mb-6px text-12px">每篇评论上限</div>
          <NInputNumber v-model:value="limit" :min="5" :max="100" class="w-140px" />
        </div>
        <NButton type="primary" :disabled="status?.running" @click="handleCollect">
          {{ status?.running ? '采集中…' : '开始采集' }}
        </NButton>
      </div>

      <div v-if="schedule" class="mt-14px flex flex-wrap items-end gap-12px border-t border-gray-100 pt-12px dark:border-gray-800">
        <div>
          <div class="mb-6px text-12px">每日定时关键词</div>
          <NInput v-model:value="schedule.Keywords" placeholder="鸡蛋批发,鸡蛋价格" class="w-260px" />
        </div>
        <div>
          <div class="mb-6px text-12px">每天几点跑</div>
          <NInput v-model:value="schedule.RunAt" placeholder="09:30" class="w-110px" />
        </div>
        <div>
          <div class="mb-6px text-12px">笔记数 / 评论上限</div>
          <div class="flex gap-6px">
            <NInputNumber v-model:value="schedule.Notes" :min="1" :max="10" class="w-90px" />
            <NInputNumber v-model:value="schedule.CommentLimit" :min="5" :max="100" class="w-90px" />
          </div>
        </div>
        <div class="flex items-center gap-6px">
          <NSwitch v-model:value="schedule.Enabled" :checked-value="1" :unchecked-value="0" />
          <span class="text-12px">启用定时</span>
        </div>
        <NButton :loading="scheduleSaving" @click="saveScheduleCfg">保存定时</NButton>
      </div>

      <div v-if="status?.log" class="mt-12px max-h-200px overflow-auto rounded bg-gray-50 p-10px text-11px leading-relaxed dark:bg-gray-800">
        <pre class="whitespace-pre-wrap">{{ status.log }}</pre>
      </div>
    </NCard>

    <NCard title="采集历史" :bordered="false" size="small">
      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>开始时间</th>
            <th>关键词</th>
            <th class="center">笔记</th>
            <th class="center">评论</th>
            <th class="center">状态</th>
            <th>说明</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in runs" :key="row.Id" >
            <td class="py-8px text-12px op-70">{{ (row.StartedAt || '').replace('T', ' ').slice(0, 19) }}</td>
            <td>{{ row.Keywords }}</td>
            <td class="center">{{ row.NotesDone }}</td>
            <td class="center">{{ row.CommentsKept }}</td>
            <td class="center">
              <NTag size="small" :type="row.Status === 1 ? 'success' : row.Status === 0 ? 'info' : 'error'">
                {{ row.Status === 1 ? '成功' : row.Status === 0 ? '运行中' : '失败' }}
              </NTag>
            </td>
            <td class="py-8px text-12px op-60">
              {{ row.Message }}
              <NButton text type="primary" size="tiny" class="ml-6px" @click="openLog(row)">日志</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <EmptyState v-if="!runs.length" title="还没有采集记录" desc="上面点「开始采集」，跑完这里会有历史" />
    </NCard>

    <NCard title="采集关键词库" :bordered="false" size="small">
      <div class="flex flex-wrap items-end gap-12px">
        <NInput v-model:value="keywordInput" placeholder="新增关键词，如 土鸡蛋批发" class="w-320px" @keyup.enter="addKeyword" />
        <NButton type="primary" @click="addKeyword">添加</NButton>
        <NButton class="ml-auto" type="primary" ghost @click="handleCollectWithLibrary">用词库全部采集</NButton>
      </div>

      <div class="mt-12px flex flex-wrap items-center gap-8px">
        <NTag v-for="item in kwList" :key="item.Id" closable @close="removeKeyword(item.Id)">
          {{ item.Keyword }}
        </NTag>
        <span v-if="!kwList.length" class="text-12px op-40">还没有关键词</span>
      </div>
    </NCard>

    <NCard title="采集黑名单（同行 / 已成交 / 勿扰）" :bordered="false" size="small">
      <div class="flex flex-wrap items-end gap-12px">
        <NInput v-model:value="blackValue" placeholder="昵称或关键词" class="w-260px" @keyup.enter="addBlack" />
        <NInput v-model:value="blackReason" placeholder="原因（可选）" class="w-220px" />
        <NButton type="primary" @click="addBlack">添加</NButton>
      </div>

      <table class="data-table mt-12px">
        <thead>
          <tr>
            <th style="width: 90px">类型</th>
            <th>内容</th>
            <th>原因</th>
            <th style="width: 80px" class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in blackList" :key="row.Id">
            <td>
              <NTag size="small">{{ row.Kind === 'nickname' ? '昵称' : '词' }}</NTag>
            </td>
            <td>{{ row.Value }}</td>
            <td class="text-12px op-60">{{ row.Reason || '-' }}</td>
            <td class="num">
              <NButton text type="error" size="tiny" @click="removeBlack(row.Id)">删除</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!blackList.length" class="py-16px text-center text-12px op-40">名单为空</div>
    </NCard>

    <NCard title="主动私信" :bordered="false" size="small">
      <div class="flex flex-wrap items-end gap-12px">
        <div>
          <div class="mb-6px text-12px">选线索</div>
          <NSelect
            v-model:value="leadTarget"
            :options="leads.map(l => ({ label: `[${l.Grade}类 ${l.Score}分] ${l.ContactName}｜${(l.Remark || '').slice(0, 18)}`, value: l.Id }))"
            placeholder="选一条小红书线索"
            class="w-320px"
            filterable
          />
        </div>
        <div class="flex-1">
          <div class="mb-6px text-12px">消息内容（留空则 AI 生成）</div>
          <NInput v-model:value="leadMessage" placeholder="留空则 AI 生成" />
        </div>
        <NButton :loading="leadLoading" @click="handleGenLeadScript">生成话术</NButton>
        <NButton :loading="leadLoading" @click="handlePreviewLeadDm">预览</NButton>
        <NButton type="primary" :loading="leadLoading" @click="handleSendLeadDm">发送</NButton>
      </div>
      <div class="mt-8px text-11px op-40">
        列表只显示有价值线索（S/A/B 类，按分数排序）。先「生成话术」→「预览」确认 → 再「发送」。
      </div>
      <pre v-if="leadPreview" class="mt-10px max-h-200px overflow-auto whitespace-pre-wrap rounded bg-gray-50 p-10px text-11px dark:bg-gray-800">{{ leadPreview }}</pre>
    </NCard>

    <NCard title="小红书账号" :bordered="false" size="small">
      <template #header-extra>
        <NButton size="small" type="primary" @click="openDevice()">新增账号</NButton>
      </template>
      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>账号</th>
            <th>小红书号</th>
            <th class="center">浏览器</th>
            <th class="center">CDP 端口</th>
            <th class="center">状态</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in devices" :key="row.Id" >
            <td class="font-medium">{{ row.Name }}</td>
            <td>{{ row.DouyinId || '-' }}</td>
            <td class="center">{{ row.BrowserName || 'Edge' }}</td>
            <td class="center">{{ row.CdpPort }}</td>
            <td class="center">
              <NTag size="small" :type="row.Status === 1 ? 'success' : 'default'">
                {{ row.Status === 1 ? '启用' : '停用' }}
              </NTag>
            </td>
            <td class="num">
              <NButton text type="primary" size="tiny" @click="toggleDevice(row)">
                {{ row.Status === 1 ? '停用' : '启用' }}
              </NButton>
            </td>
          </tr>
        </tbody>
      </table>
    </NCard>

    <NCard title="自动回复名单（也是测试发送目标）" :bordered="false" size="small">
      <template #header-extra>
        <NButton size="small" type="primary" @click="openWhitelist()">添加白名单</NButton>
      </template>

      <table class="data-table compact-mobile">
        <thead>
          <tr>
            <th>昵称</th>
            <th>小红书号</th>
            <th>主页</th>
            <th class="center">启用</th>
            <th class="num">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in whitelist" :key="row.Id" >
            <td class="font-medium">{{ row.Nickname }}</td>
            <td>{{ row.DouyinId }}</td>
            <td class="max-w-380px truncate py-8px text-11px text-primary" :title="row.ProfileUrl">{{ row.ProfileUrl }}</td>
            <td class="center">
              <NTag size="small" type="success">是</NTag>
            </td>
            <td class="num">
              <NButton text type="primary" size="tiny" class="mr-6px" @click="openWhitelist(row)">编辑</NButton>
              <NButton text type="error" size="tiny" @click="removeWhitelist(row)">移除</NButton>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!whitelist.length" class="py-20px text-center text-12px op-40">
        还没有白名单（只有名单里的人才会被自动回复）
      </div>
    </NCard>

    <NModal v-model:show="showDevice" preset="card" class="w-560px" title="小红书账号">
      <div class="flex-col-stretch gap-12px">
        <div>
          <div class="mb-6px text-12px">备注名</div>
          <NInput v-model:value="deviceForm.Name" placeholder="如：小红书主号" />
        </div>
        <div>
          <div class="mb-6px text-12px">小红书号</div>
          <NInput v-model:value="deviceForm.DouyinId" placeholder="如：947928823" />
        </div>
        <div>
          <div class="mb-6px text-12px">浏览器程序路径</div>
          <NInput v-model:value="deviceForm.BrowserPath" placeholder="C:\...\msedge.exe" />
        </div>
        <div>
          <div class="mb-6px text-12px">用户数据目录（每个账号独立）</div>
          <NInput v-model:value="deviceForm.ProfileDir" placeholder="E:\ai\scraper\edge-profile-xhs2" />
        </div>
        <div>
          <div class="mb-6px text-12px">CDP 端口（每个账号不同）</div>
          <NInputNumber v-model:value="deviceForm.CdpPort" :min="9000" :max="9999" class="w-full" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showDevice = false">取消</NButton>
          <NButton type="primary" @click="submitDevice">保存</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="showWhitelist" preset="card" class="w-560px" title="小红书私信白名单">
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
          <NInput v-model:value="whitelistForm.Nickname" placeholder="如：冬冬瓜咚" />
        </div>
        <div>
          <div class="mb-6px text-12px">小红书号</div>
          <NInput v-model:value="whitelistForm.DouyinId" placeholder="如：947928823" />
        </div>
        <div>
          <div class="mb-6px text-12px">主页地址</div>
          <NInput v-model:value="whitelistForm.ProfileUrl" placeholder="https://www.xiaohongshu.com/user/profile/..." />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showWhitelist = false">取消</NButton>
          <NButton type="primary" @click="submitWhitelist">保存</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>
