#!/usr/bin/env bash
set -e

echo "=========================================="
echo "  Firecrawl Go API 自动化测试套件"
echo "=========================================="

cd "$(dirname "$0")/../go-api"

echo "1. 执行代码静态分析 (go vet)..."
go vet ./...
echo "   ✅ 静态检查通过 (0 warnings)"

echo "2. 执行全量单元测试与路由集成测试 (go test)..."
go test -v ./...
echo "   ✅ 全部单元测试与 TCP 实时网络集成测试通过"

echo "3. 执行二进制可执行文件编译验证..."
go build -o /tmp/firecrawl-go-api main.go
echo "   ✅ Go API 网关二进制构建成功 (/tmp/firecrawl-go-api)"

echo "=========================================="
echo "  🎉 全部测试已通过！系统已具备上线条件！"
echo "=========================================="
