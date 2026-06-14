// Command sandbox is a terminal physics playground: bouncing balls in a box
// with gravity, walls, and elastic collisions, rendered as truecolor pixels via
// half-blocks. Move a cursor, fling balls, flip gravity, watch them settle.
package main

import (
	"github.com/danielriddell21/toolshed/internal/buildinfo"

	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/physics"
	"github.com/danielriddell21/toolshed/internal/render"
	"github.com/danielriddell21/toolshed/internal/style"
	"github.com/spf13/cobra"
)

const (
	fps        = 60
	fixedDt    = 1.0 / 120 // physics substep
	maxDt      = 0.05      // clamp elapsed time to avoid the spiral of death
	statusRows = 2         // text rows reserved below the canvas
	cursorStep = 3         // pixels moved per arrow key press
	minRadius  = 3
	maxRadius  = 7
)

var bg = render.RGB{R: 16, G: 18, B: 24}

type model struct {
	anim.Size
	world  *physics.World
	rng    *rand.Rand
	frame  *render.Frame
	cursor physics.Vec
	last   time.Time // timestamp of the previous tick
	accum  float64   // leftover time awaiting a fixed step

	gravity     float64
	restitution float64
	count       int
}

func newModel(gravity, restitution float64, count int, seed int64) model {
	return model{
		rng:         rand.New(rand.NewSource(seed)),
		frame:       render.NewFrame(2, 2),
		gravity:     gravity,
		restitution: restitution,
		count:       count,
	}
}

func (m model) Init() tea.Cmd { return anim.Frames(fps) }

// randColor returns a bright, saturated color so balls pop against the dark bg.
func randColor(rng *rand.Rand) render.RGB {
	hue := rng.Float64() * 360
	return hsv(hue, 0.7, 1.0)
}

// hsv converts HSV (h in degrees, s and v in [0,1]) to an RGB pixel.
func hsv(h, s, v float64) render.RGB {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	mm := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return render.RGB{
		R: uint8((r + mm) * 255),
		G: uint8((g + mm) * 255),
		B: uint8((b + mm) * 255),
	}
}

// initWorld builds a world to fit the current terminal and seeds it with random
// balls. Called once, on the first usable WindowSizeMsg.
func (m *model) initWorld() {
	w := float64(m.Width)
	h := float64(render.RowsToPixels(m.Height - statusRows))
	m.world = physics.NewWorld(w, h, m.gravity, m.restitution)
	m.cursor = physics.Vec{X: w / 2, Y: h / 2}
	for range m.count {
		m.spawnRandom()
	}
}

// spawnRandom adds one ball at a random spot with a gentle random velocity.
func (m *model) spawnRandom() {
	r := minRadius + m.rng.Float64()*(maxRadius-minRadius)
	pos := physics.Vec{
		X: r + m.rng.Float64()*(m.world.W-2*r),
		Y: r + m.rng.Float64()*(m.world.H-2*r),
	}
	vel := physics.Vec{
		X: (m.rng.Float64()*2 - 1) * 40,
		Y: (m.rng.Float64()*2 - 1) * 40,
	}
	m.world.Spawn(pos, vel, r, randColor(m.rng))
}

func (m *model) moveCursor(dx, dy float64) {
	if m.world == nil {
		return
	}
	m.cursor.X = math.Max(0, math.Min(m.world.W, m.cursor.X+dx))
	m.cursor.Y = math.Max(0, math.Min(m.world.H, m.cursor.Y+dy))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		if m.Ready() && m.world == nil {
			m.initWorld()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up":
			m.moveCursor(0, -cursorStep)
		case "down":
			m.moveCursor(0, cursorStep)
		case "left":
			m.moveCursor(-cursorStep, 0)
		case "right":
			m.moveCursor(cursorStep, 0)
		case " ", "f":
			if m.world != nil {
				r := minRadius + m.rng.Float64()*(maxRadius-minRadius)
				vel := physics.Vec{
					X: (m.rng.Float64()*2 - 1) * 60,
					Y: (m.rng.Float64()*2 - 1) * 60,
				}
				m.world.Spawn(m.cursor, vel, r, randColor(m.rng))
			}
		case "g":
			if m.world != nil {
				m.world.ToggleGravity()
			}
		case "c":
			if m.world != nil {
				m.world.Clear()
			}
		}
		return m, nil

	case anim.TickMsg:
		m.advance(time.Time(msg))
		return m, anim.Frames(fps)
	}
	return m, nil
}

// advance runs the fixed-timestep accumulator: turn real elapsed time into a
// whole number of fixed physics substeps.
func (m *model) advance(now time.Time) {
	if m.world == nil {
		m.last = now
		return
	}
	if m.last.IsZero() {
		m.last = now
		return
	}
	dt := now.Sub(m.last).Seconds()
	m.last = now
	m.accum += math.Min(dt, maxDt)
	for m.accum >= fixedDt {
		m.world.Step(fixedDt)
		m.accum -= fixedDt
	}
}

func (m model) View() string {
	if !m.Ready() || m.world == nil {
		return "Warming up the sandbox...\n"
	}

	m.frame.Resize(int(m.world.W), int(m.world.H))
	m.frame.Fill(bg)

	for _, b := range m.world.Balls {
		drawDisc(m.frame, b)
	}
	drawCursor(m.frame, m.cursor)

	grav := "down"
	if m.world.Gravity.Y < 0 {
		grav = "up"
	} else if m.world.Gravity.Y == 0 {
		grav = "off"
	}

	title := style.Title.Render("sandbox") + style.Note.Render("  — bouncing balls in a box")
	status := style.Help.Render(fmt.Sprintf(
		"balls: %d   gravity: %s   restitution: %.2f", len(m.world.Balls), grav, m.world.Restitution))
	hints := style.Help.Render(
		"arrows: move   space/f: spawn   g: gravity   c: clear   q: quit")

	return title + "\n" + m.frame.String() + "\n" + status + "   " + hints
}

// drawDisc paints a filled circle for a ball into the frame.
func drawDisc(f *render.Frame, b physics.Ball) {
	cx, cy, r := b.Pos.X, b.Pos.Y, b.R
	r2 := r * r
	x0, x1 := int(math.Floor(cx-r)), int(math.Ceil(cx+r))
	y0, y1 := int(math.Floor(cy-r)), int(math.Ceil(cy+r))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r2 {
				f.Set(x, y, b.Color)
			}
		}
	}
}

// drawCursor marks the cursor as a small white crosshair.
func drawCursor(f *render.Frame, c physics.Vec) {
	white := render.RGB{R: 240, G: 240, B: 240}
	x, y := int(math.Round(c.X)), int(math.Round(c.Y))
	f.Set(x, y, white)
	f.Set(x-1, y, white)
	f.Set(x+1, y, white)
	f.Set(x, y-1, white)
	f.Set(x, y+1, white)
}

func main() {
	buildinfo.HandleVersionFlag()
	var (
		gravity     float64
		restitution float64
		count       int
	)
	root := &cobra.Command{
		Use:   "sandbox",
		Short: "A terminal physics playground of bouncing balls",
		Long: "sandbox simulates balls bouncing in a box with gravity, walls, and " +
			"elastic collisions. Move the cursor, fling balls, and flip gravity.",
		RunE: func(cmd *cobra.Command, args []string) error {
			restitution = math.Max(0, math.Min(1, restitution))
			count = max(count, 0)
			m := newModel(gravity, restitution, count, time.Now().UnixNano())
			_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}
	root.Flags().Float64Var(&gravity, "gravity", 200, "downward gravity (pixels/s^2)")
	root.Flags().Float64Var(&restitution, "restitution", 0.8, "bounce energy retained [0,1]")
	root.Flags().IntVar(&count, "count", 5, "initial number of random balls")
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "sandbox:", err)
		os.Exit(1)
	}
}
