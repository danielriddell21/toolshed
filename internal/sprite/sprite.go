package sprite

import "strings"

type Direction int

const (
	Right Direction = 1
	Left  Direction = -1
)

type Sprite struct {
	X, Y      float64
	Speed     float64
	Dir       Direction
	FaceRight string
	FaceLeft  string
}

func (s Sprite) Glyph() string {
	if s.Dir == Left {
		return s.FaceLeft
	}
	return s.FaceRight
}

func (s Sprite) Width() int { return len([]rune(s.Glyph())) }

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

type Canvas struct {
	W, H int
	grid [][]rune
}

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

func (c *Canvas) Set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.grid[y][x] = r
}

func (c *Canvas) DrawString(x, y int, s string) {
	for i, r := range []rune(s) {
		c.Set(x+i, y, r)
	}
}

func (c *Canvas) DrawSprite(s Sprite) {
	c.DrawString(int(s.X+0.5), int(s.Y+0.5), s.Glyph())
}

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
