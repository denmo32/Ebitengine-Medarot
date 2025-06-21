package main

import (
	"bytes" // Required for bytes.Reader
	_ "embed" // Required for go:embed
	"log"
	"math/rand"
	"os" // 追加
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed mplus-1p-regular.ttf
var mplusFontData []byte

var MplusFont font.Face // This will be accessed by game.go

func loadFont() error {
	tt, err := opentype.Parse(mplusFontData)
	if err != nil {
		return err
	}
	const dpi = 72
	// Adjust font size as needed, 12 might be too small/large depending on new resolution
	MplusFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    10, // Changed from 12 to 10
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
	if len(gameData.AllParts["head"]) == 0 && 
	   len(gameData.AllParts["rightArm"]) == 0 &&
	   len(gameData.AllParts["leftArm"]) == 0 &&
	   len(gameData.AllParts["legs"]) == 0 {
		log.Println("Warning: No parts were loaded for any slot. Medarots might use placeholder parts.")
	}


	// Create a new game instance
	game := NewGame(gameData)
	if game == nil {
		log.Fatal("Failed to create new game instance.")
	}
	if len(game.Medarots) == 0 {
		log.Fatal("Game initialized with no Medarots.")
	}


	// Set window properties
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Medarot Ebitengine Port")

	// Start the game loop
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
