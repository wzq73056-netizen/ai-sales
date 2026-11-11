param(
  [Parameter(Mandatory=$true)][string]$Pkg,      # 包名，如 music
  [Parameter(Mandatory=$true)][string]$Struct,   # 结构体名，如 Music
  [Parameter(Mandatory=$true)][string]$Label,    # 中文名
  [Parameter(Mandatory=$true)][string]$Columns   # "Id:ID,Name:名称,Status:状态"
)

$ErrorActionPreference = 'Stop'
$root = 'E:\ai\soybean-admin'
$apiDir = Join-Path $root 'src\service\api'
$viewsDir = Join-Path $root "src\views\$Pkg"
New-Item -ItemType Directory -Force -Path $viewsDir | Out-Null

# 列定义 → tsx 数组片段
$colDefs = @()
foreach ($c in ($Columns -split ',')) {
  $p = $c -split ':'
  $key = $p[0].Trim(); $title = $p[1].Trim()
  $colDefs += "    { key: '$key', title: '$title', minWidth: 120, ellipsis: { tooltip: true } }"
}
$colTs = $colDefs -join ",`n"

# ---- api 文件 ----
$apiFile = Join-Path $apiDir "$Pkg.ts"
$api = @"
import { request } from '../request';

interface GvaList<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

/** $Label 列表（gin-vue-admin 生成的分页接口） */
export async function fetchGet${Struct}List(params: { page: number; pageSize: number }) {
  const { data, error } = await request<GvaList<any>>({
    url: '/$Pkg/get${Struct}List',
    method: 'get',
    params
  });

  if (error || !data) {
    return { data: null, error };
  }

  return {
    data: { records: data.list, current: data.page, size: data.pageSize, total: data.total },
    error: null
  };
}

/** 删除 $Label */
export function fetchDelete${Struct}(id: number) {
  return request({ url: '/$Pkg/delete${Struct}', method: 'delete', params: { Id: id } });
}
"@
$api | Out-File $apiFile -Encoding utf8

# ---- 页面 ----
$pageFile = Join-Path $viewsDir 'index.vue'
$page = @"
<script setup lang="tsx">
import { ref } from 'vue';
import { NButton, NPopconfirm } from 'naive-ui';
import { fetchDelete${Struct}, fetchGet${Struct}List } from '@/service/api/$Pkg';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { useAppStore } from '@/store/modules/app';

const appStore = useAppStore();
const searchParams = ref({ current: 1, size: 10 });

const { columns, data, loading, getData, mobilePagination } = useNaivePaginatedTable({
  api: () => fetchGet${Struct}List({ page: searchParams.value.current, pageSize: searchParams.value.size }),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.current = params.page;
    searchParams.value.size = params.pageSize;
  },
  columns: () => [
$colTs,
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

async function handleDelete(id: number) {
  const { error } = await fetchDelete${Struct}(id);
  if (error) return;
  window.`$message?.success('删除成功');
  getData();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <NCard title="$Label" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
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
  </div>
</template>

<style scoped></style>
"@
$page | Out-File $pageFile -Encoding utf8

# ---- locale ----
foreach ($f in 'zh-cn.ts','en-us.ts') {
  $path = Join-Path $root "src\locales\langs\$f"
  $text = [System.IO.File]::ReadAllText($path, [System.Text.Encoding]::UTF8)
  if (-not $text.Contains("${Pkg}:")) {
    $label = if ($f -eq 'zh-cn.ts') { $Label } else { $Pkg }
    $text = $text -replace "(fragment: '[^']*',)", "`$1`n    ${Pkg}: '$label',"
    [System.IO.File]::WriteAllText($path, $text, (New-Object System.Text.UTF8Encoding($false)))
  }
}

Write-Host "已生成 $Pkg：api/$Pkg.ts + views/$Pkg/index.vue + locale"
