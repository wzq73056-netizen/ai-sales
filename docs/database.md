# 数据库

## 支持的数据库

| 数据库 | 状态 | 说明 |
|---|---|---|
| **MySQL 8.0** | ✅ 已验证 | 开发和日常使用都在这个版本上跑 |
| PostgreSQL | ⚠️ 未验证 | 框架层有 pgsql 配置项，但业务 SQL 没适配过 |
| SQLite / Oracle / SQL Server | ❌ 不支持 | 没做适配 |

PostgreSQL 目前只是"看起来支持"：gin-vue-admin 框架自带 `pgsql` 配置分支和 `GormPgSql()`，
但业务代码里有 4 处 `DATE_FORMAT()`（MySQL 函数，PG 要换成 `to_char`），
还有一批 `COALESCE(NULLIF(...))` 之类的写法也没在 PG 上跑过。

要用 PG 的话，需要把这些 SQL 改成 GORM 表达式或按方言分支，属于改造工作量，不是配置项一改就行。

---

## 建库（MySQL）

库名 `ai_go`，字符集 `utf8mb4`。

### 方式一：让后端自动建表（推荐）

```bash
# 1. 建空库
mysql -uroot -p -e "CREATE DATABASE ai_go DEFAULT CHARSET utf8mb4;"

# 2. 填好 backend/server/config.yaml 的数据库账号

# 3. 启动后端
cd backend/server && ./bin/gva.exe

# 4. 调一次初始化接口（建系统表 + 默认管理员 admin/123456）
curl -X POST http://127.0.0.1:8888/init/initdb
```

分工：

```
系统表 sys_*   → gin-vue-admin 的 RegisterTables()
业务表 ai_*    → initialize/gorm_biz.go 的 bizModel()
```

`bizModel()` 只建**不存在的表**，已存在的表不碰。这样既能支持全新安装，
又不会在已有数据的库上触发 `Data truncated for column xxx` 导致启动失败。

### 方式二：导入结构文件

```bash
mysql -uroot -p ai_go < sql/schema.sql
```

`schema.sql` 只包含表结构，不含任何数据（50+ 张表，约 116 KB）。

---

## 表分组

| 分组 | 表名 | 说明 |
|---|---|---|
| 系统 | `sys_*` | 用户、角色、菜单、API、字典、操作记录等 |
| 线索 | `ai_lead` | 抖音/小红书采集来的线索，含方向、等级、联系方式 |
| 抖音 | `ai_douyin_account` `ai_douyin_device` `ai_douyin_task` `ai_douyin_dm` `ai_douyin_whitelist` | 账号线索、设备与 CDP 端口、采集任务、私信记录、自动回复白名单 |
| 微信 | `ai_wx_customer` `ai_wx_followup` `ai_wx_handoff` | 客户、跟进记录、转人工工单 |
| 知识库 | `ai_knowledge` `ai_knowledge_fragment` | 知识库与片段（买方 / 供方两套） |
| 采集配置 | `ai_scrape_keyword` `ai_scrape_blacklist` `ai_xhs_run` | 关键词、黑名单、小红书采集记录 |
| 表格视图 | `ai_table_view` | 列偏好和保存的筛选视图 |

## 业务模型在哪

Go 模型定义在 `backend/server/model/lead/`：

```
lead.go              线索（ai_lead）
wx_customer.go       微信客户
wx_followup.go       跟进记录
wx_handoff.go        转人工工单
douyin_account.go    抖音账号线索
douyin_device.go     设备与白名单
douyin_task.go       采集任务
douyin_dm.go         私信记录
```

`ai_table_view` 的表结构在 `backend/server/service/lead/table_view.go`（没挪进 model 目录），
全新库用方式一建表时这张表需要手动执行 `schema.sql` 里对应那一段。

## 注意

```
· 默认管理员 admin / 123456，部署后务必修改
· schema.sql 不含菜单数据，菜单需要从「系统管理 → 菜单管理」手动建，或走 initdb 初始化
· 数据库备份包含客户手机号和聊天内容，不要提交到公开仓库
```
