package list

import (
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"
)

// Fade is how far dimmed text blends from its colour towards Background, with
// Amount from 0 to 1. Foreground is the colour that plain text is faded from.
type Fade struct {
	Foreground color.Color
	Background color.Color
	Amount     float64
}

var sgrSequence = regexp.MustCompile("\x1b\\[([0-9;]*)m")

func blend(c color.Color, f Fade) (r, g, b int) {
	amount := f.Amount
	bgR, bgG, bgB, _ := f.Background.RGBA()
	cr, cg, cb, _ := c.RGBA()
	mix := func(from, to uint32) int {
		return int(float64(from>>8)*(1-amount) + float64(to>>8)*amount)
	}
	return mix(cr, bgR), mix(cg, bgG), mix(cb, bgB)
}

func Dim(view string, f Fade) string {
	if f.Foreground == nil || f.Background == nil {
		return view
	}
	r, g, b := blend(f.Foreground, f)
	base := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)

	lines := strings.Split(view, "\n")
	for i, line := range lines {
		line = sgrSequence.ReplaceAllStringFunc(line, func(seq string) string {
			params := sgrSequence.FindStringSubmatch(seq)[1]
			if params == "" || params == "0" {
				return seq + base
			}
			return "\x1b[" + dimParams(params, f) + "m"
		})
		lines[i] = base + line + "\x1b[m"
	}
	return strings.Join(lines, "\n")
}

func dimParams(params string, f Fade) string {
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
						r, g, b := blend(color.RGBA{R: uint8(rgb[0]), G: uint8(rgb[1]), B: uint8(rgb[2]), A: 255}, f)
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
