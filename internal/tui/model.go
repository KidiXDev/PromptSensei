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

type clockTickMsg struct{}

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
	contextEditor textarea.Model
	focusedEditor int
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
		homeItem{title: "Create Prompt", description: "Start from a fresh idea", action: actionCreate},
		homeItem{title: "Enhance Prompt", description: "Refine an existing prompt", action: actionEnhance},
		homeItem{title: "Settings", description: "Configure context, provider and database", action: actionSettings},
		homeItem{title: "Exit", description: "Quit PromptSensei", action: actionExit},
	}

	home := list.New(homeItems, list.NewDefaultDelegate(), 60, 14)
	home.SetShowTitle(false)
	home.SetShowStatusBar(false)
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
	editor.Placeholder = "Enter your prompt here..."
	editor.Focus()
	editor.SetWidth(80)
	editor.SetHeight(8)
	editor.ShowLineNumbers = false

	contextEditor := textarea.New()
	contextEditor.Placeholder = "Optional: Add more context or instructions (e.g. 'make it longer', 'include outdoors')..."
	contextEditor.SetWidth(80)
	contextEditor.SetHeight(4)
	contextEditor.ShowLineNumbers = false

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
		contextEditor:     contextEditor,
		focusedEditor:     0,
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

func clockTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return clockTickMsg{}
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		clockTickCmd(),
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
	case clockTickMsg:
		return m, clockTickCmd()
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
			m.notice = ""
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
		m.resultView.SetContent(buildResultText(msg.result, msg.warnings, m.resultView.Width))
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
		m.notice = ""
		m.lastErr = ""
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
			case actionSettings:
				m.settingsDraft = m.cfg
				m.settingsEdit = false
				m.settingsError = ""
				m.refreshSettingsList()
				m.screen = screenSettings
				return m, nil
			case actionExit:
				return m, tea.Quit
			}
		}
	}
	return m, cmd
}

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focusedEditor == 0 {
		m.editor, cmd = m.editor.Update(msg)
	} else {
		m.contextEditor, cmd = m.contextEditor.Update(msg)
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "tab":
			m.focusedEditor = (m.focusedEditor + 1) % 2
			if m.focusedEditor == 0 {
				m.editor.Focus()
				m.contextEditor.Blur()
			} else {
				m.editor.Blur()
				m.contextEditor.Focus()
			}
			return m, nil
		case "ctrl+s":
			prompt := strings.TrimSpace(m.editor.Value())
			context := strings.TrimSpace(m.contextEditor.Value())
			if prompt == "" {
				m.lastErr = "prompt cannot be empty"
				return m, nil
			}
			req := &domain.EnhanceRequest{
				Prompt:         prompt,
				Context:        context,
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
			m.syncKnowledgeListSelection()
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
			m.contextEditor.SetValue("")
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
		case "esc", "q":
			m.screen = m.knowledgeReturn
			return m, nil
		case " ", "enter":
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
			if item.selected {
				m.notice = "Knowledge enabled: " + item.name
			} else {
				m.notice = "Knowledge disabled: " + item.name
			}
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
				m.settingsEdit = false
				m.settingsError = ""
				m.refreshSettingsList()
				return m, saveSettingsCmd(m.saveConfig, m.settingsDraft)
			}
		}
		return m, cmd
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "ctrl+r":
			m.settingsDraft = m.cfg
			m.settingsError = ""
			m.refreshSettingsList()
			m.notice = "Discarded unsaved settings changes."
			return m, nil
		case "enter":
			return m.activateOrEditSetting(0)
		case "right", "l", " ":
			return m.activateOrEditSetting(1)
		case "left", "h":
			return m.activateOrEditSetting(-1)
		}
	}

	var cmd tea.Cmd
	m.settingsList, cmd = m.settingsList.Update(msg)
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
	header := titleStyle.Render(domain.AppName)
	subtitle := subtitleStyle.Render("AI prompt crafting with local booru-aware retrieval")

	body := ""
	panelW := max(40, m.width-2)
	switch m.screen {
	case screenHome:
		body = panelStyle.Width(panelW).Render(m.renderHome())
	case screenEditor:
		body = panelStyle.Width(panelW).Render(m.renderEditor())
	case screenKnowledge:
		body = panelStyle.Width(panelW).Render(m.renderKnowledge())
	case screenDataset:
		body = panelStyle.Width(panelW).Render(m.renderDataset())
	case screenSettings:
		body = panelStyle.Width(panelW).Render(m.renderSettings())
	case screenResult:
		body = panelStyle.Width(panelW).Render(m.renderResult())
	case screenBusy:
		body = panelStyle.Width(panelW).Render(m.renderBusy())
	}

	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, subtitle, "", body, "", footer)
}

func (m *model) resizeComponents() {
	if m.width == 0 || m.height == 0 {
		return
	}

	listWidth := max(40, m.width-4)
	listHeight := max(10, m.height-14)
	m.homeList.SetSize(listWidth, listHeight)
	m.knowledgeList.SetSize(listWidth, listHeight)
	m.settingsList.SetSize(listWidth, listHeight)

	totalEditorHeight := max(12, m.height-20)
	promptHeight := int(float64(totalEditorHeight) * 0.7)
	contextHeight := totalEditorHeight - promptHeight

	m.editor.SetWidth(max(40, m.width-14))
	m.editor.SetHeight(max(6, promptHeight))
	m.contextEditor.SetWidth(max(40, m.width-14))
	m.contextEditor.SetHeight(max(3, contextHeight))

	viewportWidth := max(40, m.width-12)
	viewportHeight := max(8, m.height-18)
	m.datasetView.Width = viewportWidth
	m.datasetView.Height = viewportHeight
	m.resultView.Width = viewportWidth
	m.resultView.Height = viewportHeight
}

func (m model) renderHome() string {
	return m.homeList.View()
}

func (m model) renderEditor() string {
	header := highlightStyle.Render("✍ PROMPT CRAFTING")

	modeType := ternary(m.createMode, noticeStyle.Render("CREATE"), accentStyle.Render("ENHANCE"))

	info := fmt.Sprintf("Task: %s  |  Mode: %s  |  Strict: %s",
		modeType,
		accentStyle.Render(string(m.mode)),
		ternary(m.strict, noticeStyle.Render("ON"), helpStyle.Render("OFF")))

	knowledge := "None"
	selected := m.selectedKnowledgeList()
	if len(selected) > 0 {
		knowledge = strings.Join(selected, ", ")
	}

	lines := []string{
		header,
		info,
		"Knowledge: " + helpStyle.Render(knowledge),
		"",
		accentStyle.Render("Prompt"),
		m.editor.View(),
		"",
		accentStyle.Render("Context (Optional)"),
		m.contextEditor.View(),
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s submit  %s switch  %s mode  %s strict  %s task  %s knowledge  %s clear  %s back",
				keyStyle.Render("ctrl+s"),
				keyStyle.Render("tab"),
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
	header := highlightStyle.Render("📚 KNOWLEDGE SELECTION")
	info := helpStyle.Render("Select files to include in the generation context.")

	lines := []string{
		header,
		info,
		"",
		m.knowledgeList.View(),
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s toggle  %s all  %s clear  %s back",
				keyStyle.Render("enter/space"),
				keyStyle.Render("a"),
				keyStyle.Render("c"),
				keyStyle.Render("esc/q"),
			),
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderDataset() string {
	header := highlightStyle.Render("📊 DATASET EXPLORER")

	lines := []string{
		header,
		"",
		m.datasetView.View(),
		"",
		helpStyle.Render(fmt.Sprintf("%s rebuild cache  %s back", keyStyle.Render("r"), keyStyle.Render("esc"))),
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
			accentStyle.Render("EDIT SETTING"),
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

	header := highlightStyle.Render("⚙ CONFIGURATION")

	lines := []string{
		header,
		"",
		m.settingsList.View(),
		"",
		accentStyle.Render("HELP") + " " + helpStyle.Render(selectedDesc),
		"",
		helpStyle.Render(
			fmt.Sprintf(
				"%s edit/toggle  %s cycles  %s reset  %s back",
				keyStyle.Render("enter/space"),
				keyStyle.Render("←/→"),
				keyStyle.Render("ctrl+r"),
				keyStyle.Render("esc"),
			),
		),
	}
	return strings.Join(lines, "\n")
}

func (m model) renderResult() string {
	if m.lastResult == nil {
		return "No result available."
	}

	header := highlightStyle.Render("✨ GENERATED PROMPT")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
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
	)
}

func buildResultText(result *domain.EnhanceResult, warnings []string, width int) string {
	if result == nil {
		return "No result available."
	}

	// Main Prompt Panel
	promptContent := promptPanelStyle.Width(width - 4).Render(formatPromptForDisplay(result.Output, width-8))

	// Technical sub-section
	techLines := []string{
		"",
		accentStyle.Render("DETAILS"),
		fmt.Sprintf("  Provider:   %s", result.ProviderName),
		fmt.Sprintf("  Validated:  %t", result.ValidationApplied),
		"",
		accentStyle.Render("RETRIEVAL"),
		fmt.Sprintf("  Character:  %d", len(result.Retrieval.CharacterTags)),
		fmt.Sprintf("  Confirmed:  %d", len(result.Retrieval.ConfirmedTags)),
		fmt.Sprintf("  Suggested:  %d", len(result.Retrieval.SuggestedTags)),
	}

	if len(warnings) > 0 {
		techLines = append(techLines, "", warningStyle.Render("WARNINGS"))
		for _, w := range warnings {
			techLines = append(techLines, "  - "+w)
		}
	}

	techPanel := techInfoStyle.Render(strings.Join(techLines, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left,
		promptContent,
		techPanel,
	)
}

func buildDatasetText(status services.DatasetStatus) string {
	lines := []string{
		accentStyle.Render("DATASET STATISTICS"),
		fmt.Sprintf("  Tags:           %d", status.Counts.Tags),
		fmt.Sprintf("  Aliases:        %d", status.Counts.TagAliases),
		fmt.Sprintf("  Characters:     %d", status.Counts.Characters),
		fmt.Sprintf("  Core Tags:      %d", status.Counts.CharacterCoreTags),
	}

	if status.RebuildNeeded {
		lines = append(lines, "", warningStyle.Render("! REBUILD RECOMMENDED"), "Reasons: "+strings.Join(status.RebuildReasons, ", "))
	}

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

func formatPromptForDisplay(prompt string, maxWidth int) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "(empty)"
	}
	parts := strings.Split(prompt, ",")
	lines := make([]string, 0, len(parts))
	current := ""
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if current == "" {
			current = part
			continue
		}
		next := current + ", " + part
		if len(next) > maxWidth {
			lines = append(lines, current+",")
			current = part
			continue
		}
		current = next
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return prompt
	}
	return strings.Join(lines, "\n")
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
func (m model) renderBusy() string {
	spin := m.spin.View()
	label := highlightStyle.Render(m.busyLabel)

	lines := []string{
		fmt.Sprintf("%s %s", spin, label),
		"",
	}

	stage := busyStageLabel(m.busyStage)
	if stage != "" {
		lines = append(lines, "Stage:  "+accentStyle.Render(stage))
	}

	if m.busyTotal > 0 {
		percent := float64(m.busyCurrent) / float64(m.busyTotal)
		width := max(20, m.width-20)
		filled := int(float64(width) * percent)
		if filled > width {
			filled = width
		}

		bar := selectedCheckboxStyle.Render(strings.Repeat("█", filled)) +
			helpStyle.Render(strings.Repeat("░", width-filled))

		lines = append(lines,
			fmt.Sprintf("Progress: %d / %d", m.busyCurrent, m.busyTotal),
			bar,
		)
	} else if m.busyDetail != "" {
		lines = append(lines, "Detail: "+helpStyle.Render(m.busyDetail))
	}

	if !m.busyElapsed.IsZero() {
		lines = append(lines, "", fmt.Sprintf("Time: %s", highlightStyle.Render(time.Since(m.busyElapsed).Round(100*time.Millisecond).String())))
	}

	if m.busyMode == "rebuild" && m.busyTip != "" {
		panelStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Margin(0, 0)
		lines = append(lines, "", panelStyle.Render(warningStyle.Render("💡 ")+helpStyle.Render(m.busyTip)))
	}

	return strings.Join(lines, "\n")
}

func (m model) renderFooter() string {
	left := helpStyle.Render(time.Now().Format("2006-01-02 15:04:05"))
	right := helpStyle.Render(fmt.Sprintf("%s %s", domain.AppName, domain.AppVersion))

	msg := ""
	if m.notice != "" {
		msg = noticeStyle.Render(m.notice)
	} else if m.lastErr != "" {
		msg = errorStyle.Render(m.lastErr)
	}

	totalWidth := m.width
	if totalWidth == 0 {
		totalWidth = 80
	}

	space := totalWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 0 {
		space = 0
	}

	if msg != "" {
		msgWidth := lipgloss.Width(msg)
		// Try to center the message
		midSpace := (space - msgWidth) / 2
		if midSpace < 1 {
			midSpace = 1
		}
		afterSpace := space - midSpace - msgWidth
		if afterSpace < 1 {
			afterSpace = 1
		}
		return left + strings.Repeat(" ", midSpace) + msg + strings.Repeat(" ", afterSpace) + right
	}

	return left + strings.Repeat(" ", space) + right
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

	// Case 1: Arrow keys (direction != 0) - Only for cycling/numeric
	if direction != 0 {
		next := m.settingsDraft
		changed, err := cycleSettingValue(&next, current.field, direction)
		if err != nil {
			m.lastErr = err.Error()
			return m, nil
		}
		if changed {
			m.settingsDraft = next
			return m, saveSettingsCmd(m.saveConfig, m.settingsDraft)
		}
		// If not cycleable, arrows do nothing
		return m, nil
	}

	// Case 2: Enter/Space (direction == 0)
	switch current.field.kind {
	case settingKindBool, settingKindEnum:
		// For toggle types, cycle forward
		next := m.settingsDraft
		_, err := cycleSettingValue(&next, current.field, 1)
		if err != nil {
			m.lastErr = err.Error()
			return m, nil
		}
		m.settingsDraft = next
		return m, saveSettingsCmd(m.saveConfig, m.settingsDraft)
	case settingKindAction:
		if current.field.key == settingActionDatasetStatus {
			m.startBusy("status", "Loading dataset status")
			return m, tea.Batch(m.spin.Tick, datasetStatusCmd(m.ctx, m.datasetService))
		}
	case settingKindString, settingKindSecret, settingKindFloat, settingKindInt:
		// Open text editor for precision input
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
	}

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

func (m *model) syncKnowledgeListSelection() {
	items := m.knowledgeList.Items()
	for i := range items {
		item, ok := items[i].(knowledgeItem)
		if !ok {
			continue
		}
		_, selected := m.selectedKnowledge[item.name]
		item.selected = selected
		items[i] = item
	}
	m.knowledgeList.SetItems(items)
}
