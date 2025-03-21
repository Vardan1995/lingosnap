# LingoSnap - Quick Text Translator

A lightweight desktop application that instantly translates selected text between Armenian (including Latin transliteration) and English.

## Features

- 🚀 One-click translation with Right Shift key
- 🔄 System tray integration for easy access
- 🌐 Support for both English and Armenian (including Latin transliteration)
- ✍️ Grammar and spelling corrections for English text
- 📝 Instant replacement of selected text with translation
- 💻 Cross-platform support (Windows, macOS, Linux)

## Quick Start

1. Get your Gemini API key from [Google AI Studio](https://makersuite.google.com/app/apikey)

2. Clone and set up:
```bash
git clone https://github.com/Vardan1995/lingosnap
cd lingosnap
copy .env.example .env
```

3. Add your API key to `.env`:
```bash
GEMINI_API_KEY=your-api-key-here
```

4. Build and run:
```bash
go build
.\lingosnap.exe
```

## How to Use

1. Select any text in any application
2. Press and release the Right Shift key
3. Watch the magic happen! ✨

## System Requirements

- Go 1.23+
- Internet connection
- Windows/macOS/Linux
- Administrative privileges (for keyboard hooks)

## Development Setup

```bash
# Install dependencies
go mod download

# Run in development mode
go run main.go

# Build for production
go build
```

## Keyboard Shortcuts

| Action | Windows | macOS | Linux |
|--------|---------|-------|-------|
| Translate | Right Shift | Right Shift | Right Shift |
| Copy | Ctrl+C | Cmd+C | Ctrl+C |
| Paste | Ctrl+V | Cmd+V | Ctrl+V |

## Troubleshooting

### Common Issues

- **Translation not working?**
  - Check your API key in `.env`
  - Ensure internet connectivity
  - Run as administrator

- **System tray icon missing?**
  - Restart the application
  - Check system tray settings

- **Keyboard shortcuts not responding?**
  - Run as administrator
  - Check for conflicts with other applications

## Technical Details

### Dependencies

```go
github.com/atotto/clipboard      // Clipboard operations
github.com/getlantern/systray    // System tray integration
github.com/go-vgo/robotgo       // Keyboard simulation
github.com/joho/godotenv        // Environment configuration
github.com/robotn/gohook        // Keyboard event handling
```


## License

MIT License - See [LICENSE](LICENSE) for details

---

Made with ❤️ by [Vardan1995](https://github.com/Vardan1995)

## Need Help?

- 🐛 [Report a bug](https://github.com/Vardan1995/lingosnap/issues)
- 💡 [Request a feature](https://github.com/Vardan1995/lingosnap/issues)
- 📧 [Contact the developer](mailto:your.email@example.com)