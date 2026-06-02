// Package maze provides pure (no I/O, no animation) maze parsing, generation,
// and shortest-path solving. The solver emits an ordered event log so callers
// can replay the search visually.
package maze

import (
	"bufio"
	"container/heap"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
)

// Cell is the kind of a single maze square.
type Cell uint8

const (
	Open Cell = iota
	Wall
	Start
	End
)

// Point is an (x, y) grid coordinate.
type Point struct{ X, Y int }

// Maze is a rectangular grid of cells with a designated start and end.
type Maze struct {
	W, H       int
	cells      []Cell
	Start, End Point
}

// At returns the cell at (x, y). Out-of-bounds coordinates read as Wall.
func (m *Maze) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return Wall
	}
	return m.cells[y*m.W+x]
}

func (m *Maze) set(x, y int, c Cell) {
	m.cells[y*m.W+x] = c
}

func (m *Maze) passable(x, y int) bool {
	return m.At(x, y) != Wall
}

// Parse reads a maze from r. '#' = wall, ' ' or '.' = open, 'S' = start,
// 'E' = end. Ragged lines are padded on the right with walls. It returns an
// error if the input is empty or is missing a start or end cell.
func Parse(r io.Reader) (*Maze, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	var rows []string
	width := 0
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		rows = append(rows, line)
		width = max(width, len(line))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading maze: %w", err)
	}
	if len(rows) == 0 || width == 0 {
		return nil, errors.New("empty maze input")
	}

	m := &Maze{W: width, H: len(rows), cells: make([]Cell, width*len(rows))}
	haveStart, haveEnd := false, false
	for y, row := range rows {
		for x := range width {
			var ch byte = '#'
			if x < len(row) {
				ch = row[x]
			}
			switch ch {
			case '#':
				m.set(x, y, Wall)
			case ' ', '.':
				m.set(x, y, Open)
			case 'S', 's':
				if haveStart {
					return nil, errors.New("maze has more than one start")
				}
				m.set(x, y, Start)
				m.Start = Point{x, y}
				haveStart = true
			case 'E', 'e':
				if haveEnd {
					return nil, errors.New("maze has more than one end")
				}
				m.set(x, y, End)
				m.End = Point{x, y}
				haveEnd = true
			default:
				return nil, fmt.Errorf("unrecognised character %q at row %d col %d", rune(ch), y, x)
			}
		}
	}
	if !haveStart {
		return nil, errors.New("maze has no start (S)")
	}
	if !haveEnd {
		return nil, errors.New("maze has no end (E)")
	}
	return m, nil
}

// Generate builds a solvable maze using an iterative recursive backtracker.
// Dimensions are forced odd and at least 3 so the wall/passage grid is well
// formed. Start is placed at the top-left open cell, End at the bottom-right
// open cell. rng makes generation reproducible.
func Generate(w, h int, rng *rand.Rand) *Maze {
	w = max(w, 3)
	h = max(h, 3)
	if w%2 == 0 {
		w++
	}
	if h%2 == 0 {
		h++
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	m := &Maze{W: w, H: h, cells: make([]Cell, w*h)}
	for i := range m.cells {
		m.cells[i] = Wall
	}

	// Carve on odd coordinates; walls separate the cells.
	start := Point{1, 1}
	m.set(start.X, start.Y, Open)
	stack := []Point{start}
	dirs := []Point{{0, -2}, {2, 0}, {0, 2}, {-2, 0}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		// Collect unvisited (wall) neighbours two cells away.
		var nbrs []Point
		for _, d := range dirs {
			nx, ny := cur.X+d.X, cur.Y+d.Y
			if nx > 0 && ny > 0 && nx < w-1 && ny < h-1 && m.At(nx, ny) == Wall {
				nbrs = append(nbrs, Point{nx, ny})
			}
		}
		if len(nbrs) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		next := nbrs[rng.Intn(len(nbrs))]
		// Knock down the wall between cur and next.
		m.set((cur.X+next.X)/2, (cur.Y+next.Y)/2, Open)
		m.set(next.X, next.Y, Open)
		stack = append(stack, next)
	}

	m.Start = Point{1, 1}
	m.End = Point{w - 2, h - 2}
	m.set(m.Start.X, m.Start.Y, Start)
	m.set(m.End.X, m.End.Y, End)
	return m
}

// Algo selects the search algorithm.
type Algo int

const (
	BFS Algo = iota
	AStar
)

// ParseAlgo converts a flag string into an Algo.
func ParseAlgo(s string) (Algo, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bfs":
		return BFS, nil
	case "astar", "a*":
		return AStar, nil
	default:
		return 0, fmt.Errorf("unknown algorithm %q (want bfs or astar)", s)
	}
}

// StepKind classifies an event in the search replay log.
type StepKind uint8

const (
	Visit StepKind = iota
	Frontier
	Path
)

// Step is one entry in the search event log.
type Step struct {
	P    Point
	Kind StepKind
}

var moves = [4]Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// Solve runs the chosen algorithm and returns the ordered visit/frontier event
// log, the reconstructed shortest path (Start..End inclusive), and whether a
// path was found. It is pure: no animation, no time, no I/O. Movement is
// 4-connected with equal edge weights, so both BFS and A* yield a shortest
// path. A* uses the Manhattan heuristic via container/heap.
func Solve(m *Maze, algo Algo) (steps []Step, path []Point, found bool) {
	if m == nil || m.W == 0 || m.H == 0 {
		return nil, nil, false
	}
	switch algo {
	case AStar:
		return solveAStar(m)
	default:
		return solveBFS(m)
	}
}

func index(m *Maze, p Point) int { return p.Y*m.W + p.X }

func reconstruct(m *Maze, parent []int, end Point) []Point {
	var path []Point
	for cur := index(m, end); cur != -1; cur = parent[cur] {
		path = append(path, Point{cur % m.W, cur / m.W})
	}
	// Reverse in place.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func solveBFS(m *Maze) (steps []Step, path []Point, found bool) {
	parent := make([]int, m.W*m.H)
	for i := range parent {
		parent[i] = -1
	}
	seen := make([]bool, m.W*m.H)

	queue := []Point{m.Start}
	seen[index(m, m.Start)] = true
	steps = append(steps, Step{P: m.Start, Kind: Frontier})

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		steps = append(steps, Step{P: cur, Kind: Visit})
		if cur == m.End {
			return steps, reconstruct(m, parent, m.End), true
		}
		for _, d := range moves {
			n := Point{cur.X + d.X, cur.Y + d.Y}
			if !m.passable(n.X, n.Y) {
				continue
			}
			ni := index(m, n)
			if seen[ni] {
				continue
			}
			seen[ni] = true
			parent[ni] = index(m, cur)
			queue = append(queue, n)
			steps = append(steps, Step{P: n, Kind: Frontier})
		}
	}
	return steps, nil, false
}

func manhattan(a, b Point) int {
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// pqItem is a node on the A* priority queue.
type pqItem struct {
	p     Point
	f     int // g + heuristic
	g     int
	index int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].f != pq[j].f {
		return pq[i].f < pq[j].f
	}
	return pq[i].g > pq[j].g // prefer deeper nodes on ties
}
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x any) {
	it := x.(*pqItem)
	it.index = len(*pq)
	*pq = append(*pq, it)
}
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return it
}

func solveAStar(m *Maze) (steps []Step, path []Point, found bool) {
	n := m.W * m.H
	parent := make([]int, n)
	gScore := make([]int, n)
	closed := make([]bool, n)
	for i := range parent {
		parent[i] = -1
		gScore[i] = -1
	}

	si := index(m, m.Start)
	gScore[si] = 0
	pq := &priorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &pqItem{p: m.Start, g: 0, f: manhattan(m.Start, m.End)})
	steps = append(steps, Step{P: m.Start, Kind: Frontier})

	for pq.Len() > 0 {
		cur := heap.Pop(pq).(*pqItem)
		ci := index(m, cur.p)
		if closed[ci] {
			continue
		}
		closed[ci] = true
		steps = append(steps, Step{P: cur.p, Kind: Visit})
		if cur.p == m.End {
			return steps, reconstruct(m, parent, m.End), true
		}
		for _, d := range moves {
			np := Point{cur.p.X + d.X, cur.p.Y + d.Y}
			if !m.passable(np.X, np.Y) {
				continue
			}
			ni := index(m, np)
			if closed[ni] {
				continue
			}
			tentative := cur.g + 1
			if gScore[ni] != -1 && tentative >= gScore[ni] {
				continue
			}
			gScore[ni] = tentative
			parent[ni] = ci
			heap.Push(pq, &pqItem{p: np, g: tentative, f: tentative + manhattan(np, m.End)})
			steps = append(steps, Step{P: np, Kind: Frontier})
		}
	}
	return steps, nil, false
}
