package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

/*
steps for dev:
x 1. Static scene - Draw centered borders and fixed terrain just to get layout right
x 2. Add ship - Draw ship at fixed position, allow up/down movement with bounds checking
x 3. Scrolling terrain - Make terrain scroll left, generate simple flat terrain on right
x 4. Random generation - Add randomness to terrain with constraints
5. Collision detection - Check ship vs terrain, show game over
6. Game loop timing - Add proper frame timing
7. Polish - Score, restart, difficulty, colors, instructions
*/

// ================================ CONSTS ==================================

const MIN_HEIGTH = 25
const MIN_WIDTH = 60

// ================================= TYPES ==================================

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

	// borders
	topB    [MIN_WIDTH]int
	bottomB [MIN_WIDTH]int
}

// ================================= MAIN ===================================

func main() {
	gameState := setup()

	// key events
	inputCh := make(chan tcell.Key, 10)
	go handleInput(gameState.s, inputCh)

	// update frame based on ticker
	ticker := time.NewTicker(66 * time.Millisecond)
	defer ticker.Stop()

	// game loop
	for !gameState.exit {
		<-ticker.C

		update(gameState, inputCh)
		draw(gameState)
	}

	gameState.s.Fini()
	os.Exit(0)
}

// ================================= SETUP ==================================

func setup() *GameState {
	// create screen
	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// setup style
	borderStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorPink)
	gameAreaStyle := tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorBlueViolet)
	s.SetStyle(borderStyle)

	// basic gameState
	gameState := GameState{
		s:             s,
		borderStyle:   borderStyle,
		gameAreaStyle: gameAreaStyle,
		minWidth:      MIN_WIDTH,
		minHeight:     MIN_HEIGTH,
		player:        position{10, MIN_HEIGTH / 2},
	}

	// add borders
	for i := range gameState.bottomB {
		gameState.topB[i] = 2
		gameState.bottomB[i] = MIN_HEIGTH - 2
	}

	return &gameState
}

// =============================== UPDATES ==================================

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
	// player position
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

	// borders
	updateAllBorders(&g.bottomB, &g.topB)
}

func updateAllBorders(bB *[MIN_WIDTH]int, tB *[MIN_WIDTH]int) {
	choice1 := rand.Float32()
	direction1 := 0

	choice2 := rand.Float32()
	direction2 := 0

	// check top border range
	if choice1 < 0.4 && (tB[MIN_WIDTH-1]) < MIN_HEIGTH-3 {
		direction1 = -1
	} else if choice1 > 0.6 && tB[MIN_WIDTH-1] > 3 {
		direction1 = 1
	}

	// check bottom border range
	if choice2 < 0.3 && bB[MIN_WIDTH-1] < MIN_HEIGTH-3 {
		direction2 = -1
	} else if choice2 > 0.6 && (bB[MIN_WIDTH-1]) > 3 {
		direction2 = 1
	}

	// avoid borders overlap
	if tB[MIN_WIDTH-1]+5 >= (bB[MIN_WIDTH-1]) {
		direction1 = 1
		direction2 = -1
	}

	updateBorder(tB, direction1)
	updateBorder(bB, direction2)
}

func updateBorder(b *[MIN_WIDTH]int, direction int) {
	var newBorder int
	for i := range MIN_WIDTH - 1 {
		b[i] = b[i+1]
	}

	switch direction {
	case 0:
		newBorder = b[MIN_WIDTH-2]
	case 1:
		newBorder = b[MIN_WIDTH-2] - 1
	case -1:
		newBorder = b[MIN_WIDTH-2] + 1
	}

	b[MIN_WIDTH-1] = newBorder
}

// ================================= DRAW ===================================

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
			// draw land if other part of the border
			if (y-playAreaY) > g.bottomB[x-playAreaX] ||
				(y-playAreaY) <= g.topB[x-playAreaX] {
				g.s.SetContent(x, y, '#', nil, g.gameAreaStyle)
			} else {
				g.s.SetContent(x, y, ' ', nil, g.gameAreaStyle)
			}
		}
	}

	// player
	g.s.SetContent(playerX, playerY, '>', nil, g.gameAreaStyle.Foreground(tcell.ColorPink))

	// terminal size check
	if termWidth < g.minWidth || termHeight < g.minHeight {
		text := fmt.Sprintf("Terminal too small! Need at least 60x30, now is %dx%d", termWidth, termHeight)
		drawText(g.s, (termWidth-len(text))/2, 0, g.borderStyle, text)
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
