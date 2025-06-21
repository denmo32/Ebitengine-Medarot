package main

import (
	"fmt" // 重複を削除
	"image/color"
	"log"
	"math/rand"
	"sort" // For sorting Medarots by ID for stable display order

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector" // For drawing shapes like circles
	"golang.org/x/image/font/basicfont"       // Using a basic font for now
)

const (
	ScreenWidth          = 960 // Changed to 16:9
	ScreenHeight         = 540 // Changed to 16:9
	PlayersPerTeam       = 3   // main.goに合わせるか、ここで定義するか要検討
	IconRadius           = 15
	IconDiameter         = IconRadius * 2
	Team1HomeX           = 100
	Team2HomeX           = ScreenWidth - 100
	// ExecutionLineX       = ScreenWidth / 2 // Will be split for Team1 and Team2
	MedarotIconExecutionOffset = float32(IconRadius + 5) // Offset for icons at execution line
	Team1ExecutionLineX      = float32(ScreenWidth/2) - MedarotIconExecutionOffset
	Team2ExecutionLineX      = float32(ScreenWidth/2) + MedarotIconExecutionOffset
	BattlefieldHeight        = float32(ScreenHeight * 0.4) // Reduced to 40% of ScreenHeight
	InfoPanelHeight          = float32(ScreenHeight * 0.6) // Increased to 60% of ScreenHeight
	MedarotVerticalSpacing   = BattlefieldHeight / (PlayersPerTeam + 1) // Distribute vertically in battlefield

	// ChargeBarHeight      = 5 // No longer used
	// HPBarHeight          = 5 // No longer used for overall HP bar
	// BarWidth             = IconDiameter * 2 // No longer used for overall HP bar

	// Info Panel Layout Constants
	InfoPanelStartY        = BattlefieldHeight
	InfoPanelPadding       = float32(10)
	MedarotInfoBlockWidth  = (float32(ScreenWidth) - InfoPanelPadding*3) / 2
	MedarotInfoBlockHeight = (InfoPanelHeight - InfoPanelPadding*2) / float32(PlayersPerTeam) // Recalculated based on new InfoPanelHeight
	PartHPGaugeWidth       = float32(100)
	PartHPGaugeHeight      = float32(7)  // Reduced from 8
	TextLineHeight         = float32(12) // Reduced from 14, to match font size 10 better
	HomeMarkerRadius       = float32(IconRadius / 3)
	LineWidth              = float32(1) // Line thickness for borders etc.
)

var (
	ColorWhite    = color.White
	ColorBlack    = color.Black
	ColorRed      = color.RGBA{R: 255, G: 100, B: 100, A: 255}
	ColorBlue     = color.RGBA{R: 100, G: 100, B: 255, A: 255}
	ColorGreen    = color.RGBA{R: 100, G: 255, B: 100, A: 255}
	ColorYellow   = color.RGBA{R: 255, G: 255, B: 100, A: 255}
	ColorGray     = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	Team1Color    = ColorBlue
	Team2Color    = ColorRed
	LeaderColor   = ColorYellow                                  // For leader distinction
	BrokenColor   = ColorGray
	HPColor       = ColorGreen
	ChargeColor   = ColorYellow
	CooldownColor = color.RGBA{R: 180, G: 180, B: 255, A: 255} // Light blue for cooldown bar
	FontColor     = ColorWhite
	BGColor       = color.NRGBA{R: 0x1a, G: 0x20, B: 0x2c, A: 0xff} // Dark background
)

// Game implements ebiten.Game interface.
type Game struct {
	Medarots  []*Medarot
	GameData  *GameData
	TickCount int
	DebugMode bool // To toggle debug messages
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
		Medarots:  medarots,
		GameData:  gameData,
		TickCount: 0,
		DebugMode: true, // Enable debug prints by default initially
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
	screen.Fill(BGColor) // Use defined background color

	team1Count := 0
	team2Count := 0

	for _, medarot := range g.Medarots {
		// Determine base Y position based on team and order
		var baseYPos float32
		if medarot.Team == Team1 {
			baseYPos = float32(MedarotVerticalSpacing * (team1Count + 1))
			team1Count++
		} else {
			baseYPos = float32(MedarotVerticalSpacing * (team2Count + 1))
			team2Count++
		}

		// Determine X position based on state and gauge
		var currentX float32
		progress := 0.0
		if medarot.State == StateActionCharging && medarot.CurrentActionCharge > 0 {
			progress = medarot.Gauge / medarot.CurrentActionCharge
		} else if medarot.State == StateActionCooldown && medarot.CurrentActionCooldown > 0 {
			progress = medarot.Gauge / medarot.CurrentActionCooldown
		} else if (medarot.State == StateIdleCharging || medarot.State == StateReadyToSelectAction) && medarot.MaxGauge > 0 {
			// For idle, let's consider it "progress towards selection" but positionally static at home
			// progress = medarot.Gauge / medarot.MaxGauge // This would make it move during idle, not desired for home.
		}

		switch medarot.State {
		case StateIdleCharging, StateReadyToSelectAction:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		case StateActionCharging:
			if medarot.Team == Team1 {
				currentX = float32(Team1HomeX + progress*(Team1ExecutionLineX-float32(Team1HomeX)))
			} else {
				currentX = float32(Team2HomeX - progress*(float32(Team2HomeX)-Team2ExecutionLineX))
			}
		case StateReadyToExecuteAction:
			if medarot.Team == Team1 {
				currentX = Team1ExecutionLineX
			} else {
				currentX = Team2ExecutionLineX
			}
		case StateActionCooldown:
			// Moving back to home position
			if medarot.Team == Team1 {
				currentX = float32(Team1ExecutionLineX - progress*(Team1ExecutionLineX-float32(Team1HomeX)))
			} else {
				currentX = float32(Team2ExecutionLineX + progress*(float32(Team2HomeX)-Team2ExecutionLineX))
			}
		case StateBroken:
			// Stay at home position if broken, or last known if that's preferred
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		default:
			currentX = float32(Team1HomeX) // Default fallback
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		}

		// Clamp X to screen bounds just in case
		if currentX < IconRadius {
			currentX = IconRadius
		}
		if currentX > ScreenWidth-IconRadius {
			currentX = ScreenWidth - IconRadius
		}

		// --- Draw Medarot Icon (Circle) ---
		iconColor := Team1Color
		if medarot.Team == Team2 {
			iconColor = Team2Color
		}
		if medarot.State == StateBroken {
			iconColor = BrokenColor
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, float32(IconRadius), iconColor, true)

		if medarot.IsLeader { // Draw leader indicator (e.g., a yellow border or smaller circle)
			// ★★★ 修正点3: 関数名を `StrokeCircle` に変更 ★★★
			vector.StrokeCircle(screen, currentX, baseYPos, float32(IconRadius+2), float32(2), LeaderColor, true)
		}

		// --- Draw Medarot Icon (Circle) --- (Name and other texts are moved to info panel)
		iconColor := Team1Color
		if medarot.Team == Team2 {
			iconColor = Team2Color
		}
		if medarot.State == StateBroken {
			iconColor = BrokenColor
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, float32(IconRadius), iconColor, true)

		if medarot.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, float32(IconRadius+2), LineWidth*2, LeaderColor, true) // Use LineWidth
		}
	}

	// --- Draw Battlefield Borders ---
	vector.StrokeRect(screen, 0, 0, float32(ScreenWidth), BattlefieldHeight, LineWidth, FontColor, false)

	// --- Draw Home Markers ---
	// Team 1 Home Markers
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := float32(MedarotVerticalSpacing * (float32(i) + 1))
		vector.StrokeCircle(screen, float32(Team1HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
	}
	// Team 2 Home Markers
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := float32(MedarotVerticalSpacing * (float32(i) + 1))
		vector.StrokeCircle(screen, float32(Team2HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
	}

	// --- Draw Execution Lines ---
	vector.StrokeLine(screen, Team1ExecutionLineX, 0, Team1ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
	vector.StrokeLine(screen, Team2ExecutionLineX, 0, Team2ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)


	// --- Draw Info Panel Area (Lower Half) ---
	// This will be implemented by re-adding drawMedarotInfo and calling it in a loop for each medarot,
	// similar to the previous new-battle-ui-layout branch, but adapted for current codebase.
	// For now, let's add a placeholder.
	// ebitenutil.DebugPrintAt(screen, "Info Panel Area", int(InfoPanelPadding), int(InfoPanelStartY+InfoPanelPadding))

	team1InfoCount := 0
	team2InfoCount := 0
	// Ensure Medarots are sorted by ID for consistent display order if not already
	// sort.SliceStable(g.Medarots, func(i, j int) bool { return g.Medarots[i].ID < g.Medarots[j].ID })


	for _, medarot := range g.Medarots {
		var panelX, panelY float32
		var blockWidth, blockHeight float32 = MedarotInfoBlockWidth, MedarotInfoBlockHeight // Use defined constants

		if medarot.Team == Team1 {
			panelX = InfoPanelPadding
			panelY = InfoPanelStartY + InfoPanelPadding + (MedarotInfoBlockHeight * float32(team1InfoCount)) + (InfoPanelPadding * float32(team1InfoCount))
			team1InfoCount++
		} else { // Team2
			panelX = InfoPanelPadding*2 + MedarotInfoBlockWidth
			panelY = InfoPanelStartY + InfoPanelPadding + (MedarotInfoBlockHeight * float32(team2InfoCount)) + (InfoPanelPadding * float32(team2InfoCount))
			team2InfoCount++
		}

		// Prevent drawing outside allocated screen space (simple check)
		if panelY + blockHeight > float32(ScreenHeight) {
			// log.Printf("Warning: Medarot info for %s might be drawn off-screen.", medarot.Name)
			continue // Skip drawing if it's going to be completely off-screen or overlaps too much
		}

		drawMedarotInfo(screen, medarot, panelX, panelY, blockWidth, blockHeight)
	}


	// Draw Tick Count for debugging
	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d", g.TickCount), 10, ScreenHeight-15) // 少し上に調整
	}
}


// Helper function to draw individual Medarot's info in the lower panel
func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY, blockWidth, blockHeight float32) {
	// Draw Medarot Name
	text.Draw(screen, medarot.Name, MplusFont, int(startX), int(startY)+int(TextLineHeight), FontColor)

	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	partSlotDisplayNames := map[string]string{
		"head":     "頭部",
		"rightArm": "右腕",
		"leftArm":  "左腕",
		"legs":     "脚部",
	}

	currentInfoY := startY + TextLineHeight*2 // Start Y for parts info, below name

	for _, slotKey := range partSlots {
		// Ensure we don't draw more than 4 parts if somehow more are defined, or if space runs out
		if currentInfoY + TextLineHeight + PartHPGaugeHeight > startY + blockHeight {
			break
		}

		displayName := partSlotDisplayNames[slotKey]
		hpText := "N/A"
		var hpPercentage float64 = 0
		currentHP, maxHP := 0, 0

		if part, ok := medarot.Parts[slotKey]; ok && part != nil {
			currentHP = part.HP
			maxHP = part.MaxHP
			hpText = fmt.Sprintf("%s: %d/%d", displayName, currentHP, maxHP)
			if maxHP > 0 {
				hpPercentage = float64(currentHP) / float64(maxHP)
			}
		} else {
			hpText = fmt.Sprintf("%s: N/A", displayName)
		}

		// Draw Part HP Text
		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY)+int(TextLineHeight), FontColor)

		// Draw Part HP Gauge
		gaugeX := startX + 80 // Position gauge to the right of text (adjusted)
		// gaugeY := currentInfoY + (TextLineHeight / 2) // Aligns gauge middle with text middle
		gaugeY := currentInfoY + TextLineHeight - PartHPGaugeHeight // Aligns gauge bottom with text baseline

		// Background of the gauge
		vector.DrawFilledRect(screen, gaugeX, gaugeY, PartHPGaugeWidth, PartHPGaugeHeight, color.RGBA{50, 50, 50, 255}, true)
		// Foreground of the gauge
		barFillColor := HPColor // Default to green
		if currentHP == 0 {
			barFillColor = BrokenColor
		} else if hpPercentage < 0.3 {
			barFillColor = ColorRed
		} else if hpPercentage < 0.6 {
			barFillColor = ColorYellow
		}
		vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(PartHPGaugeWidth)*hpPercentage), PartHPGaugeHeight, barFillColor, true)

		currentInfoY += TextLineHeight + PartHPGaugeHeight // No extra margin between part entries
	}
}


// Layout takes the outside size (e.g., window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}