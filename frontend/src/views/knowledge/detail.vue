<script setup lang="tsx">
import { computed, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { NButton, NInput, NPopconfirm } from 'naive-ui';
import {
  fetchDeleteDocument,
  fetchGetDocumentList,
  fetchGetFragmentList,
  fetchGetKnowledgeList,
  fetchUpdateKnowledge,
  fetchUploadDocument,
  type FragmentRow,
  type KnowledgeDocument,
  type KnowledgeRow
} from '@/service/api/knowledge';
import { fetchSearchFragments } from '@/service/api/retrieval';

const props = defineProps<{ id: number }>();

const router = useRouter();

const knowledgeId = computed(() => props.id);
const kb = ref<KnowledgeRow | null>(null);
const documents = ref<KnowledgeDocument[]>([]);
const fragments = ref<FragmentRow[]>([]);
const loading = ref(false);
const uploading = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);
const activeTab = ref<'docs' | 'search' | 'settings'>('docs');
const docKeyword = ref('');
const editForm = reactive({ Name: '', Description: '' });

const searchText = ref('');
const searchResults = ref<FragmentRow[]>([]);
const searching = ref(false);
const minScore = ref(0);

const filteredDocs = computed(() => {
  const kw = docKeyword.value.trim();

  if (!kw) return documents.value;

  return documents.value.filter(item => item.Name?.includes(kw));
});

async function load() {
  if (!knowledgeId.value) return;

  loading.value = true;

  const [{ data }, { data: docs }, { data: frag }] = await Promise.all([
    fetchGetKnowledgeList({ page: 1, pageSize: 200 }),
    fetchGetDocumentList({ knowledgeId: knowledgeId.value, page: 1, pageSize: 100 }),
    fetchGetFragmentList({ knowledgeId: knowledgeId.value, page: 1, pageSize: 200 })
  ]);

  loading.value = false;

  kb.value = data?.records?.find((item: KnowledgeRow) => item.Id === knowledgeId.value) ?? null;

  if (kb.value) {
    editForm.Name = kb.value.Name;
    editForm.Description = kb.value.Description ?? '';
  }

  if (docs?.records) documents.value = docs.records;
  if (frag?.records) fragments.value = frag.records;
}

function pickFile() {
  fileInput.value?.click();
}

async function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];

  if (!file) return;

  uploading.value = true;
  const { error } = await fetchUploadDocument(knowledgeId.value, file);
  uploading.value = false;
  input.value = '';

  if (error) return;

  window.$message?.success(`已上传「${file.name}」，自动切分完成`);
  load();
}

async function handleDeleteDocument(id: number) {
  const { error } = await fetchDeleteDocument(id);

  if (error) return;

  window.$message?.success('已删除');
  load();
}

async function handleSearch() {
  if (!searchText.value.trim()) {
    window.$message?.warning('请输入检索内容');
    return;
  }

  searching.value = true;
  const { data, error } = await fetchSearchFragments({
    knowledgeId: knowledgeId.value,
    query: searchText.value,
    topK: 5,
    minScore: minScore.value
  });
  searching.value = false;

  if (error) return;

  searchResults.value = (data as unknown as FragmentRow[]) ?? [];
}

async function handleSaveSettings() {
  if (!kb.value) return;

  const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
  const { error } = await fetchUpdateKnowledge({
    Id: kb.value.Id,
    Name: editForm.Name,
    Description: editForm.Description,
    Share: kb.value.Share,
    Status: kb.value.Status,
    UpdatedAt: now
  });

  if (error) return;

  window.$message?.success('已保存');
  load();
}

function formatSize(size: number) {
  if (!size) return '-';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;

  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

function goBack() {
  router.push('/knowledge');
}

watch(
  () => props.id,
  id => {
    if (id) load();
  },
  { immediate: true }
);
</script>

<template>
  <div class="flex gap-16px">
    <!-- 二级菜单 -->
    <div class="w-140px flex-none">
      <div
        v-for="item in [
          { key: 'docs', label: '文档内容' },
          { key: 'search', label: '搜索测试' },
          { key: 'settings', label: '设置' }
        ]"
        :key="item.key"
        class="mb-4px cursor-pointer rounded-8px px-12px py-9px text-13px transition"
        :class="activeTab === item.key ? 'bg-primary/10 text-primary font-medium' : 'hover:bg-gray-100 dark:hover:bg-gray-800'"
        @click="activeTab = item.key as any"
      >
        {{ item.label }}
      </div>
    </div>

    <!-- 内容区 -->
    <div class="min-w-0 flex-1">
      <NCard :bordered="false" size="small">
        <div style="display: flex; align-items: center; width: 100%; gap: 12px">
          <div style="flex: 1; min-width: 0">
            <div class="flex items-center gap-8px">
              <span class="text-16px font-semibold">{{ kb?.Name || '知识库' }}</span>
              <NTag size="small" type="info" :bordered="false">VECTOR DB</NTag>
            </div>
            <div class="mt-4px text-12px op-60">
              管理知识库文件，系统将根据这些文档进行语义索引与对话回复。
            </div>
          </div>
          <NButton v-if="activeTab === 'docs'" type="primary" :loading="uploading" @click="pickFile">
            <template #icon>
              <SvgIcon icon="carbon:add" />
            </template>
            添加新文件
          </NButton>
          <NButton v-else text @click="goBack">返回知识库列表</NButton>
        </div>
      </NCard>

      <NCard v-if="activeTab === 'docs'" :bordered="false" size="small" class="mt-16px">
        <div class="mb-12px flex items-center gap-12px">
          <div class="whitespace-nowrap text-14px font-medium">
            <span class="text-primary">|</span> 所有文档 ({{ documents.length }})
          </div>
          <div class="flex-1" />
          <NInput v-model:value="docKeyword" placeholder="快速检索文档..." clearable class="w-260px" />
        </div>

        <table class="w-full text-13px">
          <thead>
            <tr class="border-b border-gray-200 text-12px op-60 dark:border-gray-700">
              <th class="py-10px text-left font-normal">文档名称</th>
              <th class="py-10px text-left font-normal">格式</th>
              <th class="py-10px text-left font-normal">大小</th>
              <th class="py-10px text-left font-normal">片段数</th>
              <th class="py-10px text-left font-normal">上传时间</th>
              <th class="py-10px text-right font-normal">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="doc in filteredDocs" :key="doc.Id" class="border-b border-gray-100 dark:border-gray-800">
              <td class="py-10px">{{ doc.Name }}</td>
              <td class="py-10px uppercase op-70">{{ doc.Format || '-' }}</td>
              <td class="py-10px op-70">{{ formatSize(doc.Size) }}</td>
              <td class="py-10px op-70">{{ doc.ChunkCount }}</td>
              <td class="py-10px op-70">{{ doc.CreatedAt }}</td>
              <td class="py-10px text-right">
                <NPopconfirm @positive-click="handleDeleteDocument(doc.Id)">
                  <template #trigger>
                    <NButton text type="error" size="tiny">删除</NButton>
                  </template>
                  确认删除该文档及其片段？
                </NPopconfirm>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="!filteredDocs.length" class="py-60px text-center text-12px op-50">
          没有任何文档，点击右上角「添加新文件」开始上传
        </div>

        <input ref="fileInput" type="file" class="hidden" accept=".txt,.md,.csv,.pdf,.docx" @change="onFileChange" />
      </NCard>

      <NCard v-else-if="activeTab === 'search'" :bordered="false" size="small" class="mt-16px">
        <div class="flex items-center gap-12px">
          <NInput v-model:value="searchText" placeholder="输入问题，测试检索效果" @keyup.enter="handleSearch" />
          <div class="flex items-center gap-6px whitespace-nowrap text-12px op-60">
            相似度≥
            <NInputNumber v-model:value="minScore" :min="0" :max="100" :step="5" class="w-90px" />
          </div>
          <NButton type="primary" :loading="searching" @click="handleSearch">检索</NButton>
        </div>

        <div class="mt-16px flex-col-stretch gap-10px">
          <NEmpty v-if="!searchResults.length" description="输入内容后点击检索" />
          <div
            v-for="(hit, index) in searchResults"
            :key="hit.Id"
            class="rounded-8px border border-gray-200 p-12px dark:border-gray-700"
          >
            <div class="mb-8px flex flex-wrap items-center gap-8px text-12px">
              <NTag size="small" type="info">#{{ index + 1 }}</NTag>
              <span class="op-60">相似度</span>
              <NTag size="small" :type="hit.Similarity >= 80 ? 'success' : hit.Similarity >= 50 ? 'warning' : 'default'">
                {{ hit.Similarity }}%
              </NTag>
              <span class="op-60">来源</span>
              <span class="font-medium">{{ hit.DocumentName }}</span>
              <span class="op-40">第 {{ hit.Idx }} 段</span>
            </div>
            <div class="text-13px leading-relaxed">{{ hit.Content }}</div>
          </div>
        </div>
      </NCard>

      <NCard v-else :bordered="false" size="small" class="mt-16px">
        <div class="w-520px">
          <div class="mb-6px text-13px">知识库名称</div>
          <NInput v-model:value="editForm.Name" :maxlength="20" show-count />
          <div class="mb-6px mt-16px text-13px">知识库描述</div>
          <NInput
            v-model:value="editForm.Description"
            type="textarea"
            :maxlength="200"
            show-count
            :autosize="{ minRows: 4 }"
          />
          <NButton type="primary" class="mt-16px" @click="handleSaveSettings">保存</NButton>
        </div>
      </NCard>
    </div>
  </div>
</template>

<style scoped></style>
