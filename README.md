# IGCmail IMAP

A desktop application for Windows, macOS, and Linux that monitors an IMAP mailbox and automatically extracts IGC (flight track) file attachments. Built with Go and Fyne for a native cross-platform experience.

## ✨ Features

- **🔒 Secure IMAP Connection**: TLS-encrypted connection (port 993) to any IMAP server
- **📬 Smart Incremental Sync**: Only fetches new messages using UID-based tracking
- **🎯 IGC File Extraction**: Automatically extracts .igc attachments to a configurable folder
- **🔄 Duplicate Handling**: Same filenames get timestamped to avoid overwrites
- **📱 System Tray Integration**: Minimizes to tray with comprehensive menu controls
- **📝 Comprehensive Logging**: Detailed operation logs with configurable output
- **🔔 Desktop Notifications**: Optional notifications for errors and status updates
- **🚀 Auto-Startup**: Platform-specific startup integration (macOS Launch Agents, Windows Registry)
- **⏯️ Polling Controls**: Start/stop polling from both main UI and system tray menu
- **🔧 User-Friendly Configuration**: GUI-based settings with persistent storage
- **🌍 Cross-Platform**: Native builds for Windows, macOS, and Linux

## 🚀 Quick Start

### Prerequisites

- **Go 1.21 or newer**
- For GUI support: **Fyne v2** (automatically resolved via Go modules)

Check your Go version:
```bash
go version
```

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/snip/igcmail-imap.git
   cd igcmail-imap
   ```

2. **Build the application:**
   ```bash
   go mod tidy
   go build -o igcmailimap .
   ```

   **macOS linker note:** If you see duplicate library warnings, use:
   ```bash
   CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries" go build -o igcmailimap .
   ```

3. **Run the application:**
   ```bash
   ./igcmailimap
   ```

## ⚙️ Configuration

The application uses a GUI for configuration. Key settings include:

- **IMAP Server**: Host:port (e.g., `imap.gmail.com:993`)
- **Credentials**: Username and password
- **Output Folder**: Directory for extracted IGC files
- **Polling Interval**: Seconds between checks
- **Auto-startup**: Platform-specific startup integration
- **Logging**: Enable/disable detailed logging
- **Notifications**: Enable/disable desktop notifications

### Configuration Files Location

- **macOS**: `~/Library/Application Support/igcMailImap/`
- **Windows**: Next to the executable
- **Linux**: `~/.config/igcMailImap/` (or `$XDG_CONFIG_HOME`)

## 📋 Usage

1. Launch the application and configure your IMAP settings
2. Set your output folder for IGC files
3. Optionally enable logging and notifications
4. Start polling to begin monitoring
5. The app minimizes to system tray - use tray menu to control polling
6. IGC files are automatically extracted as emails arrive

## 🖥️ System Tray

The application provides comprehensive system tray controls:

- **Show**: Open the main configuration window
- **Start polling**: Begin monitoring for new emails
- **Stop polling**: Pause email monitoring
- **Quit**: Exit the application

Menu items are dynamically enabled/disabled based on current state.

## 📝 Logging

When enabled, detailed logs are written to `igcmailimap.log` in your output folder:

- Connection attempts and status
- Message fetching details (UID, subject, sender)
- File extraction results
- Error conditions and troubleshooting information

## 🔔 Notifications

Optional desktop notifications for:

- IMAP connection errors
- File extraction failures
- Polling start/stop events

## 🔧 Development

### Project Structure

```
igcmail-imap/
├── main.go                 # Application entry point
├── ui/                     # Fyne-based GUI components
├── imap/                   # IMAP client and fetching logic
├── extract/                # IGC file extraction utilities
├── logger/                 # Logging functionality
├── config/                 # Configuration management
├── state/                  # UID tracking for incremental sync
└── startup/                # Platform-specific auto-startup
```

### Building for Different Platforms

```bash
# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o igcmailimap-mac-intel .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o igcmailimap-mac-arm64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o igcmailimap.exe .

# Linux
GOOS=linux GOARCH=amd64 go build -o igcmailimap-linux .
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## 📄 License

This project is open source. Please check the license file for details.

## ⚠️ Security Note

IMAP passwords are stored in plain text in the configuration file. Ensure your config files are properly secured and consider using application-specific passwords when available.