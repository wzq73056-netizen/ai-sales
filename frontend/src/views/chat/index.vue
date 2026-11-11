<script setup lang="tsx">
import { nextTick, onMounted, ref } from 'vue';
import { NButton, NInput, NSelect, NTag } from 'naive-ui';
import { fetchChat, type ChatResult } from '@/service/api/chat';
import { fetchGetKnowledgeList, type KnowledgeRow } from '@/service/api/knowledge';

interface ChatTurn {
  role: 'user' | 'assistant';
  text: string;
  result?: ChatResult;
}

const question = ref('');
const topK = ref(3);
const knowledgeId = ref(0);
const loading = ref(false);
const turns = ref<ChatTurn[]>([]);
const listRef = ref<HTMLElement | null>(null);

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

async function scrollToBottom() {
  await nextTick();
  listRef.value?.scrollTo({ top: listRef.value.scrollHeight, behavior: 'smooth' });
}

async function handleSend() {
  const q = question.value.trim();

  if (!q || loading.value) return;

  turns.value.push({ role: 'user', text: q });
  question.value = '';
  loading.value = true;
  await scrollToBottom();

  const { data, error } = await fetchChat({
    knowledgeId: knowledgeId.value,
    question: q,
    topK: topK.value || 3
  });

  loading.value = false;

  if (error) {
    turns.value.push({ role: 'assistant', text: '请求失败，请稍后重试' });
  } else {
    turns.value.push({ role: 'assistant', text: (data as unknown as ChatResult)?.Answer ?? '', result: data as unknown as ChatResult });
  }

  await scrollToBottom();
}
</script>

<template>
  <div class="app-page">
    <NCard title="知识库问答" :bordered="false" size="small">
      <div class="flex flex-wrap items-center gap-12px">
        <NSelect v-model:value="knowledgeId" :options="knowledgeOptions" class="w-200px" />
        <span class="text-12px op-60">检索条数</span>
        <NInput v-model:value="topK" class="w-80px" />
        <span class="text-12px op-60">
          回答由知识库检索结果生成；检索不到时模型会直接说不知道
        </span>
      </div>
    </NCard>

    <div ref="listRef" class="flex-1 overflow-auto rounded-8px border border-gray-200 p-16px dark:border-gray-700">
      <NEmpty v-if="turns.length === 0" description="问点什么，比如：这个知识库讲了什么" />
      <div v-for="(turn, index) in turns" :key="index" class="mb-16px">
        <div v-if="turn.role === 'user'" class="flex justify-end">
          <div class="max-w-70% rounded-10px bg-primary px-14px py-10px text-white">{{ turn.text }}</div>
        </div>
        <div v-else class="flex flex-col gap-8px">
          <div class="max-w-85% rounded-10px bg-gray-100 px-14px py-10px leading-relaxed dark:bg-gray-800">
            <template v-if="turn.result?.Mode === 'llm'">
              {{ turn.result.Answer }}
            </template>
            <template v-else>
              <div class="mb-8px flex items-center gap-8px">
                <NTag type="warning" size="small">未配置大模型</NTag>
                <span class="text-12px op-60">以下是检索到的资料与将要发送的提示词</span>
              </div>
              <pre class="whitespace-pre-wrap text-12px leading-relaxed">{{ turn.result?.Prompt }}</pre>
            </template>
          </div>

          <div v-if="turn.result?.Sources?.length" class="flex flex-col gap-6px">
            <div class="text-12px op-60">引用来源</div>
            <div
              v-for="src in turn.result.Sources"
              :key="src.Id"
              class="rounded-6px border border-gray-200 px-10px py-8px text-12px dark:border-gray-700"
            >
              <NTag size="small" type="info">score {{ src.Score }}</NTag>
              <span class="ml-8px">{{ src.Content }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flex gap-12px">
      <NInput v-model:value="question" placeholder="请输入问题，回车发送" @keyup.enter="handleSend" />
      <NButton type="primary" :loading="loading" @click="handleSend">发送</NButton>
    </div>
  </div>
</template>

<style scoped></style>
