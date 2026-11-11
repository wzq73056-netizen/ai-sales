<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { NButton, NCard, NInput, NInputNumber, NTag, useMessage } from 'naive-ui';
import PageHeader from '@/components/business/PageHeader.vue';
import { fetchGetLLMConfig, fetchSaveLLMConfig, fetchTestLLMConfig, type LLMConfigForm } from '@/service/api/llm';

defineOptions({ name: 'LLMSettings' });

const message = useMessage();

const loading = ref(false);
const saving = ref(false);
const testing = ref(false);
const testResult = ref('');
const filePath = ref('');

const form = ref<LLMConfigForm>({
  baseUrl: 'https://api.deepseek.com/v1',
  apiKey: '',
  model: 'deepseek-chat',

  temperature: 0
});

async function load() {
  loading.value = true;

  const { data } = await fetchGetLLMConfig();

  if (data) {
    form.value = {
      baseUrl: data.baseUrl || '',
      apiKey: data.apiKey || '',
      model: data.model || '',

      temperature: data.temperature || 0
    };
    filePath.value = data.filePath || '';
  }

  loading.value = false;
}

async function save() {
  saving.value = true;
  testResult.value = '';

  const { error } = await fetchSaveLLMConfig({
    baseUrl: form.value.baseUrl.trim(),
    apiKey: form.value.apiKey.trim(),
    model: form.value.model.trim(),

    temperature: form.value.temperature || 0
  });

  saving.value = false;

  if (error) return;

  message.success('已保存，立即生效');
  load();
}

async function test() {
  testing.value = true;
  testResult.value = '';

  const { data, error } = await fetchTestLLMConfig();

  testing.value = false;

  if (error) {
    testResult.value = `失败：${(error as any)?.message || '未知错误'}`;

    return;
  }

  testResult.value = `连接正常，模型回复：${data?.reply ?? ''}`;
}

onMounted(load);
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="模型配置" desc="AI 回复、话术生成都用这里的模型，改完即时生效">
      <template #actions>
        <NButton size="small" :loading="testing" @click="test">测试连接</NButton>
        <NButton size="small" type="primary" :loading="saving" @click="save">保存</NButton>
      </template>
    </PageHeader>

    <NCard :bordered="false" size="small" class="flex-1">
      <div class="llm-form">
        <div class="row">
          <label>接口地址</label>
          <NInput v-model:value="form.baseUrl" placeholder="https://api.deepseek.com/v1" />
          <span class="hint">OpenAI 兼容接口，结尾不带 /chat/completions</span>
        </div>

        <div class="row">
          <label>API Key</label>
          <NInput
            v-model:value="form.apiKey"
            type="password"
            show-password-on="click"
            placeholder="sk-..."
          />
          <span class="hint">显示为 sk-c****e771 表示已保存，不改就留着不动</span>
        </div>

        <div class="row">
          <label>模型名</label>
          <NInput v-model:value="form.model" placeholder="deepseek-chat" />
          <span class="hint">例如 deepseek-chat / gpt-4o-mini / qwen-plus</span>
        </div>

        <div class="row">
          <label>温度</label>
          <NInputNumber v-model:value="form.temperature" :min="0" :max="2" :step="0.1" style="width: 160px" />
          <span class="hint">0 最稳定，私信场景建议 0.5–0.8</span>
        </div>


        <div v-if="testResult" class="result" :class="{ ok: testResult.startsWith('连接正常') }">
          {{ testResult }}
        </div>

        <div v-if="filePath" class="path">
          <NTag size="small">配置文件</NTag>
          <code>{{ filePath }}</code>
        </div>
      </div>
    </NCard>
  </div>
</template>

<style scoped>
.llm-form {
  max-width: 720px;
}

.row {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 8px 12px;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid var(--ds-border);
}

.row label {
  font-size: 13px;
  color: var(--ds-text-2);
}

.row .hint {
  grid-column: 2;
  font-size: 11px;
  color: var(--ds-text-3);
}

.result {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 12px;
  color: #b91c1c;
  background: rgba(185, 28, 28, 0.1);
}

.result.ok {
  color: #15803d;
  background: rgba(21, 128, 61, 0.1);
}

.path {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--ds-text-3);
}

@media (max-width: 767px) {
  .row {
    grid-template-columns: 1fr;
  }

  .row .hint {
    grid-column: 1;
  }
}
</style>
