# toolshed

> *A shed full of terminal toys.*

[![CI](https://github.com/danielriddell21/toolshed/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/toolshed/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/danielriddell21/toolshed/graph/badge.svg)](https://codecov.io/gh/danielriddell21/toolshed)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danielriddell21_toolshed&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=danielriddell21_toolshed)
[![Go 1.26](https://img.shields.io/badge/go-1.26-blue)](https://go.dev)
[![MIT License](https://img.shields.io/badge/licence-MIT-green)](LICENSE)

A collection of terminal toys and generative tools, written in Go. Fractals you
can fly through, simulations that paint themselves, a maze solver that shows its
work, and a handful of small ambient things to leave running in a corner of the
screen.

Twelve binaries, one module, no fuss.

## Install

### Homebrew
```sh
brew install danielriddell21/tap/toolshed
```

### Go
```sh
go install github.com/danielriddell21/toolshed/cmd/...@latest
```

Or build from a clone:

```sh
go build ./...
```

## The tools

### Visual & generative

| Tool | What it does |
| --- | --- |
| `fractal` | Mandelbrot/Julia explorer. Truecolor half-block rendering, pan + zoom, switch to Julia using the current point as the constant. |
| `turing` | Gray-Scott reaction-diffusion. Runs the simulation and writes a PNG (and optionally an animated GIF) of the Turing pattern that emerges. |

### Simulation

| Tool | What it does |
| --- | --- |
| `life` | Conway's Game of Life, seeded deterministically from a real directory tree — the same folder always grows the same way. |
| `sandbox` | A physics sandbox. Drop balls into a box and watch them fall, bounce, collide, and settle. |
| `battery` | A cell charging and draining, drawn as a multi-storey car park you can orbit in 3D, with cars that drive themselves in and out. Kinetic battery model, Shepherd voltage curve, Arrhenius thermals, and a Peukert report. |

### Text

| Tool | What it does |
| --- | --- |
| `markov` | Trains an n-gram Markov chain on text you point it at and generates new text in that style. |

### Maze

| Tool | What it does |
| --- | --- |
| `maze` | Reads (or generates) a maze and animates BFS or A* solving it cell by cell, then traces the shortest path. |

### Ambient toys

| Tool | What it does |
| --- | --- |
| `crabs` | Races ASCII crabs across the terminal at randomised speeds. |
| `fish` | A tank containing exactly one increasingly bored fish. |
| `snail` | A Pomodoro timer in which a snail crosses the screen once per session. |
| `garden` | A zen sand garden you rake with the arrow keys. |
| `duck` | A rubber-duck debugging companion that only ever says "mhm". |

## Quick start
```sh
# Explore the Mandelbrot set (arrows pan, +/- zoom, j toggles Julia, q quits)
$ fractal

# Grow a coral reaction-diffusion pattern into a PNG and a GIF
$ turing --preset coral --steps 4000 --out coral.png --gif coral.gif

# Generate text in the style of a file
$ markov --in corpus.txt --words 120 --seed 1

# Generate a maze and watch A* solve it
$ maze --generate 41x21 --algo astar

# Hammer a lead-acid cell at 2C and watch it "run out" while still half full
$ battery --chem lead --load 2 --speed 240

# Measure the rate-capacity effect and fit a Peukert exponent
$ battery --report --chem lead
```

The interactive tools (`fractal`, `life`, `sandbox`, `battery`, `maze`, and the
ambient toys) run in the alternate screen; press `q` (or `ctrl+c`) to leave.

See [docs/demos.md](docs/demos.md) for recordings of each tool.

## Layout

A single Go module. Each tool is a thin `cmd/<name>` binary over a real,
unit-tested engine in `internal/`:

- `internal/render` — truecolor framebuffer shared by `fractal`, `life`, `sandbox`, `battery`; half-block or quadrant output, plus box-filter downsampling for anti-aliasing.
- `internal/scene` — small software 3D rasteriser: perspective camera, depth buffer, flat-shaded boxes, and a camera that frames a subject by itself.
- `internal/palette` — float→RGB gradients shared by `fractal` and `turing`.
- `internal/grid` — generic toroidal grid shared by `life` and `turing`.
- `internal/fractal`, `internal/grayscott`, `internal/maze`, `internal/markov`,
  `internal/life`, `internal/physics`, `internal/battery`, `internal/carpark` — the
  per-tool engines.
- `internal/anim`, `internal/sprite`, `internal/style` — the Bubble Tea + Lipgloss
  helpers behind the ambient toys.

```sh
go test ./...
```

## Documentation

- [Demos](docs/demos.md)
