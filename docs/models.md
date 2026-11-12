# 用到的模型

一共三类：一个大语言模型（走 API）、一个语音识别模型（本地）、一个检索算法（不需要模型）。

## 总览

| 能力 | 模型 | 部署方式 | 必需 | 不装的影响 |
|---|---|---|---|---|
| 私信回复、话术生成、意图分类 | deepseek-chat（可换任意 OpenAI 兼容模型） | API 调用 | 是 | AI 不会自动回复，只能人工接待 |
| 微信语音转文字 | SenseVoice（sherpa-onnx int8） | 本地进程，端口 8898 | 否 | 语音消息看不到文字，但能播放 |
| 知识库检索 | BM25 全文检索 | MySQL 内置 | 自动可用 | — |

---

## 一、大语言模型

### 用途

```
backend/server/service/lead/dm_ai.go         私信自动回复（买方/供方两套人设 + 知识库素材）
backend/server/service/lead/lead_dm_script.go 批量生成抖音私信开场话术
backend/server/service/lead/classify.go       线索与消息的意图分类
```

### 配置方式

**方式一：后台页面（推荐）**

```
系统管理 → 模型配置
填接口地址、API Key、模型名，点「保存」，点「测试连接」验证
```

**方式二：配置文件**

编辑 `E:\ai\data\llm.json`：

```json
{
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "sk-...",
  "model": "deepseek-chat",
  "kb_id_buyer": "2003",
  "kb_id_seller": "2002",
  "temperature": 0.6
}
```

**方式三：环境变量**（优先级最低，文件会覆盖它）

```
RAG_CHAT_BASE_URL / RAG_CHAT_API_KEY / RAG_CHAT_MODEL
```

改完即时生效，不用重启后端——每次调用都会重新读配置。

### 换成别的模型

只要是 OpenAI 兼容接口（`/chat/completions`）就能直接用，改三个字段即可：

| 服务商 | base_url | model 示例 |
|---|---|---|
| DeepSeek | https://api.deepseek.com/v1 | deepseek-chat |
| 阿里通义 | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-plus |
| 月之暗面 | https://api.moonshot.cn/v1 | moonshot-v1-8k |
| 本地 Ollama | http://127.0.0.1:11434/v1 | qwen2.5:7b |

### 费用

按 token 计费，由服务商收。私信场景每条回复几百 token，量级很小。

### 不配置会怎样

```
后端能正常启动，页面能正常用
AI 回复不可用 → 私信不会自动回
话术生成不可用 → 点「生成话术」会报「未配置大模型」
```

---

## 二、语音识别模型（可选）

### 用途

微信语音消息转文字。转写结果存成 `[语音] 转写内容` 的跟进记录，前端客户详情里能看文字、也能播放原音频。

### 技术选型

```
sherpa-onnx + SenseVoice 模型，纯 onnxruntime 推理，不需要 PyTorch
不用腾讯云转写（原方案走设备端调用，有风控风险）
全部本地推理，不联网、不产生 API 费用
```

### 文件与位置

```
scripts/models/model.int8.onnx    约 228 MB
scripts/models/tokens.txt         词表
```

也可以放到任意目录，用 `ASR_MODEL_DIR` 环境变量指定。

### 安装步骤

```bash
# 1. 装依赖（需要 Python 3.11，因为 pilk 只有 cp311 的 Windows wheel）
pip install sherpa-onnx numpy scipy pilk

# 2. 下载 SenseVoice 中文模型（k2-fsa/sherpa-onnx 发布的 sherpa-onnx-sense-voice 系列）
#    解压后把 model.int8.onnx 和 tokens.txt 放进 scripts/models/

# 3. 启动服务
python scripts/serve_audio.py --port 8898

# 4. 验证
curl -s http://127.0.0.1:8898/health
# {"status": "ok", "pcm_rate": 24000, "asr": true}
```

### 接口

```
GET  /health    → {"status":"ok","pcm_rate":24000,"asr":true|false}
POST /convert   原始 SILK 字节 → audio/wav（微信语音是 SILK 格式，要先转码）
POST /asr       音频字节 → {"text":"转写结果"}
```

### 性能实测

```
模型加载      约 1 秒
单条 3 秒语音  0.1 秒内推理完成（HTTP 往返约 2 秒）
常驻内存      约 400 MB
```

### 不安装会怎样

```
转码仍然可用（SILK → WAV 能播）
/asr 返回 503，语音消息只显示「[语音]」不显示文字
文字消息、图片、其他功能完全不受影响
```

### 为什么模型不进仓库

```
228 MB 的二进制文件
git 每存一次就是一份完整副本，几次提交就上 GB
体积和许可证都不适合放代码仓库，由使用者自行下载
```

---

## 三、知识库检索：不需要模型

```
backend/server/service/kbfrag/search.go
// SearchFragments 在知识库内做 BM25 检索
```

用 MySQL 全文索引做 BM25 排序，取前几条片段塞进提示词。

**没有用到：**

```
embedding 模型
向量数据库（Milvus / Qdrant / pgvector 都没有）
```

所以知识库检索不产生额外费用，也不需要额外服务。

## 四、明确没有使用的

| 能力 | 状态 |
|---|---|
| 向量检索 / embedding | 未使用，BM25 够用 |
| 语音合成 TTS | 未实现 |
| 数字人 | 未实现 |
| 图片识别 / OCR | 未实现，微信图片直接转人工 |
| 本地部署大模型 | 未使用，默认走云端 API（可换 Ollama） |
