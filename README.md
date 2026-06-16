# MiMo WebUI

基于 Go + Alpine.js 构建的 Xiaomi MiMo V2.5 多模态 API Web 界面。单二进制部署，零前端构建依赖。

## 功能

| 功能 | 模型 | 说明 |
|------|------|------|
| 💬 对话 | `mimo-v2.5` / `mimo-v2.5-pro` | 多 session、流式输出、推理过程展示、附件上传、自动标题生成 |
| 🖼️ 图片理解 | `mimo-v2.5` | 上传图片或输入 URL，AI 分析图片内容 |
| 🎵 音频理解 | `mimo-v2.5` | 上传音频，AI 分析音频内容 |
| 🎬 视频理解 | `mimo-v2.5` | 上传视频或输入 URL，可调帧率和分辨率 |
| 🎤 语音识别 | `mimo-v2.5-asr` | WAV/MP3 语音转文字，支持中英文 |
| 🔊 语音合成 | `mimo-v2.5-tts` | 8 种预置音色 + 声音设计 + 声音克隆 |

## 架构

```mermaid
graph TB
    subgraph Browser["浏览器"]
        direction TB
        Chat["💬 对话<br/>INFLIGHT per-session 流式"]
        Multi["🖼️🎵🎬🎤🔊<br/>多模态理解"]
        Auth["🔒 登录/注册"]

        Chat --> SSE["SSE 流式解析<br/>parseSSEStream"]
        Multi --> SSE
    end

    subgraph GoServer["Go 后端 (Gin)"]
        direction TB
        MW["JWT 认证中间件"]
        Upload["文件上传<br/>MIME 检测 + 大小校验"]
        Relay["SSE Relay<br/>原始转发，零重编码"]
        Client["MiMo API Client<br/>context.Background()<br/>浏览器断开不影响 API"]
        Title["标题生成<br/>首条消息 → MiMo → 标题"]
        Cleanup["临时文件清理<br/>定时删除过期文件"]
    end

    subgraph Storage["存储"]
        SQLite[("SQLite<br/>用户·会话·消息")]
        TmpFS["/tmp/mimo-uploads<br/>临时文件"]
    end

    subgraph MiMo["MiMo API"]
        ChatAPI["/v1/chat/completions<br/>流式 SSE"]
        ASRAPI["mimo-v2.5-asr"]
        TTSAPI["mimo-v2.5-tts<br/>tts-voicedesign<br/>tts-voiceclone"]
    end

    Browser -->|"HTTP/HTTPS"| MW
    MW --> Upload
    MW --> Relay
    MW --> Title
    Upload --> TmpFS
    Relay --> Client
    Title --> Client
    Client -->|OpenAI Chat API| ChatAPI
    Client --> ChatAPI
    Client --> ASRAPI
    Client --> TTSAPI
    ChatAPI -->|SSE| Relay
    Relay -->|"text/event-stream"| Browser
    MW --> SQLite
    Cleanup -.->|定期清理| TmpFS
```

## 技术栈

```mermaid
graph LR
    subgraph Backend["后端 (Go)"]
        Gin["Gin<br/>Web 框架"]
        SQLite[("modernc.org/sqlite<br/>纯 Go SQLite")]
        JWT["golang-jwt/v5<br/>JWT 认证"]
        Embed["embed.FS<br/>静态资源嵌入"]
    end

    subgraph Frontend["前端 (零构建)"]
        Alpine["Alpine.js 3<br/>响应式交互"]
        Tailwind["Tailwind CSS CDN<br/>实用优先样式"]
        Marked["marked.js<br/>Markdown 渲染"]
    end

    subgraph Infra["基础设施"]
        Config["config.toml<br/>TOML 配置"]
        Binary["单二进制<br/>无运行时依赖"]
    end

    Gin --> SQLite
    Gin --> Embed
    Embed --> Alpine
    Embed --> Tailwind
    Embed --> Marked
```

## 快速开始

```bash
# 1. 克隆
git clone https://github.com/GreyRaphael/mimo-webui-go.git
cd mimo-webui-go

# 2. 配置
cp config.toml.example config.toml
# 编辑 config.toml，填入你的 MiMo API Key

# 3. 构建 & 运行
go build -o mimo-webui .
./mimo-webui

# 4. 访问
# http://localhost:3000
# 默认账号：admin / config.toml 中的 admin_password
```

## 配置

```toml
[server]
host = "0.0.0.0"
port = 3000

[mimo]
api_key = "your-mimo-api-key"        # 或 MIMO_API_KEY 环境变量
base_url = "https://api.xiaomimimo.com/v1"

[auth]
jwt_secret = "change-me"              # 生产环境必须修改
admin_password = "your-password"      # 首次启动创建 admin 账户

[upload]
max_image_mb = 50
max_audio_mb = 100
max_video_mb = 500
temp_dir = "/tmp/mimo-uploads"

[database]
path = "mimo-webui.db"
```

## 项目结构

```
mimo-webui-go/
├── main.go                      # 入口 + 路由注册
├── config.toml.example          # 配置模板
├── internal/
│   ├── config/config.go         # TOML 配置解析
│   ├── auth/                    # JWT + bcrypt
│   ├── db/                      # SQLite CRUD (users/sessions/messages)
│   ├── mimo/                    # MiMo API Client + SSE + TTS
│   ├── handlers/                # HTTP Handlers
│   │   ├── chat.go              # 对话 (INFLIGHT 流式 + 标题生成)
│   │   ├── image/audio/video.go # 多模态理解
│   │   ├── asr.go               # 语音识别
│   │   ├── tts.go               # 语音合成
│   │   ├── upload.go            # 文件上传 + 媒体服务
│   │   └── relay.go             # SSE 原始转发
│   └── middleware/              # JWT 认证 + 临时文件清理
├── templates/pages/             # Go HTML 模板 (Alpine.js)
├── static/
│   ├── css/custom.css           # 全局样式 + 移动端适配
│   └── js/app.js                # IndexedDB + 移动端导航
└── mimo-webui.db                # SQLite 数据库 (自动生成)
```

## 关键设计

### INFLIGHT 流式模式

借鉴 [nesquena/hermes-webui](https://github.com/nesquena/hermes-webui) 的 per-session 状态管理：

```mermaid
sequenceDiagram
    participant U as 用户
    participant FE as 前端 (Alpine.js)
    participant BE as 后端 (Go)
    participant API as MiMo API

    U->>FE: 发送消息 (session1)
    FE->>FE: _inflight[session1] = {buffer, reasoning, active}
    FE->>BE: POST /api/sessions/1/messages
    BE->>API: ChatCompletion(context.Background())
    
    loop SSE 流式
        API-->>BE: data: {"delta":{"content":"你"}}
        BE-->>FE: data: {"delta":{"content":"你"}}
        FE->>FE: inf.buffer += "你"
        FE->>FE: _scheduleStreamRender() (rAF 节流)
    end

    U->>FE: 切换到 session2
    FE->>FE: currentSession = session2
    FE->>FE: 流式气泡消失 (只渲染 _current)
    Note over FE: session1 的流继续在后台累积

    API-->>BE: data: [DONE]
    BE->>BE: db.CreateMessage(session1, content)
    
    U->>FE: 切回 session1
    FE->>BE: loadMessages(session1)
    BE-->>FE: 包含 assistant 回复 ✓
```

### 浏览器断开恢复

```mermaid
sequenceDiagram
    participant U as 用户
    participant FE as 前端
    participant BE as 后端
    participant API as MiMo API

    U->>FE: 对话中，AI 正在输出
    FE->>BE: SSE 流式连接
    BE->>API: context.Background()
    
    U->>FE: 导航到"图片理解" (页面跳转)
    FE--xBE: 浏览器关闭连接
    
    Note over BE: handler 继续运行<br/>relaySSEStream 继续读取 API<br/>浏览器写入失败但被忽略
    
    API-->>BE: SSE 流继续...
    API-->>BE: data: [DONE]
    BE->>BE: db.CreateMessage ✓
    
    U->>FE: 回到"对话"页面
    FE->>FE: init() → loadMessages()
    FE->>BE: GET /api/sessions/1/messages
    BE-->>FE: 包含完整回复 ✓
    
    Note over FE: 如果回到时流还没结束<br/>_startReplyPolling 每2秒轮询<br/>直到回复出现
```

### 移动端适配

```mermaid
graph TB
    subgraph Desktop["PC 端 (≥768px)"]
        D1["w-56 导航栏"] --> D2["w-64 Session 列表"]
        D2 --> D3["flex-1 聊天区域"]
    end

    subgraph Mobile["移动端 (<768px)"]
        M1["☰ 汉堡菜单"] --> M2["全宽内容区"]
        M3["📋 Session 切换"] --> M2
        M2 --> M4["输入区"]
        
        MDrawer["导航抽屉<br/>(点击☰滑出)"]
        MSession["Session 抽屉<br/>(点击📋滑出)"]
    end
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/register` | 注册 |
| POST | `/api/login` | 登录 |
| GET | `/api/sessions` | 列出 sessions |
| POST | `/api/sessions` | 创建 session |
| DELETE | `/api/sessions/:id` | 删除 session |
| GET | `/api/sessions/:id/messages` | 获取消息 |
| POST | `/api/sessions/:id/messages` | 发送消息 (SSE 流式) |
| POST | `/api/sessions/:id/generate-title` | 自动生成标题 |
| POST | `/api/upload` | 上传文件 |
| GET | `/api/media/:file_id` | 获取上传文件 |
| POST | `/api/image` | 图片理解 |
| POST | `/api/audio` | 音频理解 |
| POST | `/api/video` | 视频理解 |
| POST | `/api/asr` | 语音识别 |
| POST | `/api/tts` | 语音合成 |

## License

MIT
