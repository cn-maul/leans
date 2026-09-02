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

**方式一：界面设置（推荐）**：点击右上角「⚙ 设置」，填写服务商 / Base URL / 模型 / API Key，保存到 `backend/data/leans.db`，立即生效、无需重启。可用「测试连接」按钮验证配置是否可用。

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

界面保存的设置优先于配置文件。

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
4. 点击"开始分析"
5. 左侧查看标注原文（关键词高亮直接打在题干/选项上），右侧查看分析结果：
   - **题目匹配**：判断本题属于哪一类题型，列出用于判断的词句（蓝色）
   - **适用规则**：文段与选项中命中该题型的技巧规则及具体词句（绿色）
   - **注释和思路**：本题答案与完整解题思路（紫色）
   - 顶部显示本次处理的**耗时与 token 用量**
6. 标注颜色与右侧模块颜色一一对应：蓝色=题型判断依据、绿色=适用规则命中、紫色=答案/思路、红色=错误/转折、黄色=关键信息
7. 顶栏「讲义」抽屉按章节树浏览讲义，「历史」抽屉点击任意记录可一键回填题目与完整分析结果

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

任何兼容OpenAI Chat Completion API的服务：
- OpenAI (GPT-4o)
- DeepSeek
- 通义千问
- 本地部署的OpenAI兼容API

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

- 后端：`cd backend && go run .`
- 前端：`cd frontend && npm run dev`（Vite 开发服务器，代理 `/api` 到 8080）
- 测试：`cd backend && go test ./...`
