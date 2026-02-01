# IGCmail IMAP

A desktop application for Windows, macOS, and Linux that monitors an IMAP mailbox and automatically extracts IGC (flight track) file attachments. Built with Go and Fyne for a native cross-platform experience. It's main purpose is a workaround to Naviter's IGCmail which only work with POP3 protocol.

## ✨ Features

- **🔒 Secure IMAP Connection**: TLS-encrypted connection (port 993) to any IMAP server
- **📬 Smart Incremental Sync**: Only fetches new messages using UID-based tracking
- **🎯 IGC File Extraction**: Automatically extracts .igc attachments to a configurable folder
- **🔄 Duplicate Handling**: Same filenames get timestamped to avoid overwrites
- **📱 System Tray Integration**: Minimizes to tray with comprehensive menu controls
- **📝 Comprehensive Logging**: Detailed operation logs with configurable output (app lifecycle, polling details, server info)
- **🔔 Desktop Notifications**: Optional notifications for errors, polling events, and window management
- **🚀 Auto-Startup**: Platform-specific startup integration (macOS Launch Agents, Windows Registry, Linux not supported)
- **⏯️ Polling Controls**: Start/stop polling from both main UI and system tray menu (disabled by default until configuration is complete)
- **🔧 Smart Configuration UI**: GUI-based settings with change detection, directory browser, and persistent storage
- **🌍 Cross-Platform**: Native builds for Windows, macOS, and Linux (both 64-bit and 32-bit architectures)
- **🚀 CI/CD**: Automated testing and releases with GitHub Actions
- **🖥️ Native GUI Experience**: No terminal windows on Windows, proper app bundles on macOS

## 🚀 Quick Start

### Prerequisites

- **Go 1.21 or newer**
- For GUI support: **Fyne v2** (automatically resolved via Go modules)
- **Platform-specific GUI libraries** (automatically installed on supported platforms)

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

The application uses an intuitive GUI for configuration with smart features:

- **IMAP Server**: Host:port (e.g., `imap.gmail.com:993`)
- **Credentials**: Username and password
- **Output Folder**: Directory browser with create new folder capability
- **Polling Interval**: Seconds between checks
- **Auto-startup**: Platform-specific startup integration (not available on Linux)
- **Logging**: Enable/disable detailed logging with enhanced app lifecycle tracking
- **Notifications**: Enable/disable desktop notifications (errors, polling events, UI feedback)

### Smart UI Features

- **Change Detection**: Save button only enables when settings are modified
- **Directory Browser**: Browse existing folders or create new ones
- **Configuration Validation**: Start polling button is disabled until server, username, and password are filled
- **Minimize to Tray**: Dedicated button to minimize to system tray
- **Quit Button**: Clean application exit with proper shutdown logging

### Configuration Files Location

- **Linux**: `~/.config/igcMailImap/` or next to executable
- **macOS**: `~/Library/Application Support/igcMailImap/`
- **Windows**: Next to the executable

## 📋 Usage

1. Launch the application and fill in your IMAP settings (server, username, password)
2. Set your output folder using the directory browser (can create new folders)
3. Optionally enable logging and notifications
4. Click "Start polling" to begin monitoring - the button is enabled once all required fields are filled
5. Use "Minimize to Tray" button or close window to minimize to system tray
6. Use tray menu to control polling or access settings
7. IGC files are automatically extracted as emails arrive
8. Use "Quit" button or tray menu to exit with proper shutdown logging

**Note**: Polling is disabled by default to allow configuration before starting monitoring.

## 🖥️ System Tray & UI Controls

### System Tray Menu
The application provides comprehensive system tray controls:

- **Show**: Open the main configuration window
- **Start polling**: Begin monitoring for new emails (with notification)
- **Stop polling**: Pause email monitoring (with notification)
- **Quit**: Exit the application (with shutdown logging)

Menu items are dynamically enabled/disabled based on current state.

### Main UI Buttons
- **Start/Stop Polling**: Control IMAP monitoring
- **Save**: Only enabled when configuration changes are detected
- **Minimize to Tray**: Hide window to system tray (with notification)
- **Quit**: Clean application exit

## 📝 Logging

When enabled, detailed logs are written to `igcmailimap.log` in your output folder:

- **Application Lifecycle**: Start and shutdown events
- **Connection Details**: Server, username, and status for each polling session
- **Message Processing**: UID, subject, sender for fetched emails
- **File Operations**: Extraction results and duplicate handling
- **Error Tracking**: Comprehensive error logging and troubleshooting information
- **Polling Events**: Start/stop timing with interval information

## 🔔 Notifications

Optional desktop notifications for:

- **Connection Issues**: IMAP connection errors and authentication problems
- **File Operations**: Extraction failures and duplicate file handling
- **Polling Events**: Start/stop notifications with server and interval details
- **UI Feedback**: Window minimization and application state changes

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
├── startup/                # Platform-specific auto-startup
├── .github/workflows/      # CI/CD pipeline configuration
│   ├── ci.yml             # Testing workflow
│   └── build.yml          # Build and release workflow
└── README.md              # This file
```

### Building for Different Platforms

The application supports automated multi-platform builds via GitHub Actions CI/CD.

#### Manual Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o igcmailimap .
GOOS=linux GOARCH=386 go build -o igcmailimap-386 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o igcmailimap .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o igcmailimap .

# Windows (GUI application, no terminal window)
GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o igcmailimap.exe .
GOOS=windows GOARCH=386 go build -ldflags="-H windowsgui" -o igcmailimap-386.exe .
```

#### CI/CD

Automated testing and releases are handled by GitHub Actions:

- **Testing**: Runs on Windows, macOS, and Linux for every push/PR
- **Building**: Cross-compiles for all platforms (6 binaries total)
- **Releases**: Automated when version tags are pushed (e.g., `v1.0.0`)

### Configuration Files Location

- **Linux**: `~/.config/igcMailImap/` or next to executable
- **macOS**: `~/Library/Application Support/igcMailImap/`
- **Windows**: Next to the executable

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## 📄 License

This project is open source. Please check the license file for details.

## ⚠️ Security Note

IMAP passwords are stored in plain text in the configuration file. Ensure your config files are properly secured and consider using application-specific passwords when available.