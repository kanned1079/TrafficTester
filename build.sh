#!/bin/bash
set -e

# 镜像名称
IMAGE_NAME="traffic-tester:latest"
# 导出镜像 tar.gz 文件
OUTPUT_DIR="./bin"
OUTPUT_FILE="${OUTPUT_DIR}/traffic-tester.tar.gz"

echo "Step 1: Build Go executable..."
make build

echo "Step 2: Build Docker image..."
docker build -t $IMAGE_NAME .

echo "Step 3: Save Docker image and compress to ${OUTPUT_FILE}..."
mkdir -p $OUTPUT_DIR
docker save $IMAGE_NAME | gzip > $OUTPUT_FILE

echo "Build completed successfully!"
echo "Executable: ./bin/exec"
echo "Docker image tar.gz: $OUTPUT_FILE"