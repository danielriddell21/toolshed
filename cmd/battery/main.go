package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/battery"
	"github.com/danielriddell21/toolshed/internal/carpark"
	"github.com/danielriddell21/toolshed/internal/cli"
	"github.com/danielriddell21/toolshed/internal/render"
	"github.com/danielriddell21/toolshed/internal/scene"
)

const (
	fps          = 20
	maxStepS     = 1.0
	maxFrameS    = 0.25
	history      = 400
	rateStep     = 0.25
	minCRate     = 0.05
	maxCRate     = 10
	tempStep     = 5
	minAmbient   = -20
	maxAmbient   = 55
	kelvin       = 273.15
	sceneChrome  = 10
	orbitStep    = 0.18
	zoomStep     = 0.12
	maxDriveGain = 6.0
)

var speeds = []float64{1, 10, 60, 300, 900, 3600}

type model struct {
	anim.Size
	sim    *battery.Sim
	speed  float64
	paused bool
	last   time.Time
	lane   float64
	volts  []float64
	amps   []float64

	park     *carpark.Park
	frame    *render.Frame
	hires    *render.Frame
	target   *scene.Target
	solid    bool
	spin     bool
	camYaw   float64
	camPitch float64
	camZoom  float64
}

func newModel(sim *battery.Sim, speed float64, solid bool) model {
	frame := render.NewFrame(2, 2)
	hires := render.NewFrame(2, 2)
	target := scene.NewTarget(hires)
	target.PixelAspect = pixelAspect
	m := model{
		sim:      sim,
		speed:    speed,
		solid:    solid,
		spin:     true,
		frame:    frame,
		hires:    hires,
		target:   target,
		camYaw:   -0.85,
		camPitch: 0.24,
		camZoom:  1.04,
	}
	m.rebuildPark()
	return m
}

func (m *model) rebuildPark() {
	m.park = carpark.New(deckLevels, deckPerSide, m.sim.Cell.Chem.FrontFraction)
	m.park.Sync(m.sim.Cell.SurfaceSoC(), m.sim.Cell.DeepAh/m.sim.Cell.DeepCapacityAh())
}

func (m model) Init() tea.Cmd { return anim.Frames(fps) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case anim.TickMsg:
		m.advance(time.Time(msg))
		return m, anim.Frames(fps)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case " ":
		m.paused = !m.paused
	case "c":
		m.sim.SetMode(battery.Charging)
	case "d":
		m.sim.SetMode(battery.Draining)
	case "r":
		m.sim.SetMode(battery.Rest)
	case "+", "=":
		m.sim.LoadCRate = clampRate(m.sim.LoadCRate + rateStep)
	case "-", "_":
		m.sim.LoadCRate = clampRate(m.sim.LoadCRate - rateStep)
	case "]":
		m.sim.ChargeCRate = clampRate(m.sim.ChargeCRate + rateStep)
	case "[":
		m.sim.ChargeCRate = clampRate(m.sim.ChargeCRate - rateStep)
	case ">", ".":
		m.setAmbient(m.sim.Cell.AmbientK - kelvin + tempStep)
	case "<", ",":
		m.setAmbient(m.sim.Cell.AmbientK - kelvin - tempStep)
	case "f":
		m.speed = nextSpeed(m.speed)
	case "n":
		m.reset(nextChemistry(m.sim.Cell.Chem.Name))
	default:
		m.handleCamera(msg.String())
	}
	return m, nil
}

func (m *model) handleCamera(key string) {
	switch key {
	case "v":
		m.solid = !m.solid
	case "o":
		m.spin = !m.spin
	case "left":
		m.orbit(-orbitStep, 0, 0)
	case "right":
		m.orbit(orbitStep, 0, 0)
	case "up":
		m.orbit(0, orbitStep/2, 0)
	case "down":
		m.orbit(0, -orbitStep/2, 0)
	case "z":
		m.orbit(0, 0, zoomStep)
	case "x":
		m.orbit(0, 0, -zoomStep)
	}
}

// reset swaps in another chemistry and starts it full, keeping the load, the
// charge rate and the weather exactly as they were so the two cells can be
// compared under identical conditions.
func (m *model) reset(chem battery.Chemistry) {
	sim := battery.NewSim(chem, 1, m.sim.Cell.AmbientK, m.sim.ChargeCRate, m.sim.LoadCRate)
	sim.SetMode(m.sim.Mode)
	m.sim = sim
	m.volts, m.amps = nil, nil
	m.rebuildPark()
}

func nextChemistry(current string) battery.Chemistry {
	names := battery.Names()
	next := names[0]
	for i, name := range names {
		if name == current {
			next = names[(i+1)%len(names)]
			break
		}
	}
	chem, _ := battery.Named(next)
	return chem
}

func (m *model) setAmbient(celsius float64) {
	m.sim.Cell.AmbientK = math.Max(minAmbient, math.Min(maxAmbient, celsius)) + kelvin
}

func (m *model) advance(now time.Time) {
	if m.last.IsZero() {
		m.last = now
		return
	}
	elapsed := now.Sub(m.last).Seconds()
	m.last = now
	if m.paused {
		return
	}

	// Sub-step so that winding the clock forward never coarsens the simulation:
	// the model always sees steps of at most a second.
	remaining := math.Min(elapsed, maxFrameS) * m.speed
	for remaining > 0 {
		step := math.Min(remaining, maxStepS)
		m.sim.Advance(step)
		remaining -= step
	}

	m.lane += math.Abs(m.sim.CurrentA)/m.sim.Cell.Chem.CapacityAh + 0.2
	m.volts = push(m.volts, m.sim.Cell.Terminal(m.sim.CurrentA))
	m.amps = push(m.amps, m.sim.CurrentA)

	// Traffic runs close to real time whatever the clock is doing: cars cannot
	// drive at 240x, so winding forward settles bays directly instead.
	m.park.Sync(m.sim.Cell.SurfaceSoC(), m.sim.Cell.DeepAh/m.sim.Cell.DeepCapacityAh())
	m.park.Step(math.Min(elapsed, maxFrameS) * math.Max(1, math.Min(maxDriveGain, m.speed/60)))
	if m.spin {
		m.camYaw += spinRate * elapsed
	}
}

func push(series []float64, v float64) []float64 {
	series = append(series, v)
	if len(series) > history {
		series = series[len(series)-history:]
	}
	return series
}

func (m model) View() string {
	if !m.Ready() {
		return "Opening the car park...\n"
	}
	g := layout(m.Width, m.sim.Cell.Chem.FrontFraction)
	parts := []string{m.headerView(), m.settingsView(), ""}
	if m.solid {
		parts = append(parts, m.sceneView(m.Height-sceneChrome))
	} else {
		parts = append(parts, m.garageView(g), m.laneView(g))
	}
	return strings.Join(append(parts, "", m.panelView(g.inner()), "", m.helpView()), "\n")
}

func clampRate(v float64) float64 {
	return math.Max(minCRate, math.Min(maxCRate, math.Round(v/rateStep)*rateStep))
}

func nextSpeed(current float64) float64 {
	for _, s := range speeds {
		if s > current {
			return s
		}
	}
	return speeds[0]
}

func parseMode(name string) (battery.Mode, error) {
	switch name {
	case "rest":
		return battery.Rest, nil
	case "charge":
		return battery.Charging, nil
	case "drain":
		return battery.Draining, nil
	}
	return battery.Rest, fmt.Errorf("unknown mode %q; choose one of: rest, charge, drain", name)
}

var reportColumns = []struct {
	head  string
	width int
}{
	{"rate", 6},
	{"current", 9},
	{"runtime", 8},
	{"delivered", 9},
	{"of rated", 8},
	{"energy", 8},
	{"mean V", 8},
	{"end T", 8},
}

func reportRow(cells []string, note string) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = fmt.Sprintf("%*s", reportColumns[i].width, c)
	}
	return "  " + strings.Join(parts, "  ") + note
}

func report(chem battery.Chemistry, ambientC float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n%.2f Ah rated, %.2f V nominal, discharged to %.2f V at %.1f °C\n\n",
		chem.Name, chem.Blurb, chem.CapacityAh, chem.VNom, chem.VCut, ambientC)

	heads := make([]string, len(reportColumns))
	for i, c := range reportColumns {
		heads[i] = c.head
	}
	fmt.Fprintln(&b, reportRow(heads, ""))

	rates := []float64{0.1, 0.2, 0.5, 1, 2, 3, 5}
	runs := make([]battery.Run, 0, len(rates))
	var throttled bool
	for _, rate := range rates {
		r := battery.RunToEmpty(chem, rate, ambientC+kelvin, 1)
		runs = append(runs, r)
		throttled = throttled || r.Throttled

		mark := ""
		if r.Throttled {
			mark = " †"
		}
		fmt.Fprintln(&b, reportRow([]string{
			fmt.Sprintf("%.2fC", r.CRate),
			fmt.Sprintf("%.3f A", r.CurrentA),
			runtime(r.Hours),
			fmt.Sprintf("%.3f Ah", r.DeliveredAh),
			fmt.Sprintf("%.1f%%", 100*r.DeliveredAh/chem.CapacityAh),
			fmt.Sprintf("%.2f Wh", r.EnergyWh),
			fmt.Sprintf("%.3f V", r.MeanV),
			fmt.Sprintf("%.1f °C", r.EndTempC),
		}, mark))
	}

	fmt.Fprintf(&b, "\n  Peukert exponent n = %.3f\n"+
		"  Runtime falls off as t = a·I⁻ⁿ. n = 1 is a cell that never notices how hard\n"+
		"  you pull on it; every real cell sits above it.\n",
		battery.PeukertExponent(runs))
	if throttled {
		fmt.Fprint(&b, "  † throttled by pack temperature, so it never held its nominal\n"+
			"    current and is left out of the fit.\n")
	}
	return b.String()
}

func runtime(hours float64) string {
	d := time.Duration(hours * float64(time.Hour))
	if d >= time.Hour {
		return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func main() {
	var (
		chemName string
		soc      float64
		ambient  float64
		load     float64
		charge   float64
		speed    float64
		modeName string
		flat     bool
		asReport bool
	)

	root := &cobra.Command{
		Use:   "battery",
		Short: "Simulate a battery charging and draining as a multi-storey car park",
		Long: "battery models a cell as a car park: charge parks in bays, the ones nearest " +
			"the ramp can leave immediately, and the rest need time to reach it. That is the " +
			"kinetic battery model, and it is why a hard drain reaches empty early and why a " +
			"flat cell recovers if you leave it alone.",
		RunE: func(cmd *cobra.Command, args []string) error {
			chem, ok := battery.Named(chemName)
			if !ok {
				return fmt.Errorf("unknown chemistry %q; choose one of: %s",
					chemName, strings.Join(battery.Names(), ", "))
			}
			mode, err := parseMode(modeName)
			if err != nil {
				return err
			}

			if asReport {
				fmt.Print(report(chem, ambient))
				return nil
			}

			sim := battery.NewSim(chem, soc, ambient+kelvin, clampRate(charge), clampRate(load))
			sim.SetMode(mode)
			m := newModel(sim, speed, !flat)
			if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
				return fmt.Errorf("run program: %w", err)
			}
			return nil
		},
	}

	root.Flags().StringVar(&chemName, "chem", "liion",
		"cell chemistry ("+strings.Join(battery.Names(), ", ")+")")
	root.Flags().Float64Var(&soc, "soc", 1, "starting state of charge [0,1]")
	root.Flags().Float64Var(&ambient, "ambient", 25, "ambient temperature (°C)")
	root.Flags().Float64Var(&load, "load", 1, "discharge rate (C)")
	root.Flags().Float64Var(&charge, "charge", 1, "charge rate (C)")
	root.Flags().Float64Var(&speed, "speed", 60, "simulated seconds per real second")
	root.Flags().StringVar(&modeName, "mode", "drain", "starting mode (rest, charge, drain)")
	root.Flags().BoolVar(&flat, "flat", false,
		"draw the flat instrument view instead of the 3D structure")
	root.Flags().BoolVar(&asReport, "report", false,
		"print a rate-capacity table and Peukert exponent instead of running the demo")

	cli.Execute(root)
}
