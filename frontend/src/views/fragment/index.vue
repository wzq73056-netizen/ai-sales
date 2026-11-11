<script setup lang="tsx">
import { computed, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { NButton, NPopconfirm } from 'naive-ui';
import {
  fetchCreateFragment,
  fetchDeleteFragment,
  fetchGetFragmentList
} from '@/service/api/knowledge';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { useAppStore } from '@/store/modules/app';

const appStore = useAppStore();
const route = useRoute();
const router = useRouter();

/** 从 URL 拿父级知识库 id（知识库列表点「分片」进来） */
const knowledgeId = computed(() => Number(route.query.knowledgeId ?? 0));

const searchParams = ref({ current: 1, size: 10 });

const { columns, data, loading, getData, mobilePagination } = useNaivePaginatedTable({
  api: () =>
    fetchGetFragmentList({
      knowledgeId: knowledgeId.value,
      page: searchParams.value.current,
      pageSize: searchParams.value.size
    }),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.current = params.page;
    searchParams.value.size = params.pageSize;
  },
  columns: () => [
    { key: 'Id', title: 'ID', width: 70, align: 'center' },
    { key: 'Idx', title: '序号', width: 80, align: 'center' },
    { key: 'DocId', title: '文档ID', width: 140 },
    {
      key: 'Content',
      title: '内容',
      minWidth: 460,
      ellipsis: { tooltip: true }
    },
    { key: 'CreatedAt', title: '创建时间', width: 180 },
    {
      key: 'operate',
      title: '操作',
      width: 100,
      align: 'center',
      render: row => (
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
      )
    }
  ]
});

const showModal = ref(false);
const submitting = ref(false);
const form = reactive({ Idx: 0, DocId: '', Content: '' });

function openAdd() {
  Object.assign(form, { Idx: 0, DocId: '', Content: '' });
  showModal.value = true;
}

async function handleSubmit() {
  if (!form.Content.trim()) {
    window.$message?.warning('内容不能为空');
    return;
  }

  submitting.value = true;

  const now = new Date().toISOString().slice(0, 19).replace('T', ' ');

  const { error } = await fetchCreateFragment({
    KnowledgeId: knowledgeId.value,
    Idx: Number(form.Idx) || 0,
    DocId: form.DocId,
    Content: form.Content,
    CreatedAt: now,
    UpdatedAt: now
  });

  submitting.value = false;

  if (error) return;

  window.$message?.success('新增成功');
  showModal.value = false;
  getData();
}

async function handleDelete(id: number) {
  const { error } = await fetchDeleteFragment(id);
  if (error) return;
  window.$message?.success('删除成功');
  getData();
}

function backToList() {
  router.push('/knowledge');
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard
      :title="`知识库分片（知识库 #${knowledgeId}）`"
      :bordered="false"
      size="small"
      class="card-wrapper sm:flex-1-hidden"
    >
      <template #header-extra>
        <div class="flex gap-8px">
          <NButton size="small" @click="backToList">返回知识库</NButton>
          <NButton type="primary" size="small" @click="openAdd">新增分片</NButton>
        </div>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="1000"
        :loading="loading"
        remote
        :row-key="row => row.Id"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>

    <NModal v-model:show="showModal" preset="card" class="w-620px" title="新增分片">
      <NForm label-placement="left" label-width="80" :show-feedback="false">
        <NFormItem label="序号">
          <NInput v-model:value="form.Idx" placeholder="片段序号，从 0 开始" />
        </NFormItem>
        <NFormItem label="文档ID">
          <NInput v-model:value="form.DocId" placeholder="可空" />
        </NFormItem>
        <NFormItem label="内容" required>
          <NInput v-model:value="form.Content" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" placeholder="分片正文" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton @click="showModal = false">取消</NButton>
          <NButton type="primary" :loading="submitting" @click="handleSubmit">确认</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped></style>
