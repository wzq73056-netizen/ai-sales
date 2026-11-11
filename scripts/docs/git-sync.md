# 自动同步到 Gitee（单个仓库汇总前后端代码 + 脚本）

每天定时把三处代码汇总到 `E:\ai\repo`，有变化就自动 commit + push。

## 目录结构

```
E:\ai\repo
├─ backend/      ← 来自 E:\ai\gin-vue-admin（排除 bin、node_modules、config.yaml）
├─ frontend/     ← 来自 E:\ai\soybean-admin（排除 node_modules、dist）
├─ scripts/      ← 来自 E:\ai\scripts（监听脚本、健康检查、启动脚本）
├─ sql/          ← 数据库结构与种子数据导出
├─ docs/         ← 本文件与说明
└─ .gitignore
```

## 手动执行

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File E:\ai\scripts\git-sync.ps1
```

会输出：同步了多少文件、是否有改动、是否推送成功。

## 首次配置（只做一次）

```powershell
cd E:\ai\repo
git init
git branch -M main
git remote add origin https://gitee.com/<你的账号>/<仓库名>.git
git add -A
git commit -m "chore(other): initial sync"
git push -u origin main
```

令牌不想每次输入，可以记住凭据：

```powershell
git config --global credential.helper manager
# 第一次 push 输入 Gitee 用户名 + 私人令牌，之后自动记住
```

## 定时任务

```powershell
schtasks /Create /TN "ai-git-sync" /TR "powershell -NoProfile -ExecutionPolicy Bypass -File E:\ai\scripts\git-sync.ps1" /SC DAILY /ST 22:30 /F
```

## 不会上传的东西

```
server/config.yaml    数据库密码
data/                 客户聊天、语音、图片
*.log / logs/         运行日志
bin/                  gva.exe（193MB，没必要进仓库）
node_modules/ / dist/ 依赖和构建产物
```

这些要么是密钥，要么是运行时数据，要么可以用 `go build` / `pnpm build` 重新生成。
