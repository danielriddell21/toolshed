// Package sprite renders ASCII sprites onto a text canvas and handles simple
// bounded movement. It is shared by the ambient toys.
package sprite

import "strings"

// Direction of travel along the x-axis.
type Direction int

// Facing directions along the x-axis.
const (
	Right Direction = 1
	Left  Direction = -1
)

// Sprite is a movable bit of ASCII art with a left- and right-facing glyph.
type Sprite struct {
	X, Y      float64
	Speed     float64 // cells per step, always positive
	Dir       Direction
	FaceRight string
	FaceLeft  string
}

// Glyph returns the sprite art for its current facing.
func (s Sprite) Glyph() string {
	if s.Dir == Left {
		return s.FaceLeft
	}
	return s.FaceRight
}

// Width reports the rendered width of the current glyph.
func (s Sprite) Width() int { return len([]rune(s.Glyph())) }

// Advance moves the sprite by one step in its current direction, clamping to
// [0, maxX] and reversing direction on contact with either wall.
func (s *Sprite) Advance(maxX int) {
	s.X += s.Speed * float64(s.Dir)
	right := float64(maxX - s.Width())
	if right < 0 {
		right = 0
	}
	switch {
	case s.X <= 0:
		s.X = 0
		s.Dir = Right
	case s.X >= right:
		s.X = right
		s.Dir = Left
	}
}

// Canvas is a fixed-size grid of runes for compositing sprites and scenery.
type Canvas struct {
	W, H int
	grid [][]rune
}

// NewCanvas returns a blank canvas filled with spaces.
func NewCanvas(w, h int) *Canvas {
	w = max(w, 1)
	h = max(h, 1)
	grid := make([][]rune, h)
	for y := range grid {
		row := make([]rune, w)
		for x := range row {
			row[x] = ' '
		}
		grid[y] = row
	}
	return &Canvas{W: w, H: h, grid: grid}
}

// Set writes a single rune if it lies within bounds.
func (c *Canvas) Set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.grid[y][x] = r
}

// DrawString writes s starting at (x, y), clipping at the right edge.
func (c *Canvas) DrawString(x, y int, s string) {
	for i, r := range []rune(s) {
		c.Set(x+i, y, r)
	}
}

// DrawSprite composites a sprite's current glyph at its rounded position.
func (c *Canvas) DrawSprite(s Sprite) {
	c.DrawString(int(s.X+0.5), int(s.Y+0.5), s.Glyph())
}

// String renders the canvas to a newline-joined block.
func (c *Canvas) String() string {
	var b strings.Builder
	for y, row := range c.grid {
		b.WriteString(strings.TrimRight(string(row), " "))
		if y < len(c.grid)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
