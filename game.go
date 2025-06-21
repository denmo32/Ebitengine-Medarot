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
				currentX = float32(Team1HomeX + progress*(ExecutionLineX-Team1HomeX))
			} else {
				currentX = float32(Team2HomeX - progress*(Team2HomeX-ExecutionLineX))
			}
		case StateReadyToExecuteAction:
			currentX = float32(ExecutionLineX)
		case StateActionCooldown:
			// Moving back to home position
			if medarot.Team == Team1 {
				currentX = float32(ExecutionLineX - progress*(ExecutionLineX-Team1HomeX))
			} else {
				currentX = float32(ExecutionLineX + progress*(Team2HomeX-ExecutionLineX))
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

		// --- Draw Name and HP ---
		nameStr := medarot.Name
		hpStr := fmt.Sprintf("HP: %d/%d", medarot.Parts["head"].HP, medarot.Parts["head"].MaxHP) // Simplification: show head HP as main HP
		if medarot.Parts["head"] == nil { // Safety check
			hpStr = "HP: N/A"
		}

		textYOffset := float32(IconRadius + 5)
		text.Draw(screen, nameStr, basicfont.Face7x13, int(currentX)-IconRadius*2, int(baseYPos-textYOffset), FontColor)

		// ★★★ 修正点4: hpStr変数を描画する処理を追加 ★★★
		// ※Y座標はよしなに変更してください
		text.Draw(screen, hpStr, basicfont.Face7x13, int(currentX)-IconRadius*2, int(baseYPos-textYOffset+12), FontColor)

		// --- Draw HP Bar ---
		hpBarY := baseYPos + float32(IconRadius) + 3
		hpPercentage := 0.0
		if headPart := medarot.GetPart("head"); headPart != nil && headPart.MaxHP > 0 {
			hpPercentage = float64(headPart.HP) / float64(headPart.MaxHP)
		}
		vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), hpBarY, float32(BarWidth), ChargeBarHeight, color.RGBA{50, 50, 50, 255}, true) // BG for HP Bar
		vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), hpBarY, float32(BarWidth*hpPercentage), ChargeBarHeight, HPColor, true)

		// --- Draw Charge/Cooldown Bar ---
		chargeBarY := hpBarY + HPBarHeight + 2
		barColor := ChargeColor
		currentGaugeVal := medarot.Gauge
		maxGaugeForBar := medarot.MaxGauge

		switch medarot.State {
		case StateActionCharging:
			maxGaugeForBar = medarot.CurrentActionCharge
			barColor = ChargeColor
		case StateActionCooldown:
			maxGaugeForBar = medarot.CurrentActionCooldown
			barColor = CooldownColor
		case StateIdleCharging, StateReadyToSelectAction:
			// For idle/ready, it's the "time to select" gauge
			barColor = ColorGreen // Or another distinct color for "ready"
		case StateBroken, StateReadyToExecuteAction:
			currentGaugeVal = 0 // No bar needed or full bar for ready execute
			maxGaugeForBar = 0  // Avoid division by zero
		}

		gaugePercentage := 0.0
		if maxGaugeForBar > 0 {
			gaugePercentage = currentGaugeVal / maxGaugeForBar
			if gaugePercentage > 1.0 {
				gaugePercentage = 1.0
			}
		} else if medarot.State == StateReadyToExecuteAction {
			gaugePercentage = 1.0 // Full bar when ready to execute
			barColor = ChargeColor
		}

		if medarot.State != StateBroken {
			vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), chargeBarY, float32(BarWidth), ChargeBarHeight, color.RGBA{50, 50, 50, 255}, true) // BG for Charge Bar
			vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), chargeBarY, float32(BarWidth*gaugePercentage), ChargeBarHeight, barColor, true)
		}

		// --- Draw State Text (optional, for debugging or clarity) ---
		if g.DebugMode {
			stateStr := string(medarot.State)
			if medarot.State == StateActionCharging || medarot.State == StateActionCooldown {
				if part := medarot.GetPart(medarot.SelectedPartKey); part != nil {
					stateStr += fmt.Sprintf(" (%s)", part.Name)
				}
			}
			text.Draw(screen, stateStr, basicfont.Face7x13, int(currentX)-IconRadius*3, int(baseYPos+textYOffset+15), FontColor)
		}
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