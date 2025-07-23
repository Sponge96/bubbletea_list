package list

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"slices"
)

type Orientation int

const (
	Vertical Orientation = iota
	Horizontal
)

type Model struct {
	Items             []Item
	orientation       Orientation
	width             int
	height            int
	activeIndex       int
	itemStyle         lipgloss.Style
	activeItemStyle   lipgloss.Style
	selectedItemStyle lipgloss.Style
	selectionMode     bool
	selectionStart    int
	selectionEnd      int
}

type Item interface {
	View() string
	Update(tea.Msg) tea.Cmd
	SetSize(w, h int)
	GetID() int64
	ToggleExpand()
}

func New(items []Item, orientation Orientation, w, h int, itemStyle lipgloss.Style, activeItemStyle lipgloss.Style) *Model {
	return &Model{
		Items:             items,
		orientation:       orientation,
		width:             w,
		height:            h,
		activeIndex:       0,
		selectionMode:     false,
		selectionStart:    0,
		selectionEnd:      0,
		itemStyle:         itemStyle,
		activeItemStyle:   activeItemStyle,
		selectedItemStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("22")),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// simply propergates the update down to items
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
	}

	return ""
}

func (m *Model) renderActiveItem() string {
	if !m.hasValidActiveIndex() {
		return ""
	}

	m.Items[m.activeIndex].SetSize(m.width, m.height)
	if m.selectionMode {
		return m.activeItemStyle.Render(m.selectedItemStyle.Render(m.Items[m.activeIndex].View()))
	}
	return m.activeItemStyle.Render(m.Items[m.activeIndex].View())
}

func (m *Model) renderItem(i int) string {
	m.Items[i].SetSize(m.width, m.height)
	if m.selectionMode && ((m.selectionStart <= i && i <= m.selectionEnd) || (m.selectionEnd <= i && i <= m.selectionStart)) {
		return m.itemStyle.Render(m.selectedItemStyle.Render(m.Items[i].View()))
	}
	return m.itemStyle.Render(m.Items[i].View())
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

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
	for i := m.activeIndex - 1; i >= 0; i-- {
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
	for i := m.activeIndex + 1; i < len(m.Items); i++ {
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
	scroll := m.scrollView()
	return lipgloss.JoinVertical(lipgloss.Left, main_view, lipgloss.NewStyle().Padding(0, lipgloss.Width(main_view)/2-lipgloss.Width(scroll)/2).Render(scroll))
}

func (m *Model) scrollView() string {
	var s string
	for i := 0; i < m.ItemsLen(); i++ {
		if i == m.ActiveIndex() {
			s += "●"
			continue
		}
		s += "○"
	}
	return s
}

// Movement
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
	if m.selectionMode {
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
	if m.selectionMode {
		m.selectionEnd = m.activeIndex
	}
}

func (m *Model) GotoStartItem() {
	if len(m.Items) == 0 {
		m.activeIndex = -1
		return
	}
	m.activeIndex = 0
	if m.selectionMode {
		m.selectionEnd = 0
	}
}

func (m *Model) GotoEndItem() {
	if len(m.Items) == 0 {
		m.activeIndex = -1
		return
	}
	m.activeIndex = len(m.Items) - 1
	if m.selectionMode {
		m.selectionEnd = m.activeIndex
	}
}

func (m *Model) ClearItems() {
	m.Items = nil
	m.activeIndex = -1
}

func (m *Model) InsertItem(item Item, index int) {
	if index > len(m.Items) {
		index = len(m.Items)
	}

	m.Items = slices.Insert(m.Items, index, item)
	if m.activeIndex < 0 {
		m.activeIndex = 0
	}
}

func (m *Model) EnabledSelectionMode() {
	m.selectionStart = m.activeIndex
	m.selectionEnd = m.activeIndex
	m.selectionMode = true
}

func (m *Model) DisableSelectionMode() {
	m.selectionStart = 0
	m.selectionEnd = 0
	m.selectionMode = false
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

func (m *Model) GetSelection() []int64 {
	if len(m.Items) == 0 || !m.hasValidActiveIndex() {
		return nil
	}

	var selection []int64
	if !m.selectionMode {
		selection = append(selection, m.ActiveItemID())
		return selection
	}

	m.sortSelection()
	for _, item := range m.Items[m.selectionStart : m.selectionEnd+1] {
		selection = append(selection, item.GetID())
	}
	return selection
}

func (m *Model) DeleteSelection() {
	if len(m.Items) == 0 || !m.hasValidActiveIndex() {
		return
	}
	if m.selectionMode {
		m.sortSelection()
		m.Items = slices.Delete(m.Items, m.selectionStart, m.selectionEnd+1)
	} else {
		m.Items = slices.Delete(m.Items, m.activeIndex, m.activeIndex+1)
	}
	m.activeIndex = 0
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
