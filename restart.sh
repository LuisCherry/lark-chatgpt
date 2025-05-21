#!/bin/bash

# 1. 删除正在运行的 docker 容器
if docker ps -q -f name=feishu-chatgpt > /dev/null; then
    echo "停止并删除正在运行的容器 feishu-chatgpt..."
    docker stop feishu-chatgpt
    docker rm feishu-chatgpt
fi

# 2. 进入指定目录
cd /media/nvme/data/lark-chat || { echo "目录不存在"; exit 1; }

# 3. 拉取最新代码
echo "拉取最新代码..."
git pull

# 4. 构建 Docker 镜像
echo "构建镜像..."
docker build -t feishu-chatgpt:latest .

# 5. 启动容器
echo "启动容器..."
docker run -d --name feishu-chatgpt -p 7000:9000 \
    --env APP_ID=cli_a8a6907d60b81029 \
    --env APP_SECRET=AvA98VCzp4nK8VMTzeCE5dcscshcJRq8 \
    --env APP_ENCRYPT_KEY=8wp9f7lWOqyViSTniRL93gLbLFNS1Qnv \
    --env APP_VERIFICATION_TOKEN=jYEr3PuR1bgEWyIR818eIh0YSVxCHFsO \
    --env BOT_NAME=chatGpt \
    --env OPENAI_KEY="sk-895e2ca99ab04024a7c468f68797554a" \
    --env API_URL="https://api.deepseek.com" \
    --env HTTP_PROXY="" \
    --env MODEL="deepseek-chat" \
    feishu-chatgpt:latest

# 6. 打印容器日志
echo "容器启动成功，正在查看日志..."
docker logs -f feishu-chatgpt
