# ai-sales · 本地部署的 AI 获客系统

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
| 大模型 | 任意 OpenAI 兼容接口（默认 DeepSeek） |
| 语音 | 本地 whisper 转写（可选） |

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

### 3. 大模型

在 `E:\ai\data\llm.json`（Linux 下改脚本里的路径）建：

```json
{
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "sk-your-key",
  "model": "deepseek-chat"
}
```

不填也能跑，但 AI 回复和话术生成会不可用。

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

## 数据库支持

| 数据库 | 状态 |
|---|---|
| MySQL 8.0 | ✅ 已验证 |
| PostgreSQL | ⚠️ 未验证（业务 SQL 有 MySQL 专用函数，需要改造） |

建库、表结构、业务模型位置见 [docs/database.md](./docs/database.md)。

## 已知限制

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

## 许可证

业务代码以 MIT 发布，见 [LICENSE](./LICENSE)。

仓库包含两个上游项目的修改版，其许可证保留在 [LICENSES](./LICENSES)：

- gin-vue-admin —— Apache License 2.0
- soybean-admin —— MIT License

使用时请一并遵守这两份许可证的要求。
## 联系方式

```
Issue   https://github.com/wzq73056-netizen/ai-sales/issues
邮箱    <待填写>
```

遇到问题优先提 Issue，能附上报错日志和复现步骤最好。