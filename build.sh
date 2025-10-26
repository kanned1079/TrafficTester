#!/bin/bash
set -e

# 镜像名称
IMAGE_NAME="traffic-tester:latest"
# 输出目录和文件
OUTPUT_DIR="./bin"
OUTPUT_FILE="${OUTPUT_DIR}/traffic-tester.tar.gz"

echo "🛠 Step 1: Build Go executable (amd64)..."
make build

echo "🐳 Step 2: Build Docker image for amd64..."
# 使用 buildx 可确保在 ARM 等平台也能构建 amd64 镜像
docker buildx build --platform linux/amd64 -t $IMAGE_NAME --load .

echo "📦 Step 3: Save Docker image and compress to ${OUTPUT_FILE}..."
mkdir -p "$OUTPUT_DIR"
docker save "$IMAGE_NAME" | gzip > "$OUTPUT_FILE"

echo "✅ Build completed successfully!"
echo "Executable: ./bin/traffic-tester-exec"
echo "Docker image tar.gz: $OUTPUT_FILE"