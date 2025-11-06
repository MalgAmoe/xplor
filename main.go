package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ================================ CONSTS ==================================

const MIN_HEIGTH = 25
const MIN_WIDTH = 60

// ================================= TYPES ==================================

type position struct {
	x int
	y int
}

type treasure struct {
	position
	visible bool
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
	rnd       *rand.Rand
	score     int
	counter   int
	delay     int
	treasure  treasure
	ticker    *time.Ticker

	// player
	player position
	alive  bool

	// borders
	topB    [MIN_WIDTH]int
	bottomB [MIN_WIDTH]int
}

// ================================= MAIN ===================================

func main() {
	gameState := setup()
	defer gameState.ticker.Stop()

	// key events
	inputCh := make(chan tcell.Key, 10)
	go handleInput(gameState.s, inputCh)

	// game loop
	for !gameState.exit {
		<-gameState.ticker.C

		update(gameState, inputCh)
		draw(gameState)

		// update speed
		if gameState.delay > 24 &&
			gameState.counter <= 0 {
			gameState.delay -= 2
			gameState.ticker.Reset(time.Duration(gameState.delay) * time.Millisecond)
		}
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
	}
	restartGame(&gameState)

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

func restartGame(g *GameState) {
	for i := range g.bottomB {
		g.topB[i] = 2
		g.bottomB[i] = MIN_HEIGTH - 2
	}
	g.rnd = rand.New(rand.NewSource(0xDEADBEEF))
	g.player = position{10, MIN_HEIGTH / 2}
	g.alive = true
	g.score = 0
	g.counter = 100
	g.delay = 70
	g.treasure.visible = false

	// when first starting there is no ticker yet
	if g.ticker != nil {
		// stop ticker for when restarting game
		g.ticker.Stop()
	}
	g.ticker = time.NewTicker(time.Duration(g.delay) * time.Millisecond)
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
			case tcell.KeyBackspace2:
				if !g.alive {
					restartGame(g)
				}
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

	if !g.alive {
		return
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

	// borders and treasures
	updateLand(g.rnd, &g.bottomB, &g.topB, &g.treasure, &g.counter)

	// check if player crash into land
	if g.player.y <= g.topB[g.player.x] || g.player.y > g.bottomB[g.player.x] {
		g.alive = false
	}

	// update game variables
	g.score += 1
	g.counter -= 1
	if g.player == g.treasure.position {
		g.treasure.visible = false
		g.score += 500
	}
}

func updateLand(rnd *rand.Rand, bB *[MIN_WIDTH]int, tB *[MIN_WIDTH]int, treasure *treasure, counter *int) {
	choice1 := rnd.Float32()
	direction1 := 0

	choice2 := rnd.Float32()
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

	// spawn new treasure
	if *counter <= 0 {
		treasure.position.x = MIN_WIDTH - 1
		treasure.position.y = tB[MIN_WIDTH-1] + 1 + int((choice1+choice2)/2*float32(bB[MIN_WIDTH-1]-tB[MIN_WIDTH-1]))
		treasure.visible = true
		*counter = 100
	}

	// update treasure
	if treasure.position.x == 0 {
		treasure.visible = false
	} else {
		treasure.position.x--
	}
}

func updateBorder(b *[MIN_WIDTH]int, direction int) {
	var newBorder int
	for i := range MIN_WIDTH - 1 {
		b[i] = b[i+1]
	}

	// direction 1 is up, 0 is flat, -1 is down
	// terminal coordinates are inverted for y axis
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

	if g.treasure.visible {
		g.s.SetContent(playAreaX+g.treasure.x, playAreaY+g.treasure.y, '@', nil, g.gameAreaStyle.Foreground(tcell.ColorGold))
	}

	// terminal size check
	if termWidth < g.minWidth || termHeight < g.minHeight {
		text := fmt.Sprintf("Terminal too small! Need at least 60x30, now is %dx%d", termWidth, termHeight)
		drawText(g.s, (termWidth-len(text))/2, 0, g.borderStyle, text)
	}

	// info
	infoText := "move with arrows. press esc to quit"
	if !g.alive {
		infoText = "press backspace to restart. press esc to quit"
	}
	drawText(g.s, (termWidth-len(infoText))/2, termHeight-1, g.borderStyle, infoText)

	scoreText := fmt.Sprintf("score: %d", g.score)
	drawText(g.s, (termWidth-len(scoreText))/2, 0, g.borderStyle, scoreText)

	if !g.alive {
		drawText(g.s, playAreaX+int(rand.Int31n(MIN_WIDTH-4)), playAreaY+int(rand.Int31n(MIN_HEIGTH)), g.borderStyle, "DEAD")
	}

	g.s.Show()
}

func drawText(s tcell.Screen, x, y int, style tcell.Style, text string) {
	for i, r := range text {
		s.SetContent(x+i, y, r, nil, style)
	}
}
