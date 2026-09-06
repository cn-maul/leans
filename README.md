# 公考题目分析系统

基于讲义的智能题目解析与技巧匹配系统。

## 快速开始

### 1. 编译并运行（推荐，单二进制）

```powershell
# 编译前端并嵌入后端
.\build.ps1 -BuildOnly

# 运行
cd backend
.\leans.exe
```

访问 http://localhost:8080

> 运行时只依赖一个 `leans.exe` 二进制，前端已嵌入其中，属于纯 Go 运行时（SQLite 用纯 Go 驱动，无需 CGO）。`config.yaml` 缺失时自动使用内置默认值，单文件即可运行。

### 2. 配置 AI（两种方式）

**方式一：界面设置（推荐）**：点击右上角「⚙ 设置」管理**多个 AI 供应商**，每个供应商填写名称 / 接口类型 / Base URL / API Key，模型可手动输入或点「获取模型」从服务端在线拉取（在待选列表中点选添加）。主页「开始分析」按钮旁有二级下拉：一级选供应商、二级选模型，切换即时生效并持久化到 `backend/data/leans.db`。每个供应商支持「测试连接」在保存前验证配置。

**方式二：配置文件（作为默认值兜底）**：编辑 `backend/config/config.yaml`：

```yaml
ai:
  provider: "openai"
  api_key: "sk-your-api-key"
  base_url: "https://api.openai.com/v1"
  model: "gpt-4o"

data:
  dir: "data"        # SQLite 数据库目录
subjects:
  dir: "../subjects" # 讲义目录
```

配置文件的 AI 段作为开箱默认供应商；界面保存后以界面配置为准。

### 3. 添加讲义

将讲义Markdown文件放入 `subjects/` 目录：

```
subjects/
├── 言语理解.md
├── 判断推理.md
└── 数量关系.md
```

## 使用方法

1. 打开浏览器访问 http://localhost:8080
2. 在顶栏选择科目
3. 在中间粘贴题目（带选项）
4. 通过「开始分析」旁的二级下拉选择供应商与模型（在设置中添加的供应商会出现在一级下拉里）
5. 点击"开始分析"；等待过程中按钮变为「中止」，随时可取消——后端会同步中断 AI 调用，已收到的部分内容丢弃、不写入历史
6. 左侧查看标注原文（关键词高亮直接打在题干/选项上），右侧查看分析结果：
   - **题目匹配**：判断本题属于哪一类题型，列出用于判断的词句（蓝色）
   - **适用规则**：文段与选项中命中该题型的技巧规则及具体词句（绿色）
   - **注释和思路**：本题答案与完整解题思路（紫色）
   - 顶部显示本次处理的**耗时与 token 用量**
7. 标注颜色与右侧模块颜色一一对应：蓝色=题型判断依据、绿色=适用规则命中、紫色=答案/思路、红色=错误/转折、黄色=关键信息
8. 顶栏「讲义」抽屉按章节树浏览讲义，「历史」抽屉点击任意记录可一键回填题目与完整分析结果

## 分析质量说明

- **按需检索讲义**：分析时不会把整本讲义塞进 Prompt，而是从题目提取关键词，检索最相关的讲义章节（含整体概述）注入，减少 token 浪费、避免无关章节干扰判断
- **三块结构化输出**：AI 按「题目匹配 / 适用规则 / 注释和思路」三块返回，其中题型判断依据、每条规则命中的词句都会精确指向题目原文位置
- **输出容错**：AI 返回的 JSON 会做容错解析（剥离代码围栏、注释、前后缀文字，深度匹配花括号），颜色/位置/模块自动规范化；解析失败自动重试一次
- **高亮模糊匹配**：前端对 AI 标注的文本做全角/半角、标点归一化后匹配原文，提高标注命中率
- **耗时与用量**：每次分析记录处理耗时与 token 用量并在结果页展示

## Token 用量控制

单次分析的 token 占用可以通过 `backend/config/config.yaml` 的 `analysis` 段调节（实测默认配置约 4100 token/次）：

| 参数 | 默认 | 作用 |
|------|------|------|
| `lecture_budget` | 6000 | 讲义注入字符上限（输入 token 大头），调小最有效 |
| `max_sections` | 2 | 检索注入的相关章节数 |
| `max_chars` | 1500 | 每个章节最大字符 |
| `overview_max` | 1 | 讲义整体概述章节数 |
| `max_tokens` | 4096 | 输出 token 硬顶，防模型输出失控 |

> 注意：`max_tokens` 不要设太小——输出 JSON 被截断会导致解析失败并重试一次，反而更费 token。

## 数据存储

- 设置与历史：SQLite 数据库 `backend/data/leans.db`（首次运行自动创建，历史记录会保存完整分析结果以便回填）
- 讲义：从 `subjects/` 目录的 Markdown 文件读取

## 支持的AI服务

AI 接入基于自研的 [rosetta](https://github.com/cn-maul/rosetta) 库（纯 Go、零第三方依赖），统一封装三种协议，每个供应商可在设置中指定或自动检测：

- **OpenAI Chat Completions**（绝大多数服务的兼容协议）：OpenAI、DeepSeek、通义千问、Moonshot、GLM、OpenRouter、vLLM、Ollama 等
- **OpenAI Responses**
- **Anthropic Messages**

「接口类型」选自动检测时，后端会探测端点的模型目录并识别协议，失败回落 OpenAI 兼容。库内建第三方兼容处理（`max_tokens` 字段探测降级、`stream_options` 兼容、错误体双格式解析）与传输层自动重试；页面「中止」或连接断开会立即中断出站 AI 请求，不产生多余 token 消耗。

## 项目结构

```
leans/
├── backend/          # Go 后端（纯 Go，无 CGO）
│   ├── handler/      # API 路由处理
│   ├── service/      # 业务逻辑（AI 调用、分析、检索、Prompt 构建）
│   ├── subject/      # 讲义解析与章节检索
│   ├── model/        # 数据模型
│   ├── storage/      # SQLite 存储（设置/历史）
│   └── config/       # 配置管理
├── frontend/         # React 前端（构建后嵌入 Go 二进制）
│   └── src/
│       ├── components/  # UI 组件
│       ├── hooks/       # 数据获取 hooks
│       ├── api/         # API 调用
│       └── types/       # TypeScript 类型
└── subjects/         # 讲义文件
```

## 开发

- 后端：`cd backend && go run .`（首次需先编译前端并把 `frontend/dist` 复制到 `backend/static`——`backend/static` 不入库，推荐直接用根目录的 `build.ps1` 一步完成）
- 前端：`cd frontend && npm run dev`（Vite 开发服务器，代理 `/api` 到 8080）
- 测试：`cd backend && go test ./...`
