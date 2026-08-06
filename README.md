# toolshed

> *toolshed* — a shed full of terminal toys.

[![CI](https://github.com/danielriddell21/toolshed/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/toolshed/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/danielriddell21/toolshed/graph/badge.svg)](https://codecov.io/gh/danielriddell21/toolshed)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danielriddell21_toolshed&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=danielriddell21_toolshed)
[![Go 1.26](https://img.shields.io/badge/go-1.26-blue)](https://go.dev)
[![MIT License](https://img.shields.io/badge/licence-MIT-green)](LICENSE)

A collection of terminal toys and generative tools, written in Go. Fractals you can fly through, simulations that paint themselves, a maze solver that shows its work, and a handful of small ambient things to leave running in a corner of the screen.

Eleven binaries, one module, no fuss.

## Tools

| Tool | What it does |
|---|---|
| `fractal` | Mandelbrot/Julia explorer with pan, zoom and Julia toggle |
| `turing` | Gray-Scott reaction-diffusion, written to a PNG or GIF |
| `life` | Conway's Game of Life, seeded deterministically from a directory tree |
| `sandbox` | A physics sandbox — drop balls and watch them settle |
| `markov` | Trains an n-gram Markov chain and generates text in that style |
| `maze` | Animates BFS or A* solving a maze, then traces the shortest path |
| `crabs` | Races ASCII crabs across the terminal |
| `fish` | A tank containing exactly one increasingly bored fish |
| `snail` | A Pomodoro timer in which a snail crosses the screen once per session |
| `garden` | A zen sand garden you rake with the arrow keys |
| `duck` | A rubber-duck debugging companion that only ever says "mhm" |

## Install

### Homebrew
```sh
brew install danielriddell21/tap/toolshed
```

### Go install
```sh
go install github.com/danielriddell21/toolshed/cmd/...@latest
```

### From source
```sh
go build ./...
```

## Quick start

```sh
# Explore the Mandelbrot set (arrows pan, +/- zoom, j toggles Julia, q quits)
fractal

# Grow a coral reaction-diffusion pattern into a PNG and a GIF
turing --preset coral --steps 4000 --out coral.png --gif coral.gif

# Generate text in the style of a file
markov --in corpus.txt --words 120 --seed 1

# Generate a maze and watch A* solve it
maze --generate 41x21 --algo astar
```

The interactive tools run in the alternate screen; press `q` (or `ctrl+c`) to leave.

## Documentation

Full documentation lives in the [toolshed wiki](https://github.com/danielriddell21/toolshed/wiki) — usage for every tool, the shared engines behind them, and recordings of each.
