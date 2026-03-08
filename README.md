# 🔀 Chat Aggregator

**Combine Twitch and Kick chat into one beautiful view!**

Perfect for streamers who multi-stream. See all your chat messages in a single window, making it easier to interact with your community across platforms.

---

## 🎯 What Does This Do?

Have you ever streamed on both Twitch AND Kick at the same time? It's annoying to watch two separate chat windows, right?

**This tool fixes that!** It combines both chats into one clean display that you can:

- Watch in a browser window
- Add as an overlay in OBS for your viewers to see
- Customize to match your stream's style

---

## ⚡ Quick Start (5 Minutes)

### Step 1: Download

Pick the right file for your computer:

| Your Computer | Download This |
|---------------|---------------|
| 🪟 Windows | `chat-aggregator-windows-amd64.exe` |
| 🍎 Mac (M1/M2/M3) | `chat-aggregator-darwin-arm64` |
| 🍎 Mac (Intel) | `chat-aggregator-darwin-amd64` |
| 🐧 Linux | `chat-aggregator-linux-amd64` |

### Step 2: Run It!

**Windows:**
- Double-click the downloaded `.exe` file
- Follow the setup wizard that appears

**Mac:**
1. Open Terminal
2. Run these commands:
```bash
chmod +x chat-aggregator-darwin-arm64
./chat-aggregator-darwin-arm64
```

**Linux:**
```bash
chmod +x chat-aggregator-linux-amd64
./chat-aggregator-linux-amd64
```

### Step 3: Open in Browser

After running, open this address in your browser:

```
http://localhost:8080
```

**That's it!** Your combined chat is now live! 🎉

---

## 📺 Add to OBS (Browser Overlay)

Want your viewers to see the combined chat on stream?

1. In OBS, click the **+** in Sources
2. Select **Browser Source**
3. Name it "Chat Aggregator"
4. Set URL to: `http://localhost:8080?obs=true`
5. Set Width: `400` and Height: `600`
6. Click OK

Now your combined chat appears on your stream!

---

## 🔑 Getting Your Twitch Token

The setup wizard will ask for a Twitch OAuth token. Here's how to get one:

1. Go to **[twitchtokengenerator.com](https://twitchtokengenerator.com/)**
2. Click **"Connect with Twitch"**
3. Log in with your Twitch account
4. Check the box for **"Chat:Read"**
5. Click **"Generate Token"**
6. Copy the token (it starts with `oauth:`)

**Note:** Kick doesn't need any token - just enter your channel name!

---

## 🛠️ Manual Setup (Optional)

If you prefer to set up manually instead of using the wizard:

1. Create a file called `.env` in the same folder as the program
2. Add your settings:

```env
# Twitch Settings
TWITCH_USERNAME=your_username
TWITCH_OAUTH_TOKEN=oauth:your_token_here
TWITCH_CHANNEL=your_channel

# Kick Settings (no token needed!)
KICK_CHANNEL=your_channel

# Port (optional, default is 8080)
PORT=8080
```

---

## 🎨 What It Looks Like

Messages from each platform have different colors:

- 🟣 **Purple** = Twitch messages
- 🟢 **Green** = Kick messages

This makes it easy to see at a glance where each message came from!

---

## ❓ Common Questions

### "It says 'Port already in use'"

Another program is using port 8080. Either:
- Close that program, or
- Change the port in your `.env` file

### "Twitch won't connect"

- Make sure your OAuth token starts with `oauth:`
- Check that your token hasn't expired
- Try generating a new token

### "Kick messages aren't showing"

- Make sure the channel name is correct
- The channel must be live for messages to appear

### "How do I stop it?"

- Press `Ctrl+C` in the terminal/command prompt window

---

## 📊 Stats API

Want to see how many messages you've received? Open:

```
http://localhost:8080/api/stats
```

You'll see a JSON response with message counts and other info.

---

## 🔒 Is It Safe?

Yes! Here's why:

- ✅ Your Twitch token is only sent to Twitch servers
- ✅ All connections use secure HTTPS/WSS
- ✅ Your token is never logged or saved insecurely
- ✅ The tool runs locally on your computer only

---

## 🚀 For Developers

Want to contribute or build from source?

### Requirements
- Go 1.21+

### Build Commands
```bash
# Clone the repo
git clone https://github.com/Aliexe-code/ChatAggregator.git
cd ChatAggregator

# Install dependencies
go mod download

# Run tests
go test ./...

# Build for your platform
go build -o chat-aggregator .

# Build for all platforms
GOOS=windows go build -o chat-aggregator-windows.exe .
GOOS=linux go build -o chat-aggregator-linux .
GOOS=darwin go build -o chat-aggregator-mac .
```

### Test Coverage
The project has ~70% test coverage. Some code (network handlers, signal handlers, infinite loops) is inherently difficult to unit test without integration tests.

---

## 📝 License

MIT License - free to use, modify, and distribute.

---

## 💜 Made for Multi-Platform Streamers

If this tool helped your stream, consider:
- ⭐ Starring the repo on GitHub
- 🐛 Reporting any bugs you find
- 💡 Suggesting new features

**Happy streaming!** 🎮