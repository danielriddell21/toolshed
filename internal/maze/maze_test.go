package maze

import (
	"math/rand"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	src := strings.Join([]string{
		"#####",
		"#S..#",
		"#.#.#",
		"#..E#",
		"#####",
	}, "\n")
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.W != 5 || m.H != 5 {
		t.Fatalf("dims = %dx%d, want 5x5", m.W, m.H)
	}
	if m.Start != (Point{1, 1}) {
		t.Errorf("Start = %v, want {1,1}", m.Start)
	}
	if m.End != (Point{3, 3}) {
		t.Errorf("End = %v, want {3,3}", m.End)
	}
	if m.At(0, 0) != Wall {
		t.Errorf("corner = %v, want Wall", m.At(0, 0))
	}
	if m.At(1, 1) != Start {
		t.Errorf("At(1,1) = %v, want Start", m.At(1, 1))
	}
	if m.At(2, 1) != Open {
		t.Errorf("At(2,1) = %v, want Open", m.At(2, 1))
	}
	if m.At(2, 2) != Wall {
		t.Errorf("At(2,2) = %v, want Wall", m.At(2, 2))
	}
}

func TestParseRaggedPadding(t *testing.T) {
	// Short lines should pad with walls; bounds read as Wall.
	src := "S\n.E"
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.W != 2 || m.H != 2 {
		t.Fatalf("dims = %dx%d, want 2x2", m.W, m.H)
	}
	if m.At(1, 0) != Wall {
		t.Errorf("padded cell = %v, want Wall", m.At(1, 0))
	}
}

func TestParseMalformed(t *testing.T) {
	cases := map[string]string{
		"empty":    "",
		"no start": "###\n#E#\n###",
		"no end":   "###\n#S#\n###",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(src)); err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestParseNoPanicOnGarbage(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parse panicked: %v", r)
		}
	}()
	if _, err := Parse(strings.NewReader("S@E")); err == nil {
		t.Fatalf("expected error for garbage char")
	}
}

func TestGenerateSolvable(t *testing.T) {
	for seed := int64(0); seed < 25; seed++ {
		rng := rand.New(rand.NewSource(seed))
		m := Generate(21, 21, rng)
		if m.At(m.Start.X, m.Start.Y) != Start {
			t.Fatalf("seed %d: start cell not Start", seed)
		}
		if m.At(m.End.X, m.End.Y) != End {
			t.Fatalf("seed %d: end cell not End", seed)
		}
		for _, algo := range []Algo{BFS, AStar} {
			_, path, found := Solve(m, algo)
			if !found {
				t.Fatalf("seed %d algo %d: no path in generated maze", seed, algo)
			}
			if path[0] != m.Start || path[len(path)-1] != m.End {
				t.Fatalf("seed %d: path endpoints wrong: %v..%v", seed, path[0], path[len(path)-1])
			}
		}
	}
}

func TestGenerateOddDims(t *testing.T) {
	m := Generate(20, 10, rand.New(rand.NewSource(1)))
	if m.W%2 == 0 || m.H%2 == 0 {
		t.Fatalf("dims not forced odd: %dx%d", m.W, m.H)
	}
}

func TestSolveShortestPath(t *testing.T) {
	// Hand-built maze. Shortest path S(1,1) -> E(3,3) length is 5 cells.
	src := strings.Join([]string{
		"#####",
		"#S..#",
		"#.#.#",
		"#..E#",
		"#####",
	}, "\n")
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	_, bfsPath, bfsFound := Solve(m, BFS)
	_, aPath, aFound := Solve(m, AStar)
	if !bfsFound || !aFound {
		t.Fatalf("both algorithms must find a path: bfs=%v astar=%v", bfsFound, aFound)
	}
	if len(bfsPath) != len(aPath) {
		t.Fatalf("path lengths differ: bfs=%d astar=%d", len(bfsPath), len(aPath))
	}
	if len(bfsPath) != 5 {
		t.Fatalf("shortest path length = %d, want 5", len(bfsPath))
	}
	if bfsPath[0] != m.Start || bfsPath[len(bfsPath)-1] != m.End {
		t.Fatalf("bfs endpoints wrong: %v..%v", bfsPath[0], bfsPath[len(bfsPath)-1])
	}
}

func TestSolveUnreachable(t *testing.T) {
	src := strings.Join([]string{
		"#####",
		"#S#E#",
		"#####",
	}, "\n")
	m, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, _, found := Solve(m, BFS); found {
		t.Fatalf("expected no path through a wall")
	}
	if _, _, found := Solve(m, AStar); found {
		t.Fatalf("expected no path through a wall (astar)")
	}
}

func TestSolveRecordsVisits(t *testing.T) {
	m := Generate(15, 15, rand.New(rand.NewSource(3)))
	steps, _, found := Solve(m, BFS)
	if !found {
		t.Fatal("no path")
	}
	var visits int
	for _, s := range steps {
		if s.Kind == Visit {
			visits++
		}
	}
	if visits == 0 {
		t.Fatal("no Visit steps recorded")
	}
}

func TestParseAlgo(t *testing.T) {
	if a, err := ParseAlgo("bfs"); err != nil || a != BFS {
		t.Errorf("bfs => %v, %v", a, err)
	}
	if a, err := ParseAlgo("ASTAR"); err != nil || a != AStar {
		t.Errorf("ASTAR => %v, %v", a, err)
	}
	if _, err := ParseAlgo("dijkstra"); err == nil {
		t.Errorf("expected error for unknown algo")
	}
}
