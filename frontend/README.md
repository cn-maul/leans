# leans 前端

公考题目分析系统的 Web 界面：React 19 + TypeScript + Vite + Tailwind CSS 4，构建产物由后端通过 `go:embed` 嵌入二进制（`backend/static/`）。

## 开发

```bash
npm install
npm run dev   # Vite 开发服务器 :3000，/api 代理到 localhost:8080（需先启动后端）
```

## 构建

```bash
npm run build   # tsc 类型检查 + vite 构建，输出到 dist/
npm run lint    # oxlint
```

发布流程见根目录 `build.ps1`：编译前端 → 复制 `dist/` 到 `backend/static/` → 编译 Go 二进制。

## 目录结构

```
src/
├── api/client.ts       # 后端 API 封装（含 SSE 流式读取与取消）
├── components/         # UI 组件（输入卡、设置弹窗、结果面板、抽屉等）
├── hooks/              # useAnalysis / useSettings / useHistory 等状态逻辑
├── types/analysis.ts   # 与后端 model 包对应的类型定义
└── index.css           # Tailwind 入口与主题变量
```
