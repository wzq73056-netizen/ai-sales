# ai-sales · 本地部署的 AI 获客系统

[![CI](https://github.com/wzq73056-netizen/ai-sales/actions/workflows/ci.yml/badge.svg)](https://github.com/wzq73056-netizen/ai-sales/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883.svg)](https://vuejs.org/)

一套跑在自己电脑上的双向获客工具：从抖音、小红书、微信收集线索，用大模型自动回复和筛选，人工跟进成交。

不是 SaaS，不依赖云服务，数据全部留在本机。

> [!IMPORTANT]
> **本项目包含平台自动化脚本，使用前请先阅读文末[免责声明](#免责声明)。**
>
> 抖音 / 小红书的数据采集脚本在 `scraper/` 目录，需要自己配置浏览器登录态；
> 登录态、API Key、数据库密码、任何客户数据都不在仓库里。
>
> 平台可能对自动化操作做出限制或处罚，使用前请确认你能接受这个风险。

## 它能做什么

```
采集    抖音评论区 / 小红书笔记评论、私信 → 自动入库并打分
对话    本机客户端 + 平台私信由 AI 起草回复，遇打款/合同/看货转人工
分层    按 S / A / B / C / D 分级，A 类可一键生成开场话术          （本仓库）
跟进    客户看板、跟进提醒、成交阶段、时间线                      （本仓库）
双向    同时收「买我们货的客户」和「给我们供货的货源」              （本仓库）
报表    来源分布、跟进健康度、阶段转化、话术效果、方向对比          （本仓库）
```

## 技术栈

| 层 | 选型 |
|---|---|
| 后端 | Go + Gin + GORM（基于 [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin)） |
| 前端 | Vue 3 + Naive UI（基于 [soybean-admin](https://github.com/soybeanjs/soybean-admin)） |
| 数据库 | MySQL 8 |
| 浏览器自动化 | Playwright over CDP（连本机 Edge） |
| 本机客户端集成 | Python + Windows 自动化（读取本机聊天客户端消息） |
| 大模型 | deepseek-chat 等任意 OpenAI 兼容接口 |
| 语音 | SenseVoice（sherpa-onnx）本地转写，可选 |

## 采集模块怎么跑

`scraper/` 里的脚本不直接调 API，靠 Playwright 连一个**已经登录好**的浏览器。所以顺序是：

### 1. 用独立 profile 启动 Edge（带调试端口）

```powershell
# 抖音（端口 9333）
msedge.exe --remote-debugging-port=9333 `
  --user-data-dir=D:\ai\browser\profile-douyin

# 小红书（端口 9334）
msedge.exe --remote-debugging-port=9334 `
  --user-data-dir=D:\ai\browser\profile-xiaohongshu
```

`--user-data-dir` 一定要分开：同一个目录只能起一个实例，而且两个账号的
Cookie 混在一起容易互相影响。

### 2. 在打开的窗口里手动登录一次

抖音扫码、小红书扫码，登录态会存进各自的 profile 目录。
之后脚本复用的是这份登录态，不用每次登录。

### 3. 装依赖

```powershell
cd scraper
npm init -y
npm install playwright-core
```

只需要 `playwright-core`，不需要下载浏览器（用的是你本机的 Edge）。

### 4. 跑脚本

```powershell
# 抖音：按关键词搜视频
node dy_collect.mjs 关键词

# 抖音：抓某个视频的评论
node dy_comments.mjs <视频id> --limit 20

# 采完推到后台
node dy_push.mjs

# 小红书：搜笔记
node xhs_collect.mjs 关键词

# 小红书：抓评论
node xhs_comments.mjs <笔记id> --limit 20

# 小红书：推后台
node xhs_push.mjs
```

每个脚本头部都有一段注释说明参数，`node xxx.mjs --help` 不一定有，
直接看文件开头。

### 5. 私信监听（可选）

```powershell
node dy_dm_watch.mjs --once           # 跑一轮看看会做什么
node dy_dm_watch.mjs --interval 300 --send   # 常驻并真实发送
```

私信只回白名单里的人，名单在后台「系统管理 → 客户」里维护。
名单外的一律不动。

### 端口和路径都能改

```
CDP 端口   环境变量 CDP / CDP_XHS
           默认 抖音 9333、小红书 9334
浏览器     连不上会报错，先确认端口通了：
           Test-NetConnection 127.0.0.1 -Port 9333
```

### 常见问题

| 现象 | 原因 | 处理 |
|---|---|---|
| `connect ECONNREFUSED 127.0.0.1:9333` | 浏览器没起或端口不对 | 按第 1 步重起，确认端口 |
| 页面打开是登录页 | profile 里没有登录态 | 在第 2 步的窗口里手动登录 |
| 抓不到数据 | 平台改版，选择器失效 | 需要改脚本里的选择器 |
| 提示配额已满 | 每日上限到了 | 见 `humanize.mjs` 里的 LIMITS |

## 目录结构

```
backend/          Go 服务（同时托管前端 dist，单进程跑整站）
frontend/         Vue 3 管理端
scripts/          监听脚本、健康检查、备份、通知、启动脚本
sql/              数据库结构（不含数据）
LICENSES/         上游项目许可证
```

## 快速开始

### 1. 准备环境

```
Go 1.21+
Node 20+ / pnpm 9+
MySQL 8
Python 3.11（语音转写、本机客户端监听需要）
Edge / Chrome（采集需要）
```

### 2. 数据库

```sql
CREATE DATABASE ai_go DEFAULT CHARSET utf8mb4;
```

把 `backend/server/config.example.yaml` 复制成 `backend/server/config.yaml`，填入数据库账号密码。

表可以自动建（后端启动时 AutoMigrate 业务表，再调一次 `/init/initdb` 建系统表），
也可以直接导入 `sql/schema.sql`（只有结构、不含数据）。详见 [docs/database.md](./docs/database.md)。

### 3. 大模型（必需，否则 AI 不回复）

两种配法，改完即时生效：

**后台页面**：系统管理 → 模型配置，填接口地址、API Key、模型名，点「测试连接」验证。
**配置文件**：编辑 `data/llm.json`

```json
{
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "sk-your-key",
  "model": "deepseek-chat",
  "temperature": 0.6
}
```

支持任何 OpenAI 兼容接口（DeepSeek / 通义 / Kimi / 本地 Ollama）。

### 3.1 语音转文字（可选）

微信语音默认能播放但不转文字。要转文字需装本地模型：

```bash
pip install sherpa-onnx numpy scipy pilk
# 下载 SenseVoice 模型（model.int8.onnx + tokens.txt）放到 scripts/models/
python scripts/serve_audio.py --port 8898
```

228 MB 的模型文件不进仓库，需自行下载。

完整说明见 [docs/models.md](./docs/models.md)。

### 4. 构建

```bash
# 后端（会同时提供前端页面）
cd backend/server
go build -o bin/gva.exe .

# 前端
cd frontend
pnpm install
pnpm build          # 产物在 frontend/dist，后端会自动发现
```

前端 `frontend/.env.prod` 里的接口地址保持 `/proxy-default`，由 Go 进程自己转发，不需要 nginx。

### 5. 启动

```bash
backend/server/bin/gva.exe
```

打开 <http://127.0.0.1:8888/>，默认账号 `admin / 123456`（首次登录后请立即修改）。

## 配置说明

| 配置 | 位置 | 说明 |
|---|---|---|
| 数据库 | `backend/server/config.yaml` | 不进仓库 |
| 大模型 | `data/llm.json` | 不进仓库 |
| 关键词 | 后台「线索池 → 关键词」 | 分买方 / 供方两组 |
| 知识库 | 后台「知识库」 | 私信回复从这里取素材 |
| 人设 | `backend/server/service/lead/dm_ai.go` | 修改买家 / 卖家两套角色描述 |

## 定时任务（Windows）

| 任务 | 时间 | 作用 |
|---|---|---|
| 开机自启 | 登录时 | 起后端 + 三个监听 |
| 健康检查 | 每 2 分钟 | 掉线自动拉起，桌面弹窗告警 |
| 数据库备份 | 每天 03:30 | 导出到本地 backup 目录 |
| 代码同步 | 每天 22:30 | 推送到自己的 Git 仓库（可选） |

## 用到的模型

| 能力 | 模型 | 部署 | 必需 |
|---|---|---|---|
| 私信回复 / 话术生成 | deepseek-chat（可换任意 OpenAI 兼容模型） | API | 是 |
| 语音转文字 | SenseVoice（sherpa-onnx） | 本地 8898 | 否 |
| 知识库检索 | BM25（无模型） | MySQL 内置 | 自动 |

详见 [docs/models.md](./docs/models.md)。

## 文档索引

- [docs/database.md](./docs/database.md) — 数据库结构、建库方式、业务模型位置
- [docs/models.md](./docs/models.md) — 用到的模型、配置方式、换模型、未使用的模型
- [docs/full-version.md](./docs/full-version.md) — 开源版与完整版的差异、环境要求、授权范围
- [scripts/docs/git-sync.md](./scripts/docs/git-sync.md) — 代码自动同步到 git 的配置

## 数据库支持

| 数据库 | 状态 |
|---|---|
| MySQL 8.0 | ✅ 已验证 |
| PostgreSQL | ⚠️ 未验证（业务 SQL 有 MySQL 专用函数，需要改造） |

建库、表结构、业务模型位置见 [docs/database.md](./docs/database.md)。

## 已知限制

### 仓库里没有的东西

```
运行时配置   config.yaml / llm.json / db.json（有模板）
浏览器登录态 需要自己扫码登录
任何客户数据 一条都没有
```

### 环境与运行限制

```
· 依赖本机安装的客户端程序，不能跑在无桌面的服务器上
· 本机客户端掉登录后必须人工重新登录，程序只能告警
· 单用户设计，没有多租户和权限隔离
· 浏览器自动化随平台改版需要维护选择器
```

两个版本的具体差异见 [docs/full-version.md](./docs/full-version.md)。

## 安全提醒

```
· 默认账号密码务必修改
· config.yaml、data/llm.json、data/ 目录都已被 .gitignore 排除
· 数据库备份包含客户手机号和聊天内容，不要提交到公开仓库
```

## 免责声明

> 使用本项目前请认真阅读以下条款。**如果你不同意其中任何一条，请不要使用。**
> 你下载、运行或修改本项目，即视为已完整阅读并接受本声明。

**1. 风险自担**

使用者对本项目的使用由使用者自行决定，并自行承担全部风险。作者对使用者使用本项目所产生的任何损失、责任或风险不承担任何义务。

**2. 代码质量**

本项目基于现有知识和技术的开发成果。作者按现有技术水平努力确保代码的正确性和安全性，但不保证代码完全没有错误或缺陷，也不承诺任何时间内功能可用。

**3. 第三方依赖**

本项目依赖的第三方库、组件和服务（包括但不限于 Go、Node.js、MySQL、Playwright、各类大模型 API）各自遵循其原始开源或商业许可，使用者需自行查阅并遵守相应协议。作者不对第三方组件的稳定性、安全性及合规性承担责任。

**4. 平台条款**

本项目包含对第三方平台（抖音、小红书等）的自动化操作代码。**这些平台的用户协议普遍禁止使用自动化工具**。使用者需自行确认自己的使用行为是否符合所在平台的服务条款，并承担由此产生的一切后果，包括但不限于账号限流、功能限制、账号封禁。

**5. 法律法规**

使用者在使用本项目的代码和功能时，必须自行研究并遵守所在地的法律法规。任何因违反法律法规而产生的法律责任和风险，均由使用者自行承担。

**6. 数据处理合规**

本项目不对使用者涉及的数据收集、存储、传输等处理活动的合规性承担责任。使用者应自行遵守个人信息保护、数据安全等相关法律，确保处理行为合法正当；因违规操作导致的法律责任由使用者自行承担。

**7. 不得侵犯他人权益**

使用者不得使用本工具从事任何侵犯他人合法权益的行为，包括但不限于：未经授权采集他人个人信息、批量注册账号、刷量刷评论、冒充他人身份、向未授权对象发送营销信息。

**8. 不得牵连作者**

使用者在任何情况下均不得将本项目的作者、贡献者或其他相关方与使用者的使用行为联系起来，或要求其对使用者使用本项目所产生的任何损失或损害负责。

**9. 不提供相关服务**

作者不提供任何形式的账号解封、代运营、代发内容、数据采集定制服务，也不对使用者的账号状态、经营结果作任何承诺。

**10. 二次开发**

基于本项目进行的任何二次开发、修改或编译的程序与原作者无关。原创作者不承担与二次开发行为或其结果相关的任何责任，使用者应自行承担全部责任。

**11. 专利与商标**

本项目不授予使用者任何专利许可。若使用本项目导致专利纠纷或侵权，使用者自行承担全部风险和责任。未经作者书面授权，不得使用本项目名称进行商业宣传、推广或再授权。

**12. 终止与变更**

作者保留随时终止向任何违反本声明的使用者提供服务的权利，并可能要求其销毁已获取的代码及衍生作品。作者保留在不另行通知的情况下更新本声明的权利，使用者持续使用即视为接受修订后的条款。

**13. 最终确认**

在使用本项目的代码和功能之前，请你认真考虑并接受以上免责声明。如你对上述声明有任何疑问或不同意，请不要使用本项目的代码和功能。使用本项目即视为你已完全理解并接受上述全部条款。

## 许可证

业务代码以 MIT 发布，见 [LICENSE](./LICENSE)。

仓库包含两个上游项目的修改版，其许可证保留在 [LICENSES](./LICENSES)：

- gin-vue-admin —— Apache License 2.0
- soybean-admin —— MIT License

使用时请一并遵守这两份许可证的要求。
## 联系方式

```
微信    wengzhaoqin123
Issue   https://github.com/wzq73056-netizen/ai-sales/issues
```

加微信请备注「AI 获客」，否则可能不通过。功能问题和 bug 优先提 Issue，
方便留档和搜索。
