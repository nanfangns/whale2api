# Whale2API

DeepSeek 反代，支持 896K 上下文的 `deepseek-flash` (v4.1)

## 致谢与声明

本项目在 **[ds2api](https://github.com/CJackHwang/ds2api)**（作者 [CJackHwang](https://github.com/CJackHwang)）基础上二次开发并重命名为 Whale2API

【感谢Linux.do社区对项目的支持，感谢上游作者及贡献者的开源工作】

CJackHwang佬的作品我个人使用过很长一段时间，为我减轻了许多经济上的负担，中转太多导致官方出手很是可惜

本项目重新提供更稳定版本的开源。网页端限制导致模型效果并不好，还为了防封禁做了很多限制，希望自用而非盈利

## 快速开始

### Docker

#### 源码构建

```
copy .env.example .env
# 编辑 .env：设置 POOL_UI_ADMIN_TOKEN
docker compose up -d --build
```

#### release

```
docker network create whale2api

docker run -d --name device-harvest --restart unless-stopped --network whale2api \
  -e PORT=8090 -e HARVEST_CONCURRENCY=2 \
  ghcr.io/tangsong404/whale2api-device-harvest:latest

docker run -d --name whale2api --restart unless-stopped --network whale2api \
  -p 5103:5001 \
  -e PORT=5001 \
  -e WHALE2API_DATABASE_PATH=/data/whale2api.db \
  -e DEVICE_HARVEST_URL=http://device-harvest:8090 \
  -v whale2api-data:/data \
  ghcr.io/tangsong404/whale2api:latest

docker run -d --name poolui --restart unless-stopped --network whale2api \
  -p 5010:5010 \
  -e POOL_UI_PORT=5010 \
  -e WHALE2API_DATABASE_PATH=/data/whale2api.db \
  -e DEVICE_HARVEST_URL=http://device-harvest:8090 \
  -e POOL_UI_ADMIN_TOKEN=change-me \
  -v whale2api-data:/data \
  ghcr.io/tangsong404/whale2api:latest /usr/local/bin/poolui
```

### Vercel Hobby

Vercel uses the container deployment in `vercel.json`: one service runs the Go
gateway and pool UI, and a private service runs the on-demand Chromium device
harvester. This is separate from Docker Compose; pool state is stored in
PostgreSQL rather than an ephemeral container file.

In Vercel Project Settings, add these environment variables before deploying:

| Variable | Purpose |
| --- | --- |
| `WHALE2API_DATABASE_URL` | PostgreSQL connection URL. The app creates its tables on first start. |
| `POOL_UI_ADMIN_TOKEN` | Strong password for the pool UI, served at `/`. |
| `DEVICE_HARVEST_TOKEN` | Shared secret between the gateway and private harvester service. |

The service binding injects `DEVICE_HARVEST_URL`; it must not be set to a public
URL. Do not put credentials in `vercel.json`, commit them, or use the SQLite
file path on Vercel. The pool UI's bulk account-test job is process-local and
may be interrupted if a Vercel instance scales down; use it for low-volume
personal testing, not as a durable background queue.

Vercel Services and container images are currently in beta. Hobby is limited
to personal, non-commercial use and has finite compute/function quotas. Vercel
also limits function request/response bodies to 4.5 MB, so very large prompts
or base64 images may need smaller inputs. See [Vercel Services](https://vercel.com/docs/services),
[Hobby limits](https://vercel.com/docs/plans/hobby), and [Function limits](https://vercel.com/docs/functions/limitations).

### 本地开发

```
# 先起采集服务
cd tools/device-harvest && npm install && xvfb-run -a node harvest.mjs serve --port 8090 &
# 网关/面板需指向它
DEVICE_HARVEST_URL=http://127.0.0.1:8090 go run ./cmd/whale2api   # :5001
DEVICE_HARVEST_URL=http://127.0.0.1:8090 go run ./cmd/poolui      # :5010
go test ./...
```

## 使用说明

| 地址                                                                                     | 用途       |
| -------------------------------------------------------------------------------------- | -------- |
| [http://127.0.0.1:5103/v1/chat/completions](http://127.0.0.1:5103/v1/chat/completions) | 网关 API   |
| [http://127.0.0.1:5010](http://127.0.0.1:5010)                                         | 号池 WebUI |

![号池 WebUI 预览](docs/screenshot.png)

建议50个号起用（批量注册参考我的仓库 `signup-god`）

导入csv格式: `email,password[,device_id]`（第三列可留空，留空即按需采集）


## 改了什么

### 优化

1.去除所有和`DS2API`相关文本，大改提示词、工具调用符号降低封禁率

2.号池增加了对`禁言`（不是`封禁`）机制的检测，且持久化由json改为sqlite

3.增添v4.1模型 `deepseek-flash`，支持多模态（图片输入）

4.优化工具调用，减少`光说不做`与`假想完成`的情况

### 限制

1.暂时仅支持OpenAI Chat Completions兼容

2.所有模型上下文限制为 896K

## 参与贡献

欢迎通过 Pull Request 参与改进，包括但不限于：

- Bug 修复与测试补充
- 文档与使用说明完善
- 号池、网关稳定性与可观测性优化

提交 PR 前建议：

1. 在本地执行 `go test ./...` 确保通过
2. 保持改动聚焦，说明修改动机与验证方式
3. Fork 后从功能分支发起 PR，便于 review

如有较大改动，建议先开 Issue 简要讨论方案，避免重复劳动。
