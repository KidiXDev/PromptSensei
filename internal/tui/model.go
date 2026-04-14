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

type model struct {
	ctx context.Context

	promptService    *services.PromptService
	datasetService   *services.DatasetService
	knowledgeService *services.KnowledgeService
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
	settingsView  viewport.Model
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

	selectedKnowledge map[string]struct{}
	rebuildProgressCh <-chan services.DatasetRebuildProgress
	tipIndex          int
}

func Run(
	ctx context.Context,
	promptService *services.PromptService,
	datasetService *services.DatasetService,
	knowledgeService *services.KnowledgeService,
	cfg config.Config,
	in io.Reader,
	out io.Writer,
) error {
	knowledgeFiles, err := knowledgeService.List()
	if err != nil {
		return err
	}

	m := newModel(ctx, promptService, datasetService, knowledgeService, cfg, knowledgeFiles)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(in), tea.WithOutput(out))
	_, err = p.Run()
	return err
}

func newModel(
	ctx context.Context,
	promptService *services.PromptService,
	datasetService *services.DatasetService,
	knowledgeService *services.KnowledgeService,
	cfg config.Config,
	knowledgeFiles []string,
) model {
	homeItems := []list.Item{
		homeItem{title: "Create Prompt", description: "Generate from a rough idea", action: actionCreate},
		homeItem{title: "Enhance Prompt", description: "Improve an existing prompt", action: actionEnhance},
		homeItem{title: "Dataset Status", description: "Inspect CSV/SQLite cache health", action: actionDatasetStatus},
		homeItem{title: "Knowledge Files", description: "Manage selected knowledge docs", action: actionKnowledge},
		homeItem{title: "Settings", description: "View provider and mode defaults", action: actionSettings},
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

	settingsText := buildSettingsText(cfg)
	settingsView := viewport.New(80, 14)
	settingsView.SetContent(settingsText)

	return model{
		ctx:               ctx,
		promptService:     promptService,
		datasetService:    datasetService,
		knowledgeService:  knowledgeService,
		cfg:               cfg,
		screen:            screenBusy,
		knowledgeReturn:   screenHome,
		homeList:          home,
		knowledgeList:     knowledge,
		editor:            editor,
		datasetView:       viewport.New(80, 14),
		resultView:        viewport.New(80, 14),
		settingsView:      settingsView,
		spin:              spin,
		mode:              cfg.General.DefaultMode,
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
				m.settingsView.SetContent(buildSettingsText(m.cfg))
				m.settingsView.GotoTop()
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
		case "m":
			m.mode = nextMode(m.mode)
			return m, nil
		case "s":
			m.strict = !m.strict
			return m, nil
		case "k":
			m.knowledgeReturn = screenEditor
			m.screen = screenKnowledge
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
	var cmd tea.Cmd
	m.settingsView, cmd = m.settingsView.Update(msg)
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "esc" {
			m.screen = screenHome
			return m, nil
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
		body = panelStyle.Render(m.homeList.View())
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
	listHeight := max(10, m.height-14)
	m.homeList.SetSize(listWidth, listHeight)
	m.knowledgeList.SetSize(listWidth, listHeight)
	m.editor.SetWidth(max(40, m.width-14))
	m.editor.SetHeight(max(8, m.height-20))

	viewportWidth := max(40, m.width-12)
	viewportHeight := max(8, m.height-18)
	m.datasetView.Width = viewportWidth
	m.datasetView.Height = viewportHeight
	m.resultView.Width = viewportWidth
	m.resultView.Height = viewportHeight
	m.settingsView.Width = viewportWidth
	m.settingsView.Height = viewportHeight
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
				"%s submit  %s mode  %s strict  %s knowledge  %s clear  %s back",
				keyStyle.Render("ctrl+s"),
				keyStyle.Render("m"),
				keyStyle.Render("s"),
				keyStyle.Render("k"),
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
	lines := []string{
		m.settingsView.View(),
		"",
		helpStyle.Render(fmt.Sprintf("%s back", keyStyle.Render("esc"))),
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

func buildSettingsText(cfg config.Config) string {
	maskedKey := "(empty)"
	if strings.TrimSpace(cfg.Provider.APIKey) != "" {
		maskedKey = "********"
	}
	lines := []string{
		"Settings",
		"--------",
		fmt.Sprintf("default mode: %s", cfg.General.DefaultMode),
		fmt.Sprintf("strict booru validation: %t", cfg.General.StrictBooruValidation),
		fmt.Sprintf("preferred provider: %s", cfg.General.PreferredProvider),
		fmt.Sprintf("preferred model: %s", cfg.General.PreferredModel),
		"",
		"Provider",
		"--------",
		fmt.Sprintf("enabled: %t", cfg.Provider.Enabled),
		fmt.Sprintf("name: %s", cfg.Provider.Name),
		fmt.Sprintf("model: %s", cfg.Provider.Model),
		fmt.Sprintf("api base url: %s", cfg.Provider.APIBaseURL),
		fmt.Sprintf("api key: %s", maskedKey),
		fmt.Sprintf("temperature: %.2f", cfg.Provider.Temperature),
		fmt.Sprintf("max tokens: %d", cfg.Provider.MaxTokens),
		fmt.Sprintf("timeout seconds: %d", cfg.Provider.TimeoutSeconds),
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
