param(
  [Parameter(Mandatory=$true)][string]$Source,     # 源表，如 wm_material_music
  [Parameter(Mandatory=$true)][string]$Target,     # 目标表，如 ai_music
  [Parameter(Mandatory=$true)][string]$Pkg,        # 包名，如 music
  [Parameter(Mandatory=$true)][string]$Struct,     # 结构体名，如 Music
  [string]$Label = ''                               # 中文名，默认同 Target
)

# 从本机 db.json 读取数据库密码（db.json 不进仓库）
function Get-DbPassword {
  $file = 'E:\ai\data\db.json'
  if (-not (Test-Path $file)) { throw "找不到 $file，请参照 docs/db.example.json 创建" }
  return (Get-Content $file -Raw -Encoding UTF8 | ConvertFrom-Json).password
}
$ErrorActionPreference = 'Stop'
$mysql = 'D:\Program Files (x86)\mysql-8.0.26-winx64\bin\mysql.exe'
$env:MYSQL_PWD = (Get-DbPassword)
$server = 'E:\ai\gin-vue-admin\server'
if (-not $Label) { $Label = $Target }

function Invoke-Sql([string]$sql, [string]$db = 'ai_go') {
  $tmp = [System.IO.Path]::GetTempFileName() + '.sql'
  [System.IO.File]::WriteAllText($tmp, $sql, (New-Object System.Text.UTF8Encoding($false)))
  $mArgs = @('-uroot', '--default-character-set=utf8mb4')
  if ($db) { $mArgs += $db }
  $p = Start-Process -FilePath $mysql -ArgumentList $mArgs -RedirectStandardInput $tmp -RedirectStandardError "$tmp.err" -NoNewWindow -PassThru -Wait
  $err = Get-Content "$tmp.err" -Raw -ErrorAction SilentlyContinue
  Remove-Item $tmp, "$tmp.err" -Force -ErrorAction SilentlyContinue
  if ($p.ExitCode -ne 0) { throw "SQL 失败: $err" }
  return $err
}

# ---------- 1. 读源表结构 ----------
$colSql = "SELECT column_name, column_type, is_nullable, column_default, column_comment, column_key, extra FROM information_schema.columns WHERE table_schema='ruoyi-ai' AND table_name='$Source' ORDER BY ordinal_position;"
$raw = & $mysql -uroot -N --default-character-set=utf8mb4 -e $colSql 2>$null
$cols = $raw | Where-Object { $_ } | ForEach-Object {
  $f = $_ -split "`t"
  [pscustomobject]@{ Name=$f[0]; Type=$f[1]; Nullable=$f[2]; Default=$f[3]; Comment=$f[4]; Key=$f[5]; Extra=$f[6] }
}

# 丢弃的列（与“无多租户 / 统一审计”规范冲突）
$drop = @('tenant_id','create_dept','create_by','update_by','is_delete','del_flag')
$rename = @{ 'create_time'='created_at'; 'update_time'='updated_at' }

$defs = @()
$srcCols = @()
$dstCols = @()
foreach ($c in $cols) {
  if ($drop -contains $c.Name) { continue }
  $name = if ($rename.ContainsKey($c.Name)) { $rename[$c.Name] } else { $c.Name }
  $type = $c.Type
  $comment = ($c.Comment -replace "'", "''")
  $null = if ($c.Nullable -eq 'NO') { 'NOT NULL' } else { 'NULL' }
  $extra = ''
  if ($c.Name -eq 'id') { $extra = 'AUTO_INCREMENT'; $null = 'NOT NULL' }
  if ($c.Name -eq 'created_at') { $extra = 'DEFAULT CURRENT_TIMESTAMP(3)'; $type = 'datetime(3)'; $null = 'NOT NULL' }
  if ($c.Name -eq 'updated_at') { $extra = 'DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)'; $type = 'datetime(3)'; $null = 'NOT NULL' }
  if ($c.Name -eq 'deleted_at') { $type = 'datetime(3)'; $null = 'NULL'; $extra = '' }
  $def = "  ``$name`` $type $null $extra COMMENT '$comment'".TrimEnd()
  $defs += $def
  $srcCols += "``$($c.Name)``"
  $dstCols += "``$name``"
  if ($c.Name -eq 'id') { $defs += '  PRIMARY KEY (`id`)' }
}
# 常量路由兜底：没有 created_at 的表补上
if (-not ($dstCols -contains '`created_at`')) {
  $defs += '  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)'
  $defs += '  `updated_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)'
  $defs += '  `deleted_at` datetime(3) NULL'
}

$ddl = "DROP TABLE IF EXISTS ``$Target``;`nCREATE TABLE ``$Target`` (`n" + ($defs -join ",`n") + "`n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='$Label';"
Invoke-Sql "SET NAMES utf8mb4;`n$ddl" | Out-Null
Write-Host "[1/6] 建表 $Target 完成"

# ---------- 2. 迁数据 ----------
$insert = "SET NAMES utf8mb4;`nINSERT INTO ``ai_go``.``$Target`` (" + ($dstCols -join ',') + ") SELECT " + ($srcCols -join ',') + " FROM ``ruoyi-ai``.``$Source``;"
try { Invoke-Sql $insert | Out-Null } catch { Write-Warning "数据迁移失败（可忽略空表）: $_" }
$cnt = & $mysql -uroot -N -e "SELECT COUNT(*) FROM ai_go.$Target;" 2>$null
Write-Host "[2/6] 数据迁移完成，$cnt 行"

# ---------- 3. 调生成器 ----------
$login = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/base/login' -Method Post -Body (@{username='admin';password='123456';captcha='';captchaId=''} | ConvertTo-Json) -ContentType 'application/json'
$h = @{ 'x-token' = $login.data.token }

$pkgBody = @{ packageName=$Pkg; label=$Label; desc=$Label; template='package' } | ConvertTo-Json
try { Invoke-RestMethod -Uri 'http://127.0.0.1:8888/autoCode/createPackage' -Method Post -Headers $h -Body ([System.Text.Encoding]::UTF8.GetBytes($pkgBody)) -ContentType 'application/json; charset=utf-8' | Out-Null } catch {}

function To-Camel([string]$s) { ($s -split '_' | ForEach-Object { if ($_ -ne '') { $_.Substring(0,1).ToUpper() + $_.Substring(1) } }) -join '' }
$newCols = (Invoke-RestMethod -Uri "http://127.0.0.1:8888/autoCode/getColumn?tableName=$Target" -Headers $h).data.columns
$fields = @()
foreach ($c in $newCols) {
  $ft='string'; if ($c.dataType -match 'int') { $ft='int' } elseif ($c.dataType -match 'float|double|decimal') { $ft='float64' }
  $fields += @{ fieldName=(To-Camel $c.columnName); columnName=$c.columnName; fieldDesc=$c.columnComment; fieldJson=(To-Camel $c.columnName); fieldType=$ft; require=$false; primaryKey=[bool]$c.primaryKey; sort=[bool]$c.primaryKey }
}
$body = @{ tableName=$Target; structName=$Struct; abbreviation=$Pkg; humpPackageName=$Pkg; description=$Label; packageName=$Pkg; package=$Pkg; businessDB=''; autoMigrate=$false; generateServer=$true; generateWeb=$true; gvaModel=$false; isTree=$false; onlyTemplate=$false; isAdd=$false; autoCreateApiToSql=$false; autoCreateMenuToSql=$false; autoCreateBtnAuth=$false; autoCreateResource=$false; fields=$fields }
$r = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/autoCode/createTemp' -Method Post -Headers $h -Body ([System.Text.Encoding]::UTF8.GetBytes(($body|ConvertTo-Json -Depth 6))) -ContentType 'application/json; charset=utf-8'
Write-Host "[3/6] 生成代码: $($r.code) $($r.msg)"

# ---------- 4. 修生成产物 ----------
$svc = Join-Path $server "service\$Pkg\$Pkg.go"
if (Test-Path $svc) {
  $t = [System.IO.File]::ReadAllText($svc, [System.Text.Encoding]::UTF8)
  # 参数名与包名同名 → 改成 data
  # 只做四类精确替换，避免正则误伤函数签名
  $t = $t.Replace(", $Pkg $Pkg.", ", data $Pkg.")
  $t = $t.Replace("($Pkg $Pkg.", "(data $Pkg.")
  $t = $t.Replace("$Pkg.Id", "data.Id")
  $t = $t.Replace("First(&$Pkg)", "First(&data)")
  $t = $t.Replace("Updates(&$Pkg)", "Updates(&data)")
  [System.IO.File]::WriteAllText($svc, $t, (New-Object System.Text.UTF8Encoding($false)))
}
# router_biz.go 去重
$rb = Join-Path $server 'initialize\router_biz.go'
$t = [System.IO.File]::ReadAllText($rb, [System.Text.Encoding]::UTF8)
$lines = $t -split "`n"
$seen = @{}; $out = @()
foreach ($l in $lines) {
  if ($l -match 'Init\w+Router\(') { if ($seen.ContainsKey($l.Trim())) { continue }; $seen[$l.Trim()] = 1 }
  $out += $l
}
[System.IO.File]::WriteAllText($rb, ($out -join "`n"), (New-Object System.Text.UTF8Encoding($false)))
Write-Host "[4/6] 生成产物已修正"

# ---------- 5. 编译 ----------
$env:Path += ';D:\Program Files\go\bin'; $env:GOPROXY='https://goproxy.cn,direct'
Push-Location $server
$build = & go build -o bin\gva.exe . 2>&1
$buildExit = $LASTEXITCODE
Pop-Location
if ($buildExit -ne 0) { Write-Warning "编译失败：`n$($build | Select-Object -First 6 | Out-String)" } else { Write-Host "[5/6] 编译通过" }

# ---------- 6. 权限 ----------
$vals = foreach ($m in 'GET','POST','PUT','DELETE') { "('p','888','/$Pkg/*','$m')" }
Invoke-Sql ("INSERT INTO ai_go.casbin_rule (ptype,v0,v1,v2) VALUES " + ($vals -join ',') + ";") | Out-Null
Write-Host "[6/6] casbin 规则已加 → 需要重启后端生效"

Write-Host ""
Write-Host "下一步（前端）："
Write-Host "  1. src/service/api/$Pkg.ts  —— 抄 knowledge.ts 改 4 个函数"
Write-Host "  2. src/views/$Pkg/index.vue —— 抄 knowledge/index.vue 改列定义"
Write-Host "  3. locales 加 route.$Pkg"
Write-Host "  4. sys_base_menus 插一条 name='$Pkg'"
