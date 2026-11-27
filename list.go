package list

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"slices"
)

type Orientation int

const (
	Vertical Orientation = iota
	Horizontal
)

type Model struct {
	items          []ListItem
	orientation    Orientation
	width          int
	height         int
	activeIndex    int
	styles         Styles
	selectionStart int
	selectionEnd   int
	ScrollBar      bool
	FillWidth      bool
	Fade           *Fade
	Paged          bool
}

type ListItem interface {
	Init() tea.Cmd
	View() string
	Update(tea.Msg) tea.Cmd
}

// Sizer is implemented by items that need to know the list's size.
type Sizer interface {
	SetSize(w, h int)
}

type Styles struct {
	Item           lipgloss.Style
	Active         lipgloss.Style
	Selected       lipgloss.Style
	SelectedActive lipgloss.Style
	Indicator      lipgloss.Style
}

func New(items []ListItem, orientation Orientation, w, h int, s Styles) *Model {
	m := &Model{
		items:          items,
		orientation:    orientation,
		width:          w,
		height:         h,
		activeIndex:    -1,
		selectionStart: -1,
		selectionEnd:   -1,
		styles:         s,
	}
	for _, item := range items {
		m.sizeItem(item)
	}
	return m
}

func (m *Model) Items() []ListItem {
	return m.items
}

func (m *Model) ItemAt(i int) ListItem {
	if i < 0 || i >= len(m.items) {
		return nil
	}
	return m.items[i]
}

func (m *Model) sizeItem(item ListItem) {
	if s, ok := item.(Sizer); ok {
		s.SetSize(m.width, m.height)
	}
}

func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, item := range m.items {
		cmds = append(cmds, item.Init())
	}
	if m.hasValidActiveIndex() {
		m.activeIndex = 0
	}
	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	for _, item := range m.items {
		cmd = item.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) View() string {
	if len(m.items) == 0 {
		if m.ScrollBar && m.orientation == Vertical {
			return lipgloss.JoinHorizontal(lipgloss.Left, m.emptyColumn(), m.verticalScrollView())
		}
		return ""
	}

	switch m.orientation {
	case Horizontal:
		return m.horizontalView()
	case Vertical:
		return m.verticalView()
	default:
		return ""
	}
}

func (m *Model) emptyColumn() string {
	frame := max(m.styles.Item.GetHorizontalFrameSize(), m.styles.Active.GetHorizontalFrameSize())
	column := lipgloss.NewStyle().Width(m.width + frame).Height(max(0, m.height))
	return m.fill(column).Render("")
}

func (m *Model) fill(style lipgloss.Style) lipgloss.Style {
	if !m.FillWidth || m.orientation != Vertical || m.width <= 0 {
		return style
	}
	w := m.width
	if m.ScrollBar {
		w--
	}
	return style.Width(max(w, 1))
}

func (m *Model) renderActiveItem() string {
	if !m.hasValidActiveIndex() {
		return ""
	}
	if m.selecting() {
		return m.fill(m.styles.SelectedActive).Render(m.items[m.activeIndex].View())
	}
	return m.fill(m.styles.Active).Render(m.items[m.activeIndex].View())
}

func (m *Model) renderItem(i int) string {
	if m.selecting() && ((m.selectionStart <= i && i <= m.selectionEnd) || (m.selectionEnd <= i && i <= m.selectionStart)) {
		return m.fill(m.styles.Selected).Render(m.items[i].View())
	}
	view := m.items[i].View()
	if m.Fade != nil {
		view = Dim(view, *m.Fade)
	}
	return m.fill(m.styles.Item).Render(view)
}

func (m *Model) renderVerticalSections() []string {
	var sections []string
	var item string
	remainingSpace := m.height

	item = m.renderActiveItem()
	itemSize := lipgloss.Height(item)
	remainingSpace -= itemSize
	sections = append(sections, item)

	for i := m.activeIndex - 1; i >= 0 && !m.Paged; i-- {
		item = m.renderItem(i)
		itemSize := lipgloss.Height(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = slices.Insert(sections, 0, item)
		} else {
			break
		}
	}

	for i := m.activeIndex + 1; i < len(m.items) && !m.Paged; i++ {
		item = m.renderItem(i)
		itemSize := lipgloss.Height(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = append(sections, item)
		} else {
			break
		}
	}

	return sections
}

func (m *Model) verticalView() string {
	sections := m.renderVerticalSections()
	mainView := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.ScrollBar {
		return lipgloss.JoinHorizontal(lipgloss.Left, mainView, m.verticalScrollView())
	}
	return mainView
}

func (m *Model) renderHorizontalSections() []string {
	var sections []string
	var item string
	remainingSpace := m.width

	item = m.renderActiveItem()
	itemSize := lipgloss.Width(item)
	remainingSpace -= itemSize
	sections = append(sections, item)

	for i := m.activeIndex - 1; i >= 0 && !m.Paged; i-- {
		item = m.renderItem(i)
		itemSize := lipgloss.Width(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = slices.Insert(sections, 0, item)
		} else {
			break
		}
	}

	for i := m.activeIndex + 1; i < len(m.items) && !m.Paged; i++ {
		item = m.renderItem(i)
		itemSize := lipgloss.Width(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = append(sections, item)
		} else {
			break
		}
	}

	return sections
}

func (m *Model) horizontalView() string {
	sections := m.renderHorizontalSections()
	mainView := lipgloss.JoinHorizontal(lipgloss.Left, sections...)
	if m.Paged {
		mainView = lipgloss.NewStyle().Height(lipgloss.Height(m.renderActiveItem())).Render(mainView)
	} else {
		mainView = lipgloss.NewStyle().Height(m.tallestItemHeight()).Render(mainView)
	}
	if m.ScrollBar {
		dots := lipgloss.PlaceHorizontal(lipgloss.Width(mainView), lipgloss.Center, m.horizontalScrollView())
		return lipgloss.JoinVertical(lipgloss.Left, mainView, dots)
	}
	return lipgloss.JoinVertical(lipgloss.Left, mainView)
}

func (m *Model) tallestItemHeight() int {
	tallest := 0
	for i := range m.items {
		var item string
		if i == m.activeIndex {
			item = m.renderActiveItem()
		} else {
			item = m.renderItem(i)
		}
		tallest = max(tallest, lipgloss.Height(item))
	}
	return tallest
}

func (m *Model) IncrementItem(step int) {
	len := len(m.items)
	if len == 0 {
		return
	}
	if m.activeIndex+step >= len {
		m.GotoEndItem()
		return
	}
	m.activeIndex += step
	if m.selecting() {
		m.selectionEnd = m.activeIndex
	}
}

func (m *Model) DecrementItem(step int) {
	if len(m.items) == 0 {
		return
	}
	if 0 > m.activeIndex-step {
		m.GotoStartItem()
		return
	}
	m.activeIndex -= step
	if m.selecting() {
		m.selectionEnd = m.activeIndex
	}
}

func (m *Model) GotoStartItem() {
	if len(m.items) == 0 {
		m.activeIndex = -1
		return
	}
	m.activeIndex = 0
	if m.selecting() {
		m.selectionEnd = 0
	}
}

func (m *Model) GotoEndItem() {
	if len(m.items) == 0 {
		m.activeIndex = -1
		return
	}
	m.activeIndex = len(m.items) - 1
	if m.selecting() {
		m.selectionEnd = m.activeIndex
	}
}

func (m *Model) ClearItems() {
	m.items = nil
	m.activeIndex = -1
}

func (m *Model) InsertItem(item ListItem, index int) {
	if index > len(m.items) {
		index = len(m.items)
	}
	if index < 0 {
		index = 0
	}
	m.sizeItem(item)
	m.items = slices.Insert(m.items, index, item)
	m.activeIndex = index
}

func (m *Model) SetItem(index int, item ListItem) {
	if index < 0 || index >= len(m.items) {
		return
	}
	m.sizeItem(item)
	m.items[index] = item
}

func (m *Model) Selecting() bool {
	return m.selecting()
}

func (m *Model) selecting() bool {
	return m.selectionStart >= 0
}

func (m *Model) EnableSelectionMode() {
	if !m.hasValidActiveIndex() {
		return
	}
	m.selectionStart = m.activeIndex
	m.selectionEnd = m.activeIndex
}

func (m *Model) SelectAll() {
	if len(m.items) == 0 {
		return
	}
	m.selectionStart = 0
	m.selectionEnd = len(m.items) - 1
	m.activeIndex = m.selectionEnd
}

func (m *Model) DisableSelectionMode() {
	m.selectionStart = -1
	m.selectionEnd = -1
}

func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	for _, item := range m.items {
		m.sizeItem(item)
	}
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) Selection() []ListItem {
	if len(m.items) == 0 || !m.hasValidActiveIndex() {
		return nil
	}

	var selection []ListItem
	if !m.selecting() {
		selection = append(selection, m.ActiveItem())
		return selection
	}

	m.sortSelection()
	for _, item := range m.items[m.selectionStart : m.selectionEnd+1] {
		selection = append(selection, item)
	}
	return selection
}

func (m *Model) DeleteSelection() {
	if len(m.items) == 0 || !m.hasValidActiveIndex() {
		return
	}
	if m.selecting() {
		m.sortSelection()
		m.items = slices.Delete(m.items, m.selectionStart, m.selectionEnd+1)
		m.activeIndex = m.selectionStart
		m.DisableSelectionMode()
	} else {
		m.items = slices.Delete(m.items, m.activeIndex, m.activeIndex+1)
	}
	m.SetActiveIndex(m.activeIndex)
}

func (m *Model) SetActiveIndex(i int) {
	switch {
	case len(m.items) == 0:
		m.activeIndex = -1
	case i < 0:
		m.activeIndex = 0
	case i >= len(m.items):
		m.activeIndex = len(m.items) - 1
	default:
		m.activeIndex = i
	}
}

func (m *Model) SelectWhere(match func(ListItem) bool) bool {
	for i, item := range m.items {
		if match(item) {
			m.activeIndex = i
			return true
		}
	}
	return false
}

func (m *Model) sortSelection() {
	if m.selectionStart > m.selectionEnd {
		m.selectionStart, m.selectionEnd = m.selectionEnd, m.selectionStart
	}
}

func (m *Model) ItemsLen() int {
	return len(m.items)
}

func (m *Model) ActiveIndex() int {
	return m.activeIndex
}

func (m *Model) Orientation() Orientation {
	return m.orientation
}

func (m *Model) hasValidActiveIndex() bool {
	return len(m.items) > 0 && m.activeIndex >= 0 && m.activeIndex < len(m.items)
}

func (m *Model) DeleteWhere(match func(ListItem) bool) {
	for i, item := range m.items {
		if match(item) {
			m.items = slices.Delete(m.items, i, i+1)
			if m.activeIndex >= len(m.items) {
				m.activeIndex = len(m.items) - 1
			}
			if m.activeIndex < 0 {
				m.activeIndex = 0
			}
			break
		}
	}
}

func (m *Model) ActiveItem() ListItem {
	if !m.hasValidActiveIndex() {
		return nil
	}
	return m.items[m.activeIndex]
}

func (m *Model) horizontalScrollView() string {
	var s string
	for i := 0; i < m.itemsLen(); i++ {
		if i == m.ActiveIndex() {
			s += m.styles.Indicator.Render("●")
		} else {
			s += m.styles.Indicator.Render("○")
		}
	}
	return s
}

func (m *Model) verticalScrollView() string {
	indicatorHeight, indicatorPosition := 0, 0
	if m.itemsLen() > 0 {
		indicatorHeight = max(1, m.height/m.itemsLen())
		indicatorPosition = (m.ActiveIndex() * m.height) / m.itemsLen()
	}

	sections := make([]string, max(0, m.height))
	for i := range sections {
		sections[i] = " "
		if i >= indicatorPosition && i < indicatorPosition+indicatorHeight {
			sections[i] = m.styles.Indicator.Render("┃")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
