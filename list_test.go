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
	out := Dim("\x1b[38;2;255;121;198mpink\x1b[m")

	if strings.Contains(out, "38;2;255;121;198") {
		t.Errorf("the original colour should be gone: %q", out)
	}
	r, g, b := blend(color.RGBA{R: 255, G: 121, B: 198, A: 255}, dimAmount)
	if want := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b); !strings.Contains(out, want) {
		t.Errorf("the colour should be blended towards the background, want %q in %q", want, out)
	}
	if ansi.Strip(out) != "pink" {
		t.Errorf("the text must be unchanged, got %q", ansi.Strip(out))
	}
}

func TestDimGivesPlainTextAFadedColourAfterResets(t *testing.T) {
	for _, reset := range []string{"\x1b[m", "\x1b[0m"} {
		out := Dim("\x1b[38;2;255;0;0ma" + reset + " plain")

		if !strings.Contains(out, reset+fadedForeground()+" plain") {
			t.Errorf("plain text after a %q reset should be faded too: %q", reset, out)
		}
		if !strings.HasPrefix(out, fadedForeground()) {
			t.Errorf("a line should start with the faded default colour: %q", out)
		}
	}
}

func TestDimLeavesBackgroundsAlone(t *testing.T) {
	out := Dim("\x1b[48;2;1;2;3mx\x1b[m")
	if !strings.Contains(out, "\x1b[48;2;1;2;3m") {
		t.Errorf("backgrounds are kept: %q", out)
	}
}

func fadedForeground() string {
	r, g, b := blend(fadeForeground, dimAmount)
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
				l.EnabledSelectionMode()
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
func (f *fillingItem) GetID() int64           { return 0 }

type textItem struct{ text string }

func (t *textItem) Init() tea.Cmd          { return nil }
func (t *textItem) Update(tea.Msg) tea.Cmd { return nil }
func (t *textItem) View() string           { return t.text }
func (t *textItem) SetSize(int, int)       {}
func (t *textItem) GetID() int64           { return 0 }

func TestOnlyTheActiveAndSelectedItemsAreBright(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	bright := lipgloss.NewStyle().Foreground(lipgloss.RGBColor{R: 255, G: 121, B: 198})
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, Styles{Item: item, Active: item, Selected: bright})
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

	l.EnabledSelectionMode()
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
	down.EnabledSelectionMode()
	down.IncrementItem(1)
	check("selecting down", down)

	up := newList()
	up.SetActiveIndex(1)
	up.EnabledSelectionMode()
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
	l.EnabledSelectionMode()
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
