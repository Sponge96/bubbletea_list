package list

import (
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	fadeForeground = lipgloss.RGBColor{R: 0xf8, G: 0xf8, B: 0xf2}
	fadeBackground = lipgloss.RGBColor{R: 0x28, G: 0x2a, B: 0x36}
)

// dimAmount is how far dimmed items fade towards the background, 0 to 1.
const dimAmount = 0.6

var sgrSequence = regexp.MustCompile("\x1b\\[([0-9;]*)m")

func blend(c color.Color, amount float64) (r, g, b int) {
	bgR, bgG, bgB, _ := fadeBackground.RGBA()
	cr, cg, cb, _ := c.RGBA()
	mix := func(from, to uint32) int {
		return int(float64(from>>8)*(1-amount) + float64(to>>8)*amount)
	}
	return mix(cr, bgR), mix(cg, bgG), mix(cb, bgB)
}

func Dim(view string) string {
	r, g, b := blend(fadeForeground, dimAmount)
	base := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)

	lines := strings.Split(view, "\n")
	for i, line := range lines {
		line = sgrSequence.ReplaceAllStringFunc(line, func(seq string) string {
			params := sgrSequence.FindStringSubmatch(seq)[1]
			if params == "" || params == "0" {
				return seq + base
			}
			return "\x1b[" + dimParams(params) + "m"
		})
		lines[i] = base + line + "\x1b[m"
	}
	return strings.Join(lines, "\n")
}

func dimParams(params string) string {
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "38", "48":
			if i+1 >= len(parts) {
				break
			}
			switch parts[i+1] {
			case "2":
				if i+4 < len(parts) && parts[i] == "38" {
					rgb := make([]int, 3)
					ok := true
					for k := range rgb {
						v, err := strconv.Atoi(parts[i+2+k])
						if err != nil {
							ok = false
							break
						}
						rgb[k] = v
					}
					if ok {
						r, g, b := blend(color.RGBA{R: uint8(rgb[0]), G: uint8(rgb[1]), B: uint8(rgb[2]), A: 255}, dimAmount)
						parts[i+2], parts[i+3], parts[i+4] = strconv.Itoa(r), strconv.Itoa(g), strconv.Itoa(b)
					}
				}
				i += 4
			case "5":
				i += 2
			}
		}
	}
	return strings.Join(parts, ";")
}
