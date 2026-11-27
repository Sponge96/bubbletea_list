package list

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestDimFadesColouredText(t *testing.T) {
	out := Dim("\x1b[38;2;255;121;198mpink\x1b[m", testFade)

	if strings.Contains(out, "38;2;255;121;198") {
		t.Errorf("the original colour should be gone: %q", out)
	}
	r, g, b := blend(color.RGBA{R: 255, G: 121, B: 198, A: 255}, testFade)
	if want := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b); !strings.Contains(out, want) {
		t.Errorf("the colour should be blended towards the background, want %q in %q", want, out)
	}
	if ansi.Strip(out) != "pink" {
		t.Errorf("the text must be unchanged, got %q", ansi.Strip(out))
	}
}

func TestDimGivesPlainTextAFadedColourAfterResets(t *testing.T) {
	for _, reset := range []string{"\x1b[m", "\x1b[0m"} {
		out := Dim("\x1b[38;2;255;0;0ma"+reset+" plain", testFade)

		if !strings.Contains(out, reset+fadedForeground()+" plain") {
			t.Errorf("plain text after a %q reset should be faded too: %q", reset, out)
		}
		if !strings.HasPrefix(out, fadedForeground()) {
			t.Errorf("a line should start with the faded default colour: %q", out)
		}
	}
}

func TestDimLeavesBackgroundsAlone(t *testing.T) {
	out := Dim("\x1b[48;2;1;2;3mx\x1b[m", testFade)
	if !strings.Contains(out, "\x1b[48;2;1;2;3m") {
		t.Errorf("backgrounds are kept: %q", out)
	}
}

var testFade = Fade{
	Foreground: lipgloss.RGBColor{R: 0xf8, G: 0xf8, B: 0xf2},
	Background: lipgloss.RGBColor{R: 0x28, G: 0x2a, B: 0x36},
	Amount:     0.6,
}

func TestDimLeavesTheViewAloneWithoutBothColours(t *testing.T) {
	view := "\x1b[38;2;255;0;0mred\x1b[m plain"
	for name, f := range map[string]Fade{
		"no foreground": {Background: testFade.Background, Amount: 0.6},
		"no background": {Foreground: testFade.Foreground, Amount: 0.6},
	} {
		if got := Dim(view, f); got != view {
			t.Errorf("%s: the view should be unchanged, got %q", name, got)
		}
	}
}

func TestAListWithoutAFadeLeavesInactiveItemsBright(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	l := newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}}, Vertical, 20, 20, Styles{Item: item, Active: item})
	l.SetActiveIndex(0)

	if strings.Contains(l.View(), fadedForeground()) {
		t.Error("without a Fade no item should carry a faded colour")
	}
}

func fadedForeground() string {
	r, g, b := blend(testFade.Foreground, testFade)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func TestAnEmptyVerticalListKeepsItsScrollbarColumn(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	full := newTestList([]ListItem{&fillingItem{}}, Vertical, 20, 10, Styles{Item: item, Active: item})
	full.SetScrollBar(true)
	full.SetActiveIndex(0)
	empty := newTestList([]ListItem{}, Vertical, 20, 10, Styles{Item: item, Active: item})
	empty.SetScrollBar(true)

	if got, want := lipgloss.Width(empty.View()), lipgloss.Width(full.View()); got != want {
		t.Errorf("an empty list should be as wide as a full one (%d), got %d", want, got)
	}
	if got := lipgloss.Height(empty.View()); got != 10 {
		t.Errorf("an empty list should fill its 10 rows, got %d", got)
	}
	if strings.Contains(ansi.Strip(empty.View()), "┃") {
		t.Error("an empty list has nothing to scroll")
	}
}

func TestFillWidthStretchesVerticalItemsToTheListWidth(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	long := strings.Repeat("long title ", 20)

	for _, scroll := range []bool{true, false} {
		for _, selecting := range []bool{false, true} {
			l := newTestList([]ListItem{&textItem{"one"}, &textItem{long}, &textItem{"three"}}, Vertical, 30, 40, Styles{Item: item, Active: item})
			l.SetScrollBar(scroll)
			l.SetFillWidth(true)
			l.SetActiveIndex(0)
			if selecting {
				l.EnableSelectionMode()
				l.IncrementItem(1)
			}
			for i, line := range strings.Split(l.View(), "\n") {
				if got := lipgloss.Width(line); got != 30 {
					t.Errorf("scrollbar=%v selecting=%v line %d is %d wide, want 30: %q", scroll, selecting, i, got, ansi.Strip(line))
				}
			}
		}
	}
}

func TestAnEmptyFilledListIsAsWideAsAFullOne(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	full := newTestList([]ListItem{&textItem{"one"}}, Vertical, 20, 10, Styles{Item: item, Active: item})
	empty := newTestList([]ListItem{}, Vertical, 20, 10, Styles{Item: item, Active: item})
	for _, l := range []*Model{full, empty} {
		l.SetScrollBar(true)
		l.SetFillWidth(true)
	}
	full.SetActiveIndex(0)

	if got, want := lipgloss.Width(empty.View()), lipgloss.Width(full.View()); got != want {
		t.Errorf("an empty list should be as wide as a full one (%d), got %d", want, got)
	}
}

func TestFillWidthLeavesHorizontalListsAndUnsizedListsAlone(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	want := lipgloss.Width(item.Render("one"))

	horizontal := newTestList([]ListItem{&textItem{"one"}}, Horizontal, 30, 10, Styles{Item: item, Active: item})
	horizontal.SetFillWidth(true)
	horizontal.SetActiveIndex(0)
	if got := lipgloss.Width(horizontal.View()); got != want {
		t.Errorf("a horizontal list keeps its item width %d, got %d", want, got)
	}

	unsized := newTestList([]ListItem{&textItem{"one"}}, Vertical, 0, 10, Styles{Item: item, Active: item})
	unsized.SetFillWidth(true)
	unsized.SetActiveIndex(0)
	if got := lipgloss.Width(unsized.View()); got != want {
		t.Errorf("a zero-width list keeps its item width %d, got %d", want, got)
	}
}

type fillingItem struct{ width int }

func (f *fillingItem) Init() tea.Cmd          { return nil }
func (f *fillingItem) Update(tea.Msg) tea.Cmd { return nil }
func (f *fillingItem) View() string           { return lipgloss.NewStyle().Width(f.width).Render("one") }
func (f *fillingItem) SetSize(w, _ int)       { f.width = w }

type textItem struct{ text string }

func (t *textItem) Init() tea.Cmd          { return nil }
func (t *textItem) Update(tea.Msg) tea.Cmd { return nil }
func (t *textItem) View() string           { return t.text }

func TestOnlyTheActiveAndSelectedItemsAreBright(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	bright := lipgloss.NewStyle().Foreground(lipgloss.RGBColor{R: 255, G: 121, B: 198})
	l := newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, Styles{Item: item, Active: item, Selected: bright})
	l.SetFade(testFade)
	l.SetActiveIndex(0)

	faded := func(text string) bool {
		for _, line := range strings.Split(l.View(), "\n") {
			if strings.Contains(ansi.Strip(line), text) {
				return strings.Contains(line, fadedForeground())
			}
		}
		t.Fatalf("%q not rendered", text)
		return false
	}
	if faded("one") {
		t.Error("the active item should not be faded")
	}
	if !faded("two") || !faded("three") {
		t.Error("the other items should be faded")
	}

	l.EnableSelectionMode()
	l.IncrementItem(1)
	if faded("one") || faded("two") {
		t.Error("selected items should not be faded")
	}
	if !faded("three") {
		t.Error("unselected items should be faded")
	}
}

const redSGR = "38;2;255;0;0"

func lineWith(t *testing.T, view, text string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(ansi.Strip(line), text) {
			return line
		}
	}
	t.Fatalf("%q not rendered in %q", text, ansi.Strip(view))
	return ""
}

func TestSelectedRowsUseTheSelectedStyle(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.RGBColor{R: 255, G: 0, B: 0})
	newList := func() *Model {
		return newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, Styles{Selected: red, SelectedActive: red})
	}
	check := func(name string, l *Model) {
		view := l.View()
		for _, text := range []string{"one", "two"} {
			if !strings.Contains(lineWith(t, view, text), redSGR) {
				t.Errorf("%s: %q should be in the selected style", name, text)
			}
		}
		if strings.Contains(lineWith(t, view, "three"), redSGR) {
			t.Errorf("%s: \"three\" is outside the selection", name)
		}
	}

	down := newList()
	down.SetActiveIndex(0)
	down.EnableSelectionMode()
	down.IncrementItem(1)
	check("selecting down", down)

	up := newList()
	up.SetActiveIndex(1)
	up.EnableSelectionMode()
	up.DecrementItem(1)
	check("selecting up", up)
}

func TestTheActiveRowWhileSelectingUsesSelectedActive(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}}, Vertical, 20, 20,
		Styles{SelectedActive: lipgloss.NewStyle().Border(lipgloss.ThickBorder())})
	l.SetActiveIndex(0)

	if strings.Contains(l.View(), "┏") {
		t.Error("the thick border belongs to the selecting state only")
	}
	l.EnableSelectionMode()
	if !strings.Contains(l.View(), "┏") {
		t.Error("the active row should use SelectedActive while selecting")
	}
}

func TestIndicatorStyleColoursTheScrollbar(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.RGBColor{R: 255, G: 0, B: 0})
	items := func() []ListItem { return []ListItem{&textItem{"one"}, &textItem{"two"}} }

	styled := newTestList(items(), Vertical, 4, 4, Styles{Indicator: red})
	styled.SetScrollBar(true)
	styled.SetActiveIndex(0)
	if !strings.Contains(lineWith(t, styled.View(), "┃"), redSGR) {
		t.Error("the thumb should carry the Indicator colour")
	}

	plain := newTestList(items(), Vertical, 4, 4, Styles{})
	plain.SetScrollBar(true)
	plain.SetActiveIndex(0)
	if strings.Contains(lineWith(t, plain.View(), "┃"), "\x1b") {
		t.Error("a zero Indicator leaves the thumb unstyled")
	}
}

func isText(text string) func(ListItem) bool {
	return func(it ListItem) bool {
		ti, ok := it.(*textItem)
		return ok && ti.text == text
	}
}

func TestSelectWhereMovesTheCursorToTheFirstMatch(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"a"}, &textItem{"b"}, &textItem{"x"}, &textItem{"x"}}, Vertical, 20, 20, Styles{})
	l.SetActiveIndex(1)

	if !l.SelectWhere(isText("x")) {
		t.Fatal("expected a match")
	}
	if l.ActiveIndex() != 2 {
		t.Errorf("the cursor should be on the first match, got %d", l.ActiveIndex())
	}
	if l.SelectWhere(isText("missing")) {
		t.Error("no match should return false")
	}
	if l.ActiveIndex() != 2 {
		t.Errorf("no match should not move the cursor, got %d", l.ActiveIndex())
	}
}

func TestDeleteWhereRemovesOnlyTheFirstMatch(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"a"}, &textItem{"x"}, &textItem{"b"}, &textItem{"x"}}, Vertical, 20, 20, Styles{})

	l.DeleteWhere(isText("x"))

	var got []string
	for _, it := range l.Items() {
		got = append(got, it.(*textItem).text)
	}
	if strings.Join(got, ",") != "a,b,x" {
		t.Errorf("only the first match should go, got %v", got)
	}
}

func TestDeleteWhereKeepsTheCursorValid(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"a"}, &textItem{"b"}, &textItem{"c"}}, Vertical, 20, 20, Styles{})
	l.SetActiveIndex(2)
	l.DeleteWhere(isText("c"))
	if l.ActiveIndex() != 1 || l.ActiveItem().(*textItem).text != "b" {
		t.Errorf("deleting the active last item should move to the new last, got %d", l.ActiveIndex())
	}

	only := newTestList([]ListItem{&textItem{"a"}}, Vertical, 20, 20, Styles{})
	only.SetActiveIndex(0)
	only.DeleteWhere(isText("a"))
	if only.ActiveItem() != nil {
		t.Error("deleting the only item should leave no active item")
	}
}

func threeLineItems() []ListItem {
	return []ListItem{&textItem{"one"}, &textItem{"two\ntwo\ntwo"}, &textItem{"three\nthree"}}
}

func TestPagerShowsOnlyTheActiveItem(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Horizontal, 30, 10, Styles{})
	l.SetPaged(true)
	l.SetActiveIndex(1)

	view := ansi.Strip(l.View())
	if !strings.Contains(view, "two") {
		t.Errorf("the active item should show: %q", view)
	}
	if strings.Contains(view, "one") || strings.Contains(view, "three") {
		t.Errorf("only the active item should show: %q", view)
	}
}

func TestPagerIsAsTallAsTheActiveItem(t *testing.T) {
	l := newTestList(threeLineItems(), Horizontal, 30, 10, Styles{})
	l.SetPaged(true)

	for i, want := range []int{1, 3, 2} {
		l.SetActiveIndex(i)
		if got := lipgloss.Height(l.View()); got != want {
			t.Errorf("item %d: the pager should be %d lines tall, got %d", i, want, got)
		}
	}
}

func TestPagerShowsDotsOnlyWithAScrollBar(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Horizontal, 30, 10, Styles{})
	l.SetPaged(true)
	l.SetActiveIndex(0)

	if strings.Contains(ansi.Strip(l.View()), "●○○") {
		t.Error("no dots without a ScrollBar")
	}
	l.SetScrollBar(true)
	if !strings.Contains(ansi.Strip(l.View()), "●○○") {
		t.Errorf("dots expected with a ScrollBar: %q", ansi.Strip(l.View()))
	}
}

func TestHorizontalStillShowsNeighbours(t *testing.T) {
	l := newTestList(threeLineItems(), Horizontal, 30, 10, Styles{})
	l.SetActiveIndex(1)

	view := ansi.Strip(l.View())
	for _, text := range []string{"one", "two", "three"} {
		if !strings.Contains(view, text) {
			t.Errorf("%q should show in the strip: %q", text, view)
		}
	}
	l.SetActiveIndex(0)
	if got := lipgloss.Height(l.View()); got != 3 {
		t.Errorf("a horizontal strip is as tall as its tallest item, got %d", got)
	}
}

func TestEmptyListsRenderInEveryOrientation(t *testing.T) {
	for name, o := range map[string]Orientation{"vertical": Vertical, "horizontal": Horizontal} {
		l := newTestList([]ListItem{}, o, 20, 5, Styles{})
		l.SetScrollBar(true)
		got := l.View()
		if o == Vertical {
			if lipgloss.Height(got) != 5 {
				t.Errorf("%s: an empty vertical list keeps its column, got height %d", name, lipgloss.Height(got))
			}
		} else if got != "" {
			t.Errorf("%s: an empty list renders nothing, got %q", name, got)
		}
	}
}

type sizeRecorder struct {
	textItem
	calls         int
	width, height int
}

func (s *sizeRecorder) SetSize(w, h int) {
	s.calls++
	s.width, s.height = w, h
}

func TestViewDoesNotSizeItems(t *testing.T) {
	l := newTestList(nil, Vertical, 0, 0, Styles{})
	a, b := &sizeRecorder{textItem: textItem{"a"}}, &sizeRecorder{textItem: textItem{"b"}}
	l.InsertItem(a, 0)
	l.InsertItem(b, 1)
	l.SetSize(30, 10)

	if a.width != 30 || a.height != 10 || b.width != 30 || b.height != 10 {
		t.Fatalf("items should have been sized (30, 10), got (%d, %d) and (%d, %d)", a.width, a.height, b.width, b.height)
	}
	before := a.calls + b.calls
	l.View()
	l.View()
	if got := a.calls + b.calls; got != before {
		t.Errorf("View sized items: %d calls became %d", before, got)
	}
}

func TestSetSizeReachesTheItemsGivenToNew(t *testing.T) {
	a := &sizeRecorder{}
	l := New([]ListItem{a})
	l.SetSize(25, 7)

	if a.width != 25 || a.height != 7 {
		t.Errorf("want (25, 7), got (%d, %d)", a.width, a.height)
	}
}

func TestSetItemSizesTheReplacementAndKeepsTheCursor(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"a"}, &textItem{"b"}, &textItem{"c"}}, Vertical, 20, 5, Styles{})
	l.SetActiveIndex(2)
	fresh := &sizeRecorder{textItem: textItem{"fresh"}}

	l.SetItem(1, fresh)

	if l.ItemAt(1) != ListItem(fresh) {
		t.Error("the item at index 1 should be replaced")
	}
	if fresh.width != 20 || fresh.height != 5 {
		t.Errorf("replacement should be sized (20, 5), got (%d, %d)", fresh.width, fresh.height)
	}
	if l.ActiveIndex() != 2 {
		t.Errorf("the cursor should stay on 2, got %d", l.ActiveIndex())
	}
}

func TestSetItemIgnoresAnOutOfRangeIndex(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"a"}}, Vertical, 20, 5, Styles{})
	fresh := &sizeRecorder{}

	l.SetItem(-1, fresh)
	l.SetItem(1, fresh)

	if l.Len() != 1 || l.ItemAt(0).(*textItem).text != "a" {
		t.Error("the list should be unchanged")
	}
	if fresh.calls != 0 {
		t.Errorf("an ignored item should not be sized, got %d calls", fresh.calls)
	}
}

type bareItem struct{}

func (bareItem) Init() tea.Cmd          { return nil }
func (bareItem) Update(tea.Msg) tea.Cmd { return nil }
func (bareItem) View() string           { return "bare" }

func TestItemsWithoutSetSizeStillRender(t *testing.T) {
	l := newTestList([]ListItem{bareItem{}}, Vertical, 20, 5, Styles{})
	l.InsertItem(bareItem{}, 1)
	l.SetItem(0, bareItem{})
	l.SetSize(30, 6)
	l.SetActiveIndex(0)

	if !strings.Contains(l.View(), "bare") {
		t.Errorf("an item without SetSize should render, got %q", l.View())
	}
}

func TestSelectAllSelectsEveryItemAndParksTheCursorOnTheLast(t *testing.T) {
	l := newTestList([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, Styles{})
	l.SetActiveIndex(1)

	l.SelectAll()

	if !l.Selecting() {
		t.Error("select all should start a selection")
	}
	if got := len(l.Selection()); got != 3 {
		t.Errorf("selection has %d items, want 3", got)
	}
	if l.ActiveIndex() != 2 {
		t.Errorf("cursor on %d, want 2", l.ActiveIndex())
	}
}

func TestSelectAllOnAnEmptyListDoesNothing(t *testing.T) {
	l := newTestList(nil, Vertical, 20, 20, Styles{})

	l.SelectAll()

	if l.Selecting() {
		t.Error("an empty list has nothing to select")
	}
	if l.Selection() != nil {
		t.Errorf("selection = %v, want nil", l.Selection())
	}
}

func textList(names ...string) *Model {
	items := make([]ListItem, len(names))
	for i, name := range names {
		items[i] = &textItem{name}
	}
	return newTestList(items, Vertical, 20, 20, Styles{})
}

func texts(items []ListItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.(*textItem).text
	}
	return out
}

func TestClearItemsThenInsertDoesNotKeepAStaleSelection(t *testing.T) {
	l := textList("a", "b")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.IncrementItem(1)
	l.ClearItems()
	if l.ActiveIndex() != -1 || l.Selecting() || l.Selection() != nil {
		t.Fatalf("a cleared list has no cursor or selection: %d %v", l.ActiveIndex(), l.Selecting())
	}
	l.InsertItem(&textItem{"x"}, 0)
	if got := texts(l.Selection()); len(got) != 1 || got[0] != "x" || l.Selecting() {
		t.Errorf("want just the active item and no selection, got %v selecting=%v", got, l.Selecting())
	}
}

func TestNewPutsTheCursorOnTheFirstItem(t *testing.T) {
	if got := textList("a", "b").ActiveIndex(); got != 0 {
		t.Errorf("a list with items starts on 0, got %d", got)
	}
	if got := newTestList(nil, Vertical, 20, 20, Styles{}).ActiveIndex(); got != -1 {
		t.Errorf("an empty list has no cursor, got %d", got)
	}
}

func TestInitLeavesTheCursorAlone(t *testing.T) {
	l := textList("a", "b")
	l.SetActiveIndex(1)
	l.Init()
	if got := l.ActiveIndex(); got != 1 {
		t.Errorf("Init must not move the cursor, got %d", got)
	}
}

func TestNewCopiesTheSliceItIsGiven(t *testing.T) {
	original := &textItem{"a"}
	items := []ListItem{original, &textItem{"b"}}
	l := newTestList(items, Vertical, 20, 20, Styles{})
	items[0] = &textItem{"z"}
	if l.ItemAt(0) != ListItem(original) {
		t.Error("changing the caller's slice must not change the list")
	}
}

func TestDeleteWhereToEmptyLeavesTheCursorAtMinusOne(t *testing.T) {
	l := textList("a")
	l.DeleteWhere(isText("a"))
	if got := l.ActiveIndex(); got != -1 {
		t.Errorf("an emptied list has no cursor, got %d", got)
	}
}

func TestSetActiveIndexWhileSelectingExtendsTheSelection(t *testing.T) {
	l := textList("a", "b", "c")
	l.EnableSelectionMode()
	l.SetActiveIndex(2)
	if got := strings.Join(texts(l.Selection()), ","); got != "a,b,c" {
		t.Errorf("the selection should follow the cursor, got %s", got)
	}
}

func TestSelectWhereWhileSelectingExtendsTheSelection(t *testing.T) {
	l := textList("a", "b", "c")
	l.EnableSelectionMode()
	l.SelectWhere(isText("c"))
	if got := strings.Join(texts(l.Selection()), ","); got != "a,b,c" {
		t.Errorf("the selection should follow the cursor, got %s", got)
	}
}

func activeText(l *Model) string {
	if it := l.ActiveItem(); it != nil {
		return it.(*textItem).text
	}
	return ""
}

func selectedTexts(l *Model) string { return strings.Join(texts(l.Selection()), ",") }

func TestInsertBeforeTheCursorKeepsItOnTheSameItem(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(2)
	l.InsertItem(&textItem{"x"}, 0)
	if activeText(l) != "c" || l.ActiveIndex() != 3 {
		t.Errorf("cursor should follow c to index 3, got %q at %d", activeText(l), l.ActiveIndex())
	}
}

func TestInsertAtTheCursorPushesItDown(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(1)
	l.InsertItem(&textItem{"x"}, 1)
	if activeText(l) != "b" || l.ActiveIndex() != 2 {
		t.Errorf("cursor should follow b to index 2, got %q at %d", activeText(l), l.ActiveIndex())
	}
}

func TestInsertAfterTheCursorLeavesItAlone(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(0)
	l.InsertItem(&textItem{"x"}, 2)
	if l.ActiveIndex() != 0 {
		t.Errorf("cursor should stay on 0, got %d", l.ActiveIndex())
	}
}

func TestInsertIntoAnEmptyListPutsTheCursorOnIt(t *testing.T) {
	l := textList()
	l.InsertItem(&textItem{"x"}, 0)
	if l.ActiveIndex() != 0 {
		t.Errorf("cursor should be on 0, got %d", l.ActiveIndex())
	}
}

func TestInsertShiftsASelection(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.IncrementItem(1)
	l.InsertItem(&textItem{"x"}, 0)
	if got := selectedTexts(l); got != "a,b" || activeText(l) != "b" {
		t.Errorf("selection should still be a,b with the cursor on b, got %s on %q", got, activeText(l))
	}
}

func TestInsertWithAnOutOfRangeIndexIsClamped(t *testing.T) {
	l := textList("a", "b")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.InsertItem(&textItem{"x"}, -5)
	if got := strings.Join(texts(l.Items()), ","); got != "x,a,b" || selectedTexts(l) != "a" {
		t.Errorf("want x,a,b with a selected, got %s with %s selected", got, selectedTexts(l))
	}
	l.InsertItem(&textItem{"y"}, 99)
	if got := strings.Join(texts(l.Items()), ","); got != "x,a,b,y" || selectedTexts(l) != "a" {
		t.Errorf("want x,a,b,y with a selected, got %s with %s selected", got, selectedTexts(l))
	}
}

func TestDeleteBeforeTheCursorKeepsItOnTheSameItem(t *testing.T) {
	l := textList("a", "b", "c", "d")
	l.SetActiveIndex(2)
	l.DeleteWhere(isText("a"))
	if activeText(l) != "c" || l.ActiveIndex() != 1 {
		t.Errorf("cursor should follow c to index 1, got %q at %d", activeText(l), l.ActiveIndex())
	}
}

func TestDeleteTheCursorItemMovesFocusToTheNext(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(1)
	l.DeleteWhere(isText("b"))
	if activeText(l) != "c" || l.ActiveIndex() != 1 {
		t.Errorf("focus should move to c at index 1, got %q at %d", activeText(l), l.ActiveIndex())
	}
}

func TestDeleteInsideASelectionShrinksIt(t *testing.T) {
	l := textList("a", "b", "c", "d")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.IncrementItem(2)
	l.DeleteWhere(isText("b"))
	if got := selectedTexts(l); got != "a,c" || activeText(l) != "c" {
		t.Errorf("selection should be a,c with the cursor on c, got %s on %q", got, activeText(l))
	}
}

func TestDeletingTheCursorEndOfASelectionOnlyLeavesSelectedItems(t *testing.T) {
	l := textList("a", "b", "c", "d")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.IncrementItem(2)
	l.DeleteWhere(isText("c"))
	if got := selectedTexts(l); got != "a,b" || activeText(l) != "b" {
		t.Errorf("selection should be a,b with the cursor on b, got %s on %q", got, activeText(l))
	}
}

func TestDeletingEverySelectedItemClearsTheSelection(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(1)
	l.EnableSelectionMode()
	l.DeleteWhere(isText("b"))
	if l.Selecting() || activeText(l) != "c" {
		t.Errorf("selection should be cleared with the cursor on c, got selecting=%v on %q", l.Selecting(), activeText(l))
	}
}

func TestDeleteWhereWithNoMatchChangesNothing(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(1)
	l.EnableSelectionMode()
	l.IncrementItem(1)
	l.DeleteWhere(isText("zzz"))
	if got := strings.Join(texts(l.Items()), ","); got != "a,b,c" || l.ActiveIndex() != 2 || selectedTexts(l) != "b,c" {
		t.Errorf("nothing should change, got %s cursor %d selection %s", got, l.ActiveIndex(), selectedTexts(l))
	}
}

func TestDeleteSelectionRemovesTheRangeAndClearsIt(t *testing.T) {
	l := textList("a", "b", "c", "d")
	l.SetActiveIndex(1)
	l.EnableSelectionMode()
	l.IncrementItem(1)
	l.DeleteSelection()
	if got := strings.Join(texts(l.Items()), ","); got != "a,d" || activeText(l) != "d" || l.Selecting() {
		t.Errorf("want a,d with the cursor on d and no selection, got %s on %q selecting=%v", got, activeText(l), l.Selecting())
	}
}

func TestSetItemsReplacesAndSizesTheItems(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetSize(20, 5)
	a, b := &sizeRecorder{textItem: textItem{"x"}}, &sizeRecorder{textItem: textItem{"y"}}

	l.SetItems([]ListItem{a, b})

	if l.Len() != 2 || l.ItemAt(0) != ListItem(a) || l.ItemAt(1) != ListItem(b) {
		t.Fatalf("the list should hold exactly the new items, got %d", l.Len())
	}
	if a.width != 20 || a.height != 5 || b.width != 20 || b.height != 5 {
		t.Errorf("new items should be sized (20, 5), got (%d, %d) and (%d, %d)", a.width, a.height, b.width, b.height)
	}
}

func TestSetItemsKeepsTheCursor(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(1)
	l.SetItems([]ListItem{&textItem{"x"}, &textItem{"y"}, &textItem{"z"}})
	if l.ActiveIndex() != 1 {
		t.Errorf("cursor should stay on 1, got %d", l.ActiveIndex())
	}
}

func TestSetItemsClampsTheCursorWhenShorter(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(2)
	l.SetItems([]ListItem{&textItem{"x"}})
	if l.ActiveIndex() != 0 {
		t.Errorf("cursor should clamp to 0, got %d", l.ActiveIndex())
	}
}

func TestSetItemsFromEmptyPutsTheCursorOnTheFirstItem(t *testing.T) {
	l := newTestList(nil, Vertical, 20, 20, Styles{})
	l.SetItems([]ListItem{&textItem{"x"}, &textItem{"y"}})
	if l.ActiveIndex() != 0 {
		t.Errorf("cursor should be on 0, got %d", l.ActiveIndex())
	}
}

func TestSetItemsKeepsASelection(t *testing.T) {
	l := textList("a", "b", "c")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.IncrementItem(1)
	l.SetItems([]ListItem{&textItem{"x"}, &textItem{"y"}, &textItem{"z"}})
	if !l.Selecting() || len(l.Selection()) != 2 {
		t.Errorf("selection of 2 should survive, got selecting=%v len=%d", l.Selecting(), len(l.Selection()))
	}
}

func TestSetItemsShrinkingBelowTheSelectionStaysValid(t *testing.T) {
	l := textList("a", "b", "c", "d")
	l.SelectAll()
	l.SetItems([]ListItem{&textItem{"x"}, &textItem{"y"}})
	if !l.Selecting() || len(l.Selection()) != 2 || l.ActiveIndex() != 1 {
		t.Errorf("want a valid 2-item selection with the cursor on 1, got selecting=%v len=%d cursor=%d", l.Selecting(), len(l.Selection()), l.ActiveIndex())
	}
}

func TestSetItemsWithNothingClearsCursorAndSelection(t *testing.T) {
	l := textList("a", "b")
	l.SetActiveIndex(0)
	l.EnableSelectionMode()
	l.IncrementItem(1)
	l.SetItems(nil)
	if l.ActiveIndex() != -1 || l.Selecting() || l.Selection() != nil {
		t.Errorf("an empty list has no cursor or selection, got cursor=%d selecting=%v", l.ActiveIndex(), l.Selecting())
	}
	l.SetItems([]ListItem{&textItem{"x"}})
	if l.ActiveIndex() != 0 {
		t.Errorf("cursor should be on 0 after refilling, got %d", l.ActiveIndex())
	}
}

func TestSetItemsCopiesTheSlice(t *testing.T) {
	original := &textItem{"x"}
	items := []ListItem{original, &textItem{"y"}}
	l := textList("a")
	l.SetItems(items)
	items[0] = &textItem{"z"}
	if l.ItemAt(0) != ListItem(original) {
		t.Error("changing the caller's slice must not change the list")
	}
}

func TestLenCountsItems(t *testing.T) {
	if got := textList("a", "b").Len(); got != 2 {
		t.Errorf("want 2, got %d", got)
	}
	if got := newTestList(nil, Vertical, 20, 20, Styles{}).Len(); got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestAllYieldsIndexAndItemInOrder(t *testing.T) {
	var got []string
	for i, it := range textList("a", "b", "c").All() {
		got = append(got, fmt.Sprintf("%d%s", i, it.(*textItem).text))
	}
	if strings.Join(got, ",") != "0a,1b,2c" {
		t.Errorf("want 0a,1b,2c, got %v", got)
	}
}

func TestAllStopsWhenTheLoopBreaks(t *testing.T) {
	visited := 0
	for range textList("a", "b", "c").All() {
		visited++
		break
	}
	if visited != 1 {
		t.Errorf("break should stop after one item, visited %d", visited)
	}
	for range textList().All() {
		t.Error("an empty list yields nothing")
	}
}

func TestAllAllowsSetItemDuringIteration(t *testing.T) {
	l := textList("a", "b", "c")
	for i := range l.All() {
		l.SetItem(i, &textItem{"z"})
	}
	if got := strings.Join(texts(l.Items()), ","); got != "z,z,z" {
		t.Errorf("want z,z,z, got %s", got)
	}
}

type msgRecorder struct{ got []tea.Msg }

func (r *msgRecorder) Init() tea.Cmd { return nil }
func (r *msgRecorder) View() string  { return "recorder" }
func (r *msgRecorder) Update(msg tea.Msg) tea.Cmd {
	r.got = append(r.got, msg)
	return func() tea.Msg { return struct{}{} }
}

func recorderList() (*Model, []*msgRecorder) {
	recorders := []*msgRecorder{{}, {}, {}}
	items := make([]ListItem, len(recorders))
	for i, r := range recorders {
		items[i] = r
	}
	l := newTestList(items, Vertical, 20, 20, Styles{})
	l.SetActiveIndex(1)
	return l, recorders
}

func TestKeyPressesOnlyReachTheActiveItem(t *testing.T) {
	l, rs := recorderList()
	msg := tea.KeyPressMsg(tea.Key{Code: 'a'})

	if cmd := l.Update(msg); cmd == nil {
		t.Error("the active item's command should be returned")
	}
	if len(rs[0].got) != 0 || len(rs[2].got) != 0 {
		t.Errorf("inactive items must not see key presses: %v %v", rs[0].got, rs[2].got)
	}
	if len(rs[1].got) != 1 || rs[1].got[0] != tea.Msg(msg) {
		t.Errorf("the active item should get the key press, got %v", rs[1].got)
	}
}

func TestKeyReleasesAndPastesOnlyReachTheActiveItem(t *testing.T) {
	for name, msg := range map[string]tea.Msg{
		"key release": tea.KeyReleaseMsg(tea.Key{Code: 'a'}),
		"paste":       tea.PasteMsg{Content: "pasted"},
	} {
		l, rs := recorderList()
		if cmd := l.Update(msg); cmd == nil {
			t.Errorf("%s: the active item's command should be returned", name)
		}
		if len(rs[0].got) != 0 || len(rs[2].got) != 0 {
			t.Errorf("%s: inactive items must not see it: %v %v", name, rs[0].got, rs[2].got)
		}
		if len(rs[1].got) != 1 || rs[1].got[0] != msg {
			t.Errorf("%s: the active item should get it, got %v", name, rs[1].got)
		}
	}
}

func TestOtherMessagesReachEveryItem(t *testing.T) {
	l, rs := recorderList()
	msg := struct{ n int }{1}

	l.Update(msg)

	for i, r := range rs {
		if len(r.got) != 1 || r.got[0] != tea.Msg(msg) {
			t.Errorf("item %d should get the message, got %v", i, r.got)
		}
	}
}

func TestMouseMessagesStayBroadcast(t *testing.T) {
	l, rs := recorderList()
	msg := tea.MouseClickMsg(tea.Mouse{X: 1, Y: 1, Button: tea.MouseLeft})

	l.Update(msg)

	for i, r := range rs {
		if len(r.got) != 1 || r.got[0] != tea.Msg(msg) {
			t.Errorf("item %d should get the mouse message, got %v", i, r.got)
		}
	}
}

func TestKeysToAnEmptyListAreDropped(t *testing.T) {
	l := newTestList(nil, Vertical, 20, 20, Styles{})
	for name, msg := range map[string]tea.Msg{
		"key press":   tea.KeyPressMsg(tea.Key{Code: 'a'}),
		"key release": tea.KeyReleaseMsg(tea.Key{Code: 'a'}),
		"paste":       tea.PasteMsg{Content: "pasted"},
	} {
		if cmd := l.Update(msg); cmd != nil {
			t.Errorf("%s: an empty list has nowhere to send it, got a command", name)
		}
	}
}

func BenchmarkView(b *testing.B) {
	items := make([]ListItem, 50)
	for i := range items {
		items[i] = &textItem{fmt.Sprintf("item %d", i)}
	}
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	l := newTestList(items, Vertical, 20, 40, Styles{Item: item, Active: item})
	l.SetFade(testFade)
	l.SetScrollBar(true)
	l.SetActiveIndex(10)

	b.ReportAllocs()
	for b.Loop() {
		l.View()
	}
}

func optionTestItems() []ListItem {
	return []ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}
}

func sizedList(opts ...Option) *Model {
	l := New(optionTestItems(), opts...)
	l.SetSize(30, 20)
	return l
}

func TestNewDefaults(t *testing.T) {
	l := New(nil)

	if l.Orientation() != Vertical || l.Width() != 0 || l.Height() != 0 {
		t.Errorf("want vertical 0x0, got %v %dx%d", l.Orientation(), l.Width(), l.Height())
	}
	if l.ScrollBar() || l.FillWidth() || l.Paged() {
		t.Error("scrollbar, fill width and paged all default to off")
	}
	if _, ok := l.Fade(); ok {
		t.Error("a list has no fade by default")
	}
}

func TestEachOptionMatchesItsSetter(t *testing.T) {
	thick := Styles{Active: lipgloss.NewStyle().Border(lipgloss.ThickBorder())}
	cases := []struct {
		name string
		opt  Option
		set  func(*Model)
		get  func(*Model) any
	}{
		{"scrollbar", WithScrollBar(true), func(l *Model) { l.SetScrollBar(true) }, func(l *Model) any { return l.ScrollBar() }},
		{"fill width", WithFillWidth(true), func(l *Model) { l.SetFillWidth(true) }, func(l *Model) any { return l.FillWidth() }},
		{"paged", WithPaged(true), func(l *Model) { l.SetPaged(true) }, func(l *Model) any { return l.Paged() }},
		{"styles", WithStyles(thick), func(l *Model) { l.SetStyles(thick) }, func(l *Model) any { return l.Styles().Active.Render("x") }},
		{"fade", WithFade(testFade), func(l *Model) { l.SetFade(testFade) }, func(l *Model) any {
			f, ok := l.Fade()
			return [2]any{f, ok}
		}},
	}
	for _, c := range cases {
		viaOption := sizedList(c.opt)
		viaSetter := sizedList()
		c.set(viaSetter)

		if c.get(viaOption) != c.get(viaSetter) {
			t.Errorf("%s: getters differ: %v vs %v", c.name, c.get(viaOption), c.get(viaSetter))
		}
		if viaOption.View() != viaSetter.View() {
			t.Errorf("%s: views differ", c.name)
		}
		if viaOption.View() == sizedList().View() && c.name != "fade" {
			t.Errorf("%s: the option should change the view", c.name)
		}
	}
}

func TestOptionsApplyInAnyOrderAndLastWins(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	s := Styles{Item: item, Active: item}

	a := sizedList(WithFade(testFade), WithStyles(s))
	b := sizedList(WithStyles(s), WithFade(testFade))

	if a.View() != b.View() {
		t.Error("option order must not change the result")
	}
	if !strings.Contains(a.View(), fadedForeground()) {
		t.Error("the fade should be in effect")
	}
	if New(nil, WithScrollBar(true), WithScrollBar(false)).ScrollBar() {
		t.Error("the last option should win")
	}
}

func TestSetFadeCopiesTheValue(t *testing.T) {
	l := New(nil)
	f := testFade
	l.SetFade(f)
	f.Amount = 0

	if got, ok := l.Fade(); !ok || got.Amount != 0.6 {
		t.Errorf("the list should keep its own copy, got %v %v", got.Amount, ok)
	}
}

func TestClearFadeAfterWithFade(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}}, WithFade(testFade), WithStyles(Styles{Item: item, Active: item}))
	l.SetSize(20, 20)
	l.SetActiveIndex(0)
	if !strings.Contains(l.View(), fadedForeground()) {
		t.Fatal("the fade should be in effect before it is cleared")
	}

	l.ClearFade()

	if _, ok := l.Fade(); ok {
		t.Error("Fade should report no fade after ClearFade")
	}
	if strings.Contains(l.View(), fadedForeground()) {
		t.Error("no item should be faded after ClearFade")
	}
}

func TestSetStylesChangesTheRendering(t *testing.T) {
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}})
	l.SetSize(20, 20)
	if strings.Contains(l.View(), "┏") {
		t.Fatal("the zero Styles has no border")
	}

	l.SetStyles(Styles{Active: lipgloss.NewStyle().Border(lipgloss.ThickBorder())})

	if !strings.Contains(l.View(), "┏") {
		t.Error("the active row should now have a thick border")
	}
}

func newTestList(items []ListItem, o Orientation, w, h int, s Styles) *Model {
	l := New(items, WithOrientation(o), WithStyles(s))
	l.SetSize(w, h)
	return l
}
