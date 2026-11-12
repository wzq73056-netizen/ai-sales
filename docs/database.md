# 数据库

MySQL 8.0，库名 `ai_go`，字符集 `utf8mb4`。

## 两种建库方式

### 方式一：让后端自动建表（推荐）

后端启动时会 `AutoMigrate` 全部业务模型，配合 gin-vue-admin 自带的初始化接口即可从零建库：

```bash
# 1. 建空库
mysql -uroot -p -e "CREATE DATABASE ai_go DEFAULT CHARSET utf8mb4;"

# 2. 填好 backend/server/config.yaml 的数据库账号

# 3. 启动后端
cd backend/server && ./bin/gva.exe

# 4. 调一次初始化接口（会建系统表 + 默认管理员 admin/123456）
curl -X POST http://127.0.0.1:8888/init/initdb
```

系统表由 gin-vue-admin 的 `RegisterTables()` 建，业务表（`ai_*`）由 `initialize/gorm_biz.go` 的 `bizModel()` 建。

### 方式二：导入结构文件

不想跑初始化、或者想先看结构，可以直接导入：

```bash
mysql -uroot -p ai_go < sql/schema.sql
```

`schema.sql` 只包含**表结构**，不含任何数据（56 张表，约 116 KB）。

## 表分组

| 分组 | 表名 | 说明 |
|---|---|---|
| 系统 | `sys_*` | gin-vue-admin 的用户、角色、菜单、API、字典等 |
| 线索 | `ai_lead` | 抖音/小红书采集来的线索，含方向、等级、联系方式 |
| 抖音 | `ai_douyin_account` `ai_douyin_device` `ai_douyin_task` `ai_douyin_dm` `ai_douyin_whitelist` | 账号线索、设备与 CDP 端口、采集任务、私信记录、自动回复白名单 |
| 微信 | `ai_wx_customer` `ai_wx_followup` `ai_wx_handoff` | 客户、跟进记录、转人工工单 |
| 知识库 | `ai_knowledge` `ai_knowledge_fragment` | 知识库与片段（买方 / 供方两套） |
| 采集配置 | `ai_scrape_keyword` `ai_scrape_blacklist` `ai_xhs_run` | 关键词、黑名单、小红书采集记录 |
| 表格视图 | `ai_table_view` | 列偏好和保存的筛选视图 |

## 注意

```
· ai_table_view 未纳入 AutoMigrate，用方式一时请手动执行 sql/schema.sql 里的这一段
· 默认管理员 admin / 123456，部署后务必修改
· schema.sql 不含菜单数据；菜单由初始化接口写入，或从菜单管理页面手动建
```
