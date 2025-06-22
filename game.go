package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font" // Added for text measurement
)

// ★リファクタリング3: マジックナンバーを定数化
const (
	// --- Screen ---
	ScreenWidth  = 960
	ScreenHeight = 540

	// --- Game Logic ---
	PlayersPerTeam = 3

	// --- Battlefield Layout ---
	IconRadius                 = 15
	Team1HomeX                 = 100
	Team2HomeX                 = ScreenWidth - 100
	MedarotIconExecutionOffset = float32(IconRadius + 5)
	Team1ExecutionLineX        = float32(ScreenWidth/2) - MedarotIconExecutionOffset
	Team2ExecutionLineX        = float32(ScreenWidth/2) + MedarotIconExecutionOffset
	BattlefieldHeight          = float32(ScreenHeight * 0.4)
	MedarotVerticalSpacing     = BattlefieldHeight / (PlayersPerTeam + 1)
	HomeMarkerRadius           = float32(IconRadius / 3)
	LineWidth                  = float32(1)

	// --- Info Panel Layout ---
	InfoPanelStartY        = BattlefieldHeight
	InfoPanelHeight        = float32(ScreenHeight * 0.6)
	InfoPanelPadding       = float32(10)
	MedarotInfoBlockWidth  = (float32(ScreenWidth) - InfoPanelPadding*3) / 2
	MedarotInfoBlockHeight = (InfoPanelHeight - (InfoPanelPadding * (float32(PlayersPerTeam) + 1))) / float32(PlayersPerTeam)
	PartHPGaugeWidth       = float32(100)
	PartHPGaugeHeight      = float32(7)
	TextLineHeight         = float32(12)
	PartHPGaugeOffsetX     = float32(80) // 「80」という数字を定数にした
)

// (Color definitions and Game struct are unchanged)
var (
	ColorWhite    = color.White
	ColorBlack    = color.Black
	ColorRed      = color.RGBA{R: 255, G: 100, B: 100, A: 255}
	ColorBlue     = color.RGBA{R: 100, G: 100, B: 255, A: 255}
	ColorGreen    = color.RGBA{R: 100, G: 255, B: 100, A: 255}
	ColorYellow   = color.RGBA{R: 255, G: 255, B: 100, A: 255}
	ColorGray     = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	ColorOrange   = color.RGBA{R: 255, G: 165, B: 0, A: 255} // For message window
	Team1Color    = ColorBlue
	Team2Color    = ColorRed
	LeaderColor   = ColorYellow
	BrokenColor   = ColorGray
	HPColor       = ColorGreen
	ChargeColor   = ColorYellow
	CooldownColor = color.RGBA{R: 180, G: 180, B: 255, A: 255}
	FontColor     = ColorWhite
	BGColor       = color.NRGBA{R: 0x1a, G: 0x20, B: 0x2c, A: 0xff}
)

type Game struct {
	Medarots  []*Medarot
	GameData  *GameData
	TickCount int
	DebugMode bool

	// New fields for player input and message window
	CurrentGameState       GameState
	ActiveMedarotForInput *Medarot // Medarot currently waiting for player input
	Message               string
	ShowMessageWindow     bool
	InputCallback         func() // Callback to execute after input

	InitialActionSelectionPhase bool
	PendingInitialSelections    int
}

// GameState represents the different states of the game flow
type GameState int

const (
	GameStateRunning GameState = iota
	GameStateWaitForInput
	GameStatePlayerActionSelection // New state for when player is selecting an action
)

// (NewGame and Update are unchanged)
func NewGame(gameData *GameData) *Game {
	medarots := InitializeAllMedarots(gameData)
	if len(medarots) == 0 {
		log.Fatal("No medarots were initialized. Exiting.")
	}
	sort.Slice(medarots, func(i, j int) bool {
		return medarots[i].ID < medarots[j].ID
	})
	return &Game{
		Medarots:               medarots,
		GameData:               gameData,
		TickCount:              0,
		DebugMode:              true,
		CurrentGameState:       GameStateRunning,
		ActiveMedarotForInput:  nil,
		Message:                   "",
		ShowMessageWindow:         false,
		InputCallback:             nil,
		InitialActionSelectionPhase: true,
		PendingInitialSelections:    0, // Will be calculated below
	}

	// Calculate pending initial selections
	pendingSelections := 0
	for _, m := range game.Medarots {
		if m.State != StateBroken { // Assuming only non-broken medarots select actions initially
			pendingSelections++
		}
	}
	game.PendingInitialSelections = pendingSelections
	// If no medarots can select actions, bypass the initial phase
	if game.PendingInitialSelections == 0 {
		game.InitialActionSelectionPhase = false
	}


	return game
}

func (g *Game) Update() error {
	g.TickCount++

	// Handle player input if waiting for regular messages or action execution confirmation
	if g.CurrentGameState == GameStateWaitForInput {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			// If there's a specific callback (like executing an action), run it.
			// Otherwise, just resume the game.
			// Check if the input was for player action selection message
			if g.ActiveMedarotForInput != nil && g.ActiveMedarotForInput.Team == Team1 && (g.Message == fmt.Sprintf("%sの行動を選択してください。", g.ActiveMedarotForInput.Name) || g.Message == fmt.Sprintf("[初期選択] %sの行動を選択してください。(残り%d機)", g.ActiveMedarotForInput.Name, g.PendingInitialSelections)) {
				// This click is to acknowledge the "select action" message, now show parts or allow selection.
				// For now, we simplify: this click will proceed to the simplified part selection logic below.
				// If we had a more complex UI for part selection, this would transition to that UI state.
				g.ShowMessageWindow = false // Hide the initial "select action" prompt
				// The actual selection logic is handled below in GameStatePlayerActionSelection
			} else {
				g.CurrentGameState = GameStateRunning
				g.ShowMessageWindow = false
				if g.InputCallback != nil {
					g.InputCallback()
					g.InputCallback = nil
				}
			}
		}
		// If waiting for input (general message), or if it's the initial selection phase and waiting for player to start selection,
		// skip further game logic updates for this frame.
		if !g.InitialActionSelectionPhase { // If not in initial phase, and waiting for input, definitely skip.
			return nil
		}
		// If in initial phase AND waiting for input (e.g. player team1's turn message), also skip medarot updates.
		if g.InitialActionSelectionPhase && g.ShowMessageWindow {
			return nil
		}
	}

	// Handle player action selection input (this is simplified)
	if g.CurrentGameState == GameStatePlayerActionSelection {
		// This is where the player would interact with a UI to select a part.
		// Simplified: a click selects the first available part.
		// This click is DIFFERENT from the one that closes a general message window.
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.ActiveMedarotForInput != nil {
			if g.ActiveMedarotForInput.Team == Team1 {
				availableParts := g.ActiveMedarotForInput.GetAvailableAttackParts()
				if len(availableParts) > 0 {
					selectedPart := availableParts[0]
					log.Printf("Player input: %s selected %s.", g.ActiveMedarotForInput.Name, selectedPart.Name)
					success := g.ActiveMedarotForInput.SelectAction(selectedPart.Slot)
					if success {
						if g.InitialActionSelectionPhase {
							g.PendingInitialSelections--
							log.Printf("Initial Selection: Player %s selected. Pending: %d", g.ActiveMedarotForInput.Name, g.PendingInitialSelections)
						}
					}
				} else {
					log.Printf("Player input: %s has no available parts. Skipping selection.", g.ActiveMedarotForInput.Name)
					if g.InitialActionSelectionPhase {
						g.PendingInitialSelections-- // Still counts as a "turn" taken for selection
						log.Printf("Initial Selection: Player %s no parts. Pending: %d", g.ActiveMedarotForInput.Name, g.PendingInitialSelections)
					}
				}
				g.CurrentGameState = GameStateRunning
				g.ShowMessageWindow = false
				g.ActiveMedarotForInput = nil
			}
		}
		// If player is selecting, don't run other game logic for this frame to avoid interference.
		// Unless it's the initial phase and we are waiting for other AI to also select.
		if !g.InitialActionSelectionPhase && g.ActiveMedarotForInput != nil { // if not initial phase and a player is actively selecting, pause.
			return nil
		}
	}

	// Main game logic loop for Medarots
	for _, medarot := range g.Medarots {
		if g.CurrentGameState == GameStatePlayerActionSelection && g.ActiveMedarotForInput == medarot {
			// This medarot is currently being handled by player selection input block or waiting for that input.
			// If in initial phase, we let other medarots proceed with their selection logic.
			// If not in initial phase, other medarots also proceed.
			// The key is that THIS medarot's Update() is skipped later if it's the ActiveMedarotForInput.
		}

		originalState := medarot.State

		switch medarot.State {
		case StateReadyToSelectAction:
			if medarot.State == StateBroken {
				continue
			}

			if medarot.Team == Team1 {
				// Only trigger player selection if it's not already the active medarot being handled by GameStatePlayerActionSelection
				// and if game is not paused by another message window.
				if g.ActiveMedarotForInput != medarot && g.CurrentGameState != GameStateWaitForInput {
					g.CurrentGameState = GameStatePlayerActionSelection
					g.ActiveMedarotForInput = medarot
					g.ShowMessageWindow = true
					if g.InitialActionSelectionPhase {
						g.Message = fmt.Sprintf("[初期選択] %sの行動を選択してください。(残り%d機)", medarot.Name, g.PendingInitialSelections)
					} else {
						g.Message = fmt.Sprintf("%sの行動を選択してください。", medarot.Name)
					}
					log.Printf("Setup: Player %s to select action. Pending initial: %d", medarot.Name, g.PendingInitialSelections)
				}
			} else { // Team 2 (AI)
				// AI selects action only if not in a state where player input is blocking (general message or player selection)
				// and if it's its turn in the initial phase or initial phase is over.
				if g.CurrentGameState == GameStateRunning || (g.InitialActionSelectionPhase && g.CurrentGameState != GameStatePlayerActionSelection) {
					availableParts := medarot.GetAvailableAttackParts()
					if len(availableParts) > 0 {
						selectedPart := availableParts[rand.Intn(len(availableParts))]
						success := medarot.SelectAction(selectedPart.Slot)
						if success {
							log.Printf("Game Update: AI %s (%s) selected %s. Now %s.", medarot.Name, medarot.ID, selectedPart.Name, medarot.State)
							if g.InitialActionSelectionPhase && originalState == StateReadyToSelectAction && medarot.State != StateReadyToSelectAction {
								g.PendingInitialSelections--
								log.Printf("Initial Selection: AI %s selected. Pending: %d", medarot.Name, g.PendingInitialSelections)
							}
						}
					} else {
						if g.InitialActionSelectionPhase && originalState == StateReadyToSelectAction {
							// If AI has no parts, it effectively "passes".
							// We need to ensure its state changes or it's handled so PendingInitialSelections decrements.
							// For now, assume if it stays ReadyToSelectAction, it won't decrement.
							// This implies a medarot *must* select an action to count.
							// A better approach: if no parts, Medarot.SelectAction(nil) or similar should transition state.
							// Let's assume for now, if no parts, it doesn't select and thus doesn't decrement.
							// This might require adjusting PendingInitialSelections calculation or how "no action" is handled.
							// Alternative: if no parts, it still "completes" its selection turn.
							g.PendingInitialSelections--
							log.Printf("Initial Selection: AI %s no parts. Pending: %d", medarot.Name, g.PendingInitialSelections)
                            // HACK: Prevent re-selection loop if no parts by forcing a state change.
                            // This medarot effectively "passes" its turn to select.
                            // It should ideally go into a short cooldown or back to idle.
                            medarot.State = StateIdleCharging
                            medarot.Gauge = 0
						}
					}
				}
			}

		case StateReadyToExecuteAction:
			if medarot.State == StateBroken {
				continue
			}
			// If in initial selection phase, and not all have selected, Medarot.Update() will be skipped later, so charge won't start.
			// Action execution itself is also blocked by message window.
			if g.InitialActionSelectionPhase && g.PendingInitialSelections > 0 {
                 // Don't set up execution message if initial selection is pending
				continue
			}

			if !g.ShowMessageWindow && g.CurrentGameState == GameStateRunning {
				g.ShowMessageWindow = true
				g.Message = fmt.Sprintf("%sが%sで攻撃します。", medarot.Name, medarot.Parts[medarot.SelectedPartKey].Name)
				g.CurrentGameState = GameStateWaitForInput
				currentMedarot := medarot
				g.InputCallback = func() {
					log.Printf("Game Update: %s (%s) is executing action: %s.\n", currentMedarot.Name, currentMedarot.ID, currentMedarot.Parts[currentMedarot.SelectedPartKey].Name)
					success := currentMedarot.ExecuteAction()
					// Result message will be set up here
					resultMessage := ""
					if success {
						log.Printf("Game Update: %s (%s) successfully executed. Now %s.\n", currentMedarot.Name, currentMedarot.ID, currentMedarot.State)
						resultMessage = fmt.Sprintf("%sの攻撃完了！", currentMedarot.Name)
					} else {
						log.Printf("Game Update: %s (%s) failed to execute action.\n", currentMedarot.Name, currentMedarot.ID)
						resultMessage = fmt.Sprintf("%sの行動は失敗した...", currentMedarot.Name)
					}

					// Nested wait for input to show result
					g.ShowMessageWindow = true
					g.Message = resultMessage
					g.CurrentGameState = GameStateWaitForInput
					g.InputCallback = nil // This inner callback is just to proceed past result.
				}
				return nil
			}
		}

		// Conditions for skipping Medarot.Update():
		// 1. Initial selection phase is active AND not all Medarots have selected actions yet.
		// 2. Game is waiting for player's general input (message window is up).
		// 3. Game is waiting for the current Medarot (g.ActiveMedarotForInput) to make a selection.
		//    (but allow other medarots to update if it's initial phase and this isn't the one being selected)
		shouldSkipMedarotUpdate := false
		if g.InitialActionSelectionPhase && g.PendingInitialSelections > 0 {
			shouldSkipMedarotUpdate = true
		} else if g.CurrentGameState == GameStateWaitForInput && g.ShowMessageWindow {
			shouldSkipMedarotUpdate = true
		} else if g.CurrentGameState == GameStatePlayerActionSelection && g.ActiveMedarotForInput == medarot {
			shouldSkipMedarotUpdate = true
		}

		if !shouldSkipMedarotUpdate {
			medarot.Update()
		}
	}

	if g.InitialActionSelectionPhase && g.PendingInitialSelections <= 0 {
		log.Println("Initial action selection phase complete. All Medarots will now update normally.")
		g.InitialActionSelectionPhase = false
		// Ensure game state allows normal progression
		if g.CurrentGameState != GameStateWaitForInput { // Don't override if a message is already up
			g.CurrentGameState = GameStateRunning
		}
	}

	return nil
}

// ★リファクタリング1: Draw関数を責務ごとに分割
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(BGColor)

	g.drawBattlefield(screen)
	g.drawMedarotIcons(screen)
	g.drawInfoPanels(screen)

	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d", g.TickCount), 10, ScreenHeight-15)
	}

	// Draw message window if needed
	if g.ShowMessageWindow {
		g.drawMessageWindow(screen)
	}
}

func (g *Game) drawMessageWindow(screen *ebiten.Image) {
	// Window dimensions and position (centered)
	windowWidth := float32(ScreenWidth * 0.6)
	windowHeight := float32(ScreenHeight * 0.2)
	windowX := (float32(ScreenWidth) - windowWidth) / 2
	windowY := (float32(ScreenHeight) - windowHeight) / 2

	// Background
	bgColor := color.NRGBA{R: 0, G: 0, B: 0, A: 200} // Semi-transparent black
	vector.DrawFilledRect(screen, windowX, windowY, windowWidth, windowHeight, bgColor, true)

	// Border
	borderColor := ColorOrange
	vector.StrokeRect(screen, windowX, windowY, windowWidth, windowHeight, 2, borderColor, false)

	// Message Text
	if MplusFont == nil {
		log.Println("Error in drawMessageWindow: MplusFont is nil")
		ebitenutil.DebugPrintAt(screen, "Font not loaded!", int(windowX+InfoPanelPadding), int(windowY+InfoPanelPadding))
		return
	}

	// Basic text wrapping (very simple)
	// For more complex wrapping, a dedicated library or more sophisticated logic would be needed.
	maxCharsPerLine := int(windowWidth-InfoPanelPadding*2) / 6 // Approximate char width
	wrappedText := wrapText(g.Message, maxCharsPerLine)

	textY := windowY + InfoPanelPadding + TextLineHeight
	for _, line := range wrappedText {
		text.Draw(screen, line, MplusFont, int(windowX+InfoPanelPadding), int(textY), FontColor)
		textY += TextLineHeight + 2 // Line spacing
		if textY > windowY+windowHeight-InfoPanelPadding {
			break // Stop if text exceeds window height
		}
	}

	// Prompt for input
	promptMsg := "クリックして続行..."
	promptTextWidth := float32(font.MeasureString(MplusFont, promptMsg).Ceil())
	promptX := windowX + (windowWidth-promptTextWidth)/2
	promptY := windowY + windowHeight - InfoPanelPadding - TextLineHeight/2
	text.Draw(screen, promptMsg, MplusFont, int(promptX), int(promptY), FontColor)

}

// Helper function for basic text wrapping
func wrapText(message string, maxCharsPerLine int) []string {
	var lines []string
	if len(message) == 0 || maxCharsPerLine <= 0 {
		return []string{message}
	}

	words := splitIntoWords(message) // A more robust split would handle various whitespace
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 > maxCharsPerLine {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

// A simple word splitter, could be improved.
func splitIntoWords(text string) []string {
	// This is a very basic split, doesn't handle all punctuation or CJK characters well for wrapping.
	// For simplicity, we'll split by space.
	var words []string
	currentWord := ""
	for _, r := range text {
		if r == ' ' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			currentWord += string(r)
		}
	}
	if currentWord != "" {
		words = append(words, currentWord)
	}
	return words
}


// drawBattlefieldは静的な背景要素を描画する
func (g *Game) drawBattlefield(screen *ebiten.Image) {
	// Borders
	vector.StrokeRect(screen, 0, 0, float32(ScreenWidth), BattlefieldHeight, LineWidth, FontColor, false)

	// Home Markers
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, float32(Team1HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
		vector.StrokeCircle(screen, float32(Team2HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
	}

	// Execution Lines
	vector.StrokeLine(screen, Team1ExecutionLineX, 0, Team1ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
	vector.StrokeLine(screen, Team2ExecutionLineX, 0, Team2ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
}

// drawMedarotIconsはバトルフィールド上の動くアイコンを描画する
func (g *Game) drawMedarotIcons(screen *ebiten.Image) {
	team1Count := 0
	team2Count := 0

	for _, medarot := range g.Medarots {
		var yIndex int
		if medarot.Team == Team1 {
			yIndex = team1Count
			team1Count++
		} else {
			yIndex = team2Count
			team2Count++
		}
		baseYPos := MedarotVerticalSpacing * float32(yIndex+1)
		currentX := g.calculateIconX(medarot)

		// Clamp X to screen bounds
		if currentX < float32(IconRadius) {
			currentX = float32(IconRadius)
		}
		if currentX > float32(ScreenWidth-IconRadius) {
			currentX = float32(ScreenWidth - IconRadius)
		}

		iconColor := Team1Color
		if medarot.Team == Team2 {
			iconColor = Team2Color
		}
		if medarot.State == StateBroken {
			iconColor = BrokenColor
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, float32(IconRadius), iconColor, true)

		if medarot.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, float32(IconRadius+2), float32(2), LeaderColor, true)
		}
	}
}

// drawInfoPanelsは下部の情報パネルを描画する
func (g *Game) drawInfoPanels(screen *ebiten.Image) {
	team1InfoCount := 0
	team2InfoCount := 0

	for _, medarot := range g.Medarots {
		var panelX, panelY float32
		if medarot.Team == Team1 {
			panelX = InfoPanelPadding
			panelY = InfoPanelStartY + InfoPanelPadding + float32(team1InfoCount)*(MedarotInfoBlockHeight+InfoPanelPadding)
			team1InfoCount++
		} else { // Team2
			panelX = InfoPanelPadding*2 + MedarotInfoBlockWidth
			panelY = InfoPanelStartY + InfoPanelPadding + float32(team2InfoCount)*(MedarotInfoBlockHeight+InfoPanelPadding)
			team2InfoCount++
		}
		drawMedarotInfo(screen, medarot, panelX, panelY)
	}
}

// ★リファクタリング2: 複雑な計算ロジックをヘルパー関数に切り出し
func (g *Game) calculateIconX(medarot *Medarot) float32 {
	progress := 0.0
	if medarot.State == StateActionCharging && medarot.CurrentActionCharge > 0 {
		progress = medarot.Gauge / medarot.CurrentActionCharge
	} else if medarot.State == StateActionCooldown && medarot.CurrentActionCooldown > 0 {
		progress = medarot.Gauge / medarot.CurrentActionCooldown
	}

	homeX, execX := float32(Team1HomeX), Team1ExecutionLineX
	if medarot.Team == Team2 {
		homeX, execX = float32(Team2HomeX), Team2ExecutionLineX
	}

	switch medarot.State {
	case StateActionCharging:
		return homeX + float32(progress)*(execX-homeX)
	case StateReadyToExecuteAction:
		return execX
	case StateActionCooldown:
		return execX - float32(progress)*(execX-homeX)
	case StateIdleCharging, StateReadyToSelectAction, StateBroken:
		fallthrough
	default:
		return homeX
	}
}

// drawMedarotInfoは個々のメダロットの情報を描画する
func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY float32) {
	text.Draw(screen, medarot.Name, MplusFont, int(startX), int(startY)+int(TextLineHeight), FontColor)
	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	partSlotDisplayNames := map[string]string{
		"head":     "頭部", "rightArm": "右腕", "leftArm": "左腕", "legs": "脚部",
	}

	currentInfoY := startY + TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+TextLineHeight > startY+MedarotInfoBlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		var hpPercentage float64
		if part, ok := medarot.Parts[slotKey]; ok && part != nil {
			hpText = fmt.Sprintf("%s: %d/%d", displayName, part.HP, part.MaxHP)
			if part.MaxHP > 0 {
				hpPercentage = float64(part.HP) / float64(part.MaxHP)
			}
			// Draw HP Gauge
			gaugeX := startX + PartHPGaugeOffsetX
			gaugeY := currentInfoY + TextLineHeight - PartHPGaugeHeight
			vector.DrawFilledRect(screen, gaugeX, gaugeY, PartHPGaugeWidth, PartHPGaugeHeight, color.RGBA{50, 50, 50, 255}, true)
			barFillColor := HPColor
			if part.HP == 0 {
				barFillColor = BrokenColor
			} else if hpPercentage < 0.3 {
				barFillColor = ColorRed
			} else if hpPercentage < 0.6 {
				barFillColor = ColorYellow
			}
			vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(PartHPGaugeWidth)*hpPercentage), PartHPGaugeHeight, barFillColor, true)
		} else {
			hpText = fmt.Sprintf("%s: N/A", displayName)
		}

		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY)+int(TextLineHeight), FontColor)
		currentInfoY += TextLineHeight + 2
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}