package ui

import (
	_ "embed"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"igcmailimap/config"
	"igcmailimap/extract"
	"igcmailimap/imap"
	"igcmailimap/logger"
	"igcmailimap/startup"
	"igcmailimap/state"
)

//go:embed igcmailimap.png
var iconPNG []byte

//go:embed igcmailimap.ico
var iconICO []byte

// App holds the Fyne app, main window, config form, and poll state.
type App struct {
	Fyne      fyne.App
	Win       fyne.Window
	Config    *config.Config
	State     *state.State
	Logger    *logger.Logger
	statePath string
	cfgPath   string

	// Form fields (kept for read/write)
	serverEntry        *widget.Entry
	userEntry          *widget.Entry
	passEntry          *widget.Entry
	outputEntry        *widget.Entry
	outputBrowseBtn    *widget.Button
	intervalEntry      *widget.Entry
	startupCheck       *widget.Check
	loggingCheck       *widget.Check
	notificationsCheck *widget.Check
	startBtn           *widget.Button
	stopBtn            *widget.Button

	// Original values for change tracking
	originalServer      string
	originalUser        string
	originalPassword    string
	originalOutput      string
	originalInterval    int
	originalStartup     bool
	originalLogging     bool
	originalNotifications bool
	saveBtn             *widget.Button

	// Tray menu items (for dynamic updates)
	startPollItem *fyne.MenuItem
	stopPollItem  *fyne.MenuItem

	// pollStop is non-nil while the poll loop is running; close it to stop.
	pollStop    chan struct{}
	shuttingDown bool
	mu           sync.Mutex
}

// New creates and configures the app (loads config/state, builds UI).
func New() (*App, error) {
	a := app.NewWithID("igcmailimap")
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	statePath, _ := config.StatePath()
	cfgPath, _ := config.ConfigPath()
	st, err := state.Load(statePath)
	if err != nil {
		return nil, err
	}

	// Initialize logger
	log, err := logger.New(cfg.OutputFolder, cfg.LoggingEnabled)
	if err != nil {
		return nil, err
	}

	ap := &App{
		Fyne:      a,
		Config:    cfg,
		State:     st,
		Logger:    log,
		statePath: statePath,
		cfgPath:   cfgPath,
	}

	ap.Win = a.NewWindow("IGCmail IMAP")
	ap.buildConfigForm()
	ap.Win.Resize(fyne.NewSize(480, 380))
	ap.Win.CenterOnScreen()

	// Set application icon
	if icon, err := loadAppIcon(); err == nil {
		a.SetIcon(icon)
	} else {
		// Fallback to theme icon if custom icon fails to load
		a.SetIcon(theme.MailComposeIcon())
	}

	// Handle application shutdown (Command-Q, etc.)
	a.Lifecycle().SetOnStopped(func() {
		ap.cleanup()
	})

	// Close = hide to tray
	ap.Win.SetCloseIntercept(func() {
		ap.Win.Hide()
		// Check if app is still running after a short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			// If we're not shutting down and the app is still running, show notification
			ap.mu.Lock()
			stillRunning := ap.pollStop != nil || !ap.shuttingDown
			ap.mu.Unlock()
			if stillRunning {
				ap.notifyInfo("IGCmail IMAP is running in the background. Use the system tray icon to access it.")
			}
		}()
	})

	// System tray
	if desk, ok := a.(desktop.App); ok {
		ap.startPollItem = fyne.NewMenuItem("Start polling", func() { ap.StartPolling() })
		ap.stopPollItem = fyne.NewMenuItem("Stop polling", func() { ap.StopPolling() })

		m := fyne.NewMenu("IGCmail IMAP",
			fyne.NewMenuItem("Show", func() { ap.Win.Show() }),
			fyne.NewMenuItemSeparator(),
			ap.startPollItem,
			ap.stopPollItem,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() { ap.quit() }),
		)
		desk.SetSystemTrayMenu(m)
		ap.updateTrayMenu()
	}

	return ap, nil
}

func (a *App) buildConfigForm() {
	a.serverEntry = widget.NewEntry()
	a.serverEntry.SetPlaceHolder("imap.example.com:993")
	a.serverEntry.SetText(a.Config.IMAPServer)
	a.serverEntry.OnChanged = func(string) {
		a.updateSaveButtonState()
		a.updatePollButtons()
	}

	a.userEntry = widget.NewEntry()
	a.userEntry.SetPlaceHolder("user@example.com")
	a.userEntry.SetText(a.Config.IMAPUser)
	a.userEntry.OnChanged = func(string) {
		a.updateSaveButtonState()
		a.updatePollButtons()
	}

	a.passEntry = widget.NewPasswordEntry()
	a.passEntry.SetPlaceHolder("password")
	a.passEntry.SetText(a.Config.IMAPPassword)
	a.passEntry.OnChanged = func(string) {
		a.updateSaveButtonState()
		a.updatePollButtons()
	}

	a.outputEntry = widget.NewEntry()
	a.outputEntry.SetPlaceHolder("C:\\IGC or /path/to/igc")
	a.outputEntry.SetText(a.Config.OutputFolder)
	a.outputEntry.OnChanged = func(string) { a.updateSaveButtonState() }

	a.outputBrowseBtn = widget.NewButton("Browse...", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				return
			}
			if uri != nil {
				a.outputEntry.SetText(uri.Path())
			}
		}, a.Win)
		d.Show()
	})

	a.intervalEntry = widget.NewEntry()
	a.intervalEntry.SetPlaceHolder("60")
	a.intervalEntry.SetText(strconv.Itoa(a.Config.IntervalSec))
	if a.Config.IntervalSec <= 0 {
		a.intervalEntry.SetText("60")
	}
	a.intervalEntry.OnChanged = func(string) { a.updateSaveButtonState() }

	startupEnabled, _ := startup.Enabled()
	a.startupCheck = widget.NewCheck(startupCheckLabel(), nil)
	a.startupCheck.SetChecked(startupEnabled)

	a.loggingCheck = widget.NewCheck("Enable logging", nil)
	a.loggingCheck.SetChecked(a.Config.LoggingEnabled)

	a.notificationsCheck = widget.NewCheck("Enable notifications", nil)
	a.notificationsCheck.SetChecked(a.Config.NotificationsEnabled)

	a.saveBtn = widget.NewButton("Save", func() { a.save() })
	// Preserve existing startup check functionality
	originalStartupOnChanged := a.startupCheck.OnChanged
	a.startupCheck.OnChanged = func(checked bool) {
		if originalStartupOnChanged != nil {
			originalStartupOnChanged(checked)
		}
		_ = startup.SetEnabled(checked)
		a.updateSaveButtonState()
	}

	a.loggingCheck.OnChanged = func(bool) { a.updateSaveButtonState() }
	a.notificationsCheck.OnChanged = func(bool) { a.updateSaveButtonState() }

	a.startBtn = widget.NewButton("Start polling", func() { a.StartPolling() })
	a.stopBtn = widget.NewButton("Stop polling", func() { a.StopPolling() })
	a.updatePollButtons()

	quitBtn := widget.NewButton("Quit", func() { a.quit() })
	minimizeBtn := widget.NewButton("Minimize to tray", func() {
		a.Win.Hide()
		if !a.shuttingDown {
			a.notifyInfo("IGCmail IMAP is running in the background. Use the system tray icon to access it.")
		}
	})

	// Initialize original values and save button state
	a.storeOriginalValues()
	a.updateSaveButtonState()

	form := widget.NewForm(
		widget.NewFormItem("IMAP server", a.serverEntry),
		widget.NewFormItem("User", a.userEntry),
		widget.NewFormItem("Password", a.passEntry),
		widget.NewFormItem("Output folder", container.NewBorder(nil, nil, nil, a.outputBrowseBtn, a.outputEntry)),
		widget.NewFormItem("Interval (seconds)", a.intervalEntry),
		widget.NewFormItem("", a.startupCheck),
		widget.NewFormItem("", a.loggingCheck),
		widget.NewFormItem("", a.notificationsCheck),
		widget.NewFormItem("", a.startBtn),
		widget.NewFormItem("", a.stopBtn),
		widget.NewFormItem("", a.saveBtn),
		widget.NewFormItem("", minimizeBtn),
		widget.NewFormItem("", quitBtn),
	)
	a.Win.SetContent(form)
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return 60
	}
	return n
}

func startupCheckLabel() string {
	switch runtime.GOOS {
	case "darwin":
		return "Run at login"
	case "windows":
		return "Run at Windows startup"
	default:
		return "Run at startup"
	}
}

func (a *App) save() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Config.IMAPServer = a.serverEntry.Text
	a.Config.IMAPUser = a.userEntry.Text
	a.Config.IMAPPassword = a.passEntry.Text
	a.Config.OutputFolder = a.outputEntry.Text
	a.Config.IntervalSec = parseInt(a.intervalEntry.Text)
	if a.Config.IntervalSec <= 0 {
		a.Config.IntervalSec = 60
	}
	a.Config.RunAtStartup = a.startupCheck.Checked
	a.Config.LoggingEnabled = a.loggingCheck.Checked
	a.Config.NotificationsEnabled = a.notificationsCheck.Checked

	// Update logger with new settings
	if a.Logger != nil {
		a.Logger.Close()
	}
	newLogger, err := logger.New(a.Config.OutputFolder, a.Config.LoggingEnabled)
	if err != nil {
		a.notifyError("Failed to initialize logger: " + err.Error())
		return
	}
	a.Logger = newLogger

	if err := config.Save(a.Config); err != nil {
		a.notifyError("Save failed: " + err.Error())
		return
	}
	_ = startup.SetEnabled(a.Config.RunAtStartup)
	a.notifyInfo("Settings saved")

	// Reset original values and disable save button
	a.storeOriginalValues()
	a.updateSaveButtonState()
}

// loadAppIcon loads the embedded application icon
func loadAppIcon() (fyne.Resource, error) {
	// For runtime application icons, always use PNG as it's more reliable across platforms
	if len(iconPNG) > 0 {
		return fyne.NewStaticResource("icon.png", iconPNG), nil
	}

	// Fallback to ICO if PNG is not available (shouldn't happen in normal builds)
	if len(iconICO) > 0 {
		return fyne.NewStaticResource("icon.ico", iconICO), nil
	}

	return nil, fmt.Errorf("no embedded icon data available")
}

func (a *App) storeOriginalValues() {
	a.originalServer = a.Config.IMAPServer
	a.originalUser = a.Config.IMAPUser
	a.originalPassword = a.Config.IMAPPassword
	a.originalOutput = a.Config.OutputFolder
	a.originalInterval = a.Config.IntervalSec
	a.originalStartup = a.Config.RunAtStartup
	a.originalLogging = a.Config.LoggingEnabled
	a.originalNotifications = a.Config.NotificationsEnabled
}

func (a *App) hasUnsavedChanges() bool {
	return a.serverEntry.Text != a.originalServer ||
		a.userEntry.Text != a.originalUser ||
		a.passEntry.Text != a.originalPassword ||
		a.outputEntry.Text != a.originalOutput ||
		parseInt(a.intervalEntry.Text) != a.originalInterval ||
		a.startupCheck.Checked != a.originalStartup ||
		a.loggingCheck.Checked != a.originalLogging ||
		a.notificationsCheck.Checked != a.originalNotifications
}

func (a *App) updateSaveButtonState() {
	if a.saveBtn != nil {
		hasChanges := a.hasUnsavedChanges()
		if hasChanges {
			a.saveBtn.Enable()
		} else {
			a.saveBtn.Disable()
		}
	}
}

func (a *App) notifyError(msg string) {
	if a.Config.NotificationsEnabled {
		a.Fyne.SendNotification(fyne.NewNotification("IGCmail IMAP Error", msg))
	}
}

func (a *App) notifyInfo(msg string) {
	if a.Config.NotificationsEnabled {
		a.Fyne.SendNotification(fyne.NewNotification("IGCmail IMAP", msg))
	}
}

// cleanup handles the shutdown logging and cleanup operations
func (a *App) cleanup() {
	a.mu.Lock()
	if a.shuttingDown {
		a.mu.Unlock()
		return // Already cleaned up
	}
	a.shuttingDown = true
	a.mu.Unlock()

	a.Logger.Info("IGCmail IMAP application shutting down")

	a.mu.Lock()
	pollingWasRunning := a.pollStop != nil
	if a.pollStop != nil {
		close(a.pollStop)
		a.pollStop = nil
	}
	a.mu.Unlock()

	if pollingWasRunning {
		a.Logger.Info("IMAP polling stopped during application shutdown")
	}

	// Close logger
	if a.Logger != nil {
		a.Logger.Close()
	}
}

func (a *App) quit() {
	a.cleanup()
	a.Fyne.Quit()
}

// hasValidConfiguration checks if all required fields are filled
func (a *App) hasValidConfiguration() bool {
	return a.serverEntry.Text != "" && a.userEntry.Text != "" && a.passEntry.Text != ""
}

// updatePollButtons enables/disables Start and Stop based on polling state and configuration validity.
func (a *App) updatePollButtons() {
	a.mu.Lock()
	running := a.pollStop != nil
	a.mu.Unlock()
	if a.startBtn != nil {
		if running {
			a.startBtn.Disable()
			a.stopBtn.Enable()
		} else {
			// Enable only if configuration is valid
			if a.hasValidConfiguration() {
				a.startBtn.Enable()
			} else {
				a.startBtn.Disable()
			}
			a.stopBtn.Disable()
		}
	}
	a.updateTrayMenu()
}

// updateTrayMenu updates the system tray menu items based on polling state and configuration.
func (a *App) updateTrayMenu() {
	if a.startPollItem == nil || a.stopPollItem == nil {
		return
	}

	a.mu.Lock()
	running := a.pollStop != nil
	a.mu.Unlock()

	if running {
		a.startPollItem.Disabled = true
		a.stopPollItem.Disabled = false
	} else {
		// Disable start if configuration is invalid
		a.startPollItem.Disabled = !a.hasValidConfiguration()
		a.stopPollItem.Disabled = true
	}

	// Refresh the tray menu by re-setting it
	if desk, ok := a.Fyne.(desktop.App); ok {
		m := fyne.NewMenu("IGCmail IMAP",
			fyne.NewMenuItem("Show", func() { a.Win.Show() }),
			fyne.NewMenuItemSeparator(),
			a.startPollItem,
			a.stopPollItem,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() { a.quit() }),
		)
		desk.SetSystemTrayMenu(m)
	}
}

// StartPolling starts the poll loop and saves PollingEnabled = true.
func (a *App) StartPolling() {
	a.mu.Lock()
	if a.pollStop != nil {
		a.mu.Unlock()
		return
	}
	a.pollStop = make(chan struct{})
	a.mu.Unlock()
	a.Config.PollingEnabled = true
	_ = config.Save(a.Config)
	a.updatePollButtons()

	// Log that polling has started
	a.Logger.Info(fmt.Sprintf("IMAP polling started for %s from %s with %d second intervals", a.Config.IMAPUser, a.Config.IMAPServer, a.Config.IntervalSec))

	// Notify user that polling has started
	a.notifyInfo(fmt.Sprintf("IMAP polling started (%d second intervals)", a.Config.IntervalSec))

	go a.pollLoop(a.pollStop)
}

// StopPolling stops the poll loop and saves PollingEnabled = false.
func (a *App) StopPolling() {
	a.mu.Lock()
	if a.pollStop == nil {
		a.mu.Unlock()
		return
	}
	ch := a.pollStop
	a.pollStop = nil
	a.mu.Unlock()
	close(ch)
	a.Config.PollingEnabled = false
	_ = config.Save(a.Config)
	a.updatePollButtons()

	// Log that polling has stopped
	a.Logger.Info("IMAP polling stopped")

	// Notify user that polling has stopped
	a.notifyInfo("IMAP polling stopped")
}

// Run shows the window and, if config.PollingEnabled, starts the poll loop (restores previous state).
func (a *App) Run() {
	a.Logger.Info("IGCmail IMAP application started")

	if a.Config.PollingEnabled {
		a.StartPolling()
	} else {
		a.updatePollButtons()
	}
	a.Win.ShowAndRun()
}

func (a *App) pollLoop(stopCh chan struct{}) {
	interval := time.Duration(a.intervalSeconds()) * time.Second
	time.Sleep(2 * time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		a.fetchAndExtract()
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			// next iteration
		}
	}
}

func (a *App) intervalSeconds() int {
	a.mu.Lock()
	sec := a.Config.IntervalSec
	a.mu.Unlock()
	if sec <= 0 {
		return 60
	}
	return sec
}

func (a *App) fetchAndExtract() {
	a.mu.Lock()
	cfg := *a.Config
	st := a.State
	a.mu.Unlock()

	if cfg.OutputFolder == "" || cfg.IMAPServer == "" {
		return
	}

	fetcher := imap.NewFetcher(&cfg, st)
	msgs, err := fetcher.FetchNew()
	if err != nil {
		a.Logger.Error("IMAP fetch failed: " + err.Error())
		a.notifyError("IMAP: " + err.Error())
		return
	}

	if len(msgs) == 0 {
		return
	}

	// Log fetch summary and individual messages only when there are new messages
	var uids []uint32
	for _, msg := range msgs {
		uids = append(uids, msg.UID)
		a.Logger.LogMessageDetails(msg.UID, msg.Subject, msg.From)
	}
	a.Logger.LogFetch(len(msgs), cfg.OutputFolder, uids)

	saveDir := extract.NewSaveDir(cfg.OutputFolder)
	var allResults []logger.ExtractResult
	for _, m := range msgs {
		results, err := extract.ExtractIGCAttachments(m.Body, saveDir)
		if err != nil {
			a.Logger.Error(fmt.Sprintf("IGC extraction failed for UID %d: %s", m.UID, err.Error()))
			a.notifyError("Extract: " + err.Error())
			continue
		}

		// Convert extract.ExtractResult to logger.ExtractResult
		var loggerResults []logger.ExtractResult
		for _, result := range results {
			loggerResults = append(loggerResults, logger.ExtractResult{
				Filename: result.Filename,
				Path:     result.Path,
			})
		}
		allResults = append(allResults, loggerResults...)

		// Log details for this message
		if len(loggerResults) > 0 {
			a.Logger.LogMessageExtract(m.UID, m.Subject, m.From, loggerResults, cfg.OutputFolder)
		}
	}

	a.Logger.LogExtract(len(allResults), cfg.OutputFolder)
}
