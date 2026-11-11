<script setup lang="tsx">
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { NButton, NInput } from 'naive-ui';
import { fetchCreateKnowledge } from '@/service/api/knowledge';
import { useAuthStore } from '@/store/modules/auth';
import SvgIcon from '@/components/custom/svg-icon.vue';

const router = useRouter();
const authStore = useAuthStore();

const submitting = ref(false);
const form = reactive({
  Name: '',
  Description: '',
  Share: 0,
  Status: 1
});

async function handleSubmit() {
  if (!form.Name.trim()) {
    window.$message?.warning('请输入知识库名称');
    return;
  }

  submitting.value = true;

  const now = new Date().toISOString().slice(0, 19).replace('T', ' ');
  const { error } = await fetchCreateKnowledge({
    ...form,
    UserId: Number(authStore.userInfo.userId) || 1,
    RequestCount: 0,
    CreatedAt: now,
    UpdatedAt: now
  });

  submitting.value = false;

  if (error) return;

  window.$message?.success('创建成功');
  router.push('/knowledge');
}

function goBack() {
  router.back();
}
</script>

<template>
  <div class="relative h-full flex-col">
    <div class="mb-4 flex items-center">
      <NButton text @click="goBack">
        <template #icon>
          <SvgIcon icon="carbon:chevron-left" />
        </template>
        返回上一步
      </NButton>
      <div class="flex-1" />
      <NTag size="small" type="info" :bordered="false">VECTOR MODE</NTag>
    </div>

    <div class="flex-1 flex-col items-center pt-32px">
      <div class="text-center">
        <div class="text-22px font-semibold">创建新的知识库</div>
        <div class="mt-6px text-12px op-60">为您的 AI 提供精准的私有数据支撑</div>
      </div>

      <div class="mt-28px w-520px rounded-12px bg-gray-50 p-20px dark:bg-gray-800">
        <div class="mb-6px">
          <span class="text-error">*</span>
          <span class="ml-4px text-13px">知识库名称</span>
        </div>
        <NInput v-model:value="form.Name" placeholder="请输入知识库名称" :maxlength="20" show-count />

        <div class="mb-6px mt-18px text-13px">知识库描述</div>
        <NInput
          v-model:value="form.Description"
          type="textarea"
          placeholder="请输入知识库描述"
          :maxlength="200"
          show-count
          :autosize="{ minRows: 5, maxRows: 8 }"
        />

        <div class="mb-6px mt-18px text-13px">知识库类型</div>
        <NSelect
          :value="'vector'"
          :options="[{ label: '向量知识库', value: 'vector' }]"
        />
      </div>

      <div class="mt-60px w-520px">
        <NButton type="primary" size="large" block :loading="submitting" @click="handleSubmit">立即开启知识库</NButton>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
