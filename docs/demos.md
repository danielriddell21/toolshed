# Demos

Recordings of every tool. Each GIF is generated from a [VHS][vhs] tape in
[`docs/tapes/`](tapes); regenerate them with:

```sh
scripts/render-demos.sh
```

(That builds the tools into `./bin`, puts them on `PATH`, and runs `vhs` over
every tape. VHS needs `ttyd` and `ffmpeg` installed.)

## Visual & generative

### fractal

A Mandelbrot/Julia explorer rendered with truecolor half-blocks. Arrow keys pan,
`+`/`-` zoom, `j` toggles Julia mode using the current point as the constant,
`[`/`]` cycle palettes.

![fractal demo](img/fractal.gif)

### turing

A Gray-Scott reaction-diffusion simulation. It runs the model and writes the
resulting Turing pattern to a PNG, optionally with an animated GIF of its
evolution.

![turing demo](img/turing.gif)

## Simulation

### life

Conway's Game of Life, seeded from a real directory tree so the same folder
always produces the same starting pattern. Space pauses, `n` steps, `r` reseeds.

![life demo](img/life.gif)

### sandbox

A physics sandbox: drop balls into a box and watch them fall, bounce, collide,
and settle. `g` flips gravity, `c` clears.

![sandbox demo](img/sandbox.gif)

## Text

### markov

Trains an n-gram Markov chain on whatever text you point it at and generates new
text in that style. `--stream` prints it word by word.

![markov demo](img/markov.gif)

## Maze

### maze

Reads or generates a maze and animates BFS or A* solving it, then traces the
shortest path in a highlight color.

![maze demo](img/maze.gif)

## Ambient toys

### crabs

ASCII crabs race left to right at randomised speeds. There is a winner and no
prize.

![crabs demo](img/crabs.gif)

### fish

A tank with exactly one fish. It drifts, faces where it's going, blows bubbles,
and is bored.

![fish demo](img/fish.gif)

### snail

A Pomodoro timer in which a single snail crosses the screen once over the whole
session. Space pauses, `q` quits.

![snail demo](img/snail.gif)

### garden

A zen sand garden raked with the arrow keys. Cycle patterns with `p`, reset with
`r`. No score, no end.

![garden demo](img/garden.gif)

### duck

A rubber-duck debugging companion that only ever says "mhm", growing more
skeptical the more you explain.

![duck demo](img/duck.gif)

[vhs]: https://github.com/charmbracelet/vhs
