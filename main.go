package main

import (
	"log"
	"math/rand"
	"os" // 追加
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
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
