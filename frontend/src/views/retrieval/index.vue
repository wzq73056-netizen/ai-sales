<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NButton, NInput, NInputNumber, NSelect, NTag } from 'naive-ui';
import { fetchGetKnowledgeList, type KnowledgeRow } from '@/service/api/knowledge';
import { fetchSearchFragments, type SearchHit } from '@/service/api/retrieval';

const query = ref('');
const topK = ref(5);
const knowledgeId = ref(0);
const loading = ref(false);
const results = ref<SearchHit[]>([]);
const searched = ref(false);

const knowledgeOptions = ref<{ label: string; value: number }[]>([{ label: '全部知识库', value: 0 }]);

onMounted(async () => {
  const { data } = await fetchGetKnowledgeList({ page: 1, pageSize: 100 });

  if (data?.records?.length) {
    knowledgeOptions.value = [
      { label: '全部知识库', value: 0 },
      ...data.records.map((item: KnowledgeRow) => ({ label: item.Name, value: item.Id }))
    ];
  }
});

async function handleSearch() {
  const q = query.value.trim();

  if (!q) {
    window.$message?.warning('请输入检索内容');
    return;
  }

  loading.value = true;
  const { data, error } = await fetchSearchFragments({
    knowledgeId: knowledgeId.value,
    query: q,
    topK: topK.value || 5
  });
  loading.value = false;

  if (error) return;

  results.value = (data as unknown as SearchHit[]) ?? [];
  searched.value = true;
}
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <NCard title="知识检索" :bordered="false" size="small">
      <div class="flex flex-wrap items-center gap-12px">
        <NSelect v-model:value="knowledgeId" :options="knowledgeOptions" class="w-200px" />
        <NInput
          v-model:value="query"
          placeholder="输入要检索的内容，例如：产品定价"
          class="flex-1"
          @keyup.enter="handleSearch"
        />
        <NInputNumber v-model:value="topK" :min="1" :max="20" class="w-120px">
          <template #prefix>Top</template>
        </NInputNumber>
        <NButton type="primary" :loading="loading" @click="handleSearch">检索</NButton>
      </div>
    </NCard>

    <NCard title="检索结果" :bordered="false" size="small" class="flex-1">
      <NEmpty v-if="searched && results.length === 0" description="没有命中任何分片" />
      <NEmpty v-else-if="!searched" description="输入内容后点击检索" />
      <div v-else class="flex-col-stretch gap-12px">
        <div
          v-for="(hit, index) in results"
          :key="hit.Id"
          class="rounded-8px border border-gray-200 p-14px dark:border-gray-700"
        >
          <div class="mb-8px flex items-center gap-8px">
            <NTag type="info" size="small">#{{ index + 1 }}</NTag>
            <NTag size="small">分片 {{ hit.Idx }}</NTag>
            <NTag v-if="hit.DocId" size="small">{{ hit.DocId }}</NTag>
            <span class="text-12px op-60">知识库 {{ hit.KnowledgeId }}</span>
            <div class="flex-1" />
            <span class="text-13px text-primary font-semibold">score {{ hit.Score }}</span>
          </div>
          <div class="leading-relaxed">{{ hit.Content }}</div>
        </div>
      </div>
    </NCard>
  </div>
</template>

<style scoped></style>
