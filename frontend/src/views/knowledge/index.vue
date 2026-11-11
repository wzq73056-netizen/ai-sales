<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { NButton, NInput, NSpin } from 'naive-ui';
import { fetchDeleteKnowledge, fetchGetKnowledgeList, fetchGetKnowledgeStats, type KnowledgeRow } from '@/service/api/knowledge';
import SvgIcon from '@/components/custom/svg-icon.vue';
import KnowledgeDetail from './detail.vue';
import KnowledgeCreate from './create.vue';
import PageHeader from '@/components/business/PageHeader.vue';
import EmptyState from '@/components/business/EmptyState.vue';

/** ISO 时间 → 2026-11-11 12:21 */
function shortTime(value?: string | null) {
  if (!value) return '-';

  return String(value).replace('T', ' ').replace(/\.\d+/, '').replace(/[+-]\d{2}:\d{2}$/, '').slice(0, 16);
}

const route = useRoute();
const router = useRouter();

/**
 * 三个视图共用一个路由组件：
 *   /knowledge          → 卡片列表
 *   /knowledge/detail   → 详情（?id=）
 *   /knowledge/create   → 新建
 * 这样不用给每次页面改动都新增一条路由（动态路由模式下新增路由要走后端菜单）
 */
const mode = computed(() => {
  if (route.path.endsWith('/detail')) return 'detail';
  if (route.path.endsWith('/create')) return 'create';
  return 'list';
});

const loading = ref(false);
const keyword = ref('');
const list = ref<KnowledgeRow[]>([]);
const stats = ref<Record<number, number>>({});

const filtered = computed(() => {
  const kw = keyword.value.trim();

  if (!kw) return list.value;

  return list.value.filter(item => item.Name?.includes(kw) || item.Description?.includes(kw));
});

async function load() {
  loading.value = true;
  const [{ data }, { data: statMap }] = await Promise.all([
    fetchGetKnowledgeList({ page: 1, pageSize: 200 }),
    fetchGetKnowledgeStats()
  ]);
  loading.value = false;

  if (data?.records) list.value = data.records;
  if (statMap) stats.value = statMap;
}

function goCreate() {
  router.push({ path: '/knowledge/create' });
}

function goDetail(row: KnowledgeRow) {
  router.push({ path: '/knowledge/detail', query: { id: String(row.Id) } });
}

async function handleDelete(row: KnowledgeRow) {
  const { error } = await fetchDeleteKnowledge(row.Id);

  if (error) return;

  window.$message?.success('已删除');
  load();
}

watch(
  () => route.path,
  path => {
    if (path.endsWith('/knowledge')) load();
  },
  { immediate: true }
);
</script>

<template>
  <KnowledgeCreate v-if="mode === 'create'" />

  <KnowledgeDetail v-else-if="mode === 'detail'" :id="Number(route.query.id ?? 0)" />

  <div v-else class="app-page lt-sm:overflow-auto">
    <PageHeader title="知识库" desc="AI 回复和话术生成都从这里取资料，片段越具体效果越好">
      <template #actions>
        <NInput v-model:value="keyword" placeholder="搜索知识库" clearable style="width: 220px" />
        <NButton size="small" :loading="loading" @click="load">刷新</NButton>
        <NButton size="small" type="primary" @click="goCreate">新建知识库</NButton>
      </template>
    </PageHeader>

    <NCard :bordered="false" size="small" class="flex-1">
      <NSpin :show="loading">
        <table class="data-table mobile-cards">
          <thead>
            <tr>
              <th>知识库</th>
              <th class="num">片段数</th>
              <th>描述</th>
              <th class="num">创建时间</th>
              <th class="num">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filtered" :key="item.Id" class="cursor-pointer" @click="goDetail(item)">
              <td class="font-medium" data-label="知识库">
                <div class="flex items-center gap-8px">
                  <span class="h-24px w-24px flex-center rounded-6px bg-primary/10 text-13px text-primary">
                    <SvgIcon icon="carbon:document" />
                  </span>
                  <span class="truncate">{{ item.Name }}</span>
                </div>
              </td>
              <td class="num" data-label="片段数">{{ stats[item.Id] ?? 0 }}</td>
              <td class="truncate-cell text-12px op-70" data-label="描述">{{ item.Description || '暂无描述' }}</td>
              <td class="num text-12px op-50" data-label="创建时间">{{ shortTime(item.CreatedAt) }}</td>
              <td class="num" data-label="操作">
                <NButton text type="error" size="tiny" @click.stop="handleDelete(item)">删除</NButton>
              </td>
            </tr>
          </tbody>
        </table>

        <EmptyState v-if="!loading && !filtered.length" title="还没有知识库" desc="新建一个，把产品资料和话术导进去">
          <NButton size="small" type="primary" @click="goCreate">新建知识库</NButton>
        </EmptyState>
      </NSpin>
    </NCard>
  </div>
</template>