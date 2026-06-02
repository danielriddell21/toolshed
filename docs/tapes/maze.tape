# VHS tape for maze. Render with: vhs docs/tapes/maze.tape
Output docs/img/maze.gif

Set Shell "bash"
Set FontSize 16
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 10

Type "maze --generate 41x21 --algo astar --speed 40 --seed 1" Sleep 500ms
Enter
Sleep 6s
Type "q"
Sleep 500ms
