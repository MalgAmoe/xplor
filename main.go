package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

/*
some stuff:
terrain: 2 arrays or slices
gap size calculation
circular buffer for terrain
tick timing for set fps?
ticking game loop, poll events in goroutine

steps for dev:
1. Static scene - Draw centered borders and fixed terrain just to get layout right
2. Add ship - Draw ship at fixed position, allow up/down movement with bounds checking
3. Scrolling terrain - Make terrain scroll left, generate simple flat terrain on right
4. Random generation - Add randomness to terrain with constraints
5. Collision detection - Check ship vs terrain, show game over
6. Game loop timing - Add proper frame timing
7. Polish - Score, restart, difficulty, colors, instructions
*/
const MIN_HEIGTH = 25
const MIN_WIDTH = 60

type position struct {
	x int
	y int
}

type GameState struct {
	// terminal values
	s             tcell.Screen
	borderStyle   tcell.Style
	gameAreaStyle tcell.Style

	// game status
	exit      bool
	minWidth  int
	minHeight int

	// player
	player position
}

func main() {
	// init screen
	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	borderStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	gameAreaStyle := tcell.StyleDefault.Background(tcell.ColorDarkBlue)
	s.SetStyle(borderStyle)

	gameState := GameState{
		s:             s,
		borderStyle:   borderStyle,
		gameAreaStyle: gameAreaStyle,
		minWidth:      MIN_WIDTH,
		minHeight:     MIN_HEIGTH,
		player:        position{10, MIN_HEIGTH / 2},
	}

	// key events
	inputCh := make(chan tcell.Key, 10)
	go handleInput(gameState.s, inputCh)

	// update frame based on ticker
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()

	// game loop
	for !gameState.exit {
		<-ticker.C

		update(&gameState, inputCh)
		draw(&gameState)
	}

	gameState.s.Fini()
}

func handleInput(s tcell.Screen, inputCh chan tcell.Key) {
	for {
		ev := s.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			inputCh <- ev.Key()

		case *tcell.EventResize:
			s.Sync()
		}
	}
}

func update(g *GameState, inputCh <-chan tcell.Key) {

outer:
	for {
		select {
		case key := <-inputCh:
			switch key {
			case tcell.KeyESC:
				g.exit = true
			case tcell.KeyUp:
				g.player.y -= 1
			case tcell.KeyDown:
				g.player.y += 1
			case tcell.KeyLeft:
				g.player.x -= 1
			case tcell.KeyRight:
				g.player.x += 1
			}
		default:
			break outer
		}
	}

	if g.player.y <= 0 {
		g.player.y = 0
	} else if g.player.y >= g.minHeight-1 {
		g.player.y = g.minHeight - 1
	}

	if g.player.x <= 0 {
		g.player.x = 0
	} else if g.player.x >= g.minWidth-1 {
		g.player.x = g.minWidth - 1
	}
}

func draw(g *GameState) {
	// offset calculations
	termWidth, termHeight := g.s.Size()
	playAreaX := (termWidth - g.minWidth) / 2
	playAreaY := (termHeight - g.minHeight) / 2

	playerX := playAreaX + g.player.x
	playerY := playAreaY + g.player.y

	g.s.Clear()

	// draw game screen
	for y := playAreaY; y < playAreaY+g.minHeight; y++ {
		for x := playAreaX; x < playAreaX+g.minWidth; x++ {
			g.s.SetContent(x, y, ' ', nil, g.gameAreaStyle)
		}
	}

	// player
	g.s.SetContent(playerX, playerY, '>', nil, g.gameAreaStyle.Foreground(tcell.ColorYellow))

	// terminal size check
	if termWidth < g.minWidth || termHeight < g.minHeight {
		text := fmt.Sprintf("Terminal too small! Need at least 60x30, now is %dx%d", termWidth, termHeight)
		drawText(g.s, 1, 1, g.borderStyle, text)
	}

	// info
	drawText(g.s, (termWidth-35)/2, termHeight-1, g.borderStyle, "move with arrows. press esc to quit")

	g.s.Show()
}

func drawText(s tcell.Screen, x, y int, style tcell.Style, text string) {
	for i, r := range text {
		s.SetContent(x+i, y, r, nil, style)
	}
}
