package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/danielriddell21/toolshed/internal/style"
)

const (
	minBaysPerRow = 12
	maxBaysPerRow = 44
	garageRows    = 6
	carSpacing    = 4
	panelColumn   = 30
	gaugeWidth    = 14
	laneReserve   = 24
	bayCar        = "▄"
	bayLine       = "▏"
	bayFree       = " "
)

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

var (
	nearCars = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	deepCars = lipgloss.NewStyle().Foreground(lipgloss.Color("#15803D"))
	inbound  = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	vacant   = lipgloss.NewStyle().Foreground(lipgloss.Color("#39404E"))
	concrete = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	warm     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
	label    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	reading  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Bold(true)
	trace    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
)

type garage struct {
	perRow int
	front  int
	deep   int
}

func layout(width int, frontFraction float64) garage {
	perRow := clampInt((width-8)/2, minBaysPerRow, maxBaysPerRow)
	total := perRow * garageRows
	front := clampInt(int(math.Round(float64(total)*frontFraction)), perRow, total-perRow)
	return garage{perRow: perRow, front: front, deep: total - front}
}

func (g garage) inner() int { return g.perRow*2 + 2 }

func (g garage) rows(bays int) int { return (bays + g.perRow - 1) / g.perRow }

// deck draws one block of bays, bottom row first, so cars visibly stack up
// against the ramp end of the structure and the top floor is the one left half
// empty.
func (g garage) deck(bays int, fill float64, cars lipgloss.Style) []string {
	filled := clampInt(int(math.Round(float64(bays)*fill)), 0, bays)
	out := make([]string, 0, g.rows(bays))
	for row := g.rows(bays) - 1; row >= 0; row-- {
		start := row * g.perRow
		width := min(g.perRow, bays-start)
		taken := clampInt(filled-start, 0, width)
		line := cars.Render(strings.Repeat(bayCar+bayLine, taken)) +
			vacant.Render(strings.Repeat(bayFree+bayLine, width-taken)) +
			strings.Repeat(" ", (g.perRow-width)*2)
		out = append(out, concrete.Render("│")+" "+line+" "+concrete.Render("│"))
	}
	return out
}

func rule(width int, left, right, fill, text string) string {
	if pad := width - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat(fill, pad)
	}
	return concrete.Render(left) + label.Render(text) + concrete.Render(right)
}

func (m model) garageView(g garage) string {
	cell := m.sim.Cell
	near, deep := nearCars, deepCars
	if m.sim.CurrentA < 0 {
		near, deep = inbound, inbound
	}

	lines := make([]string, 0, garageRows+3)
	lines = append(lines, rule(g.inner(), "╭", "╮", "─",
		fmt.Sprintf("─ DEEP BAYS · %.3f Ah · a long walk from the ramp ", cell.DeepAh)))
	lines = append(lines, g.deck(g.deep, cell.DeepAh/cell.DeepCapacityAh(), deep)...)
	lines = append(lines, m.rampView(g))
	lines = append(lines, g.deck(g.front, cell.SurfaceSoC(), near)...)
	lines = append(lines, rule(g.inner(), "╰", "╯", "─",
		fmt.Sprintf("─ FRONT BAYS · %.3f Ah · ready to leave ", cell.FrontAh)))

	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func (m model) rampView(g garage) string {
	flow := m.sim.Cell.RampFlowA()
	glyph, direction := "─", "settled"
	if flow > 0.001 {
		glyph, direction = "▼", "down to the exit"
	} else if flow < -0.001 {
		glyph, direction = "▲", "up into the deep bays"
	}

	// The arrows thin out as the two wells equalise, so a cell that has just
	// been hammered looks busy and a rested one goes quiet.
	lit := 1 + clampInt(int(math.Abs(flow)*4), 0, 2)
	arrows := strings.Repeat(glyph, lit) + strings.Repeat(" ", 3-lit)

	text := fmt.Sprintf("═ RAMP %s %.3f A %s · k %.1f /h ", arrows, math.Abs(flow), direction, m.sim.Cell.RampRate())
	if pad := g.inner() - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat("═", pad)
	}
	return concrete.Render("├") + warm.Render(text) + concrete.Render("┤")
}

// laneView is the access road under the structure: cars stream out to the exit
// on a drain and in from the entrance on a charge, at a speed set by the current.
func (m model) laneView(g garage) string {
	road := []rune(strings.Repeat(" ", max(1, g.inner()-laneReserve)))
	glyph, gate, paint := '◄', "to the load", nearCars
	if m.sim.CurrentA < 0 {
		glyph, gate, paint = '►', "from the charger", inbound
	}

	if m.sim.CurrentA != 0 {
		for i := int(m.lane) % carSpacing; i < len(road); i += carSpacing {
			road[i] = glyph
		}
	} else {
		gate = "barrier down"
	}

	return "   " + paint.Render(string(road)) + "  " +
		reading.Render(fmt.Sprintf("%.3f A", math.Abs(m.sim.CurrentA))) + " " + label.Render(gate)
}

func (m model) panelView(g garage) string {
	cell := m.sim.Cell
	temp := reading
	if cell.TempC() > 45 {
		temp = warm
	}

	sparkCells := clampInt(g.inner()/4, 8, 16)
	rows := [][]string{{
		stat("charge", gauge(cell.SoC(), gaugeWidth)+fmt.Sprintf(" %.1f%%", 100*cell.SoC())),
		stat("front bays", fmt.Sprintf("%.1f%%", 100*cell.SurfaceSoC())),
		stat("stored", fmt.Sprintf("%.3f / %.3f Ah", cell.ChargeAh(), cell.Chem.CapacityAh)),
	}, {
		stat("terminal", fmt.Sprintf("%.3f V", cell.Terminal(m.sim.CurrentA))),
		stat("rested", fmt.Sprintf("%.3f V", cell.OpenCircuit())),
		stat("power", signed(m.sim.PowerW(), "W", "out", "in")),
	}, {
		statStyled("temperature", fmt.Sprintf("%.1f °C", cell.TempC()), temp),
		stat("resistance", fmt.Sprintf("%.1f mΩ", 1000*cell.Resistance())),
		stat("cycles", fmt.Sprintf("%.3f", m.sim.Cycles())),
	}, {
		stat("clock", fmt.Sprintf("%s ×%g", clock(m.sim.ElapsedS), m.speed)),
		statStyled("volts", trace.Render(spark(m.volts, sparkCells))+
			reading.Render(fmt.Sprintf(" %.3f V", last(m.volts))), reading),
		statStyled("amps", nearCars.Render(spark(m.amps, sparkCells))+
			reading.Render(fmt.Sprintf(" %.3f A", last(m.amps))), reading),
	}}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, "  "+strings.Join(padColumns(row), ""))
	}
	return strings.Join(lines, "\n")
}

func padColumns(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = c
		if i < len(cells)-1 {
			out[i] += strings.Repeat(" ", max(1, panelColumn-lipgloss.Width(c)))
		}
	}
	return out
}

func stat(name, text string) string { return statStyled(name, text, reading) }

func statStyled(name, text string, valueStyle lipgloss.Style) string {
	return label.Render(name+" ") + valueStyle.Render(text)
}

func signed(v float64, unit, positive, negative string) string {
	if v == 0 {
		return fmt.Sprintf("0.000 %s", unit)
	}
	direction := positive
	if v < 0 {
		direction = negative
	}
	return fmt.Sprintf("%.3f %s %s", math.Abs(v), unit, direction)
}

func gauge(fill float64, width int) string {
	filled := clampInt(int(math.Round(fill*float64(width))), 0, width)
	return nearCars.Render(strings.Repeat("█", filled)) + vacant.Render(strings.Repeat("░", width-filled))
}

func spark(series []float64, width int) string {
	if len(series) == 0 {
		return strings.Repeat(" ", width)
	}
	if len(series) > width {
		series = series[len(series)-width:]
	}

	lo, hi := series[0], series[0]
	for _, v := range series {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}

	var b strings.Builder
	b.WriteString(strings.Repeat(" ", width-len(series)))
	for _, v := range series {
		level := 0
		if hi > lo {
			level = clampInt(int((v-lo)/(hi-lo)*float64(len(sparkRunes)-1)+0.5), 0, len(sparkRunes)-1)
		}
		b.WriteRune(sparkRunes[level])
	}
	return b.String()
}

func last(series []float64) float64 {
	if len(series) == 0 {
		return 0
	}
	return series[len(series)-1]
}

func clock(seconds float64) string {
	total := int(seconds)
	return fmt.Sprintf("%02d:%02d:%02d", total/3600, total/60%60, total%60)
}

func (m model) headerView() string {
	cell := m.sim.Cell
	return style.Title.Render("battery") +
		style.Note.Render("  — a cell as a multi-storey car park") + "   " +
		label.Render(cell.Chem.Name+" · ") + reading.Render(fmt.Sprintf("%.1f Ah", cell.Chem.CapacityAh)) +
		label.Render(" · ") + reading.Render(m.sim.Mode.String()) +
		label.Render(", "+m.sim.Phase.String())
}

func (m model) settingsView() string {
	return "  " + label.Render(fmt.Sprintf("load %.2fC · charge %.2fC · ambient %.1f °C · window %.2f-%.2f V",
		m.sim.LoadCRate, m.sim.ChargeCRate, m.sim.Cell.AmbientK-kelvin,
		m.sim.Cell.Chem.VCut, m.sim.Cell.Chem.VFull))
}

func (m model) helpView() string {
	pause := "space pause"
	if m.paused {
		pause = "space resume"
	}
	return style.Help.Render("  c charge  d drain  r rest  n cell  +/- load  [/] charge  " +
		"</> ambient  f speed  " + pause + "  q quit")
}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }
