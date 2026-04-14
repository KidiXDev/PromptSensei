package tui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/services"
)

type screen int

const (
	screenHome screen = iota
	screenEditor
	screenKnowledge
	screenDataset
	screenSettings
	screenResult
	screenBusy
)

const (
	actionCreate        = "create"
	actionEnhance       = "enhance"
	actionDatasetStatus = "dataset_status"
	actionKnowledge     = "knowledge"
	actionSettings      = "settings"
	actionRebuild       = "rebuild_dataset"
	actionExit          = "exit"
)

var rebuildTips = []string{
	"Tip: CSV files are the source of truth; SQLite is generated cache.",
	"Tip: Keep aliases in tag.csv to improve recall during retrieval.",
	"Tip: Use dataset rebuild after editing tag or character CSV files.",
	"Tip: Character core tags improve automatic expansion quality.",
}

type enhanceDoneMsg struct {
	result   *domain.EnhanceResult
	warnings []string
	err      error
}

type datasetStatusDoneMsg struct {
	status services.DatasetStatus
	err    error
}

type datasetRebuildDoneMsg struct {
	err error
}

type datasetRebuildProgressMsg struct {
	progress services.DatasetRebuildProgress
}

type rebuildProgressClosedMsg struct{}

type busyTooltipTickMsg struct{}

type startupRebuildCheckDoneMsg struct {
	needed  bool
	auto    bool
	reasons []string
	err     error
}

type settingsSavedMsg struct {
	err error
}

type model struct {
	ctx context.Context

	promptService    *services.PromptService
	datasetService   *services.DatasetService
	knowledgeService *services.KnowledgeService
	saveConfig       func(config.Config) error
	configPath       string
	cfg              config.Config

	screen          screen
	knowledgeReturn screen
	width           int
	height          int

	homeList      list.Model
	knowledgeList list.Model
	editor        textarea.Model
	datasetView   viewport.Model
	resultView    viewport.Model
	settingsList  list.Model
	settingsDraft config.Config
	settingsInput textinput.Model
	spin          spinner.Model

	mode          domain.Mode
	strict        bool
	createMode    bool
	notice        string
	lastErr       string
	busyLabel     string
	busyTip       string
	busyElapsed   time.Time
	busyStage     string
	busyCurrent   int
	busyTotal     int
	busyDetail    string
	busyMode      string
	rebuildOrigin string
	lastRequest   *domain.EnhanceRequest
	lastResult    *domain.EnhanceResult
	warnings      []string
	settingsDirty bool
	settingsEdit  bool
	settingsError string

	selectedKnowledge map[string]struct{}
	rebuildProgressCh <-chan services.DatasetRebuildProgress
	tipIndex          int
}

func Run(
	ctx context.Context,
	promptService *services.PromptService,
	datasetService *services.DatasetService,
	knowledgeService *services.KnowledgeService,
	configPath string,
	cfg config.Config,
	saveConfig func(config.Config) error,
	in io.Reader,
	out io.Writer,
) error {
	knowledgeFiles, err := knowledgeService.List()
	if err != nil {
		return err
	}

	m := newModel(ctx, promptService, datasetService, knowledgeService, configPath, cfg, saveConfig, knowledgeFiles)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(in), tea.WithOutput(out))
	_, err = p.Run()
	return err
}

func newModel(
	ctx context.Context,
	promptService *services.PromptService,
	datasetService *services.DatasetService,
	knowledgeService *services.KnowledgeService,
	configPath string,
	cfg config.Config,
	saveConfig func(config.Config) error,
	knowledgeFiles []string,
) model {
	homeItems := []list.Item{
		homeItem{title: "Create Prompt", description: "Generate from a rough idea", action: actionCreate},
		homeItem{title: "Enhance Prompt", description: "Improve an existing prompt", action: actionEnhance},
		homeItem{title: "Dataset Status", description: "Inspect CSV/SQLite cache health", action: actionDatasetStatus},
		homeItem{title: "Knowledge Files", description: "Manage selected knowledge docs", action: actionKnowledge},
		homeItem{title: "Settings", description: "Edit provider, API key, mode, and paths", action: actionSettings},
		homeItem{title: "Rebuild Dataset", description: "Force CSV to SQLite rebuild", action: actionRebuild},
		homeItem{title: "Exit", description: "Quit PromptSensei", action: actionExit},
	}

	home := list.New(homeItems, list.NewDefaultDelegate(), 60, 14)
	home.Title = "Actions"
	home.SetShowHelp(false)
	home.SetFilteringEnabled(false)

	knowledgeItems := make([]list.Item, 0, len(knowledgeFiles))
	for _, file := range knowledgeFiles {
		knowledgeItems = append(knowledgeItems, knowledgeItem{name: file, selected: false})
	}
	knowledge := list.New(knowledgeItems, list.NewDefaultDelegate(), 60, 14)
	knowledge.Title = "Knowledge Selection"
	knowledge.SetShowHelp(false)
	knowledge.SetFilteringEnabled(false)

	editor := textarea.New()
	editor.Placeholder = "Describe your idea or paste a prompt..."
	editor.Focus()
	editor.SetWidth(80)
	editor.SetHeight(12)
	editor.ShowLineNumbers = false

	spin := spinner.New(spinner.WithSpinner(spinner.Dot))
	spin.Style = accentStyle

	settings := list.New(buildSettingsItems(cfg), list.NewDefaultDelegate(), 80, 14)
	settings.Title = "Settings"
	settings.SetShowHelp(false)
	settings.SetFilteringEnabled(false)

	return model{
		ctx:               ctx,
		promptService:     promptService,
		datasetService:    datasetService,
		knowledgeService:  knowledgeService,
		saveConfig:        saveConfig,
		configPath:        configPath,
		cfg:               cfg,
		screen:            screenBusy,
		knowledgeReturn:   screenHome,
		homeList:          home,
		knowledgeList:     knowledge,
		editor:            editor,
		datasetView:       viewport.New(80, 14),
		resultView:        viewport.New(80, 14),
		settingsList:      settings,
		settingsDraft:     cfg,
		spin:              spin,
		mode:              cfg.General.DefaultMode,
		strict:            cfg.General.StrictBooruValidation,
		busyMode:          "startup_check",
		busyLabel:         "Checking dataset cache",
		busyElapsed:       time.Now(),
		selectedKnowledge: map[string]struct{}{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		startupRebuildCheckCmd(m.ctx, m.datasetService),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeComponents()
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
	case startupRebuildCheckDoneMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.notice = ""
			m.screen = screenHome
			m.clearBusyState()
			return m, nil
		}
		if msg.needed && msg.auto {
			m.notice = "Startup rebuild required: " + joinReasons(msg.reasons)
			return m, m.startRebuildCmd("startup", "Building SQLite cache for startup")
		}
		if msg.needed && !msg.auto {
			m.notice = "Dataset cache is stale (" + joinReasons(msg.reasons) + "). Auto rebuild disabled."
		} else {
			m.notice = "Dataset cache is ready."
		}
		m.screen = screenHome
		m.clearBusyState()
		return m, nil
	case enhanceDoneMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.notice = ""
			m.screen = screenEditor
			m.clearBusyState()
			return m, nil
		}
		m.lastResult = msg.result
		m.warnings = msg.warnings
		m.lastErr = ""
		m.notice = "Prompt generated successfully."
		m.clearBusyState()
		m.resultView.SetContent(buildResultText(msg.result, msg.warnings))
		m.resultView.GotoTop()
		m.screen = screenResult
		return m, nil
	case datasetStatusDoneMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.screen = screenHome
			m.clearBusyState()
			return m, nil
		}
		m.lastErr = ""
		m.notice = "Dataset status loaded."
		m.clearBusyState()
		m.datasetView.SetContent(buildDatasetText(msg.status))
		m.datasetView.GotoTop()
		m.screen = screenDataset
		return m, nil
	case datasetRebuildDoneMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			m.screen = screenHome
			m.clearBusyState()
			return m, nil
		}
		if m.rebuildOrigin == "startup" {
			m.notice = "Startup dataset rebuild completed."
			m.lastErr = ""
			m.screen = screenHome
			m.clearBusyState()
			return m, nil
		}
		m.notice = "Dataset rebuilt. Loading fresh status..."
		m.lastErr = ""
		m.startBusy("status", "Loading dataset status")
		return m, tea.Batch(m.spin.Tick, datasetStatusCmd(m.ctx, m.datasetService))
	case datasetRebuildProgressMsg:
		m.busyStage = msg.progress.Stage
		m.busyCurrent = msg.progress.Current
		m.busyTotal = msg.progress.Total
		m.busyDetail = msg.progress.Detail
		if m.busyMode != "rebuild" {
			return m, nil
		}
		return m, waitRebuildProgressCmd(m.rebuildProgressCh)
	case rebuildProgressClosedMsg:
		return m, nil
	case busyTooltipTickMsg:
		if m.screen != screenBusy || m.busyMode != "rebuild" {
			return m, nil
		}
		if len(rebuildTips) > 0 {
			m.tipIndex = (m.tipIndex + 1) % len(rebuildTips)
			m.busyTip = rebuildTips[m.tipIndex]
		}
		return m, busyTooltipTickCmd()
	case settingsSavedMsg:
		if msg.err != nil {
			m.lastErr = msg.err.Error()
			return m, nil
		}
		m.cfg = m.settingsDraft
		m.mode = m.cfg.General.DefaultMode
		m.strict = m.cfg.General.StrictBooruValidation
		m.notice = "Settings saved and applied."
		m.lastErr = ""
		m.settingsDirty = false
		m.refreshSettingsList()
		return m, nil
	}

	switch m.screen {
	case screenHome:
		return m.updateHome(msg)
	case screenEditor:
		return m.updateEditor(msg)
	case screenKnowledge:
		return m.updateKnowledge(msg)
	case screenDataset:
		return m.updateDataset(msg)
	case screenSettings:
		return m.updateSettings(msg)
	case screenResult:
		return m.updateResult(msg)
	case screenBusy:
		return m.updateBusy(msg)
	default:
		return m, nil
	}
}

func (m model) updateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.homeList, cmd = m.homeList.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q":
			return m, tea.Quit
		case "enter":
			item, ok := m.homeList.SelectedItem().(homeItem)
			if !ok {
				return m, nil
			}
			switch item.action {
			case actionCreate:
				m.createMode = true
				m.lastErr = ""
				m.notice = "Create mode selected."
				m.editor.Focus()
				m.screen = screenEditor
				return m, nil
			case actionEnhance:
				m.createMode = false
				m.lastErr = ""
				m.notice = "Enhance mode selected."
				m.editor.Focus()
				m.screen = screenEditor
				return m, nil
			case actionDatasetStatus:
				m.startBusy("status", "Loading dataset status")
				return m, tea.Batch(m.spin.Tick, datasetStatusCmd(m.ctx, m.datasetService))
			case actionKnowledge:
				m.knowledgeReturn = screenHome
				m.screen = screenKnowledge
				return m, nil
			case actionSettings:
				m.settingsDraft = m.cfg
				m.settingsDirty = false
				m.settingsEdit = false
				m.settingsError = ""
				m.refreshSettingsList()
				m.screen = screenSettings
				return m, nil
			case actionRebuild:
				return m, m.startRebuildCmd("manual", "Rebuilding SQLite cache from CSV")
			case actionExit:
				return m, tea.Quit
			}
		}
	}
	return m, cmd
}

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "ctrl+s":
			prompt := strings.TrimSpace(m.editor.Value())
			if prompt == "" {
				m.lastErr = "prompt cannot be empty"
				return m, nil
			}
			req := &domain.EnhanceRequest{
				Prompt:         prompt,
				Mode:           m.mode,
				KnowledgeFiles: m.selectedKnowledgeList(),
				StrictBooru:    m.strict,
				CreateMode:     m.createMode,
			}
			m.lastRequest = req
			m.startBusy("enhance", "Generating prompt")
			m.lastErr = ""
			return m, tea.Batch(m.spin.Tick, enhanceCmd(m.ctx, m.promptService, *req))
		case "ctrl+g":
			m.mode = nextMode(m.mode)
			return m, nil
		case "ctrl+b":
			m.strict = !m.strict
			return m, nil
		case "ctrl+k":
			m.knowledgeReturn = screenEditor
			m.screen = screenKnowledge
			return m, nil
		case "ctrl+t":
			m.createMode = !m.createMode
			if m.createMode {
				m.notice = "Task switched to create mode."
			} else {
				m.notice = "Task switched to enhance mode."
			}
			return m, nil
		case "ctrl+l":
			m.editor.SetValue("")
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateKnowledge(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.knowledgeList, cmd = m.knowledgeList.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "enter":
			m.screen = m.knowledgeReturn
			return m, nil
		case " ":
			idx := m.knowledgeList.Index()
			items := m.knowledgeList.Items()
			if idx < 0 || idx >= len(items) {
				return m, nil
			}
			item, ok := items[idx].(knowledgeItem)
			if !ok {
				return m, nil
			}
			item.selected = !item.selected
			if item.selected {
				m.selectedKnowledge[item.name] = struct{}{}
			} else {
				delete(m.selectedKnowledge, item.name)
			}
			items[idx] = item
			m.knowledgeList.SetItems(items)
			return m, nil
		case "a":
			items := m.knowledgeList.Items()
			for i := range items {
				item := items[i].(knowledgeItem)
				item.selected = true
				items[i] = item
				m.selectedKnowledge[item.name] = struct{}{}
			}
			m.knowledgeList.SetItems(items)
			return m, nil
		case "c":
			items := m.knowledgeList.Items()
			for i := range items {
				item := items[i].(knowledgeItem)
				item.selected = false
				items[i] = item
			}
			m.selectedKnowledge = map[string]struct{}{}
			m.knowledgeList.SetItems(items)
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateDataset(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.datasetView, cmd = m.datasetView.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "r":
			return m, m.startRebuildCmd("manual", "Rebuilding SQLite cache from CSV")
		}
	}
	return m, cmd
}

func (m model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.settingsEdit {
		var cmd tea.Cmd
		m.settingsInput, cmd = m.settingsInput.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc":
				m.settingsEdit = false
				m.settingsError = ""
				return m, nil
			case "enter":
				current, ok := m.selectedSettingItem()
				if !ok {
					m.settingsEdit = false
					m.settingsError = ""
					return m, nil
				}
				next := m.settingsDraft
				if err := applySettingValue(&next, current.field, m.settingsInput.Value()); err != nil {
					m.settingsError = err.Error()
					return m, nil
				}
				m.settingsDraft = next
				m.settingsDirty = m.settingsDraft != m.cfg
				m.settingsEdit = false
				m.settingsError = ""
				m.refreshSettingsList()
				return m, nil
			}
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.settingsList, cmd = m.settingsList.Update(msg)
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "ctrl+r":
			m.settingsDraft = m.cfg
			m.settingsDirty = false
			m.settingsError = ""
			m.refreshSettingsList()
			m.notice = "Discarded unsaved settings changes."
			return m, nil
		case "ctrl+s":
			if !m.settingsDirty {
				m.notice = "No settings changes to save."
				return m, nil
			}
			if m.saveConfig == nil {
				m.cfg = m.settingsDraft
				m.mode = m.cfg.General.DefaultMode
				m.strict = m.cfg.General.StrictBooruValidation
				m.settingsDirty = false
				m.refreshSettingsList()
				m.notice = "Settings updated in session."
				return m, nil
			}
			return m, saveSettingsCmd(m.saveConfig, m.settingsDraft)
		case "enter":
			return m.activateOrEditSetting(1)
		case "right", "l", " ":
			return m.activateOrEditSetting(1)
		case "left", "h":
			return m.activateOrEditSetting(-1)
		}
	}
	return m, cmd
}

func (m model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.resultView, cmd = m.resultView.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "e":
			m.screen = screenEditor
			return m, nil
		case "r":
			if m.lastRequest == nil {
				return m, nil
			}
			m.startBusy("enhance", "Regenerating prompt")
			return m, tea.Batch(m.spin.Tick, enhanceCmd(m.ctx, m.promptService, *m.lastRequest))
		case "c":
			if m.lastResult == nil {
				return m, nil
			}
			if err := clipboard.WriteAll(m.lastResult.Output); err != nil {
				m.lastErr = "copy failed: " + err.Error()
				return m, nil
			}
			m.notice = "Copied result to clipboard."
			m.lastErr = ""
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateBusy(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := titleStyle.Render("PromptSensei")
	subtitle := subtitleStyle.Render("AI prompt crafting with local booru-aware retrieval")

	body := ""
	switch m.screen {
	case screenHome:
		body = panelStyle.Render(m.renderHome())
	case screenEditor:
		body = panelStyle.Render(m.renderEditor())
	case screenKnowledge:
		body = panelStyle.Render(m.renderKnowledge())
	case screenDataset:
		body = panelStyle.Render(m.renderDataset())
	case screenSettings:
		body = panelStyle.Render(m.renderSettings())
	case screenResult:
		body = panelStyle.Render(m.renderResult())
	case screenBusy:
		body = panelStyle.Render(m.renderBusy())
	}

	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, subtitle, "", body, "", footer)
}

func (m *model) resizeComponents() {
	if m.width == 0 || m.height == 0 {
		return
	}

	listWidth := max(45, m.width-8)
	homeWidth := max(34, int(float64(listWidth)*0.6))
	listHeight := max(10, m.height-14)
	m.homeList.SetSize(homeWidth, listHeight)
	m.knowledgeList.SetSize(listWidth, listHeight)
	m.settingsList.SetSize(listWidth, listHeight)
	m.editor.SetWidth(max(40, m.width-14))
	m.editor.SetHeight(max(8, m.height-20))

	viewportWidth := max(40, m.width-12)
	viewportHeight := max(8, m.height-18)
	m.datasetView.Width = viewportWidth
	m.datasetView.Height = viewportHeight
	m.resultView.Width = viewportWidth
	m.resultView.Height = viewportHeight
}

func (m model) renderHome() string {
	summaryLines := []string{
		accentStyle.Render("Session"),
		fmt.Sprintf("Mode: %s", m.mode),
		fmt.Sprintf("Task: %s", ternary(m.createMode, "Create", "Enhance")),
		fmt.Sprintf("Strict validation: %t", m.strict),
		fmt.Sprintf("Provider: %s (enabled=%t)", strings.TrimSpace(m.cfg.Provider.Name), m.cfg.Provider.Enabled),
		fmt.Sprintf("Model: %s", strings.TrimSpace(m.cfg.Provider.Model)),
		fmt.Sprintf("Knowledge selected: %d", len(m.selectedKnowledge)),
		"",
		accentStyle.Render("Navigation"),
		fmt.Sprintf("%s move  %s select", keyStyle.Render("j/k or arrows"), keyStyle.Render("enter")),
		fmt.Sprintf("%s open TUI settings editor", keyStyle.Render("Settings")),
	}

	left := m.homeList.View()
	rightWidth := max(26, m.width-46)
	right := lipgloss.NewStyle().
		Width(rightWidth).
		Render(strings.Join(summaryLines, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (m model) renderEditor() string {
	modeLabel := accentStyle.Render(string(m.mode))
	modeType := "Enhance"
	if m.createMode {
		modeType = "Create"
	}

	knowledge := "(none)"
	selected := m.selectedKnowledgeList()
	if len(selected) > 0 {
		knowledge = strings.Join(selected, ", ")
	}

	lines := []string{
		fmt.Sprintf("Task: %s", accentStyle.Render(modeType)),
		fmt.Sprintf("Mode: %s  Strict: %t", modeLabel, m.strict),
		"Knowledge: " + knowledge,
		"",
		m.editor.View(),
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s submit  %s mode  %s strict  %s task  %s knowledge  %s clear  %s back",
				keyStyle.Render("ctrl+s"),
				keyStyle.Render("ctrl+g"),
				keyStyle.Render("ctrl+b"),
				keyStyle.Render("ctrl+t"),
				keyStyle.Render("ctrl+k"),
				keyStyle.Render("ctrl+l"),
				keyStyle.Render("esc"),
			),
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderKnowledge() string {
	lines := []string{
		"Select knowledge files used during prompt assembly.",
		"",
		m.knowledgeList.View(),
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s toggle  %s all  %s clear  %s back",
				keyStyle.Render("space"),
				keyStyle.Render("a"),
				keyStyle.Render("c"),
				keyStyle.Render("esc/enter"),
			),
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderDataset() string {
	lines := []string{
		m.datasetView.View(),
		"",
		helpStyle.Render(fmt.Sprintf("%s rebuild  %s back", keyStyle.Render("r"), keyStyle.Render("esc"))),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderSettings() string {
	if m.settingsEdit {
		prompt := "Enter new value and press enter."
		if m.settingsError != "" {
			prompt = errorStyle.Render(m.settingsError)
		}
		lines := []string{
			"Edit setting",
			"",
			m.settingsInput.View(),
			"",
			prompt,
			"",
			helpStyle.Render(fmt.Sprintf("%s apply  %s cancel", keyStyle.Render("enter"), keyStyle.Render("esc"))),
		}
		return strings.Join(lines, "\n")
	}

	selectedDesc := ""
	if item, ok := m.selectedSettingItem(); ok {
		selectedDesc = item.field.description
	}
	dirtyState := "clean"
	if m.settingsDirty {
		dirtyState = "unsaved changes"
	}

	lines := []string{
		fmt.Sprintf("Config file: %s", m.configPath),
		fmt.Sprintf("Draft state: %s", accentStyle.Render(dirtyState)),
		"",
		m.settingsList.View(),
		"",
		"Selected: " + selectedDesc,
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s edit/toggle  %s cycle back  %s save  %s reload  %s back",
				keyStyle.Render("enter/space/right"),
				keyStyle.Render("left"),
				keyStyle.Render("ctrl+s"),
				keyStyle.Render("ctrl+r"),
				keyStyle.Render("esc"),
			),
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderResult() string {
	lines := []string{
		m.resultView.View(),
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s copy  %s rerun  %s edit  %s back",
				keyStyle.Render("c"),
				keyStyle.Render("r"),
				keyStyle.Render("e"),
				keyStyle.Render("esc"),
			),
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderBusy() string {
	lines := []string{
		fmt.Sprintf("%s %s", m.spin.View(), accentStyle.Render(m.busyLabel)),
	}

	stage := busyStageLabel(m.busyStage)
	if stage != "" {
		lines = append(lines, "Stage: "+stage)
	}
	if m.busyDetail != "" {
		lines = append(lines, "Detail: "+m.busyDetail)
	}
	if m.busyTotal > 0 {
		percent := float64(m.busyCurrent) / float64(m.busyTotal) * 100
		lines = append(lines, fmt.Sprintf("Progress: %d/%d (%.1f%%)", m.busyCurrent, m.busyTotal, percent))
	}
	if !m.busyElapsed.IsZero() {
		lines = append(lines, fmt.Sprintf("Elapsed: %s", time.Since(m.busyElapsed).Round(100*time.Millisecond)))
	}
	if m.busyMode == "rebuild" && m.busyTip != "" {
		lines = append(lines, "", helpStyle.Render(m.busyTip))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderFooter() string {
	parts := []string{
		helpStyle.Render(fmt.Sprintf("%s quit", keyStyle.Render("ctrl+c"))),
	}
	if m.screen == screenSettings && m.settingsDirty {
		parts = append(parts, noticeStyle.Render("Unsaved settings changes"))
	}
	if m.notice != "" {
		parts = append(parts, noticeStyle.Render(m.notice))
	}
	if m.lastErr != "" {
		parts = append(parts, errorStyle.Render(m.lastErr))
	}
	return strings.Join(parts, "  |  ")
}

func (m *model) startBusy(mode string, label string) {
	m.screen = screenBusy
	m.busyMode = mode
	m.busyLabel = label
	m.busyElapsed = time.Now()
	m.busyStage = ""
	m.busyCurrent = 0
	m.busyTotal = 0
	m.busyDetail = ""
	if mode == "rebuild" {
		m.tipIndex = 0
		if len(rebuildTips) > 0 {
			m.busyTip = rebuildTips[0]
		}
	} else {
		m.busyTip = ""
	}
}

func (m *model) clearBusyState() {
	m.busyMode = ""
	m.rebuildOrigin = ""
	m.busyLabel = ""
	m.busyStage = ""
	m.busyCurrent = 0
	m.busyTotal = 0
	m.busyDetail = ""
	m.busyTip = ""
	m.busyElapsed = time.Time{}
	m.rebuildProgressCh = nil
}

func (m *model) startRebuildCmd(origin string, label string) tea.Cmd {
	progressCh := make(chan services.DatasetRebuildProgress, 32)
	m.rebuildProgressCh = progressCh
	m.rebuildOrigin = origin
	m.startBusy("rebuild", label)
	return tea.Batch(
		m.spin.Tick,
		busyTooltipTickCmd(),
		waitRebuildProgressCmd(progressCh),
		datasetRebuildCmd(m.ctx, m.datasetService, progressCh),
	)
}

func (m *model) refreshSettingsList() {
	items := buildSettingsItems(m.settingsDraft)
	m.settingsList.SetItems(items)
	if len(items) == 0 {
		return
	}
	idx := m.settingsList.Index()
	if idx < 0 {
		idx = 0
	}
	if idx >= len(items) {
		idx = len(items) - 1
	}
	m.settingsList.Select(idx)
}

func (m model) selectedSettingItem() (settingItem, bool) {
	item, ok := m.settingsList.SelectedItem().(settingItem)
	if !ok {
		return settingItem{}, false
	}
	return item, true
}

func (m model) activateOrEditSetting(direction int) (tea.Model, tea.Cmd) {
	current, ok := m.selectedSettingItem()
	if !ok {
		return m, nil
	}

	next := m.settingsDraft
	changed, err := cycleSettingValue(&next, current.field, direction)
	if err != nil {
		m.lastErr = err.Error()
		return m, nil
	}
	if changed {
		m.settingsDraft = next
		m.settingsDirty = m.settingsDraft != m.cfg
		m.settingsError = ""
		m.refreshSettingsList()
		return m, nil
	}
	if direction < 0 {
		return m, nil
	}

	input := textinput.New()
	input.SetValue(rawSettingValue(m.settingsDraft, current.field.key))
	input.Width = max(30, m.width-20)
	input.Placeholder = "Enter value..."
	if current.field.kind == settingKindSecret {
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '*'
	}
	input.Focus()

	m.settingsInput = input
	m.settingsEdit = true
	m.settingsError = ""
	return m, nil
}

func (m model) selectedKnowledgeList() []string {
	out := make([]string, 0, len(m.selectedKnowledge))
	for name := range m.selectedKnowledge {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func nextMode(current domain.Mode) domain.Mode {
	switch current {
	case domain.ModeNatural:
		return domain.ModeBooru
	case domain.ModeBooru:
		return domain.ModeHybrid
	default:
		return domain.ModeNatural
	}
}

func enhanceCmd(ctx context.Context, service *services.PromptService, req domain.EnhanceRequest) tea.Cmd {
	return func() tea.Msg {
		result, warnings, err := service.Enhance(ctx, req)
		return enhanceDoneMsg{
			result:   result,
			warnings: warnings,
			err:      err,
		}
	}
}

func saveSettingsCmd(saveFn func(config.Config) error, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		if saveFn == nil {
			return settingsSavedMsg{}
		}
		err := saveFn(cfg)
		return settingsSavedMsg{err: err}
	}
}

func startupRebuildCheckCmd(_ context.Context, service *services.DatasetService) tea.Cmd {
	return func() tea.Msg {
		needed, reasons, err := service.NeedsRebuild()
		if err != nil {
			return startupRebuildCheckDoneMsg{err: err}
		}
		return startupRebuildCheckDoneMsg{
			needed:  needed,
			auto:    service.AutoRebuildEnabled(),
			reasons: reasons,
			err:     nil,
		}
	}
}

func datasetStatusCmd(ctx context.Context, service *services.DatasetService) tea.Cmd {
	return func() tea.Msg {
		status, err := service.Status(ctx)
		return datasetStatusDoneMsg{
			status: status,
			err:    err,
		}
	}
}

func datasetRebuildCmd(ctx context.Context, service *services.DatasetService, progressCh chan<- services.DatasetRebuildProgress) tea.Cmd {
	return func() tea.Msg {
		defer close(progressCh)
		_, err := service.RebuildWithProgress(ctx, func(p services.DatasetRebuildProgress) {
			select {
			case progressCh <- p:
			default:
				// Keep UI responsive; dropping a frame is acceptable for high-frequency progress.
			}
		})
		return datasetRebuildDoneMsg{err: err}
	}
}

func waitRebuildProgressCmd(progressCh <-chan services.DatasetRebuildProgress) tea.Cmd {
	return func() tea.Msg {
		if progressCh == nil {
			return rebuildProgressClosedMsg{}
		}
		p, ok := <-progressCh
		if !ok {
			return rebuildProgressClosedMsg{}
		}
		return datasetRebuildProgressMsg{progress: p}
	}
}

func busyTooltipTickCmd() tea.Cmd {
	return tea.Tick(900*time.Millisecond, func(_ time.Time) tea.Msg {
		return busyTooltipTickMsg{}
	})
}

func buildResultText(result *domain.EnhanceResult, warnings []string) string {
	if result == nil {
		return "No result."
	}

	lines := []string{
		"Output",
		"------",
		result.Output,
		"",
		fmt.Sprintf("Provider: %s (used=%t)", result.ProviderName, result.UsedProvider),
		fmt.Sprintf("Prompt chain: applied=%t stages=%d", result.ChainApplied, result.ChainStages),
		fmt.Sprintf("Validation applied: %t", result.ValidationApplied),
		"",
		"Retrieval Summary",
		"-----------------",
		fmt.Sprintf("Character tags: %d", len(result.Retrieval.CharacterTags)),
		fmt.Sprintf("Confirmed tags: %d", len(result.Retrieval.ConfirmedTags)),
		fmt.Sprintf("Suggested tags: %d", len(result.Retrieval.SuggestedTags)),
		fmt.Sprintf("Rejected tags: %d", len(result.Retrieval.RejectedTags)),
	}

	if len(warnings) > 0 {
		lines = append(lines, "", "Warnings", "--------")
		for _, w := range warnings {
			lines = append(lines, "- "+w)
		}
	}

	return strings.Join(lines, "\n")
}

func buildDatasetText(status services.DatasetStatus) string {
	lines := []string{
		"Dataset Status",
		"--------------",
		"tag.csv: " + status.Paths.TagCSV,
		"character.csv: " + status.Paths.CharacterCSV,
		"cache db: " + status.Paths.DBPath,
		"metadata: " + status.MetadataPath,
		fmt.Sprintf("db exists: %t", status.HasDatabase),
		fmt.Sprintf("rebuild needed: %t", status.RebuildNeeded),
	}
	if len(status.RebuildReasons) > 0 {
		lines = append(lines, "reasons: "+strings.Join(status.RebuildReasons, "; "))
	}
	lines = append(lines,
		"",
		"Row Counts",
		"----------",
		fmt.Sprintf("tags: %d", status.Counts.Tags),
		fmt.Sprintf("aliases: %d", status.Counts.TagAliases),
		fmt.Sprintf("characters: %d", status.Counts.Characters),
		fmt.Sprintf("character triggers: %d", status.Counts.CharacterTriggers),
		fmt.Sprintf("character core tags: %d", status.Counts.CharacterCoreTags),
	)
	return strings.Join(lines, "\n")
}

func busyStageLabel(stage string) string {
	switch stage {
	case "load_tags":
		return "Loading tag CSV"
	case "load_characters":
		return "Loading character CSV"
	case "create_schema":
		return "Creating SQLite schema"
	case "insert_tags":
		return "Indexing tags"
	case "insert_characters":
		return "Indexing characters"
	case "commit":
		return "Committing transaction"
	case "count_rows":
		return "Counting indexed rows"
	case "swap_db":
		return "Replacing cache database"
	case "hash_csv":
		return "Computing CSV hashes"
	case "done":
		return "Completed"
	default:
		return stage
	}
}

func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "up-to-date"
	}
	return strings.Join(reasons, "; ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ternary[T any](cond bool, left T, right T) T {
	if cond {
		return left
	}
	return right
}
