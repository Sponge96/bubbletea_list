package list

import (
	"iter"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	scrollBar      bool
	fillWidth      bool
	fade           *Fade
	paged          bool
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

// Option configures a Model at construction. Options are applied in order, so
// the last one for a setting wins.
type Option func(*Model)

func WithOrientation(o Orientation) Option { return func(m *Model) { m.orientation = o } }
func WithStyles(s Styles) Option           { return func(m *Model) { m.SetStyles(s) } }
func WithFade(f Fade) Option               { return func(m *Model) { m.SetFade(f) } }
func WithScrollBar(on bool) Option         { return func(m *Model) { m.SetScrollBar(on) } }
func WithFillWidth(on bool) Option         { return func(m *Model) { m.SetFillWidth(on) } }
func WithPaged(on bool) Option             { return func(m *Model) { m.SetPaged(on) } }

// New defaults to a vertical list with zero Styles, no fade and size 0x0; use
// SetSize to size it.
func New(items []ListItem, opts ...Option) *Model {
	m := &Model{
		items:          slices.Clone(items),
		activeIndex:    -1,
		selectionStart: -1,
	}
	for _, opt := range opts {
		opt(m)
	}
	for _, item := range m.items {
		m.sizeItem(item)
	}
	m.normalize()
	return m
}

func (m *Model) normalize() {
	if len(m.items) == 0 {
		m.activeIndex, m.selectionStart = -1, -1
		return
	}
	last := len(m.items) - 1
	m.activeIndex = min(max(m.activeIndex, 0), last)
	if m.Selecting() {
		m.selectionStart = min(max(m.selectionStart, 0), last)
	}
}

func (m *Model) selectionRange() (lo, hi int) {
	return min(m.selectionStart, m.activeIndex), max(m.selectionStart, m.activeIndex)
}

func (m *Model) Items() []ListItem {
	return slices.Clone(m.items)
}

// SetItems replaces the list's items and is the way to rebuild a list while
// keeping the user's position: the cursor and any selection are kept, clamped
// to the new length.
func (m *Model) SetItems(items []ListItem) {
	m.items = slices.Clone(items)
	for _, item := range m.items {
		m.sizeItem(item)
	}
	m.normalize()
}

// All iterates the list's items in order without copying them. SetItem during
// iteration is fine; inserting or deleting items during iteration is undefined.
func (m *Model) All() iter.Seq2[int, ListItem] {
	return func(yield func(int, ListItem) bool) {
		for i, item := range m.items {
			if !yield(i, item) {
				return
			}
		}
	}
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
	return tea.Batch(cmds...)
}

// Update sends key and paste messages to the active item only, so inactive
// items never receive keystrokes, and every other message to all items. Mouse
// messages stay broadcast because the list has no hit-testing to tell which
// item was clicked.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg, tea.PasteMsg:
		if item := m.ActiveItem(); item != nil {
			return item.Update(msg)
		}
		return nil
	}

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
		if m.scrollBar && m.orientation == Vertical {
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
	if !m.fillWidth || m.orientation != Vertical || m.width <= 0 {
		return style
	}
	w := m.width
	if m.scrollBar {
		w--
	}
	return style.Width(max(w, 1))
}

func (m *Model) renderActiveItem() string {
	if !m.hasValidActiveIndex() {
		return ""
	}
	if m.Selecting() {
		return m.fill(m.styles.SelectedActive).Render(m.items[m.activeIndex].View())
	}
	return m.fill(m.styles.Active).Render(m.items[m.activeIndex].View())
}

func (m *Model) renderItem(i int) string {
	if m.Selecting() {
		if lo, hi := m.selectionRange(); lo <= i && i <= hi {
			return m.fill(m.styles.Selected).Render(m.items[i].View())
		}
	}
	view := m.items[i].View()
	if m.fade != nil {
		view = Dim(view, *m.fade)
	}
	return m.fill(m.styles.Item).Render(view)
}

func (m *Model) renderSections(maxSize int, sizeFunc func(string) int) []string {
	var sections []string
	var item string
	remainingSpace := maxSize

	item = m.renderActiveItem()
	itemSize := sizeFunc(item)
	remainingSpace -= itemSize
	sections = append(sections, item)

	for i := m.activeIndex - 1; i >= 0 && !m.paged; i-- {
		item = m.renderItem(i)
		itemSize := sizeFunc(item)
		if itemSize <= remainingSpace {
			remainingSpace -= itemSize
			sections = slices.Insert(sections, 0, item)
		} else {
			break
		}
	}

	for i := m.activeIndex + 1; i < len(m.items) && !m.paged; i++ {
		item = m.renderItem(i)
		itemSize := sizeFunc(item)
		if itemSize <= remainingSpace {
			remainingSpace -= itemSize
			sections = append(sections, item)
		} else {
			break
		}
	}

	return sections
}

func (m *Model) renderVerticalSections() []string {
	return m.renderSections(m.height, lipgloss.Height)
}

func (m *Model) verticalView() string {
	sections := m.renderVerticalSections()
	mainView := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.scrollBar {
		return lipgloss.JoinHorizontal(lipgloss.Left, mainView, m.verticalScrollView())
	}
	return mainView
}

func (m *Model) renderHorizontalSections() []string {
	return m.renderSections(m.width, lipgloss.Width)
}

func (m *Model) horizontalView() string {
	sections := m.renderHorizontalSections()
	mainView := lipgloss.JoinHorizontal(lipgloss.Left, sections...)
	if m.paged {
		mainView = lipgloss.NewStyle().Height(lipgloss.Height(m.renderActiveItem())).Render(mainView)
	} else {
		mainView = lipgloss.NewStyle().Height(m.tallestItemHeight()).Render(mainView)
	}
	if m.scrollBar {
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
	n := len(m.items)
	if n == 0 {
		return
	}
	if m.activeIndex+step >= n {
		m.GotoEndItem()
		return
	}
	m.activeIndex += step
	m.normalize()
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
	m.normalize()
}

func (m *Model) GotoStartItem() {
	m.activeIndex = 0
	m.normalize()
}

func (m *Model) GotoEndItem() {
	m.activeIndex = len(m.items) - 1
	m.normalize()
}

func (m *Model) ClearItems() {
	m.items = nil
	m.normalize()
}

// InsertItem inserts item at index, clamped to the list. The cursor and any
// selection keep pointing at the same items; inserting into an empty list puts
// the cursor on the new item.
func (m *Model) InsertItem(item ListItem, index int) {
	if index > len(m.items) {
		index = len(m.items)
	}
	if index < 0 {
		index = 0
	}
	m.sizeItem(item)
	m.items = slices.Insert(m.items, index, item)
	if m.activeIndex >= index {
		m.activeIndex++
	}
	if m.Selecting() && m.selectionStart >= index {
		m.selectionStart++
	}
	m.normalize()
}

func (m *Model) SetItem(index int, item ListItem) {
	if index < 0 || index >= len(m.items) {
		return
	}
	m.sizeItem(item)
	m.items[index] = item
}

func (m *Model) Selecting() bool {
	return m.selectionStart >= 0
}

func (m *Model) EnableSelectionMode() {
	if !m.hasValidActiveIndex() {
		return
	}
	m.selectionStart = m.activeIndex
}

func (m *Model) SelectAll() {
	if len(m.items) == 0 {
		return
	}
	m.selectionStart = 0
	m.activeIndex = len(m.items) - 1
}

func (m *Model) DisableSelectionMode() {
	m.selectionStart = -1
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
	if !m.Selecting() {
		selection = append(selection, m.ActiveItem())
		return selection
	}

	lo, hi := m.selectionRange()
	return slices.Clone(m.items[lo : hi+1])
}

func (m *Model) DeleteSelection() {
	if len(m.items) == 0 || !m.hasValidActiveIndex() {
		return
	}
	if m.Selecting() {
		lo, hi := m.selectionRange()
		m.items = slices.Delete(m.items, lo, hi+1)
		m.activeIndex = lo
		m.DisableSelectionMode()
	} else {
		m.items = slices.Delete(m.items, m.activeIndex, m.activeIndex+1)
	}
	m.normalize()
}

func (m *Model) SetActiveIndex(i int) {
	m.activeIndex = i
	m.normalize()
}

func (m *Model) SelectWhere(match func(ListItem) bool) bool {
	for i, item := range m.items {
		if match(item) {
			m.activeIndex = i
			m.normalize()
			return true
		}
	}
	return false
}

func (m *Model) Len() int {
	return len(m.items)
}

func (m *Model) ActiveIndex() int {
	return m.activeIndex
}

func (m *Model) Orientation() Orientation {
	return m.orientation
}

func (m *Model) SetScrollBar(on bool) { m.scrollBar = on }
func (m *Model) ScrollBar() bool      { return m.scrollBar }

func (m *Model) SetFillWidth(on bool) { m.fillWidth = on }
func (m *Model) FillWidth() bool      { return m.fillWidth }

func (m *Model) SetPaged(on bool) { m.paged = on }
func (m *Model) Paged() bool      { return m.paged }

func (m *Model) SetStyles(s Styles) { m.styles = s }
func (m *Model) Styles() Styles     { return m.styles }

// SetFade makes inactive items fade according to f.
func (m *Model) SetFade(f Fade) { m.fade = &f }

func (m *Model) ClearFade() { m.fade = nil }

func (m *Model) Fade() (Fade, bool) {
	if m.fade == nil {
		return Fade{}, false
	}
	return *m.fade, true
}

func (m *Model) hasValidActiveIndex() bool {
	return len(m.items) > 0 && m.activeIndex >= 0 && m.activeIndex < len(m.items)
}

// DeleteWhere removes only the first item for which match returns true. The
// cursor and any selection keep following the same items. If the cursor's own
// item goes, focus moves to the item after it, or the one before it when the
// cursor was at the end of the list or at the high end of a selection. A
// selection whose items are all deleted is cleared.
func (m *Model) DeleteWhere(match func(ListItem) bool) {
	d := slices.IndexFunc(m.items, match)
	if d < 0 {
		return
	}
	m.items = slices.Delete(m.items, d, d+1)
	if !m.Selecting() {
		if d < m.activeIndex {
			m.activeIndex--
		}
		m.normalize()
		return
	}

	lo, hi := m.selectionRange()
	cursorIsHigh := m.activeIndex >= m.selectionStart
	switch {
	case d < lo:
		lo--
		hi--
	case d <= hi:
		hi--
	}
	switch {
	case hi < lo:
		m.selectionStart = -1
		m.activeIndex = d
	case cursorIsHigh:
		m.selectionStart, m.activeIndex = lo, hi
	default:
		m.selectionStart, m.activeIndex = hi, lo
	}
	m.normalize()
}

func (m *Model) ActiveItem() ListItem {
	if !m.hasValidActiveIndex() {
		return nil
	}
	return m.items[m.activeIndex]
}

func (m *Model) horizontalScrollView() string {
	var s string
	for i := 0; i < m.Len(); i++ {
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
	if m.Len() > 0 {
		indicatorHeight = max(1, m.height/m.Len())
		indicatorPosition = (m.ActiveIndex() * m.height) / m.Len()
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
