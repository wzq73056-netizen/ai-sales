<script setup lang="ts">
import { computed } from 'vue';
import { $t } from '@/locales';
import { useAppStore } from '@/store/modules/app';
import pkg from '~/package.json';

const appStore = useAppStore();

const column = computed(() => (appStore.isMobile ? 1 : 2));

interface PkgJson {
  name: string;
  version: string;
  dependencies: PkgVersionInfo[];
  devDependencies: PkgVersionInfo[];
}

interface PkgVersionInfo {
  name: string;
  version: string;
}

const { name, version, dependencies, devDependencies } = pkg;

function transformVersionData(tuple: [string, string]): PkgVersionInfo {
  const [$name, $version] = tuple;
  return {
    name: $name,
    version: $version
  };
}

const pkgJson: PkgJson = {
  name,
  version,
  dependencies: Object.entries(dependencies).map(item => transformVersionData(item)),
  devDependencies: Object.entries(devDependencies).map(item => transformVersionData(item))
};

const latestBuildTime = BUILD_TIME;

/** 版本记录：与「今日工作台」的项目动态共用同一套文案 */
const changelog = computed(() => [
  { version: 'v1.5.0', content: $t('page.home.projectNews.desc1'), time: '2026-11-12' },
  { version: 'v1.4.0', content: $t('page.home.projectNews.desc2'), time: '2026-11-08' },
  { version: 'v1.3.0', content: $t('page.home.projectNews.desc3'), time: '2026-11-05' },
  { version: 'v1.2.0', content: $t('page.home.projectNews.desc4'), time: '2026-10-30' },
  { version: 'v1.1.0', content: $t('page.home.projectNews.desc5'), time: '2026-10-22' }
]);
</script>

<template>
  <NSpace vertical :size="16">
    <NCard :title="$t('page.about.title')" :bordered="false" size="small" segmented class="card-wrapper">
      <p>{{ $t('page.about.introduction') }}</p>
    </NCard>
    <NCard :title="$t('page.home.projectNews.title')" :bordered="false" size="small" segmented class="card-wrapper">
      <NList>
        <NListItem v-for="item in changelog" :key="item.version">
          <div class="flex items-start justify-between gap-16px">
            <div class="flex-1">
              <NTag size="small" type="primary">{{ item.version }}</NTag>
              <span class="ml-8px text-13px">{{ item.content }}</span>
            </div>
            <span class="shrink-0 text-12px op-50">{{ item.time }}</span>
          </div>
        </NListItem>
      </NList>
    </NCard>
    <NCard :title="$t('page.about.projectInfo.title')" :bordered="false" size="small" segmented class="card-wrapper">
      <NDescriptions label-placement="left" bordered size="small" :column="column">
        <NDescriptionsItem :label="$t('page.about.projectInfo.version')">
          <NTag type="primary">{{ pkgJson.version }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.about.projectInfo.latestBuildTime')">
          <NTag type="primary">{{ latestBuildTime }}</NTag>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.about.projectInfo.githubLink')">
          <a class="text-primary" :href="pkg.homepage" target="_blank" rel="noopener noreferrer">
            {{ $t('page.about.projectInfo.githubLink') }}
          </a>
        </NDescriptionsItem>
        <NDescriptionsItem :label="$t('page.about.projectInfo.previewLink')">
          <a class="text-primary" :href="pkg.website" target="_blank" rel="noopener noreferrer">
            {{ $t('page.about.projectInfo.previewLink') }}
          </a>
        </NDescriptionsItem>
      </NDescriptions>
    </NCard>
    <NCard :title="$t('page.about.prdDep')" :bordered="false" size="small" segmented class="card-wrapper">
      <NDescriptions label-placement="left" bordered size="small" :column="column">
        <NDescriptionsItem v-for="item in pkgJson.dependencies" :key="item.name" :label="item.name">
          {{ item.version }}
        </NDescriptionsItem>
      </NDescriptions>
    </NCard>
    <NCard :title="$t('page.about.devDep')" :bordered="false" size="small" segmented class="card-wrapper">
      <NDescriptions label-placement="left" bordered size="small" :column="column">
        <NDescriptionsItem v-for="item in pkgJson.devDependencies" :key="item.name" :label="item.name">
          {{ item.version }}
        </NDescriptionsItem>
      </NDescriptions>
    </NCard>
  </NSpace>
</template>

<style scoped></style>
