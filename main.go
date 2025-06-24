package main

import (
	// "bytes" // opentype.Parseは[]byteを直接受け取るため不要
	_ "embed" // Required for go:embed
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed MPLUS1p-Regular.ttf
var mplusFontData []byte

var MplusFont font.Face // This will be accessed by game.go

func loadFont() error {
	tt, err := opentype.Parse(mplusFontData)
	if err != nil {
		return err
	}
	const dpi = 72
	// Adjust font size as needed
	MplusFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    10,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	log.Println("Custom font loaded successfully.")
	return nil
}

func main() {
	// Load font first
	if err := loadFont(); err != nil {
		log.Fatalf("フォントの読み込みに失敗しました: %v", err)
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Log current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get current working directory: %v", err)
	} else {
		log.Printf("Current working directory: %s", wd)
	}

	// Load game data
	gameData, err := LoadAllGameData()
	if err != nil {
		log.Fatalf("Failed to load game data: %v", err)
	}
	if gameData == nil {
		log.Fatal("Game data is nil after loading.")
	}

	// Check if crucial data is loaded
	if len(gameData.Medals) == 0 {
		log.Println("Warning: No medals were loaded. Medarots might use fallback medals.")
	}
	if len(gameData.AllParts) == 0 {
		log.Println("Warning: No parts were loaded. Medarots might use placeholder parts.")
	}

	// ★★★ [変更点] Configをロード ★★★
	config := LoadConfig()

	// Create a new game instance
	game := NewGame(gameData, config) // 引数にconfigを追加
	if game == nil {
		log.Fatal("Failed to create new game instance.")
	}
	if len(game.Medarots) == 0 {
		log.Fatal("Game initialized with no Medarots.")
	}

	// ★★★ [修正] Configからウィンドウサイズを設定 ★★★
	ebiten.SetWindowSize(config.UI.Screen.Width, config.UI.Screen.Height)
	ebiten.SetWindowTitle("メダロット風ゲーム (Ebitengine)")

	// Start the game loop
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
