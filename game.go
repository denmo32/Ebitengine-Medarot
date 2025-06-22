package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// --- Constants ---
const (
	ScreenWidth                = 960
	ScreenHeight               = 540
	PlayersPerTeam             = 3
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
	InfoPanelStartY            = BattlefieldHeight
	InfoPanelHeight            = float32(ScreenHeight * 0.6)
	InfoPanelPadding           = float32(10)
	MedarotInfoBlockWidth      = (float32(ScreenWidth) - InfoPanelPadding*3) / 2
	MedarotInfoBlockHeight     = (InfoPanelHeight - (InfoPanelPadding * (float32(PlayersPerTeam) + 1))) / float32(PlayersPerTeam)
	PartHPGaugeWidth           = float32(100)
	PartHPGaugeHeight          = float32(7)
	TextLineHeight             = float32(12)
	PartHPGaugeOffsetX         = float32(80)
	ActionModalButtonWidth     = 300
	ActionModalButtonHeight    = 35
	ActionModalButtonSpacing   = 5
)

// --- Colors & Fonts ---
var (
	ColorWhite    = color.White
	ColorBlack    = color.Black
	ColorRed      = color.RGBA{R: 255, G: 100, B: 100, A: 255}
	ColorBlue     = color.RGBA{R: 100, G: 100, B: 255, A: 255}
	ColorGreen    = color.RGBA{R: 100, G: 255, B: 100, A: 255}
	ColorYellow   = color.RGBA{R: 255, G: 255, B: 100, A: 255}
	ColorGray     = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	ColorOrange   = color.RGBA{R: 255, G: 165, B: 0, A: 255}
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

// --- Game State & Core Types ---
type GameState int

const (
	StatePlaying GameState = iota
	StatePlayerActionSelect
	GameStateMessage
	GameStateOver
)

type TeamID int

const (
	Team1 TeamID = iota
	Team2
)

// --- Game Struct ---
type Game struct {
	Medarots            []*Medarot
	GameData            *GameData
	TickCount           int
	DebugMode           bool
	State               GameState
	PlayerTeam          TeamID
	actionQueue         []*Medarot
	message             string
	postMessageCallback func()
	winner              TeamID
	// ★★★ 修正点1: 堅牢な実装のため、ターゲット保持用フィールドを追加 ★★★
	playerActionTarget *Medarot
}

// NewGame initializes the game.
func NewGame(gameData *GameData) *Game {
	medarots := InitializeAllMedarots(gameData)
	if len(medarots) == 0 {
		log.Fatal("No medarots were initialized. Exiting.")
	}
	sort.Slice(medarots, func(i, j int) bool {
		return medarots[i].ID < medarots[j].ID
	})

	g := &Game{
		Medarots:           medarots,
		GameData:           gameData,
		TickCount:          0,
		DebugMode:          true,
		State:              StatePlaying,
		PlayerTeam:         Team1,
		actionQueue:        make([]*Medarot, 0),
		playerActionTarget: nil, // 初期化
	}

	return g
}

// showMessage is a helper to display a message and pause the game.
func (g *Game) showMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	g.State = GameStateMessage
}

// Update proceeds the game state.
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.DebugMode = !g.DebugMode
	}

	g.TickCount++

	switch g.State {
	case StatePlaying:
		g.updatePlaying()
	case StatePlayerActionSelect:
		g.updatePlayerActionSelect()
	case GameStateMessage:
		g.updateMessage()
	case GameStateOver:
		// Game over, no updates, just wait for input to restart (optional)
		return nil
	}

	// ★★★ 修正点2: 状態遷移ロジックをupdatePlaying内に移動 ★★★
	// 理由は、モーダル表示直前にターゲットを決定する必要があるため。
	// if g.State == StatePlaying && len(g.actionQueue) > 0 {
	// 	g.State = StatePlayerActionSelect
	// }

	return nil
}

// updatePlaying handles the main game loop.
func (g *Game) updatePlaying() {
	if g.State == GameStateOver {
		return
	}

	for _, medarot := range g.Medarots {
		medarot.Update(g)

		switch medarot.State {
		case StateReadyToSelectAction:
			isQueued := false
			for _, m := range g.actionQueue {
				if m.ID == medarot.ID {
					isQueued = true
					break
				}
			}
			if !isQueued {
				if medarot.Team == g.PlayerTeam {
					g.actionQueue = append(g.actionQueue, medarot)
				} else {
					g.handleAIAction(medarot)
				}
			}
		case StateReadyToExecuteAction:
			g.setupActionExecution(medarot)
			return
		}
	}

	// ★★★ 修正点3: 状態遷移ロジックをここに移動し、ターゲット決定処理を追加 ★★★
	if g.State == StatePlaying && len(g.actionQueue) > 0 {
		// モーダル表示の直前に、今回の射撃ターゲットを決定して保持する
		targetCandidates := g.getTargetCandidates(g.actionQueue[0])
		if len(targetCandidates) > 0 {
			g.playerActionTarget = targetCandidates[rand.Intn(len(targetCandidates))]
		} else {
			g.playerActionTarget = nil
		}
		g.State = StatePlayerActionSelect
	}
}

// updatePlayerActionSelect handles player input for the modal.
func (g *Game) updatePlayerActionSelect() {
	if len(g.actionQueue) == 0 {
		g.State = StatePlaying
		g.playerActionTarget = nil // 念のためクリア
		return
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	currentMedarot := g.actionQueue[0]
	availableParts := currentMedarot.GetAvailableAttackParts()
	mx, my := ebiten.CursorPosition()
	for i, selectedPart := range availableParts {
		buttonX := float32(ScreenWidth/2 - ActionModalButtonWidth/2)
		buttonY := float32(ScreenHeight/2 - 50 + (ActionModalButtonHeight+ActionModalButtonSpacing)*float32(i))
		buttonW, buttonH := float32(ActionModalButtonWidth), float32(ActionModalButtonHeight)

		if float32(mx) >= buttonX && float32(mx) < buttonX+buttonW &&
			float32(my) >= buttonY && float32(my) < buttonY+buttonH {

			// ★★★ 修正点4: ターゲット選択を保持していたg.playerActionTargetから行う ★★★
			if selectedPart.ActionType == "shoot" {
				currentMedarot.TargetedMedarot = g.playerActionTarget
			} else {
				// Fight target is determined at execution time
				currentMedarot.TargetedMedarot = nil
			}

			log.Printf("Player selected: %s for %s", selectedPart.Name, currentMedarot.Name)
			if currentMedarot.SelectAction(selectedPart.Slot) {
				g.actionQueue = g.actionQueue[1:] // Dequeue
			} else {
				log.Printf("Player %s action selection for %s failed.", currentMedarot.Name, selectedPart.Name)
			}

			if len(g.actionQueue) == 0 {
				g.State = StatePlaying
				g.playerActionTarget = nil // キューが空になったらクリア
			}
			return
		}
	}
}

// updateMessage handles input while a message is displayed.
func (g *Game) updateMessage() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.postMessageCallback != nil {
			g.postMessageCallback()
		} else {
			g.State = StatePlaying
		}
	}
}

// setupActionExecution starts the message flow for an action.
func (g *Game) setupActionExecution(medarot *Medarot) {
	part := medarot.GetPart(medarot.SelectedPartKey)
	if part == nil {
		medarot.State = StateReadyToSelectAction
		medarot.Gauge = 0
		return
	}

	executeCallback := func() {
		success := medarot.ExecuteAction(g)
		resultMessage := medarot.LastActionLog

		if success {
			// The detailed log is already set, we can just show it.
		} else {
			if resultMessage == "" {
				resultMessage = fmt.Sprintf("%sの%sは失敗した...", medarot.Name, part.Name)
			}
		}

		g.showMessage(resultMessage, func() {
			g.State = StatePlaying
			g.checkGameEnd()
		})
	}

	actionVerb := part.ActionType
	targetInfo := ""
	if part.ActionType == "shoot" && medarot.TargetedMedarot != nil {
		targetInfo = fmt.Sprintf(" -> %s", medarot.TargetedMedarot.Name)
	} else if part.ActionType == "fight" {
		targetInfo = " (closest)"
	}

	g.showMessage(fmt.Sprintf("%s: %s (%s)%s！", medarot.Name, part.Name, actionVerb, targetInfo), executeCallback)
}

// handleAIAction encapsulates the AI's action selection logic.
func (g *Game) handleAIAction(medarot *Medarot) {
	if medarot.State != StateReadyToSelectAction {
		return
	}
	availableParts := medarot.GetAvailableAttackParts()
	if len(availableParts) == 0 {
		return
	}

	selectedPart := availableParts[rand.Intn(len(availableParts))]

	targetCandidates := g.getTargetCandidates(medarot)
	if len(targetCandidates) > 0 {
		chosenTarget := targetCandidates[rand.Intn(len(targetCandidates))]
		medarot.TargetedMedarot = chosenTarget
	} else {
		medarot.TargetedMedarot = nil
	}

	medarot.SelectAction(selectedPart.Slot)
}

// getTargetCandidates returns a list of potential targets for the actingMedarot.
func (g *Game) getTargetCandidates(actingMedarot *Medarot) []*Medarot {
	candidates := []*Medarot{}
	opponentTeam := Team2
	if actingMedarot.Team == Team2 {
		opponentTeam = Team1
	}

	for _, m := range g.Medarots {
		if m.Team == opponentTeam && m.State != StateBroken {
			candidates = append(candidates, m)
		}
	}
	return candidates
}

// findClosestOpponent finds the living opponent Medarot closest to the actingMedarot on the X-axis.
func (g *Game) findClosestOpponent(actingMedarot *Medarot) *Medarot {
	var closestOpponent *Medarot
	minDistance := float32(-1.0)

	opponentTeam := Team2
	if actingMedarot.Team == Team2 {
		opponentTeam = Team1
	}

	actingMedarotX := g.calculateIconX(actingMedarot)

	for _, opponent := range g.Medarots {
		if opponent.Team == opponentTeam && opponent.State != StateBroken {
			opponentX := g.calculateIconX(opponent)
			distance := actingMedarotX - opponentX
			if distance < 0 {
				distance = -distance
			}

			if closestOpponent == nil || distance < minDistance {
				minDistance = distance
				closestOpponent = opponent
			}
		}
	}
	return closestOpponent
}

// checkGameEnd checks if a team has lost and sets the game state accordingly.
func (g *Game) checkGameEnd() {
	if g.State == GameStateOver {
		return
	}

	team1Active := false
	team2Active := false
	for _, m := range g.Medarots {
		if m.State != StateBroken {
			if m.Team == Team1 {
				team1Active = true
			} else {
				team2Active = true
			}
		}
	}

	if !team1Active {
		g.winner = Team2
		g.State = GameStateOver
		g.showMessage("チーム2の勝利！", nil)
		log.Println("GAME OVER: Team 2 Wins!")
	} else if !team2Active {
		g.winner = Team1
		g.State = GameStateOver
		g.showMessage("チーム1の勝利！", nil)
		log.Println("GAME OVER: Team 1 Wins!")
	}
}

// Draw renders the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(BGColor)
	g.drawBattlefield(screen)
	g.drawMedarotIcons(screen)
	g.drawInfoPanels(screen)

	if g.State == StatePlayerActionSelect && len(g.actionQueue) > 0 {
		g.drawActionModal(screen, g.actionQueue[0])
	} else if g.State == GameStateMessage || g.State == GameStateOver {
		g.drawMessageWindow(screen)
	}

	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d | State: %v", g.TickCount, g.State), 10, ScreenHeight-15)
	}
}

// --- Drawing Helper Functions ---

func (g *Game) drawMessageWindow(screen *ebiten.Image) {
	windowWidth := float32(ScreenWidth * 0.7)
	windowHeight := float32(ScreenHeight * 0.25)
	windowX := (float32(ScreenWidth) - windowWidth) / 2
	windowY := BattlefieldHeight - windowHeight/2

	vector.DrawFilledRect(screen, windowX, windowY, windowWidth, windowHeight, color.NRGBA{0, 0, 0, 200}, true)
	vector.StrokeRect(screen, windowX, windowY, windowWidth, windowHeight, 2, ColorOrange, false)

	if MplusFont != nil {
		text.Draw(screen, g.message, MplusFont, int(windowX+20), int(windowY+windowHeight/2), FontColor)
		if g.State != GameStateOver {
			promptMsg := "クリックして続行..."
			bounds, _ := font.BoundString(MplusFont, promptMsg)
			promptTextWidth := float32((bounds.Max.X - bounds.Min.X).Ceil())
			promptX := windowX + windowWidth - promptTextWidth - 20
			promptY := windowY + windowHeight - 20
			text.Draw(screen, promptMsg, MplusFont, int(promptX), int(promptY), FontColor)
		}
	}
}

func (g *Game) drawActionModal(screen *ebiten.Image, medarot *Medarot) {
	overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), float32(ScreenHeight), overlayColor, false)

	boxX, boxY := float32(ScreenWidth/2-160), float32(ScreenHeight/2-100)
	boxW, boxH := float32(320), float32(200)
	vector.DrawFilledRect(screen, boxX, boxY, boxW, boxH, BGColor, true)
	vector.StrokeRect(screen, boxX, boxY, boxW, boxH, 2, Team1Color, true)

	titleStr := fmt.Sprintf("%s の行動を選択", medarot.Name)
	bounds, _ := font.BoundString(MplusFont, titleStr)
	titleWidth := float32((bounds.Max.X - bounds.Min.X).Ceil())
	text.Draw(screen, titleStr, MplusFont, int(ScreenWidth/2-int(titleWidth/2)), int(boxY+30), FontColor)

	availableParts := medarot.GetAvailableAttackParts()
	for i, part := range availableParts {
		buttonX := float32(ScreenWidth/2 - ActionModalButtonWidth/2)
		buttonY := float32(ScreenHeight/2 - 50 + (ActionModalButtonHeight+ActionModalButtonSpacing)*float32(i))
		buttonW, buttonH := float32(ActionModalButtonWidth), float32(ActionModalButtonHeight)
		vector.StrokeRect(screen, buttonX, buttonY, buttonW, buttonH, 1, FontColor, true)

		// ★★★ 修正点5: ターゲット情報を表示するロジックを追加 ★★★
		partStr := fmt.Sprintf("%s (%s) [Pow:%d, Chg:%d]", part.Name, part.Slot, part.Power, part.Charge)

		if part.ActionType == "shoot" {
			if g.playerActionTarget != nil {
				partStr += fmt.Sprintf(" -> %s", g.playerActionTarget.Name)
			} else {
				partStr += " (ターゲットなし)"
			}
		} else if part.ActionType == "fight" {
			partStr += " -> 最寄りの敵"
		}
		
		text.Draw(screen, partStr, MplusFont, int(buttonX+10), int(buttonY+22), FontColor)
	}
}

func (g *Game) drawBattlefield(screen *ebiten.Image) {
	vector.StrokeRect(screen, 0, 0, float32(ScreenWidth), BattlefieldHeight, LineWidth, FontColor, false)
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, float32(Team1HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
		vector.StrokeCircle(screen, float32(Team2HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
	}
	vector.StrokeLine(screen, Team1ExecutionLineX, 0, Team1ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
	vector.StrokeLine(screen, Team2ExecutionLineX, 0, Team2ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
}

func (g *Game) drawMedarotIcons(screen *ebiten.Image) {
	team1Indices := make(map[string]int)
	team2Indices := make(map[string]int)
	sortedMedarotsForYCalc := make([]*Medarot, len(g.Medarots))
	copy(sortedMedarotsForYCalc, g.Medarots)
	sort.Slice(sortedMedarotsForYCalc, func(i, j int) bool {
		if sortedMedarotsForYCalc[i].Team != sortedMedarotsForYCalc[j].Team {
			return sortedMedarotsForYCalc[i].Team < sortedMedarotsForYCalc[j].Team
		}
		return sortedMedarotsForYCalc[i].ID < sortedMedarotsForYCalc[j].ID
	})

	currentTeam1Count := 0
	currentTeam2Count := 0
	for _, m := range sortedMedarotsForYCalc {
		if m.Team == Team1 {
			team1Indices[m.ID] = currentTeam1Count
			currentTeam1Count++
		} else {
			team2Indices[m.ID] = currentTeam2Count
			currentTeam2Count++
		}
	}

	for _, medarot := range g.Medarots {
		var yIndex int
		if medarot.Team == Team1 {
			yIndex = team1Indices[medarot.ID]
		} else {
			yIndex = team2Indices[medarot.ID]
		}

		baseYPos := MedarotVerticalSpacing * float32(yIndex+1)
		currentX := g.calculateIconX(medarot)
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
			vector.StrokeCircle(screen, currentX, baseYPos, float32(IconRadius+2), 2, LeaderColor, true)
		}

		// ★★★ 修正点6: ターゲットラインの描画部分を削除 ★★★
		// if medarot.TargetedMedarot != nil && ... { ... } のブロック全体を削除
	}
}

func (g *Game) drawInfoPanels(screen *ebiten.Image) {
	team1InfoCount, team2InfoCount := 0, 0
	for _, medarot := range g.Medarots {
		var panelX, panelY float32
		if medarot.Team == Team1 {
			panelX = InfoPanelPadding
			panelY = InfoPanelStartY + InfoPanelPadding + float32(team1InfoCount)*(MedarotInfoBlockHeight+InfoPanelPadding)
			team1InfoCount++
		} else {
			panelX = InfoPanelPadding*2 + MedarotInfoBlockWidth
			panelY = InfoPanelStartY + InfoPanelPadding + float32(team2InfoCount)*(MedarotInfoBlockHeight+InfoPanelPadding)
			team2InfoCount++
		}
		drawMedarotInfo(screen, medarot, panelX, panelY)
	}
}

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
	default:
		return homeX
	}
}

func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY float32) {
	var nameColor color.Color = FontColor
	if medarot.State == StateBroken {
		nameColor = BrokenColor
	}
	text.Draw(screen, medarot.Name, MplusFont, int(startX), int(startY)+int(TextLineHeight), nameColor)

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		stateStr := fmt.Sprintf("St: %s", medarot.State)
		text.Draw(screen, stateStr, MplusFont, int(startX+70), int(startY)+int(TextLineHeight), ColorYellow)
	}

	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	partSlotDisplayNames := map[string]string{"head": "頭部", "rightArm": "右腕", "leftArm": "左腕", "legs": "脚部"}
	currentInfoY := startY + TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+TextLineHeight > startY+MedarotInfoBlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		if part, ok := medarot.Parts[slotKey]; ok && part != nil {
			hpText = fmt.Sprintf("%s: %d/%d", displayName, part.HP, part.MaxHP)
			if part.MaxHP > 0 {
				hpPercentage := float64(part.HP) / float64(part.MaxHP)
				gaugeX := startX + PartHPGaugeOffsetX
				// ★★★ 修正点7: HPゲージのY軸を調整 ★★★
				gaugeY := currentInfoY - TextLineHeight/2 - PartHPGaugeHeight/2
				vector.DrawFilledRect(screen, gaugeX, gaugeY, PartHPGaugeWidth, PartHPGaugeHeight, color.NRGBA{50, 50, 50, 255}, true)
				barFillColor := HPColor
				if part.IsBroken {
					barFillColor = BrokenColor
				} else if hpPercentage < 0.3 {
					barFillColor = ColorRed
				} else if hpPercentage < 0.6 {
					barFillColor = ColorYellow
				}
				vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(PartHPGaugeWidth)*hpPercentage), PartHPGaugeHeight, barFillColor, true)
			}
		} else {
			hpText = fmt.Sprintf("%s: N/A", displayName)
		}
		var textColor color.Color = FontColor
		if part, ok := medarot.Parts[slotKey]; ok && part.IsBroken {
			textColor = BrokenColor
		}
		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY), textColor)
		currentInfoY += TextLineHeight + 4
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}