package list

import (
	"image/color"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type Orientation int

const (
	Vertical Orientation = iota
	Horizontal
)

type Model struct {
	Items             []ListItem
	orientation       Orientation
	width             int
	height            int
	activeIndex       int
	itemStyle         lipgloss.Style
	activeItemStyle   lipgloss.Style
	selectedItemStyle lipgloss.Style
	selectionStart    int
	selectionEnd      int
	ScrollBar         bool
	FitActiveHeight   bool
	// ActiveOnly shows just the active item of a horizontal list, not its neighbours.
	ActiveOnly bool
}

type ListItem interface {
	Init() tea.Cmd
	View() string
	Update(tea.Msg) tea.Cmd
	SetSize(w, h int)
	GetID() int64
}

func New(items []ListItem, orientation Orientation, w, h int, itemStyle, activeItemStyle, selectionStyle lipgloss.Style) *Model {
	return &Model{
		Items:             items,
		orientation:       orientation,
		width:             w,
		height:            h,
		activeIndex:       -1,
		selectionStart:    -1,
		selectionEnd:      -1,
		itemStyle:         itemStyle,
		activeItemStyle:   activeItemStyle,
		selectedItemStyle: selectionStyle,
	}
}
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, item := range m.Items {
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
	for _, item := range m.Items {
		cmd = item.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}
func (m *Model) View() string {
	if len(m.Items) == 0 {
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
func (m *Model) renderActiveItem() string {
	if !m.hasValidActiveIndex() {
		return ""
	}
	m.Items[m.activeIndex].SetSize(m.width, m.height)
	view := m.Items[m.activeIndex].View()
	if m.selecting() {
		active := m.activeItemStyle.Border(lipgloss.ThickBorder()).BorderForeground(m.selectedItemStyle.GetForeground())
		return active.UnsetBackground().Render(fill(view, m.selectedItemStyle.GetBackground()))
	}
	return m.activeItemStyle.UnsetBackground().Render(fill(view, m.activeItemStyle.GetBackground()))
}
func (m *Model) renderItem(i int) string {
	m.Items[i].SetSize(m.width, m.height)
	view := m.Items[i].View()
	if m.selecting() && ((m.selectionStart <= i && i <= m.selectionEnd) || (m.selectionEnd <= i && i <= m.selectionStart)) {
		selected := m.itemStyle.BorderForeground(m.selectedItemStyle.GetForeground())
		return selected.UnsetBackground().Render(fill(view, m.selectedItemStyle.GetBackground()))
	}
	return m.itemStyle.UnsetBackground().Render(fill(view, m.itemStyle.GetBackground()))
}

// fill paints bg behind a rendered block. Styled text inside resets the
// background as it ends, so it is switched back on after every reset, and
// short lines are padded so the fill is a clean rectangle.
func fill(view string, bg color.Color) string {
	if bg == nil {
		return view
	}
	if _, none := bg.(lipgloss.NoColor); none {
		return view
	}
	on := ansi.NewStyle().BackgroundColor(bg).String()
	lines := strings.Split(view, "\n")
	width := 0
	for _, l := range lines {
		width = max(width, ansi.StringWidth(l))
	}
	for i, l := range lines {
		l = strings.ReplaceAll(l, "\x1b[m", "\x1b[m"+on)
		l = strings.ReplaceAll(l, "\x1b[0m", "\x1b[0m"+on)
		pad := strings.Repeat(" ", width-ansi.StringWidth(l))
		lines[i] = on + l + pad + "\x1b[m"
	}
	return strings.Join(lines, "\n")
}
func (m *Model) verticalView() string {
	var sections []string
	var item string

	remainingSpace := m.height

	item = m.renderActiveItem()
	itemSize := lipgloss.Height(item)
	remainingSpace -= itemSize
	sections = append(sections, item)

	// Render items before activeIndex
	for i := m.activeIndex - 1; i >= 0; i-- {
		item = m.renderItem(i)
		itemSize := lipgloss.Height(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = slices.Insert(sections, 0, item)
		} else {
			break
		}
	}

	// Render items after activeIndex
	for i := m.activeIndex + 1; i < len(m.Items); i++ {
		item = m.renderItem(i)
		itemSize := lipgloss.Height(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = append(sections, item)
		} else {
			break
		}
	}
	mainView := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.ScrollBar {
		return lipgloss.JoinHorizontal(lipgloss.Left, mainView, lipgloss.NewStyle().Padding(0, lipgloss.Height(mainView)/2-lipgloss.Height(m.verticalScrollView())/2).Render(m.verticalScrollView()))
	}

	return mainView
}
func (m *Model) horizontalView() string {
	var sections []string
	var item string

	remainingSpace := m.width

	item = m.renderActiveItem()
	itemSize := lipgloss.Width(item)
	remainingSpace -= itemSize
	sections = append(sections, item)

	// Render items before activeIndex
	for i := m.activeIndex - 1; i >= 0 && !m.ActiveOnly; i-- {
		item = m.renderItem(i)
		itemSize := lipgloss.Width(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = slices.Insert(sections, 0, item)
		} else {
			break
		}
	}

	// Render items after activeIndex
	for i := m.activeIndex + 1; i < len(m.Items) && !m.ActiveOnly; i++ {
		item = m.renderItem(i)
		itemSize := lipgloss.Width(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = append(sections, item)
		} else {
			break
		}
	}

	main_view := lipgloss.JoinHorizontal(lipgloss.Left, sections...)
	if m.FitActiveHeight {
		main_view = lipgloss.NewStyle().Height(lipgloss.Height(m.renderActiveItem())).Render(main_view)
	} else {
		main_view = lipgloss.NewStyle().Height(m.tallestItemHeight()).Render(main_view)
	}
	if m.ScrollBar {
		dots := lipgloss.PlaceHorizontal(lipgloss.Width(main_view), lipgloss.Center, m.horizontalScrollView())
		return lipgloss.JoinVertical(lipgloss.Left, main_view, dots)
	}
	return lipgloss.JoinVertical(lipgloss.Left, main_view)
}

func (m *Model) tallestItemHeight() int {
	tallest := 0
	for i := range m.Items {
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
	len := len(m.Items)
	if len == 0 {
		return
	}
	// Exceeded length of Items, sets ActiveIndex to final item instead.
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
	if len(m.Items) == 0 {
		return
	}
	// Went into - index, sets ActiveIndex to first item instead.
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
	if len(m.Items) == 0 {
		m.activeIndex = -1
		return
	}
	m.activeIndex = 0
	if m.selecting() {
		m.selectionEnd = 0
	}
}
func (m *Model) GotoEndItem() {
	if len(m.Items) == 0 {
		m.activeIndex = -1
		return
	}
	m.activeIndex = len(m.Items) - 1
	if m.selecting() {
		m.selectionEnd = m.activeIndex
	}
}
func (m *Model) ClearItems() {
	m.Items = nil
	m.activeIndex = -1
}
func (m *Model) InsertItem(item ListItem, index int) {
	if index > len(m.Items) {
		index = len(m.Items)
	}
	if index < 0 {
		index = 0
	}
	m.Items = slices.Insert(m.Items, index, item)
	m.activeIndex = index
}
func (m *Model) Selecting() bool {
	return m.selecting()
}
func (m *Model) selecting() bool {
	return m.selectionStart >= 0
}
func (m *Model) EnabledSelectionMode() {
	if !m.hasValidActiveIndex() {
		return
	}
	m.selectionStart = m.activeIndex
	m.selectionEnd = m.activeIndex
}
func (m *Model) DisableSelectionMode() {
	m.selectionStart = -1
	m.selectionEnd = -1
}
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
}
func (m *Model) ActiveItemID() int64 {
	if !m.hasValidActiveIndex() {
		return -1
	}
	return m.Items[m.activeIndex].GetID()
}
func (m *Model) GetSelection() []ListItem {
	if len(m.Items) == 0 || !m.hasValidActiveIndex() {
		return nil
	}

	var selection []ListItem
	if !m.selecting() {
		selection = append(selection, m.ActiveItem())
		return selection
	}

	m.sortSelection()
	for _, item := range m.Items[m.selectionStart : m.selectionEnd+1] {
		selection = append(selection, item)
	}
	return selection
}
func (m *Model) DeleteSelection() {
	if len(m.Items) == 0 || !m.hasValidActiveIndex() {
		return
	}
	if m.selecting() {
		m.sortSelection()
		m.Items = slices.Delete(m.Items, m.selectionStart, m.selectionEnd+1)
		m.activeIndex = m.selectionStart
		m.DisableSelectionMode()
	} else {
		m.Items = slices.Delete(m.Items, m.activeIndex, m.activeIndex+1)
	}
	m.SetActiveIndex(m.activeIndex)
}

func (m *Model) SetActiveIndex(i int) {
	switch {
	case len(m.Items) == 0:
		m.activeIndex = -1
	case i < 0:
		m.activeIndex = 0
	case i >= len(m.Items):
		m.activeIndex = len(m.Items) - 1
	default:
		m.activeIndex = i
	}
}

func (m *Model) SelectItemByID(id int64) bool {
	for i, item := range m.Items {
		if item.GetID() == id {
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
	return len(m.Items)
}
func (m *Model) ActiveIndex() int {
	return m.activeIndex
}
func (m *Model) hasValidActiveIndex() bool {
	return len(m.Items) > 0 && m.activeIndex >= 0 && m.activeIndex < len(m.Items)
}
func (m *Model) DeleteItemByID(id int64) {
	for i, item := range m.Items {
		if item.GetID() == id {
			m.Items = slices.Delete(m.Items, i, i+1)
			if m.activeIndex >= len(m.Items) {
				m.activeIndex = len(m.Items) - 1
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
	return m.Items[m.activeIndex]
}
func (m *Model) horizontalScrollView() string {
	var s string
	for i := 0; i < m.ItemsLen(); i++ {
		if i == m.ActiveIndex() {
			s += "●"
		} else {
			s += "○"
		}
	}
	return s
}

func (m *Model) verticalScrollView() string {
	if m.ItemsLen() == 0 {
		return ""
	}

	indicatorHeight := max(1, m.height/m.ItemsLen())
	indicatorPosition := (m.ActiveIndex() * m.height) / m.ItemsLen()

	var sections []string
	for i := 0; i < m.height; i++ {
		if i >= indicatorPosition && i < indicatorPosition+indicatorHeight {
			sections = append(sections, "┃")
		} else {
			sections = append(sections, "")
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
