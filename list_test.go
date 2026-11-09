package list

import (
	"fmt"
	"image/color"
	"soap/tui/styles"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestDimFadesColouredText(t *testing.T) {
	out := dim("\x1b[38;2;255;121;198mpink\x1b[m")

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
		out := dim("\x1b[38;2;255;0;0ma" + reset + " plain")

		if !strings.Contains(out, reset+fadedForeground()+" plain") {
			t.Errorf("plain text after a %q reset should be faded too: %q", reset, out)
		}
		if !strings.HasPrefix(out, fadedForeground()) {
			t.Errorf("a line should start with the faded default colour: %q", out)
		}
	}
}

func TestDimLeavesBackgroundsAlone(t *testing.T) {
	out := dim("\x1b[48;2;1;2;3mx\x1b[m")
	if !strings.Contains(out, "\x1b[48;2;1;2;3m") {
		t.Errorf("backgrounds are kept: %q", out)
	}
}

func fadedForeground() string {
	r, g, b := blend(styles.DraculaForeground, dimAmount)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

type textItem struct{ text string }

func (t *textItem) Init() tea.Cmd          { return nil }
func (t *textItem) Update(tea.Msg) tea.Cmd { return nil }
func (t *textItem) View() string           { return t.text }
func (t *textItem) SetSize(int, int)       {}
func (t *textItem) GetID() int64           { return 0 }

func TestOnlyTheActiveAndSelectedItemsAreBright(t *testing.T) {
	item := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	bright := lipgloss.NewStyle().Foreground(lipgloss.RGBColor{R: 255, G: 121, B: 198})
	l := New([]ListItem{&textItem{"one"}, &textItem{"two"}, &textItem{"three"}}, Vertical, 20, 20, item, item, bright)
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
