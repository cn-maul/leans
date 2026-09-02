# 公考题目分析系统 - 编译启动脚本
param(
    [switch]$BuildOnly,
    [switch]$RunOnly
)

$ErrorActionPreference = "Stop"
$Root = $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  公考题目分析系统 - 编译启动脚本" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 检查 Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[错误] 未找到 Go，请先安装 Go" -ForegroundColor Red
    exit 1
}

# 检查 Node
if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Host "[错误] 未找到 npm，请先安装 Node.js" -ForegroundColor Red
    exit 1
}

# 步骤1: 编译前端
if (-not $RunOnly) {
    Write-Host "`n[1/3] 编译前端..." -ForegroundColor Yellow
    Push-Location "$Root\frontend"
    try {
        npm install --prefer-offline 2>$null | Out-Null
        npm run build
        if ($LASTEXITCODE -ne 0) { throw "前端编译失败" }
        Write-Host "  ✓ 前端编译完成" -ForegroundColor Green
    } finally {
        Pop-Location
    }

    # 复制前端产物到后端
    Write-Host "`n[2/3] 嵌入前端产物..." -ForegroundColor Yellow
    if (Test-Path "$Root\backend\static") {
        Remove-Item -Recurse -Force "$Root\backend\static"
    }
    Copy-Item -Recurse -Force "$Root\frontend\dist" "$Root\backend\static"
    Write-Host "  ✓ 前端产物已复制到 backend/static" -ForegroundColor Green
}

# 步骤2: 编译后端
if (-not $RunOnly) {
    Write-Host "`n[3/3] 编译后端..." -ForegroundColor Yellow
    Push-Location "$Root\backend"
    try {
        $env:GOPROXY = "https://goproxy.cn,direct"
        go build -o leans.exe .
        if ($LASTEXITCODE -ne 0) { throw "后端编译失败" }
        Write-Host "  ✓ 后端编译完成: backend\leans.exe" -ForegroundColor Green
    } finally {
        Pop-Location
    }
}

# 步骤3: 启动
if (-not $BuildOnly) {
    Write-Host "`n启动服务..." -ForegroundColor Yellow
    Write-Host "  地址: http://localhost:8080" -ForegroundColor Cyan
    Write-Host "  按 Ctrl+C 停止" -ForegroundColor Gray
    Write-Host "========================================" -ForegroundColor Cyan

    Push-Location "$Root\backend"
    try {
        .\leans.exe
    } finally {
        Pop-Location
    }
}
