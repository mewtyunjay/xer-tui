package viewer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"xer-tui/internal/update"
	"xer-tui/internal/version"
)

const (
	screenBrowser = "browser"
	screenViewer  = "viewer"

	searchRowsMode   = "row"
	filterRowsMode   = "filter-rows"
	filterTablesMode = "filter-tables"
)

type keyMap struct {
	NextTable   key.Binding
	PrevTable   key.Binding
	Down        key.Binding
	Up          key.Binding
	PageDown    key.Binding
	PageUp      key.Binding
	Right       key.Binding
	Left        key.Binding
	FastRight   key.Binding
	FastLeft    key.Binding
	Top         key.Binding
	Bottom      key.Binding
	Home        key.Binding
	Search      key.Binding
	FilterRows  key.Binding
	FilterTable key.Binding
	NextMatch   key.Binding
	PrevMatch   key.Binding
	Clear       key.Binding
	BrowseFiles key.Binding
	OpenFile    key.Binding
	Update      key.Binding
	Help        key.Binding
	Quit        key.Binding
}

type Model struct {
	data *FileData
	dir  string
	mode string

	files      []string
	fileIndex  int
	fileScroll int

	width  int
	height int

	tableIndex   int
	tableScroll  int
	selectedRow  int
	rowScroll    int
	columnScroll int

	searchMode      string
	searchInput     textinput.Model
	filteredIndices []int
	rowQuery        string
	rowFilterActive bool
	filteredRows    []int
	matchedRows     []int

	checkingUpdate  bool
	UpdateRequested bool
	status          string

	showHelp bool
	keys     keyMap
	help     help.Model
}

type styles struct {
	Title         lipgloss.Style
	Muted         lipgloss.Style
	SidebarTitle  lipgloss.Style
	SidebarItem   lipgloss.Style
	SidebarActive lipgloss.Style
	Status        lipgloss.Style
	Header        lipgloss.Style
	SelectedRow   lipgloss.Style
	Empty         lipgloss.Style
}

var appStyles = styles{
	Title:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
	Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	SidebarTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
	SidebarItem:   lipgloss.NewStyle().Padding(0, 1),
	SidebarActive: lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")),
	Status:        lipgloss.NewStyle().Bold(true),
	Header:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
	SelectedRow:   lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")),
	Empty:         lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true),
}

func NewModel(data *FileData, dir string) Model {
	if dir == "" {
		switch {
		case data != nil && data.Path != "":
			dir = filepath.Dir(data.Path)
		default:
			cwd, err := os.Getwd()
			if err != nil {
				dir = "."
			} else {
				dir = cwd
			}
		}
	}

	h := help.New()
	h.ShowAll = false

	si := textinput.New()
	si.Prompt = "/"
	si.CharLimit = 64

	m := Model{
		data:        data,
		dir:         dir,
		mode:        screenBrowser,
		searchInput: si,
		keys: keyMap{
			NextTable:   key.NewBinding(key.WithKeys("tab", "]"), key.WithHelp("tab", "next table")),
			PrevTable:   key.NewBinding(key.WithKeys("shift+tab", "["), key.WithHelp("shift+tab", "prev table")),
			Down:        key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/down", "next")),
			Up:          key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k/up", "prev")),
			PageDown:    key.NewBinding(key.WithKeys("pgdown", "d"), key.WithHelp("pgdn", "page down")),
			PageUp:      key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
			Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("l/right", "scroll right")),
			Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("h/left", "scroll left")),
			FastRight:   key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "jump right")),
			FastLeft:    key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "jump left")),
			Home:        key.NewBinding(key.WithKeys("0"), key.WithHelp("0", "left edge")),
			Top:         key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
			Bottom:      key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
			Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search rows")),
			FilterRows:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter rows")),
			FilterTable: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "filter tables")),
			NextMatch:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
			PrevMatch:   key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
			Clear:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filters")),
			BrowseFiles: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "browse files")),
			OpenFile:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open file")),
			Update:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update xv")),
			Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
			Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		},
		help: h,
	}

	if data != nil {
		m.mode = screenViewer
	}

	m.refreshFiles()
	return m
}

func (m Model) ShortHelp() []key.Binding {
	if m.mode == screenBrowser {
		return []key.Binding{m.keys.OpenFile, m.keys.Down, m.keys.Update, m.keys.Help, m.keys.Quit}
	}
	return []key.Binding{m.keys.NextTable, m.keys.Down, m.keys.Right, m.keys.Search, m.keys.FilterRows, m.keys.BrowseFiles, m.keys.Quit}
}

func (m Model) FullHelp() [][]key.Binding {
	if m.mode == screenBrowser {
		return [][]key.Binding{
			{m.keys.OpenFile, m.keys.Down, m.keys.Up, m.keys.PageDown, m.keys.PageUp, m.keys.Top, m.keys.Bottom},
			{m.keys.Help, m.keys.Update, m.keys.Quit},
		}
	}
	return [][]key.Binding{
		{m.keys.NextTable, m.keys.PrevTable, m.keys.Down, m.keys.Up, m.keys.PageDown, m.keys.PageUp},
		{m.keys.Right, m.keys.Left, m.keys.FastRight, m.keys.FastLeft, m.keys.Home, m.keys.Top, m.keys.Bottom},
		{m.keys.Search, m.keys.FilterRows, m.keys.FilterTable, m.keys.NextMatch, m.keys.PrevMatch, m.keys.Clear, m.keys.BrowseFiles, m.keys.Update, m.keys.Help, m.keys.Quit},
	}
}

type updateCheckMsg struct {
	Result update.Result
	Err    error
}

func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		updater, err := update.New(update.Config{
			RepoOwner:      version.RepositoryOwner,
			RepoName:       version.RepositoryName,
			BinaryName:     version.BinaryName,
			CurrentVersion: version.Current(),
		})
		if err != nil {
			return updateCheckMsg{Err: err}
		}
		result, err := updater.Check(context.Background())
		return updateCheckMsg{Result: result, Err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.clamp()
		return m, nil

	case updateCheckMsg:
		m.checkingUpdate = false
		if typed.Err != nil {
			m.status = typed.Err.Error()
			return m, nil
		}
		if !typed.Result.Available {
			m.status = fmt.Sprintf("already up to date (%s)", displayVersion(typed.Result.LatestVersion))
			return m, nil
		}
		m.status = fmt.Sprintf("update %s -> %s available, closing to install...",
			displayVersion(typed.Result.PreviousVersion),
			displayVersion(typed.Result.LatestVersion))
		m.UpdateRequested = true
		return m, tea.Quit

	case tea.KeyMsg:
		if m.status != "" {
			m.status = ""
		}
		if m.searchMode != "" {
			return m.updateSearch(typed)
		}
		if m.mode == screenBrowser || m.data == nil {
			return m.updateBrowser(typed)
		}
		return m.updateViewer(typed)
	}

	return m, nil
}

func (m Model) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
	case key.Matches(msg, m.keys.Update):
		if m.checkingUpdate {
			return m, nil
		}
		m.status = "checking latest release..."
		m.checkingUpdate = true
		return m, checkForUpdateCmd()
	case key.Matches(msg, m.keys.OpenFile):
		if err := m.openSelectedFile(); err != nil {
			m.status = err.Error()
		}
	case key.Matches(msg, m.keys.Down):
		m.moveFile(1)
	case key.Matches(msg, m.keys.Up):
		m.moveFile(-1)
	case key.Matches(msg, m.keys.PageDown):
		m.moveFile(max(1, m.filesVisible()))
	case key.Matches(msg, m.keys.PageUp):
		m.moveFile(-max(1, m.filesVisible()))
	case key.Matches(msg, m.keys.Top):
		m.fileIndex = 0
	case key.Matches(msg, m.keys.Bottom):
		if len(m.files) > 0 {
			m.fileIndex = len(m.files) - 1
		}
	}

	m.clamp()
	return m, nil
}

func (m Model) updateViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Search):
		return m.beginSearch(searchRowsMode, "/", m.rowQuery)
	case key.Matches(msg, m.keys.FilterRows):
		return m.beginSearch(filterRowsMode, "f:", m.rowQuery)
	case key.Matches(msg, m.keys.FilterTable):
		m.filteredIndices = nil
		m.tableScroll = 0
		return m.beginSearch(filterTablesMode, "t:", "")
	case key.Matches(msg, m.keys.NextMatch):
		m.jumpToMatch(1)
	case key.Matches(msg, m.keys.PrevMatch):
		m.jumpToMatch(-1)
	case key.Matches(msg, m.keys.Clear):
		m.clearFilters()
	case key.Matches(msg, m.keys.BrowseFiles):
		m.enterBrowser()
	case key.Matches(msg, m.keys.Update):
		if m.checkingUpdate {
			return m, nil
		}
		m.status = "checking latest release..."
		m.checkingUpdate = true
		return m, checkForUpdateCmd()
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
	case key.Matches(msg, m.keys.NextTable):
		m.setTable(m.visibleTablePos() + 1)
	case key.Matches(msg, m.keys.PrevTable):
		m.setTable(m.visibleTablePos() - 1)
	case key.Matches(msg, m.keys.Down):
		m.moveRow(1)
	case key.Matches(msg, m.keys.Up):
		m.moveRow(-1)
	case key.Matches(msg, m.keys.PageDown):
		m.moveRow(max(1, m.rowsVisible()))
	case key.Matches(msg, m.keys.PageUp):
		m.moveRow(-max(1, m.rowsVisible()))
	case key.Matches(msg, m.keys.Right):
		m.moveColumn(4)
	case key.Matches(msg, m.keys.Left):
		m.moveColumn(-4)
	case key.Matches(msg, m.keys.FastRight):
		m.moveColumn(max(10, m.tableViewportWidth()/2))
	case key.Matches(msg, m.keys.FastLeft):
		m.moveColumn(-max(10, m.tableViewportWidth()/2))
	case key.Matches(msg, m.keys.Home):
		m.columnScroll = 0
	case key.Matches(msg, m.keys.Top):
		m.selectedRow = 0
		m.rowScroll = 0
	case key.Matches(msg, m.keys.Bottom):
		if rows := m.visibleRowCount(); rows > 0 {
			m.selectedRow = rows - 1
			m.ensureRowVisible()
		}
	}

	m.clamp()
	return m, nil
}

func (m Model) beginSearch(mode, prompt, value string) (tea.Model, tea.Cmd) {
	m.searchMode = mode
	m.searchInput.Prompt = prompt
	m.searchInput.SetValue(value)
	return m, m.searchInput.Focus()
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchInput.Blur()
		switch m.searchMode {
		case filterTablesMode:
			if len(m.filteredIndices) > 0 {
				m.tableIndex = m.filteredIndices[0]
			}
			m.selectedRow = 0
			m.rowScroll = 0
			m.columnScroll = 0
			m.rebuildRowState()
		case searchRowsMode:
			m.applyRowQuery(m.searchInput.Value(), false)
		case filterRowsMode:
			m.applyRowQuery(m.searchInput.Value(), true)
		}
		m.searchMode = ""
		m.clamp()
		return m, nil
	case "esc":
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		if m.searchMode == filterTablesMode {
			m.filteredIndices = nil
			m.tableScroll = 0
		}
		m.searchMode = ""
		m.clamp()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	if m.searchMode == filterTablesMode {
		m.updateTableFilter()
	}
	return m, cmd
}

func (m *Model) applyRowQuery(query string, filter bool) {
	m.rowQuery = strings.TrimSpace(query)
	m.rowFilterActive = filter && m.rowQuery != ""
	m.selectedRow = 0
	m.rowScroll = 0
	m.rebuildRowState()
	if m.rowQuery == "" {
		return
	}
	m.jumpToMatch(0)
}

func (m *Model) clearFilters() {
	actualRow := m.selectedActualRow()
	m.filteredIndices = nil
	m.tableScroll = 0
	m.rowQuery = ""
	m.rowFilterActive = false
	m.filteredRows = nil
	m.matchedRows = nil
	if actualRow >= 0 {
		m.selectedRow = actualRow
	} else {
		m.selectedRow = 0
	}
	m.rowScroll = 0
}

func (m *Model) updateTableFilter() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if query == "" {
		m.filteredIndices = nil
		m.tableScroll = 0
		return
	}

	prevTable := m.tableIndex
	m.filteredIndices = make([]int, 0)
	for i, table := range m.data.Tables {
		if strings.Contains(strings.ToLower(table.Name), query) {
			m.filteredIndices = append(m.filteredIndices, i)
		}
	}
	if len(m.filteredIndices) > 0 {
		m.tableIndex = m.filteredIndices[0]
		if m.tableIndex != prevTable {
			m.selectedRow = 0
			m.rowScroll = 0
			m.columnScroll = 0
			m.rebuildRowState()
		}
	}
	m.tableScroll = 0
}

func (m *Model) rebuildRowState() {
	m.filteredRows = m.filteredRows[:0]
	m.matchedRows = m.matchedRows[:0]

	query := strings.ToLower(strings.TrimSpace(m.rowQuery))
	if query == "" || m.data == nil {
		return
	}

	table := m.currentTable()
	for row := 0; row < table.RowCount(); row++ {
		if !rowContains(table, row, query) {
			continue
		}
		if m.rowFilterActive {
			m.filteredRows = append(m.filteredRows, row)
			m.matchedRows = append(m.matchedRows, len(m.filteredRows)-1)
			continue
		}
		m.matchedRows = append(m.matchedRows, row)
	}
}

func (m *Model) jumpToMatch(direction int) {
	if len(m.matchedRows) == 0 {
		if m.rowQuery != "" {
			m.status = "no matches"
		}
		return
	}

	targetIndex := 0
	switch {
	case direction > 0:
		targetIndex = 0
		for i, row := range m.matchedRows {
			if row > m.selectedRow {
				targetIndex = i
				break
			}
		}
		if m.matchedRows[targetIndex] <= m.selectedRow {
			targetIndex = 0
		}
	case direction < 0:
		targetIndex = len(m.matchedRows) - 1
		for i := len(m.matchedRows) - 1; i >= 0; i-- {
			if m.matchedRows[i] < m.selectedRow {
				targetIndex = i
				break
			}
		}
		if m.matchedRows[targetIndex] >= m.selectedRow {
			targetIndex = len(m.matchedRows) - 1
		}
	}

	m.selectedRow = m.matchedRows[targetIndex]
	m.ensureRowVisible()
	m.status = fmt.Sprintf("match %d/%d", targetIndex+1, len(m.matchedRows))
}

func rowContains(table TableData, row int, query string) bool {
	for col := 0; col < table.ColumnCount(); col++ {
		if strings.Contains(strings.ToLower(table.Cell(row, col)), query) {
			return true
		}
	}
	return false
}

func (m Model) visibleTables() []int {
	if m.data == nil {
		return nil
	}
	if m.filteredIndices != nil {
		return m.filteredIndices
	}
	all := make([]int, len(m.data.Tables))
	for i := range all {
		all[i] = i
	}
	return all
}

func (m Model) visibleTablePos() int {
	for i, idx := range m.visibleTables() {
		if idx == m.tableIndex {
			return i
		}
	}
	return 0
}

func displayVersion(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.width < 60 || m.height < 10 {
		return appStyles.Empty.Render("window too small for xv")
	}

	headerLine := m.renderHeaderLine()
	contentHeight := m.contentHeight()

	var body string
	if m.mode == screenBrowser || m.data == nil {
		body = m.renderBrowser(contentHeight, m.width)
	} else {
		sidebarWidth := m.sidebarWidth()
		mainWidth := max(20, m.width-sidebarWidth-1)
		left := m.renderSidebar(contentHeight, sidebarWidth)
		right := m.renderMain(contentHeight, mainWidth)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	var footer string
	if m.searchMode != "" {
		footer = m.searchInput.View()
	} else {
		footer = m.help.View(m)
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, body, footer)
}

func (m Model) renderHeaderLine() string {
	title := appStyles.Title.Render("xv")

	var subject string
	switch {
	case m.mode == screenViewer && m.data != nil:
		subject = m.data.Name
	default:
		subject = m.dir
	}
	if subject != "" {
		title += "  " + subject
	}
	if m.status != "" {
		title += "  " + appStyles.Muted.Render(m.status)
	}
	return title
}

func (m Model) renderBrowser(height, width int) string {
	lines := make([]string, 0, height)
	title := fmt.Sprintf("Files (.xer)  %d", len(m.files))
	lines = append(lines, appStyles.SidebarTitle.Width(width).Render(fitToWidth(title, width)))
	lines = append(lines, appStyles.Muted.Width(width).Render(fitToWidth(m.dir, width)))

	if len(m.files) == 0 {
		lines = append(lines, appStyles.Empty.Width(width).Render(fitToWidth("no .xer files in this directory", width)))
	} else {
		visible := max(1, height-2)
		end := min(len(m.files), m.fileScroll+visible)
		for i := m.fileScroll; i < end; i++ {
			label := filepath.Base(m.files[i])
			if m.data != nil && m.files[i] == m.data.Path {
				label += "  current"
			}
			label = fitToWidth(label, width)
			style := appStyles.SidebarItem
			if i == m.fileIndex {
				style = appStyles.SidebarActive
			}
			lines = append(lines, style.Width(width).Render(label))
		}
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (m *Model) currentTable() TableData {
	if m.data == nil || len(m.data.Tables) == 0 {
		return TableData{}
	}
	return m.data.Tables[m.tableIndex]
}

func (m *Model) setTable(index int) {
	tables := m.visibleTables()
	if len(tables) == 0 {
		return
	}
	switch {
	case index < 0:
		index = len(tables) - 1
	case index >= len(tables):
		index = 0
	}

	m.tableIndex = tables[index]
	m.selectedRow = 0
	m.rowScroll = 0
	m.columnScroll = 0
	m.rebuildRowState()
	if m.rowQuery != "" {
		m.jumpToMatch(0)
	}
	m.ensureTableVisible()
}

func (m *Model) moveFile(delta int) {
	if len(m.files) == 0 {
		m.fileIndex = 0
		m.fileScroll = 0
		return
	}
	m.fileIndex += delta
	if m.fileIndex < 0 {
		m.fileIndex = 0
	}
	if m.fileIndex >= len(m.files) {
		m.fileIndex = len(m.files) - 1
	}
	m.ensureFileVisible()
}

func (m *Model) moveRow(delta int) {
	rows := m.visibleRowCount()
	if rows == 0 {
		m.selectedRow = 0
		m.rowScroll = 0
		return
	}

	m.selectedRow += delta
	if m.selectedRow < 0 {
		m.selectedRow = 0
	}
	if m.selectedRow >= rows {
		m.selectedRow = rows - 1
	}
	m.ensureRowVisible()
}

func (m *Model) moveColumn(delta int) {
	m.columnScroll += delta
}

func (m *Model) openSelectedFile() error {
	if len(m.files) == 0 {
		return fmt.Errorf("no .xer files found in %s", m.dir)
	}

	path := m.files[m.fileIndex]
	data, err := LoadFile(path)
	if err != nil {
		return err
	}

	m.data = data
	m.dir = filepath.Dir(path)
	m.mode = screenViewer
	m.filteredIndices = nil
	m.tableIndex = 0
	m.tableScroll = 0
	m.selectedRow = 0
	m.rowScroll = 0
	m.columnScroll = 0
	m.rowQuery = ""
	m.rowFilterActive = false
	m.filteredRows = nil
	m.matchedRows = nil
	m.refreshFiles()
	return nil
}

func (m *Model) enterBrowser() {
	m.mode = screenBrowser
	m.searchMode = ""
	m.searchInput.Blur()
	m.refreshFiles()
}

func (m *Model) refreshFiles() {
	files, err := listXERFiles(m.dir)
	if err != nil {
		m.files = nil
		m.fileIndex = 0
		m.fileScroll = 0
		m.status = err.Error()
		return
	}

	m.files = files
	if len(m.files) == 0 {
		m.fileIndex = 0
		m.fileScroll = 0
		return
	}

	if m.data != nil {
		if idx := indexOf(m.files, m.data.Path); idx >= 0 {
			m.fileIndex = idx
		}
	}
	if m.fileIndex >= len(m.files) {
		m.fileIndex = len(m.files) - 1
	}
	if m.fileIndex < 0 {
		m.fileIndex = 0
	}
	m.ensureFileVisible()
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func (m Model) visibleRowCount() int {
	if m.rowFilterActive {
		return len(m.filteredRows)
	}
	return m.currentTable().RowCount()
}

func (m Model) actualRowIndex(visibleRow int) int {
	switch {
	case visibleRow < 0:
		return -1
	case m.rowFilterActive:
		if visibleRow >= len(m.filteredRows) {
			return -1
		}
		return m.filteredRows[visibleRow]
	default:
		if visibleRow >= m.currentTable().RowCount() {
			return -1
		}
		return visibleRow
	}
}

func (m Model) selectedActualRow() int {
	return m.actualRowIndex(m.selectedRow)
}

func (m *Model) clamp() {
	if m.mode == screenBrowser || m.data == nil {
		m.clampBrowser()
		return
	}
	m.clampViewer()
}

func (m *Model) clampBrowser() {
	if len(m.files) == 0 {
		m.fileIndex = 0
		m.fileScroll = 0
		return
	}
	if m.fileIndex < 0 {
		m.fileIndex = 0
	}
	if m.fileIndex >= len(m.files) {
		m.fileIndex = len(m.files) - 1
	}

	maxFileScroll := max(0, len(m.files)-m.filesVisible())
	if m.fileScroll < 0 {
		m.fileScroll = 0
	}
	if m.fileScroll > maxFileScroll {
		m.fileScroll = maxFileScroll
	}
	m.ensureFileVisible()
}

func (m *Model) clampViewer() {
	if len(m.data.Tables) == 0 {
		return
	}

	if m.tableIndex < 0 {
		m.tableIndex = 0
	}
	if m.tableIndex >= len(m.data.Tables) {
		m.tableIndex = len(m.data.Tables) - 1
	}

	rows := m.visibleRowCount()
	if rows == 0 {
		m.selectedRow = 0
		m.rowScroll = 0
	} else {
		if m.selectedRow < 0 {
			m.selectedRow = 0
		}
		if m.selectedRow >= rows {
			m.selectedRow = rows - 1
		}
		maxRowScroll := max(0, rows-m.rowsVisible())
		if m.rowScroll < 0 {
			m.rowScroll = 0
		}
		if m.rowScroll > maxRowScroll {
			m.rowScroll = maxRowScroll
		}
	}

	maxColumnScroll := m.currentTable().maxHorizontalOffset(m.tableViewportWidth())
	if m.columnScroll < 0 {
		m.columnScroll = 0
	}
	if m.columnScroll > maxColumnScroll {
		m.columnScroll = maxColumnScroll
	}

	m.ensureRowVisible()
	m.ensureTableVisible()
}

func (m *Model) ensureFileVisible() {
	visible := m.filesVisible()
	if visible <= 0 {
		m.fileScroll = 0
		return
	}
	if m.fileIndex < m.fileScroll {
		m.fileScroll = m.fileIndex
	}
	if m.fileIndex >= m.fileScroll+visible {
		m.fileScroll = m.fileIndex - visible + 1
	}
}

func (m *Model) ensureRowVisible() {
	visible := m.rowsVisible()
	if visible <= 0 {
		m.rowScroll = 0
		return
	}
	if m.selectedRow < m.rowScroll {
		m.rowScroll = m.selectedRow
	}
	if m.selectedRow >= m.rowScroll+visible {
		m.rowScroll = m.selectedRow - visible + 1
	}
}

func (m *Model) ensureTableVisible() {
	pos := m.visibleTablePos()
	visible := max(1, m.contentHeight()-1)
	if pos < m.tableScroll {
		m.tableScroll = pos
	}
	if pos >= m.tableScroll+visible {
		m.tableScroll = pos - visible + 1
	}
}

func (m Model) sidebarWidth() int {
	maxWidth := 24
	for _, table := range m.data.Tables {
		label := m.tableLabel(table)
		maxWidth = max(maxWidth, runeLen(label)+2)
	}
	return min(maxWidth, 34)
}

func (m Model) contentHeight() int {
	return max(4, m.height-2)
}

func (m Model) rowsVisible() int {
	return max(1, m.contentHeight()-3)
}

func (m Model) filesVisible() int {
	return max(1, m.contentHeight()-2)
}

func (m Model) tableViewportWidth() int {
	sidebarWidth := m.sidebarWidth()
	mainWidth := max(20, m.width-sidebarWidth-1)
	table := m.currentTable()
	fixedWidth := table.RowNumberWidth + 3
	return max(1, mainWidth-fixedWidth)
}

func (m Model) renderSidebar(height, width int) string {
	lines := make([]string, 0, height)
	tables := m.visibleTables()

	title := "Tables"
	if len(m.filteredIndices) > 0 {
		title = fmt.Sprintf("Tables (%d)", len(m.filteredIndices))
	}
	lines = append(lines, appStyles.SidebarTitle.Width(width).Render(fitToWidth(title, width)))

	visible := max(1, height-1)
	end := min(len(tables), m.tableScroll+visible)
	for si := m.tableScroll; si < end; si++ {
		realIndex := tables[si]
		label := fitToWidth(m.tableLabel(m.data.Tables[realIndex]), width)
		style := appStyles.SidebarItem
		if realIndex == m.tableIndex {
			style = appStyles.SidebarActive
		}
		lines = append(lines, style.Width(width).Render(label))
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderMain(height, width int) string {
	table := m.currentTable()
	lines := make([]string, 0, height)

	rowsLabel := fmt.Sprintf("%d", table.RowCount())
	if m.rowFilterActive {
		rowsLabel = fmt.Sprintf("%d/%d", len(m.filteredRows), table.RowCount())
	}

	currentRow := 0
	if actual := m.selectedActualRow(); actual >= 0 {
		currentRow = actual + 1
	}

	status := fmt.Sprintf("%s  rows:%s  cols:%d  row:%d", table.Name, rowsLabel, table.ColumnCount(), currentRow)
	if !m.rowFilterActive {
		status += fmt.Sprintf("/%d", max(1, table.RowCount()))
	}
	if m.rowQuery != "" && !m.rowFilterActive {
		status += fmt.Sprintf("  matches:%d", len(m.matchedRows))
	}
	status += fmt.Sprintf("  x:%d", m.columnScroll)

	lines = append(lines, appStyles.Status.Width(width).Render(fitToWidth(status, width)))

	headerLine, separator := m.renderHeader(table, width)
	lines = append(lines, headerLine, separator)

	rowLimit := m.rowsVisible()
	for row := 0; row < rowLimit; row++ {
		rowIndex := m.rowScroll + row
		actualRow := m.actualRowIndex(rowIndex)
		if actualRow < 0 {
			lines = append(lines, strings.Repeat(" ", width))
			continue
		}

		line := m.renderRow(table, actualRow, width)
		if rowIndex == m.selectedRow {
			line = appStyles.SelectedRow.Width(width).Render(line)
		}
		lines = append(lines, line)
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (m Model) renderHeader(table TableData, width int) (string, string) {
	fixedWidth := table.RowNumberWidth + 3
	dataWidth := max(1, width-fixedWidth)

	prefix := fmt.Sprintf("%*s | ", table.RowNumberWidth, "#")
	data := make([]string, len(table.Columns))
	for i, name := range table.Columns {
		data[i] = padOrTrim(name, table.ColumnWidths[i])
	}
	dataText := strings.Join(data, " | ")

	header := prefix + clipPad(dataText, m.columnScroll, dataWidth)
	separator := strings.Repeat("-", fixedWidth) + strings.Repeat("-", min(dataWidth, max(0, runeLen(dataText)-m.columnScroll)))

	return appStyles.Header.Width(width).Render(fitToWidth(header, width)), fitToWidth(separator, width)
}

func (m Model) renderRow(table TableData, row int, width int) string {
	fixedWidth := table.RowNumberWidth + 3
	dataWidth := max(1, width-fixedWidth)

	prefix := fmt.Sprintf("%*d | ", table.RowNumberWidth, row+1)
	data := make([]string, len(table.Columns))
	for col := range table.Columns {
		data[col] = padOrTrim(table.Cell(row, col), table.ColumnWidths[col])
	}
	dataText := strings.Join(data, " | ")
	return fitToWidth(prefix+clipPad(dataText, m.columnScroll, dataWidth), width)
}

func (m Model) tableLabel(table TableData) string {
	return fmt.Sprintf("%-14s %6d", table.Name, table.RowCount())
}

func runeLen(s string) int {
	return len([]rune(s))
}

func clipPad(s string, start, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if start >= len(runes) {
		return strings.Repeat(" ", width)
	}

	end := min(len(runes), start+width)
	out := string(runes[start:end])
	if pad := width - runeLen(out); pad > 0 {
		out += strings.Repeat(" ", pad)
	}
	return out
}

func fitToWidth(s string, width int) string {
	return clipPad(s, 0, width)
}

func padOrTrim(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > width {
		if width == 1 {
			return string(runes[:1])
		}
		return string(runes[:width-1]) + "…"
	}
	if pad := width - len(runes); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
