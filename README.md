# 🔀 Multi-Platform Chat Aggregator

Combine Twitch and Kick chat messages into one unified view! Perfect for streamers who stream on multiple platforms simultaneously.

![Demo](https://via.placeholder.com/800x400/0e0e10/9146FF?text=Chat+Aggregator+Demo)

## ✨ Features

- **Unified Chat View** - See messages from Twitch and Kick in one window
- **Platform Icons** - Each message shows which platform it came from
- **Real-time Updates** - Messages appear instantly via WebSocket
- **OBS Compatible** - Add as a Browser Source overlay
- **Dark Mode** - Easy on the eyes during long streams
- **Secure** - OAuth tokens are never logged or exposed

## 🚀 Quick Start

### Step 1: Download

Download the latest release for your operating system:

| Platform | Download |
|----------|----------|
| Windows | `chat-aggregator-windows.exe` |
| macOS | `chat-aggregator-mac` |
| Linux | `chat-aggregator-linux` |

### Step 2: Create Configuration

Create a file named `.env` in the same folder as the executable:

```env
# Twitch Configuration
TWITCH_USERNAME=your_bot_username
TWITCH_OAUTH_TOKEN=oauth:your_token_here
TWITCH_CHANNEL=your_channel_name

# Kick Configuration (no auth needed!)
KICK_CHANNEL=your_channel_name

# Server Port
PORT=8080
```

### Step 3: Get Twitch OAuth Token

1. Go to [Twitch Token Generator](https://twitchtokengenerator.com/)
2. Click "Connect with Twitch"
3. Select scopes: `chat:read`
4. Copy the token (starts with `oauth:`)

### Step 4: Run

**Windows:**
```bash
chat-aggregator-windows.exe
```

**macOS:**
```bash
chmod +x chat-aggregator-mac
./chat-aggregator-mac
```

**Linux:**
```bash
chmod +x chat-aggregator-linux
./chat-aggregator-linux
```

### Step 5: Open in Browser

Open `http://localhost:8080` in your browser to see the combined chat!

## 📺 OBS Setup

To add the chat as an overlay in OBS:

1. In OBS, add a new **Browser Source**
2. Set the URL to: `http://localhost:8080?obs=true`
3. Set width/height (recommended: 400x800)
4. Click OK

The `?obs=true` parameter hides the header and footer for a cleaner overlay.

## 🎨 What It Looks Like

```
┌──────────────────────────────────────────────────┐
│  🔀 Chat Aggregator           ● Connected        │
├──────────────────────────────────────────────────┤
│  14:32:05  🟣 TWITCH  xQc: LETS GOOOO           │
│  14:32:06  🟢 KICK    Train: nice stream        │
│  14:32:08  🟣 TWITCH  poki: love this game      │
│  14:32:10  🟢 KICK    nick: first time here     │
│  14:32:12  🟣 TWITCH  shroud: PogChamp          │
│  14:32:15  🟢 KICK    viewer: this is cool!     │
└──────────────────────────────────────────────────┘
```

## 🔧 Configuration Options

| Variable | Required | Description |
|----------|----------|-------------|
| `TWITCH_USERNAME` | Yes* | Your Twitch bot/account username |
| `TWITCH_OAUTH_TOKEN` | Yes* | OAuth token (from twitchtokengenerator.com) |
| `TWITCH_CHANNEL` | Yes* | Channel to join (without #) |
| `KICK_CHANNEL` | Yes* | Kick channel to join |
| `PORT` | No | Web server port (default: 8080) |

*\*At least one platform must be configured*

## 🛠️ Build from Source

If you want to build from source:

### Requirements

- Go 1.21 or higher

### Steps

```bash
# Clone the repository
git clone https://github.com/yourusername/chat-aggregator.git
cd chat-aggregator

# Install dependencies
go mod download

# Build for your platform
go build -o chat-aggregator .

# Or build for all platforms
GOOS=windows go build -o chat-aggregator-windows.exe .
GOOS=linux go build -o chat-aggregator-linux .
GOOS=darwin go build -o chat-aggregator-mac .
```

## 🔒 Security

- **OAuth tokens** are never logged or printed
- **WebSocket connections** use secure protocols (wss://)
- **HTML content** is escaped to prevent XSS
- **Rate limiting** prevents message floods

## 📊 Stats API

View server statistics at `http://localhost:8080/api/stats`:

```json
{
  "total_messages": 1234,
  "twitch_messages": 800,
  "kick_messages": 434,
  "connected_clients": 1,
  "uptime_seconds": 3600
}
```

## 🧪 Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run verbose
go test -v ./...
```

## ❓ Troubleshooting

### "Failed to connect to Twitch"

- Check your OAuth token is valid
- Make sure it starts with `oauth:`
- Verify the token has `chat:read` scope

### "Failed to connect to Kick"

- Check the channel name is correct
- The channel must exist on Kick

### "Port already in use"

- Change the `PORT` in your `.env` file
- Or close the application using that port

### Messages not appearing

- Check the console for error messages
- Verify your channel names are correct
- Make sure you're actually streaming/live

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## 💜 Acknowledgments

- Built with [Go](https://golang.org/)
- WebSocket support by [gorilla/websocket](https://github.com/gorilla/websocket)
- Inspired by streamers who stream on multiple platforms

---

**Made with 💜 for multi-platform streamers**
