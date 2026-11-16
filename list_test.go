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
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}}, Vertical, 20, 20, Styles{Item: item, Active: item})
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
	full := New([]ListItem{&fillingItem{}}, Vertical, 20, 10, Styles{Item: item, Active: item})
	full.ScrollBar = true
	full.SetActiveIndex(0)
	empty := New([]ListItem{}, Vertical, 20, 10, Styles{Item: item, Active: item})
	empty.ScrollBar = true

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
			l := New([]ListItem{&textItem{"one"}, &textItem{long}, &textItem{"three"}}, Vertical, 30, 40, Styles{Item: item, Active: item})
			l.ScrollBar = scroll
			l.FillWidth = true
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
	full := New([]ListItem{&textItem{"one"}}, Vertical, 20, 10, Styles{Item: item, Active: item})
	empty := New([]ListItem{}, Vertical, 20, 10, Styles{Item: item, Active: item})
	for _, l := range []*Model{full, empty} {
		l.ScrollBar = true
		l.FillWidth = true
	}
	full.SetActiveIndex(0)

	if got, want := lipgloss.Width(empty.View()), lipgloss.Width(full.View()); got != want {
		t.Errorf("an empty list should be as wide as a full one (%d), got %d", want, got)
	}
}

func TestFillWidthLeavesHorizontalListsAndUnsizedListsAlone(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	want := lipgloss.Width(item.Render("one"))

	horizontal := New([]ListItem{&textItem{"one"}}, Horizontal, 30, 10, Styles{Item: item, Active: item})
	horizontal.FillWidth = true
	horizontal.SetActiveIndex(0)
	if got := lipgloss.Width(horizontal.View()); got != want {
		t.Errorf("a horizontal list keeps its item width %d, got %d", want, got)
	}

	unsized := New([]ListItem{&textItem{"one"}}, Vertical, 0, 10, Styles{Item: item, Active: item})
	unsized.FillWidth = true
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
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, Styles{Item: item, Active: item, Selected: bright})
	l.Fade = &testFade
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
		return New([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, Styles{Selected: red, SelectedActive: red})
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
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}}, Vertical, 20, 20,
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

	styled := New(items(), Vertical, 4, 4, Styles{Indicator: red})
	styled.ScrollBar = true
	styled.SetActiveIndex(0)
	if !strings.Contains(lineWith(t, styled.View(), "┃"), redSGR) {
		t.Error("the thumb should carry the Indicator colour")
	}

	plain := New(items(), Vertical, 4, 4, Styles{})
	plain.ScrollBar = true
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
	l := New([]ListItem{&textItem{"a"}, &textItem{"b"}, &textItem{"x"}, &textItem{"x"}}, Vertical, 20, 20, Styles{})
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
	l := New([]ListItem{&textItem{"a"}, &textItem{"x"}, &textItem{"b"}, &textItem{"x"}}, Vertical, 20, 20, Styles{})

	l.DeleteWhere(isText("x"))

	var got []string
	for _, it := range l.Items {
		got = append(got, it.(*textItem).text)
	}
	if strings.Join(got, ",") != "a,b,x" {
		t.Errorf("only the first match should go, got %v", got)
	}
}

func TestDeleteWhereKeepsTheCursorValid(t *testing.T) {
	l := New([]ListItem{&textItem{"a"}, &textItem{"b"}, &textItem{"c"}}, Vertical, 20, 20, Styles{})
	l.SetActiveIndex(2)
	l.DeleteWhere(isText("c"))
	if l.ActiveIndex() != 1 || l.ActiveItem().(*textItem).text != "b" {
		t.Errorf("deleting the active last item should move to the new last, got %d", l.ActiveIndex())
	}

	only := New([]ListItem{&textItem{"a"}}, Vertical, 20, 20, Styles{})
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
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Pager, 30, 10, Styles{})
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
	l := New(threeLineItems(), Pager, 30, 10, Styles{})

	for i, want := range []int{1, 3, 2} {
		l.SetActiveIndex(i)
		if got := lipgloss.Height(l.View()); got != want {
			t.Errorf("item %d: the pager should be %d lines tall, got %d", i, want, got)
		}
	}
}

func TestPagerShowsDotsOnlyWithAScrollBar(t *testing.T) {
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Pager, 30, 10, Styles{})
	l.SetActiveIndex(0)

	if strings.Contains(ansi.Strip(l.View()), "●○○") {
		t.Error("no dots without a ScrollBar")
	}
	l.ScrollBar = true
	if !strings.Contains(ansi.Strip(l.View()), "●○○") {
		t.Errorf("dots expected with a ScrollBar: %q", ansi.Strip(l.View()))
	}
}

func TestHorizontalStillShowsNeighbours(t *testing.T) {
	l := New(threeLineItems(), Horizontal, 30, 10, Styles{})
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
	for name, o := range map[string]Orientation{"vertical": Vertical, "horizontal": Horizontal, "pager": Pager} {
		l := New([]ListItem{}, o, 20, 5, Styles{})
		l.ScrollBar = true
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
	l := New(nil, Vertical, 0, 0, Styles{})
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

func TestNewSizesTheItemsItIsGiven(t *testing.T) {
	a := &sizeRecorder{}
	New([]ListItem{a}, Vertical, 25, 7, Styles{})

	if a.width != 25 || a.height != 7 {
		t.Errorf("want (25, 7), got (%d, %d)", a.width, a.height)
	}
}

func TestSetItemSizesTheReplacementAndKeepsTheCursor(t *testing.T) {
	l := New([]ListItem{&textItem{"a"}, &textItem{"b"}, &textItem{"c"}}, Vertical, 20, 5, Styles{})
	l.SetActiveIndex(2)
	fresh := &sizeRecorder{textItem: textItem{"fresh"}}

	l.SetItem(1, fresh)

	if l.Items[1] != ListItem(fresh) {
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
	l := New([]ListItem{&textItem{"a"}}, Vertical, 20, 5, Styles{})
	fresh := &sizeRecorder{}

	l.SetItem(-1, fresh)
	l.SetItem(1, fresh)

	if len(l.Items) != 1 || l.Items[0].(*textItem).text != "a" {
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
	l := New([]ListItem{bareItem{}}, Vertical, 20, 5, Styles{})
	l.InsertItem(bareItem{}, 1)
	l.SetItem(0, bareItem{})
	l.SetSize(30, 6)
	l.SetActiveIndex(0)

	if !strings.Contains(l.View(), "bare") {
		t.Errorf("an item without SetSize should render, got %q", l.View())
	}
}
