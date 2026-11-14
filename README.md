# ai-sales · 本地部署的 AI 获客系统

[![CI](https://github.com/wzq73056-netizen/ai-sales/actions/workflows/ci.yml/badge.svg)](https://github.com/wzq73056-netizen/ai-sales/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883.svg)](https://vuejs.org/)

一套跑在自己电脑上的双向获客工具：从抖音、小红书、微信收集线索，用大模型自动回复和筛选，人工跟进成交。

不是 SaaS，不依赖云服务，数据全部留在本机。

## 它能做什么

```
采集    抖音评论区 / 小红书笔记评论、私信 → 自动入库并打分
对话    微信、抖音、小红书私信由 AI 接待，遇到打款、合同、看货转为人工工单
分层    按 S / A / B / C / D 分级，A 类可一键生成开场话术
跟进    客户看板、跟进提醒、成交阶段、时间线
双向    同时收「买我们货的客户」和「给我们供货的货源」，两套人设与知识库
报表    来源分布、跟进健康度、阶段转化、话术效果、方向对比
```

## 技术栈

| 层 | 选型 |
|---|---|
| 后端 | Go + Gin + GORM（基于 [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin)） |
| 前端 | Vue 3 + Naive UI（基于 [soybean-admin](https://github.com/soybeanjs/soybean-admin)） |
| 数据库 | MySQL 8 |
| 浏览器自动化 | Playwright over CDP（连本机 Edge） |
| 微信 | Python + Windows UI 自动化（wxauto 方式，非注入） |
| 大模型 | deepseek-chat 等任意 OpenAI 兼容接口 |
| 语音 | SenseVoice（sherpa-onnx）本地转写，可选 |

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
Python 3.11（语音转写和微信监听需要）
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
- [scripts/docs/git-sync.md](./scripts/docs/git-sync.md) — 代码自动同步到 git 的配置

## 数据库支持

| 数据库 | 状态 |
|---|---|
| MySQL 8.0 | ✅ 已验证 |
| PostgreSQL | ⚠️ 未验证（业务 SQL 有 MySQL 专用函数，需要改造） |

建库、表结构、业务模型位置见 [docs/database.md](./docs/database.md)。

## 已知限制

仓库里不包含平台的采集和自动私信模块（`scraper/` 目录），这部分仅本地运行，
原因见文末免责声明。开源部分包含：后台服务、管理端、数据库结构、启动与运维脚本。

```
· 依赖本机安装的微信 / 抖音客户端，不能跑在无桌面的服务器上
· 采集走浏览器自动化，平台改版可能需要维护选择器
· 微信掉登录后必须人工扫码，程序只能告警
· 单用户设计，没有多租户和权限隔离
```

## 安全提醒

```
· 默认账号密码务必修改
· config.yaml、data/llm.json、data/ 目录都已被 .gitignore 排除
· 数据库备份包含客户手机号和聊天内容，不要提交到公开仓库
```

## 免责声明

本项目用于在自己的账号上做客户线索整理和接待辅助，请勿用于以下用途：

```
批量注册账号、刷量、刷评论
冒充他人身份运营
向未授权的人发送营销信息
规避平台的实名、资质审核
```

使用者需自行遵守所在平台（抖音、小红书、微信等）的服务条款和当地法律法规。
平台可能因自动化操作对账号做出限制或处罚，**由此产生的后果由使用者自行承担**，
作者不提供任何形式的账号恢复、封禁解除承诺。

本项目按 MIT 协议「按现状」提供，不对可用性、数据完整性作任何担保。

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