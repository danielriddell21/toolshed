# VHS tape for battery. Render with: vhs docs/tapes/battery.tape
Output docs/img/battery.gif

Set Shell "bash"
Set FontSize 14
Set Width 1040
Set Height 430
Set Theme "Catppuccin Mocha"
Set Padding 10

Type "battery --chem lead --load 2 --speed 240" Sleep 500ms
Enter
Sleep 6s
Type "r" Sleep 5s
Type "d" Sleep 3s
Type "c" Sleep 7s
Type "q"
Sleep 500ms
