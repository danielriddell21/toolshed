package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/danielriddell21/toolshed/internal/style"
)

const (
	panelColumn = 30
	gaugeWidth  = 14
)

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

// Cars are painted from a fixed set of colours so a bay always holds the same
// car, which makes an arrival or a departure read as one vehicle moving rather
// than a bar changing length.
var carPaint = [][3]uint8{
	{0xE2, 0xE8, 0xF0},
	{0xEF, 0x44, 0x44},
	{0x60, 0xA5, 0xFA},
	{0x9C, 0xA3, 0xAF},
	{0x34, 0xD3, 0x99},
	{0xFB, 0xBF, 0x24},
	{0xA7, 0x8B, 0xFA},
	{0xF4, 0x72, 0xB6},
	{0x38, 0xBD, 0xF8},
	{0xCB, 0xD5, 0xE1},
}

var (
	nearCars = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
	vacant   = lipgloss.NewStyle().Foreground(lipgloss.Color("#39404E"))
	warm     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F97316"))
	label    = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	reading  = lipgloss.NewStyle().Foreground(lipgloss.Color("#E2E8F0")).Bold(true)
	trace    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
)

func (m model) panelView(width int) string {
	cell := m.sim.Cell
	temp := reading
	if cell.TempC() > 45 {
		temp = warm
	}

	sparkCells := clampInt(width/4, 8, 16)
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
	// Two lines: what the cell is doing, then where you are looking at it from.
	return style.Help.Render("  c charge  d drain  r rest  n cell  +/- load  [/] charge  "+
		"</> ambient  f speed  "+pause+"  q quit") + "\n" +
		style.Help.Render("  arrows orbit  z/x zoom  o spin")
}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }
