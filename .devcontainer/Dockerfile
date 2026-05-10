FROM golang:1.26-bookworm

RUN apt-get update && apt-get install -y \
    git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1000 vscode \
    && useradd -u 1000 -g vscode -m -s /bin/bash vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Go キャッシュディレクトリを vscode ユーザーで作成しておく
# Docker named volume は初回マウント時にここのパーミッションを引き継ぐ
RUN mkdir -p /home/vscode/go/pkg/mod /home/vscode/.cache/go-build /home/vscode/.claude \
    && chown -R vscode:vscode /home/vscode/go /home/vscode/.cache /home/vscode/.claude

WORKDIR /workspace
USER vscode
