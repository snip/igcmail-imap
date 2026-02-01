package ui

import (
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
	"fyne.io/fyne/v2/widget"
	"igcmailimap/config"
	"igcmailimap/extract"
	"igcmailimap/imap"
	"igcmailimap/logger"
	"igcmailimap/startup"
	"igcmailimap/state"
)

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

	// Tray menu items (for dynamic updates)
	startPollItem *fyne.MenuItem
	stopPollItem  *fyne.MenuItem

	// pollStop is non-nil while the poll loop is running; close it to stop.
	pollStop chan struct{}
	mu       sync.Mutex
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

	// Close = hide to tray
	ap.Win.SetCloseIntercept(func() {
		ap.Win.Hide()
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

	a.userEntry = widget.NewEntry()
	a.userEntry.SetPlaceHolder("user@example.com")
	a.userEntry.SetText(a.Config.IMAPUser)

	a.passEntry = widget.NewPasswordEntry()
	a.passEntry.SetPlaceHolder("password")
	a.passEntry.SetText(a.Config.IMAPPassword)

	a.outputEntry = widget.NewEntry()
	a.outputEntry.SetPlaceHolder("C:\\IGC or /path/to/igc")
	a.outputEntry.SetText(a.Config.OutputFolder)

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
	a.intervalEntry.SetPlaceHolder("61")
	a.intervalEntry.SetText(strconv.Itoa(a.Config.IntervalSec))
	if a.Config.IntervalSec <= 0 {
		a.intervalEntry.SetText("61")
	}

	startupEnabled, _ := startup.Enabled()
	a.startupCheck = widget.NewCheck(startupCheckLabel(), nil)
	a.startupCheck.SetChecked(startupEnabled)

	a.loggingCheck = widget.NewCheck("Enable logging", nil)
	a.loggingCheck.SetChecked(a.Config.LoggingEnabled)

	a.notificationsCheck = widget.NewCheck("Enable notifications", nil)
	a.notificationsCheck.SetChecked(a.Config.NotificationsEnabled)

	saveBtn := widget.NewButton("Save", func() { a.save() })
	a.startupCheck.OnChanged = func(checked bool) {
		_ = startup.SetEnabled(checked)
	}

	a.startBtn = widget.NewButton("Start polling", func() { a.StartPolling() })
	a.stopBtn = widget.NewButton("Stop polling", func() { a.StopPolling() })
	a.updatePollButtons()

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
		widget.NewFormItem("", saveBtn),
	)
	a.Win.SetContent(form)
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return 61
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
		a.Config.IntervalSec = 61
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

func (a *App) quit() {
	a.mu.Lock()
	if a.pollStop != nil {
		close(a.pollStop)
		a.pollStop = nil
	}
	a.mu.Unlock()

	// Close logger
	if a.Logger != nil {
		a.Logger.Close()
	}

	a.Fyne.Quit()
}

// updatePollButtons enables/disables Start and Stop based on whether polling is running.
func (a *App) updatePollButtons() {
	a.mu.Lock()
	running := a.pollStop != nil
	a.mu.Unlock()
	if a.startBtn != nil {
		if running {
			a.startBtn.Disable()
			a.stopBtn.Enable()
		} else {
			a.startBtn.Enable()
			a.stopBtn.Disable()
		}
	}
	a.updateTrayMenu()
}

// updateTrayMenu updates the system tray menu items based on polling state.
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
		a.startPollItem.Disabled = false
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
	a.Logger.Info(fmt.Sprintf("IMAP polling started with %d second intervals", a.Config.IntervalSec))

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
}

// Run shows the window and, if config.PollingEnabled, starts the poll loop (restores previous state).
func (a *App) Run() {
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
		return 61
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
