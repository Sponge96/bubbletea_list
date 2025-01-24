package list

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"slices"
)

type Orientation int

const (
	Vertical Orientation = iota
	Horizontal
)

type Model struct {
	items             []Item
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
	GetID() int64
	View() string
	SetSize(w, h int)
	ToggleExpand()
	Update(tea.Msg) tea.Cmd
}

func New(items []Item, orientation Orientation, w, h int, itemStyle lipgloss.Style, activeItemStyle lipgloss.Style) *Model {
	return &Model{
		items:             items,
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
	for _, item := range m.items {
		cmd = item.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) View() string {
	// Nothing to display
	if len(m.items) == 0 {
		return ""
	}

	if m.orientation == Horizontal {
		return m.horizontalView()
	}

	if m.orientation == Vertical {
		return m.verticalView()
	}

	return ""
}

func (m *Model) renderActiveItem() string {
	m.items[m.activeIndex].SetSize(m.width, m.height)
	if m.selectionMode {
		return m.activeItemStyle.Render(m.selectedItemStyle.Render(m.items[m.activeIndex].View()))
	}
	return m.activeItemStyle.Render(m.items[m.activeIndex].View())
}

func (m *Model) renderItem(i int) string {
	m.items[i].SetSize(m.width, m.height)
	if m.selectionMode && ((m.selectionStart <= i && i <= m.selectionEnd) || (m.selectionEnd <= i && i <= m.selectionStart)) {
		return m.itemStyle.Render(m.selectedItemStyle.Render(m.items[i].View()))
	}
	return m.itemStyle.Render(m.items[i].View())
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
	for i := m.activeIndex + 1; i < len(m.items); i++ {
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
	for i := m.activeIndex + 1; i < len(m.items); i++ {
		item = m.renderItem(i)
		itemSize := lipgloss.Width(item)
		if itemSize < remainingSpace {
			remainingSpace -= itemSize
			sections = append(sections, item)
		} else {
			break
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, sections...)
}

// Movement
func (m *Model) IncrementItem(step int) {
	len := len(m.items)
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
	if len(m.items) == 0 {
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
	m.activeIndex = 0
	if m.selectionMode {
		m.selectionEnd = 0
	}
}

func (m *Model) GotoEndItem() {
	m.activeIndex = len(m.items) - 1
	if m.selectionMode {
		m.selectionEnd = m.activeIndex
	}
}

func (m *Model) ClearItems() {
	m.items = nil
}

func (m *Model) InsertItem(item Item) {
	m.items = slices.Insert(m.items, 0, item)
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
	return m.items[m.activeIndex].GetID()
}

func (m *Model) GetSelection() []int64 {
	if len(m.items) == 0 {
		return nil
	}

	var selection []int64
	if !m.selectionMode {
		selection = append(selection, m.ActiveItemID())
		return selection
	}

	m.sortSelection()
	for _, item := range m.items[m.selectionStart:m.selectionEnd] {
		selection = append(selection, item.GetID())
	}
	return selection
}

func (m *Model) DeleteSelection() {
	if len(m.items) == 0 {
		return
	}
	// if selection mode active deletes selection otherwise delete activeID
	if m.selectionMode {
		// swap start/end if start >
		m.sortSelection()
		m.items = slices.Delete(m.items, m.selectionStart, m.selectionEnd+1)
	} else {
		m.items = slices.Delete(m.items, m.activeIndex, m.activeIndex+1)
	}
	m.activeIndex = 0
}

func (m *Model) sortSelection() {
	if m.selectionStart > m.selectionEnd {
		m.selectionStart, m.selectionEnd = m.selectionEnd, m.selectionStart
	}
}
