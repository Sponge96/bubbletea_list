package list

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var testFill = lipgloss.RGBColor{R: 1, G: 2, B: 3}

const fillOn = "\x1b[48;2;1;2;3m"

func TestFillKeepsTheBackgroundAfterStyledText(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("red") + " plain"
	out := fill(styled, testFill)

	if !strings.HasPrefix(out, fillOn) {
		t.Fatalf("the fill should start the line: %q", out)
	}
	reset := strings.Index(out, "\x1b[m")
	if reset < 0 || !strings.HasPrefix(out[reset:], "\x1b[m"+fillOn) {
		t.Fatalf("the fill should resume after the styled text resets: %q", out)
	}
}

func TestFillMakesARectangle(t *testing.T) {
	out := fill("a\nlonger line", testFill)
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w != len("longer line") {
			t.Errorf("line %q is %d wide", l, w)
		}
		if !strings.HasPrefix(l, fillOn) {
			t.Errorf("every line should start with the fill: %q", l)
		}
	}
}

func TestFillWithoutAColourChangesNothing(t *testing.T) {
	if got := fill("x\ny", lipgloss.NoColor{}); got != "x\ny" {
		t.Errorf("got %q", got)
	}
}

type textItem struct{ text string }

func (t *textItem) Init() tea.Cmd          { return nil }
func (t *textItem) Update(tea.Msg) tea.Cmd { return nil }
func (t *textItem) View() string           { return t.text }
func (t *textItem) SetSize(int, int)       {}
func (t *textItem) GetID() int64           { return 0 }

func TestActiveAndSelectedItemsAreFilled(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	active := item.Background(lipgloss.RGBColor{R: 10, G: 20, B: 30})
	selected := lipgloss.NewStyle().Background(lipgloss.RGBColor{R: 40, G: 50, B: 60})
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}}, Vertical, 20, 10, item, active, selected)
	l.SetActiveIndex(0)

	view := l.View()
	if !strings.Contains(view, "\x1b[48;2;10;20;30m") {
		t.Errorf("the active item should be filled: %q", view)
	}
	if strings.Contains(view, "\x1b[48;2;40;50;60m") {
		t.Errorf("nothing is selected yet: %q", view)
	}

	l.EnabledSelectionMode()
	l.IncrementItem(1)
	view = l.View()
	if strings.Count(view, "\x1b[48;2;40;50;60m") < 2 {
		t.Errorf("both selected items should be filled: %q", view)
	}
}
