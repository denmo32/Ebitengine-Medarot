package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sort" // For sorting Medarots by ID for stable display order

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont" // Using a basic font for now
)

const (
	ScreenWidth  = 800
	ScreenHeight = 600
)

// Game implements ebiten.Game interface.
type Game struct {
	Medarots   []*Medarot
	GameData   *GameData
	TickCount  int
	DebugMode  bool // To toggle debug messages
}

// NewGame creates a new Game instance.
func NewGame(gameData *GameData) *Game {
	medarots := InitializeAllMedarots(gameData)
	if len(medarots) == 0 {
		log.Fatal("No medarots were initialized. Exiting.")
	}

	// Sort Medarots by ID for consistent display order if needed, especially for drawing.
	sort.Slice(medarots, func(i, j int) bool {
		return medarots[i].ID < medarots[j].ID
	})

	return &Game{
		Medarots:   medarots,
		GameData:   gameData,
		TickCount:  0,
		DebugMode:  true, // Enable debug prints by default initially
	}
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	g.TickCount++

	for _, medarot := range g.Medarots {
		// Handle state-dependent actions before general update
		switch medarot.State {
		case StateReadyToSelectAction:
			if medarot.State == StateBroken { // Double check, should be caught by Medarot.Update
				continue
			}
			availableParts := medarot.GetAvailableAttackParts()
			if len(availableParts) > 0 {
				// Simple AI: select a random available part
				selectedPart := availableParts[rand.Intn(len(availableParts))]
				if g.DebugMode && g.TickCount%60 == 0 { // Log less frequently
					log.Printf("Game Update: %s (%s) is selecting action. Attempting to use %s (Slot: %s).\n", medarot.Name, medarot.ID, selectedPart.Name, selectedPart.Slot)
				}
				success := medarot.SelectAction(selectedPart.Slot)
				if success && g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) successfully selected %s. Now %s.\n", medarot.Name, medarot.ID, selectedPart.Name, medarot.State)
				} else if !success && g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) failed to select %s.\n", medarot.Name, medarot.ID, selectedPart.Name)
				}
			} else {
				// No available parts, perhaps set to broken or a special state.
				// For now, it might just stay in ReadyToSelectAction or be handled by its own Update.
				if g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) is ReadyToSelectAction but has no available attack parts.\n", medarot.Name, medarot.ID)
				}
				// If head is not broken, but no attack parts, it means arms are broken.
				// It should still be able to cool down or charge idle if it was in another state.
				// If it has no actions, it effectively becomes idle until head is broken.
				// Let Medarot.Update handle if it should become broken due to head.
			}

		case StateReadyToExecuteAction:
			if medarot.State == StateBroken { // Double check
				continue
			}
			if g.DebugMode && g.TickCount%60 == 0 {
				log.Printf("Game Update: %s (%s) is executing action: %s.\n", medarot.Name, medarot.ID, medarot.Parts[medarot.SelectedPartKey].Name)
			}
			success := medarot.ExecuteAction()
			if success && g.DebugMode && g.TickCount%60 == 0 {
				log.Printf("Game Update: %s (%s) successfully executed. Now %s.\n", medarot.Name, medarot.ID, medarot.State)
			} else if !success && g.DebugMode && g.TickCount%60 == 0 {
                 log.Printf("Game Update: %s (%s) failed to execute action.\n", medarot.Name, medarot.ID)
            }
		}

		// General update for gauge, state changes due to time passing
		medarot.Update()
	}
	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60 [s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 0x1a, G: 0x20, B: 0x2c, A: 0xff}) // Dark background

	yPos := 20
	for _, medarot := range g.Medarots {
		// Basic Medarot Info
		info := fmt.Sprintf("ID: %s (%s) - %s", medarot.ID, medarot.Name, medarot.Team)
		text.Draw(screen, info, basicfont.Face7x13, 20, yPos, color.White)
		yPos += 15

		status := fmt.Sprintf("  State: %s, Gauge: %.1f", medarot.State, medarot.Gauge)
		if medarot.State == StateActionCharging {
			status += fmt.Sprintf("/%.0f (Charging %s)", medarot.CurrentActionCharge, medarot.SelectedPartKey)
		} else if medarot.State == StateActionCooldown {
			status += fmt.Sprintf("/%.0f (Cooldown)", medarot.CurrentActionCooldown)
		} else if medarot.State == StateIdleCharging || medarot.State == StateReadyToSelectAction {
			status += fmt.Sprintf("/%.0f", medarot.MaxGauge)
		}
		text.Draw(screen, status, basicfont.Face7x13, 20, yPos, color.White)
		yPos += 15

		// Parts Info
		for _, slotKey := range []string{"head", "rightArm", "leftArm", "legs"} {
			part, exists := medarot.Parts[slotKey]
			partInfo := fmt.Sprintf("    %s: ", slotKey)
			if exists && part != nil {
				partInfo += fmt.Sprintf("%s (HP: %d/%d", part.Name, part.HP, part.MaxHP)
				if part.IsBroken {
					partInfo += " BROKEN"
				}
				if slotKey != "legs" { // Legs don't typically have charge/cooldown in the same way for "actions"
					partInfo += fmt.Sprintf(", CHG: %d, CD: %d", part.Charge, part.Cooldown)
				}
				partInfo += ")"
			} else {
				partInfo += "<NONE>"
			}
			text.Draw(screen, partInfo, basicfont.Face7x13, 20, yPos, color.White)
			yPos += 15
		}
		yPos += 10 // Extra space between medarots
	}

	// Draw Tick Count for debugging
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d", g.TickCount), 10, ScreenHeight-20)
}

// Layout takes the outside size (e.g., window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}
