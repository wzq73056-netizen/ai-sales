<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { NButton, NCard, NTag } from 'naive-ui';
import PageHeader from '@/components/business/PageHeader.vue';
import { fetchFunnelStats, fetchScriptStats, type FunnelStats, type ScriptStats } from '@/service/api/wechat';
import { fetchLeadAnalytics } from '@/service/api/lead';

defineOptions({ name: 'Analytics' });

const data = ref<FunnelStats | null>(null);
const loading = ref(false);
const scriptStats = ref<ScriptStats | null>(null);
const leadAnalytics = ref<any>(null);

const maxCategory = computed(() => {
  const list = data.value?.categories || [];

  return Math.max(...list.map(item => item.count), 1);
});

async function load() {
  loading.value = true;
  const [{ data: payload }, { data: scripts }, { data: analytics }] = await Promise.all([
    fetchFunnelStats(),
    fetchScriptStats(),
    fetchLeadAnalytics()
  ]);

  if (payload) data.value = payload;
  if (scripts) scriptStats.value = scripts;
  if (analytics) leadAnalytics.value = analytics;

  loading.value = false;
}

onMounted(load);
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="报表" desc="来源分布、跟进健康度、阶段转化、话术效果">
      <template #actions>
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
      </template>
    </PageHeader>

    <template v-if="data">
      <div class="grid grid-cols-4 gap-12px">
        <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
          <div class="text-12px op-50">客户总数</div>
          <div class="mt-6px text-28px font-semibold">{{ data.total }}</div>
        </div>
        <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
          <div class="text-12px op-50">已成交</div>
          <div class="mt-6px text-28px font-semibold text-green-600">{{ data.dealCount }}</div>
        </div>
        <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
          <div class="text-12px op-50">成交率</div>
          <div class="mt-6px text-28px font-semibold">{{ data.dealRate.toFixed(1) }}%</div>
        </div>
        <div class="rounded border border-gray-200 bg-white p-16px dark:border-gray-700 dark:bg-gray-900">
          <div class="text-12px op-50">平均分 / 工单</div>
          <div class="mt-6px text-28px font-semibold">
            {{ data.avgScore }}<span class="text-14px op-40"> / {{ data.handoffTotal }}</span>
          </div>
        </div>
      </div>

      <NCard v-if="leadAnalytics?.sides?.length" title="方向对比（谁买谁卖）" :bordered="false" size="small">
        <table class="data-table">
          <thead>
            <tr>
              <th>方向</th>
              <th class="num">线索</th>
              <th class="num">A 类</th>
              <th class="num">有联系方式</th>
              <th class="num">已转客户</th>
              <th class="num">成交</th>
              <th class="num">转化率</th>
              <th class="num">均分</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in leadAnalytics.sides" :key="row.Side">
              <td class="font-medium">
                <NTag size="small" :type="row.Side === 'supplier' ? 'warning' : 'info'">{{ row.Name }}</NTag>
              </td>
              <td class="num">{{ row.Total }}</td>
              <td class="num">{{ row.GradeA }}</td>
              <td class="num">{{ row.WithContact }}</td>
              <td class="num">{{ row.Converted }}</td>
              <td class="num">{{ row.Deals }}</td>
              <td class="num">{{ row.ConvertRate.toFixed(1) }}%</td>
              <td class="num">{{ row.AvgScore.toFixed(0) }}</td>
            </tr>
          </tbody>
        </table>

        <div v-if="!leadAnalytics.sides.some(s => s.Side === 'supplier')" class="hint-row">
          还没有供货线索。想同时收「供我方」的货源，用「鸡蛋 货源」「鸡蛋 一手货源」这类关键词采一轮。
        </div>
      </NCard>

      <div class="grid grid-cols-2 gap-16px">
        <NCard title="线索来源分布" :bordered="false" size="small">
          <table class="data-table">
            <thead>
              <tr>
                <th>来源</th>
                <th class="num">总数</th>
                <th class="num">A 类</th>
                <th class="num">B 类</th>
                <th class="num">有联系方式</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in leadAnalytics?.sources || []" :key="row.Source">
                <td class="font-medium">{{ row.Source === 'douyin' ? '抖音' : row.Source === 'xhs' ? '小红书' : row.Source === 'import' ? '导入' : row.Source }}</td>
                <td class="num">{{ row.Total }}</td>
                <td class="num text-green-600">{{ row.GradeA }}</td>
                <td class="num">{{ row.GradeB }}</td>
                <td class="num">{{ row.WithContact }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="!(leadAnalytics?.sources || []).length" class="py-16px text-center text-12px op-40">暂无数据</div>
        </NCard>

        <NCard title="跟进健康度" :bordered="false" size="small">
          <div class="grid grid-cols-2 gap-12px">
            <div>
              <div class="text-12px op-50">客户总数</div>
              <div class="text-20px font-semibold">{{ leadAnalytics?.health?.CustomerTotal ?? 0 }}</div>
            </div>
            <div>
              <div class="text-12px op-50">逾期跟进</div>
              <div class="text-20px font-semibold" :class="(leadAnalytics?.health?.Overdue ?? 0) > 0 ? 'text-red-500' : ''">
                {{ leadAnalytics?.health?.Overdue ?? 0 }}
              </div>
            </div>
            <div>
              <div class="text-12px op-50">逾期率</div>
              <div class="text-20px font-semibold">{{ (leadAnalytics?.health?.OverdueRate ?? 0).toFixed(1) }}%</div>
            </div>
            <div>
              <div class="text-12px op-50">沉默 ≥7 天</div>
              <div class="text-20px font-semibold" :class="(leadAnalytics?.health?.Silent7 ?? 0) > 0 ? 'text-orange-500' : ''">
                {{ leadAnalytics?.health?.Silent7 ?? 0 }}
              </div>
            </div>
            <div>
              <div class="text-12px op-50">平均沉默天数</div>
              <div class="text-20px font-semibold">{{ (leadAnalytics?.health?.AvgSilentDays ?? 0).toFixed(1) }}</div>
            </div>
            <div>
              <div class="text-12px op-50">已成交</div>
              <div class="text-20px font-semibold text-green-600">{{ leadAnalytics?.health?.DealCount ?? 0 }}</div>
            </div>
          </div>
        </NCard>
      </div>

      <div class="grid grid-cols-2 gap-16px">
        <NCard title="阶段转化" :bordered="false" size="small">
          <div v-for="(stage, index) in data.stages" :key="stage.name" class="mb-12px">
            <div class="mb-4px flex items-center justify-between text-12px">
              <span>{{ index + 1 }}. {{ stage.name }}</span>
              <span class="op-60">{{ stage.count }} 人 · {{ stage.rate.toFixed(0) }}%</span>
            </div>
            <div class="bar-track">
              <div
                class="bar-fill"
                :style="{ width: `${Math.max(stage.rate, 2)}%` }"
              />
            </div>
          </div>
        </NCard>

        <NCard title="工单类别" :bordered="false" size="small">
          <div v-if="!data.categories.length" class="py-20px text-center text-12px op-40">暂无工单</div>
          <div v-for="item in data.categories" :key="item.name" class="mb-10px flex items-center gap-8px">
            <NTag size="small" type="info">{{ item.name }}</NTag>
            <div class="bar-track is-thin">
              <div class="bar-fill is-thin is-orange" :style="{ width: `${(item.count / maxCategory) * 100}%` }" />
            </div>
            <span class="w-24px text-right text-12px">{{ item.count }}</span>
          </div>
        </NCard>
      </div>

      <NCard title="主动私信话术效果" :bordered="false" size="small">
        <div class="mb-12px grid grid-cols-3 gap-12px">
          <div class="rounded border border-gray-200 p-12px dark:border-gray-700">
            <div class="text-12px op-50">已发送</div>
            <div class="mt-4px text-22px font-semibold">{{ scriptStats?.Total ?? 0 }}</div>
          </div>
          <div class="rounded border border-gray-200 p-12px dark:border-gray-700">
            <div class="text-12px op-50">对方已回</div>
            <div class="mt-4px text-22px font-semibold text-green-600">{{ scriptStats?.Replied ?? 0 }}</div>
          </div>
          <div class="rounded border border-gray-200 p-12px dark:border-gray-700">
            <div class="text-12px op-50">回复率</div>
            <div class="mt-4px text-22px font-semibold">{{ (scriptStats?.Rate ?? 0).toFixed(1) }}%</div>
          </div>
        </div>

        <table class="data-table">
          <thead>
            <tr>
              <th>话术</th>
              <th style="width: 70px" class="center">平台</th>
              <th style="width: 70px" class="num">发送</th>
              <th style="width: 70px" class="num">回复</th>
              <th style="width: 80px" class="num">回复率</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(item, index) in scriptStats?.Items || []" :key="index">
              <td class="truncate-cell">{{ item.Script }}</td>
              <td class="center">{{ item.Platform === 'xhs' ? '小红书' : '抖音' }}</td>
              <td class="num">{{ item.Sent }}</td>
              <td class="num">{{ item.Replied }}</td>
              <td class="num">{{ item.Rate.toFixed(0) }}%</td>
            </tr>
          </tbody>
        </table>

        <div v-if="!(scriptStats?.Items || []).length" class="py-16px text-center text-12px op-40">
          还没有发送记录（主动私信发出后自动统计）
        </div>
      </NCard>

      <NCard title="最近 7 天" :bordered="false" size="small">
        <table class="w-full text-13px">
          <thead>
            <tr class="border-b border-gray-200 text-12px op-60 dark:border-gray-700">
              <th class="py-8px text-left font-normal">日期</th>
              <th class="py-8px text-center font-normal">新客户</th>
              <th class="py-8px text-center font-normal">新工单</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="day in data.trend" :key="day.date" class="border-b border-gray-100 dark:border-gray-800">
              <td class="py-8px">{{ day.date }}</td>
              <td class="py-8px text-center">{{ day.customers }}</td>
              <td class="py-8px text-center">{{ day.handoffs }}</td>
            </tr>
          </tbody>
        </table>

        <div v-if="!leadAnalytics.sides.some(s => s.Side === 'supplier')" class="hint-row">
          还没有供货线索。想同时收「供我方」的货源，用「鸡蛋 货源」「鸡蛋 一手货源」这类关键词采一轮。
        </div>
      </NCard>
    </template>

    <div v-else class="py-60px text-center text-12px op-40">加载中…</div>
  </div>
</template>

<style scoped>
.side-card {
  min-width: 140px;
  padding: 12px 16px;
  border: 1px solid var(--ds-border);
  border-radius: 10px;
  background: var(--ds-surface);
}
</style>
<style scoped>
.bar-track {
  height: 20px;
  width: 100%;
  overflow: hidden;
  border-radius: 6px;
  background: rgba(148, 163, 184, 0.18);
}

.bar-track.is-thin {
  height: 8px;
  flex: 1;
}

.bar-fill {
  height: 100%;
  border-radius: 6px;
  background: linear-gradient(90deg, #3b82f6, #2563eb);
  transition: width 300ms ease;
}

.bar-fill.is-thin {
  border-radius: 6px;
}

.hint-row {
  margin-top: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 12px;
  color: var(--ds-text-2);
  background: rgba(148, 163, 184, 0.1);
}

html.dark .bar-track {
  background: rgba(148, 163, 184, 0.16);
}

.bar-fill.is-orange {
  background: linear-gradient(90deg, #fb923c, #f97316);
}

html.dark .bar-fill {
  background: linear-gradient(90deg, #60a5fa, #3b82f6);
}
</style>