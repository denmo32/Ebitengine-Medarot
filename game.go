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
	ScreenWidth          = 800
	ScreenHeight         = 600
	PlayersPerTeam       = 3 // main.goに合わせるか、ここで定義するか要検討
	IconRadius           = 15
	IconDiameter         = IconRadius * 2
	Team1HomeX           = 100
	Team2HomeX           = ScreenWidth - 100
	ExecutionLineX       = ScreenWidth / 2
	MedarotVerticalSpacing = ScreenHeight / (PlayersPerTeam + 1) // Distribute vertically
	ChargeBarHeight      = 5
	HPBarHeight          = 5
	BarWidth             = IconDiameter * 2
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

// Layout constants for the new UI
const (
	BattlefieldHeight      = ScreenHeight / 2
	InfoPanelHeight        = ScreenHeight / 2
	InfoPanelStartY        = BattlefieldHeight
	InfoPanelPadding       = 10
	MedarotInfoBlockWidth  = (ScreenWidth - InfoPanelPadding*3) / 2 // For two columns
	MedarotInfoBlockHeight = (InfoPanelHeight - InfoPanelPadding*2) / PlayersPerTeam
	PartHPGaugeWidth       = 100
	PartHPGaugeHeight      = 8
	TextLineHeight         = 14
)

// Helper function to draw individual Medarot's info in the lower panel
func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY, blockWidth, blockHeight float32) {
	// Draw Medarot Name
	text.Draw(screen, medarot.Name, basicfont.Face7x13, int(startX), int(startY)+TextLineHeight, FontColor)

	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	partSlotDisplayNames := map[string]string{
		"head":     "頭",
		"rightArm": "右腕", // Changed for clarity
		"leftArm":  "左腕", // Changed for clarity
		"legs":     "脚部", // Changed for clarity
	}

	currentY := startY + float32(TextLineHeight*2) // Start Y for parts info

	for _, slotKey := range partSlots {
		displayName := partSlotDisplayNames[slotKey]
		hpText := "N/A"
		var hpPercentage float64 = 0
		var currentHP, maxHP int = 0

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
		text.Draw(screen, hpText, basicfont.Face7x13, int(startX), int(currentY)+TextLineHeight, FontColor)

		// Draw Part HP Gauge
		gaugeX := startX + 70 // Position gauge to the right of text
		gaugeY := currentY + float32(TextLineHeight/2) + 2

		// Background of the gauge
		vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(PartHPGaugeWidth), float32(PartHPGaugeHeight), color.RGBA{50, 50, 50, 255}, true)
		// Foreground of the gauge
		barColor := HPColor
		if currentHP == 0 {
			barColor = BrokenColor // Or a very dark red
		} else if hpPercentage < 0.3 {
			barColor = ColorRed
		} else if hpPercentage < 0.6 {
			barColor = ColorYellow
		}
		vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(PartHPGaugeWidth*hpPercentage), float32(PartHPGaugeHeight), barColor, true)

		currentY += float32(TextLineHeight) + PartHPGaugeHeight // Move Y for next part (text + gauge height)
		if currentY > startY+blockHeight-float32(TextLineHeight) { // Prevent drawing outside allocated block
			break
		}
	}
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60 [s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(BGColor)

	// --- Draw Battlefield Area (Upper Half) ---
	team1IconCount := 0
	team2IconCount := 0
	battlefieldIconYSpacing := BattlefieldHeight / (PlayersPerTeam + 1)

	for _, medarot := range g.Medarots {
		// Determine base Y position for icons in the battlefield
		var iconYPos float32
		if medarot.Team == Team1 {
			iconYPos = float32(battlefieldIconYSpacing * (team1IconCount + 1))
			team1IconCount++
		} else {
			iconYPos = float32(battlefieldIconYSpacing * (team2IconCount + 1))
			team2IconCount++
		}

		// Determine X position based on state and gauge (same logic as before)
		var currentX float32
		progress := 0.0
		// ... (Gauge calculation logic - this part remains the same as original)
		if medarot.State == StateActionCharging && medarot.CurrentActionCharge > 0 {
			progress = medarot.Gauge / medarot.CurrentActionCharge
		} else if medarot.State == StateActionCooldown && medarot.CurrentActionCooldown > 0 {
			progress = medarot.Gauge / medarot.CurrentActionCooldown
		}

		switch medarot.State {
		case StateIdleCharging, StateReadyToSelectAction:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		case StateActionCharging:
			if medarot.Team == Team1 {
				currentX = float32(Team1HomeX + progress*(ExecutionLineX-Team1HomeX))
			} else {
				currentX = float32(Team2HomeX - progress*(Team2HomeX-ExecutionLineX))
			}
		case StateReadyToExecuteAction:
			currentX = float32(ExecutionLineX)
		case StateActionCooldown:
			if medarot.Team == Team1 {
				currentX = float32(ExecutionLineX - progress*(ExecutionLineX-Team1HomeX))
			} else {
				currentX = float32(ExecutionLineX + progress*(Team2HomeX-ExecutionLineX))
			}
		case StateBroken:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		default:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		}
		if currentX < IconRadius { currentX = IconRadius }
		if currentX > ScreenWidth-IconRadius { currentX = ScreenWidth - IconRadius }


		// Draw Medarot Icon
		iconColor := Team1Color
		if medarot.Team == Team2 { iconColor = Team2Color }
		if medarot.State == StateBroken { iconColor = BrokenColor }
		vector.DrawFilledCircle(screen, currentX, iconYPos, float32(IconRadius), iconColor, true)
		if medarot.IsLeader {
			vector.StrokeCircle(screen, currentX, iconYPos, float32(IconRadius+2), 2, LeaderColor, true)
		}

		// }
	}

	// --- Draw Info Panel Area (Lower Half) ---
	team1InfoCount := 0
	team2InfoCount := 0

	// Sort Medarots by ID for stable display order in info panel (optional but good practice)
	// If g.Medarots is already sorted, this is not strictly necessary here.
	// sort.SliceStable(g.Medarots, func(i, j int) bool { return g.Medarots[i].ID < g.Medarots[j].ID })


	for _, medarot := range g.Medarots {
		var panelX, panelY float32
		var blockWidth, blockHeight float32 = float32(MedarotInfoBlockWidth), float32(MedarotInfoBlockHeight)

		if medarot.Team == Team1 {
			panelX = float32(InfoPanelPadding)
			panelY = float32(InfoPanelStartY + InfoPanelPadding + (MedarotInfoBlockHeight * float32(team1InfoCount)) + (InfoPanelPadding * float32(team1InfoCount)))
			team1InfoCount++
		} else { // Team2
			panelX = float32(InfoPanelPadding*2 + MedarotInfoBlockWidth)
			panelY = float32(InfoPanelStartY + InfoPanelPadding + (MedarotInfoBlockHeight * float32(team2InfoCount)) + (InfoPanelPadding * float32(team2InfoCount)))
			team2InfoCount++
		}

		// Ensure the block does not go off screen (especially the last one if spacing is tight)
		if panelY + blockHeight > ScreenHeight {
			// This might happen if too many players or not enough space.
			// Consider adjusting MedarotInfoBlockHeight or InfoPanelPadding
			// For now, we'll just let it draw, but in a real scenario, this needs robust handling.
		}

		drawMedarotInfo(screen, medarot, panelX, panelY, blockWidth, blockHeight)
	}


	// Draw Tick Count for debugging
	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d", g.TickCount), 10, ScreenHeight-20)
	}
}

// Layout takes the outside size (e.g., window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}