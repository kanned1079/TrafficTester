# 使用 Alpine 最小基础镜像
FROM alpine:3.18

# 设置工作目录
WORKDIR /app

# 拷贝可执行文件和配置文件到 /app/
COPY bin/exec /app/exec
COPY config/conf.yaml /app/config/conf.yaml

# 给 exec 可执行权限
RUN chmod +x /app/exec

# 暴露端口（如果程序需要，可选）
# EXPOSE 8080

# 启动命令
ENTRYPOINT ["/app/exec"]