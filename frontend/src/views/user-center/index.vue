<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { NButton, NCard, NInput, NTag, useMessage } from 'naive-ui';
import PageHeader from '@/components/business/PageHeader.vue';
import { request } from '@/service/request';

defineOptions({ name: 'UserCenter' });

const message = useMessage();

interface UserInfo {
  ID?: number;
  userName?: string;
  nickName?: string;
  phone?: string;
  email?: string;
  authority?: { authorityName?: string };
}

const saving = ref(false);
const info = ref<UserInfo>({});

const oldPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');

const canSubmit = computed(() => oldPassword.value.length > 0 && newPassword.value.length >= 6 && newPassword.value === confirmPassword.value);

async function load() {
  // 原始接口返回 body.data.userInfo，字段是 ID / userName / nickName / authority.authorityName
  const { data } = await request<{ userInfo: UserInfo }>({ url: '/user/getUserInfo' });

  if (data?.userInfo) {
    info.value = data.userInfo;
  }
}

async function changePassword() {
  if (newPassword.value !== confirmPassword.value) {
    message.error('两次输入的新密码不一致');

    return;
  }

  if (newPassword.value.length < 6) {
    message.error('新密码至少 6 位');

    return;
  }

  saving.value = true;

  const { error } = await request({
    url: '/user/changePassword',
    method: 'post',
    data: {
      password: oldPassword.value,
      newPassword: newPassword.value
    }
  });

  saving.value = false;

  if (error) return;

  message.success('密码已修改，下次登录用新密码');
  oldPassword.value = '';
  newPassword.value = '';
  confirmPassword.value = '';
}

onMounted(load);
</script>

<template>
  <div class="app-page lt-sm:overflow-auto">
    <PageHeader title="个人中心" desc="当前登录账号的信息，以及修改密码" />

    <div class="grid gap-16px lg:grid-cols-2">
      <NCard title="账号信息" :bordered="false" size="small">
        <div class="info-list">
          <div class="row">
            <span class="label">用户名</span>
            <span class="value">{{ info.userName || '-' }}</span>
          </div>
          <div class="row">
            <span class="label">昵称</span>
            <span class="value">{{ info.nickName || '-' }}</span>
          </div>
          <div class="row">
            <span class="label">角色</span>
            <span class="value">
              <NTag size="small" type="primary">{{ info.authority?.authorityName || '超级管理员' }}</NTag>
            </span>
          </div>
          <div class="row">
            <span class="label">手机号</span>
            <span class="value">{{ info.phone || '未填' }}</span>
          </div>
          <div class="row">
            <span class="label">邮箱</span>
            <span class="value">{{ info.email || '未填' }}</span>
          </div>
        </div>
        <p class="tip">账号信息由「系统管理 → 用户管理」维护。这个系统默认只有你一个账号，数据都在本机。</p>
      </NCard>

      <NCard title="修改密码" :bordered="false" size="small">
        <div class="flex flex-col gap-12px">
          <div class="field">
            <span class="label">当前密码</span>
            <NInput v-model:value="oldPassword" type="password" show-password-on="click" placeholder="输入现在的密码" />
          </div>
          <div class="field">
            <span class="label">新密码</span>
            <NInput v-model:value="newPassword" type="password" show-password-on="click" placeholder="至少 6 位" />
          </div>
          <div class="field">
            <span class="label">确认新密码</span>
            <NInput v-model:value="confirmPassword" type="password" show-password-on="click" placeholder="再输一次" />
          </div>
          <NButton type="primary" :block="true" :loading="saving" :disabled="!canSubmit" @click="changePassword">
            修改密码
          </NButton>
          <p class="tip">默认密码是 admin / 123456，建议换成自己记得住的。改完需要重新登录。</p>
        </div>
      </NCard>
    </div>
  </div>
</template>

<style scoped>
.info-list {
  display: flex;
  flex-direction: column;
}

.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--ds-border);
}

.label {
  width: 80px;
  flex: 0 0 80px;
  font-size: 12px;
  color: var(--ds-text-3);
}

.value {
  font-size: 13px;
  color: var(--ds-text);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tip {
  margin-top: 14px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--ds-text-3);
}
</style>
