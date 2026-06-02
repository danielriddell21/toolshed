# VHS tape for markov. Render with: vhs docs/tapes/markov.tape
Output docs/img/markov.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 360
Set Theme "Catppuccin Mocha"
Set Padding 12

# Create a small corpus to train on (hidden from the recording).
Hide
Type "printf 'The old lighthouse stood alone on the rocky shore. Every night it cast a long beam across the dark water. Sailors trusted the light and it never failed them. Children in the village told stories about the keeper who lived there.\n' > corpus.txt"
Enter
Show

Type "markov --in corpus.txt --words 60 --seed 1 --stream" Sleep 500ms
Enter
Sleep 5s
