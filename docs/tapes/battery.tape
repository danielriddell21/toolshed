# VHS tape for battery. Render with: vhs docs/tapes/battery.tape
Output docs/img/battery.gif

Set Shell "bash"
Set FontSize 12
Set Width 1100
Set Height 620
Set Theme "Catppuccin Mocha"
Set Padding 10

Type "battery --load 2 --speed 60" Sleep 500ms
Enter
Sleep 6s
Type "o" Sleep 7s
Type "c" Sleep 9s
Type "v" Sleep 4s
Type "q"
Sleep 500ms
