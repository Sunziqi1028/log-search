# Log Searcher

轻量级日志搜索工具，类似 Notepad++ 的关键字搜索高亮体验。

## 特点

- **本地读取** — 直接在服务器上读取日志文件，无需下载到本地
- **关键字搜索** — 支持多关键字（空格/逗号/分号分隔），大小写不敏感
- **Notepad++ 风格高亮** — 匹配的文本在搜索结果中高亮显示（黄色背景）
- **目录浏览** — 按目录组织日志文件，点击即可查看
- **零外部依赖** — 纯 Go 实现，无需 Elasticsearch/Kafka/JVM

## 快速开始

```bash
# 1. 编译
go build -o log-searcher ./cmd/web/

# 2. 启动（指定日志目录）
./log-searcher --dirs /var/log/myapp

# 也可以指定多个目录
./log-searcher --dirs "/var/log/app1,/var/log/app2"

# 3. 访问
# http://localhost:8080
```

## 配置

```bash
# 自定义监听端口
./log-searcher --addr :9090 --dirs /var/log/myapp
```

## 使用

1. 左侧选择目录浏览日志文件
2. 输入关键字搜索（支持多个关键字，用空格/逗号/分号分隔）
3. 搜索结果中高亮显示匹配的文本
4. 点击任意结果查看上下文（前后5行）

## 部署

```bash
# 安装到系统
sudo cp log-searcher /usr/local/bin/
sudo tee /etc/systemd/system/log-searcher.service > /dev/null <<EOF
[Unit]
Description=Log Searcher
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/log-searcher --dirs /var/log/myapp --addr :8080
Restart=always

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now log-searcher
```
