package main

import (
	"fmt"
	"image"
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

// NewGame はゲームを初期化します
func NewGame(gameData *GameData, config Config) *Game {
	medarots := InitializeAllMedarots(gameData)
	if len(medarots) == 0 {
		log.Fatal("No medarots were initialized. Exiting.")
	}
	g := &Game{
		Medarots:              medarots,
		GameData:              gameData,
		Config:                config,
		TickCount:             0,
		DebugMode:             true,
		State:                 StatePlaying,
		PlayerTeam:            Team1,
		actionQueue:           make([]*Medarot, 0),
		sortedMedarotsForDraw: make([]*Medarot, len(medarots)),
		// actionablePartsForModal: make([]*Part, 0, 3), // ★★★ [修正点] この行を削除 ★★★
		team1Leader: nil,
		team2Leader: nil,
	}
	// リーダーをキャッシュ
	for _, m := range medarots {
		if m.IsLeader {
			if m.Team == Team1 {
				g.team1Leader = m
			} else {
				g.team2Leader = m
			}
		}
	}
	// ソート済みリストの作成とDrawIndexの割り当て
	sortedMedarots := make([]*Medarot, len(medarots))
	copy(sortedMedarots, medarots)
	sort.Slice(sortedMedarots, func(i, j int) bool {
		if sortedMedarots[i].Team != sortedMedarots[j].Team {
			return sortedMedarots[i].Team < sortedMedarots[j].Team
		}
		return sortedMedarots[i].ID < sortedMedarots[j].ID
	})
	team1Count, team2Count := 0, 0
	for _, m := range sortedMedarots {
		if m.Team == Team1 {
			m.DrawIndex = team1Count
			team1Count++
		} else {
			m.DrawIndex = team2Count
			team2Count++
		}
	}
	// ソート済みリストを保持
	g.sortedMedarotsForDraw = sortedMedarots
	return g
}

// Update はゲームのメインループです
func (g *Game) Update() error {
	if g.State == GameStateOver {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cursorPos := image.Pt(ebiten.CursorPosition())
			if cursorPos.In(g.getResetButtonRect()) {
				g.restartRequested = true
			}
		}
		if g.restartRequested {
			newG := NewGame(g.GameData, g.Config)
			*g = *newG
		}
		return nil
	}
	// --- 通常の更新処理 ---
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
	}
	return nil
}

// showMessage はメッセージ表示状態に移行します
func (g *Game) showMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	g.State = GameStateMessage
}

// updatePlaying はプレイ中のロジックを処理します
func (g *Game) updatePlaying() {
	for _, medarot := range g.Medarots {
		medarot.Update(g.Config.Balance)
		g.checkAndHandleMedarotState(medarot)
	}
	g.checkGameEnd()
	g.tryEnterActionSelect()
}

// checkAndHandleMedarotState は各メダロットの状態を確認し、必要な処理を呼び出します
func (g *Game) checkAndHandleMedarotState(medarot *Medarot) {
	if medarot.State == StateReadyToExecuteAction {
		g.setupActionExecution(medarot)
		return
	}
	g.queueUpActionableMedarot(medarot)
}

// queueUpActionableMedarot は行動可能なメダロットをキューに追加します
func (g *Game) queueUpActionableMedarot(medarot *Medarot) {
	if medarot.State != StateReadyToSelectAction {
		return
	}
	for _, m := range g.actionQueue {
		if m.ID == medarot.ID {
			return
		}
	}
	if medarot.Team == g.PlayerTeam {
		g.actionQueue = append(g.actionQueue, medarot)
	} else {
		g.handleAIAction(medarot)
	}
}

// ★★★ [修正点] actionablePartsForModalへの代入を削除 ★★★
// tryEnterActionSelect はプレイヤーの行動選択状態に移行できるか試みます
func (g *Game) tryEnterActionSelect() {
	if g.State == StatePlaying && len(g.actionQueue) > 0 {
		g.playerActionTarget = nil
		targetCandidates := g.getTargetCandidates(g.actionQueue[0])
		if len(targetCandidates) > 0 {
			g.playerActionTarget = targetCandidates[rand.Intn(len(targetCandidates))]
		}
		// g.actionablePartsForModal = g.actionQueue[0].GetAvailableAttackParts() // この行を削除
		g.State = StatePlayerActionSelect
	}
}

// ★★★ [修正点] availableParts を都度取得するように変更 ★★★
// updatePlayerActionSelect はプレイヤーの行動選択を処理します
func (g *Game) updatePlayerActionSelect() {
	if len(g.actionQueue) == 0 {
		g.State = StatePlaying
		g.playerActionTarget = nil
		return
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	currentMedarot := g.actionQueue[0]
	availableParts := currentMedarot.GetAvailableAttackParts() // ★ここで直接取得
	mx, my := ebiten.CursorPosition()
	for i := range availableParts {
		btnW := g.Config.UI.ActionModal.ButtonWidth
		btnH := g.Config.UI.ActionModal.ButtonHeight
		btnSpacing := g.Config.UI.ActionModal.ButtonSpacing
		buttonX := g.Config.UI.Screen.Width/2 - int(btnW/2)
		buttonY := g.Config.UI.Screen.Height/2 - 50 + (int(btnH)+int(btnSpacing))*i
		buttonRect := image.Rect(buttonX, buttonY, buttonX+int(btnW), buttonY+int(btnH))

		if (image.Point{X: mx, Y: my}).In(buttonRect) {
			selectedPart := availableParts[i]
			if selectedPart.Category == CategoryShoot {
				currentMedarot.TargetedMedarot = g.playerActionTarget
			} else {
				currentMedarot.TargetedMedarot = nil
			}
			var slotKey string
			switch selectedPart.Type {
			case PartTypeHead:
				slotKey = "head"
			case PartTypeRArm:
				slotKey = "rightArm"
			case PartTypeLArm:
				slotKey = "leftArm"
			}
			if currentMedarot.SelectAction(slotKey) {
				g.actionQueue = g.actionQueue[1:]
			}
			if len(g.actionQueue) == 0 {
				g.State = StatePlaying
				g.playerActionTarget = nil
			}
			return
		}
	}
}

// updateMessage はメッセージ表示中のクリックを処理します
func (g *Game) updateMessage() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.postMessageCallback != nil {
			g.postMessageCallback()
		} else {
			g.State = StatePlaying
		}
	}
}

// setupActionExecution は行動実行のメッセージフローを開始します
func (g *Game) setupActionExecution(medarot *Medarot) {
	part := medarot.GetPart(medarot.SelectedPartKey)
	if part == nil {
		medarot.resetToActionSelection()
		return
	}
	executeCallback := func() {
		opponents := g.getTargetCandidates(medarot)
		medarot.ExecuteAction(g.Config.Balance, opponents)
		nextCallback := func() {
			g.State = StatePlaying
		}
		g.showMessage(medarot.LastActionLog, nextCallback)
	}
	actionVerb := string(part.Category)
	targetInfo := ""
	if part.Category == CategoryShoot && medarot.TargetedMedarot != nil {
		targetInfo = fmt.Sprintf(" -> %s", medarot.TargetedMedarot.Name)
	}
	g.showMessage(fmt.Sprintf("%s: %s (%s)%s！", medarot.Name, part.PartName, actionVerb, targetInfo), executeCallback)
}

// handleAIAction はAIの行動選択を処理します
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
		medarot.TargetedMedarot = targetCandidates[rand.Intn(len(targetCandidates))]
	}
	var slotKey string
	switch selectedPart.Type {
	case PartTypeHead:
		slotKey = "head"
	case PartTypeRArm:
		slotKey = "rightArm"
	case PartTypeLArm:
		slotKey = "leftArm"
	}
	medarot.SelectAction(slotKey)
}

// getTargetCandidates は攻撃対象の候補を返します
func (g *Game) getTargetCandidates(actingMedarot *Medarot) []*Medarot {
	candidates := []*Medarot{}
	var opponentTeam TeamID = Team2
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

// checkGameEnd はゲームの終了を判定します
func (g *Game) checkGameEnd() {
	if g.State == GameStateOver {
		return
	}
	if g.team1Leader != nil && g.team1Leader.State == StateBroken {
		g.winner = Team2
		g.State = GameStateOver
		g.message = "チーム2の勝利！"
	} else if g.team2Leader != nil && g.team2Leader.State == StateBroken {
		g.winner = Team1
		g.State = GameStateOver
		g.message = "チーム1の勝利！"
	}
}

// Draw はゲーム画面を描画します
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(g.Config.UI.Colors.Background)
	g.drawBattlefield(screen)
	g.drawMedarotIcons(screen)
	g.drawInfoPanels(screen)
	if g.State == StatePlayerActionSelect && len(g.actionQueue) > 0 {
		g.drawActionModal(screen, g.actionQueue[0])
	} else if g.State == GameStateMessage || g.State == GameStateOver {
		g.drawMessageWindow(screen)
	}
	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d | State: %v", g.TickCount, g.State), 10, g.Config.UI.Screen.Height-15)
	}
}

// getResetButtonRect はリセットボタンの表示領域を返します。
func (g *Game) getResetButtonRect() image.Rectangle {
	btnW, btnH := 100, 40
	btnX := (g.Config.UI.Screen.Width - btnW) / 2
	btnY := g.Config.UI.Screen.Height - btnH - 20
	return image.Rect(btnX, btnY, btnX+btnW, btnY+btnH)
}

func (g *Game) drawMessageWindow(screen *ebiten.Image) {
	windowWidth := int(float32(g.Config.UI.Screen.Width) * 0.7)
	windowHeight := int(float32(g.Config.UI.Screen.Height) * 0.25)
	windowX := (g.Config.UI.Screen.Width - windowWidth) / 2
	windowY := int(g.Config.UI.Battlefield.Height) - windowHeight/2
	windowRect := image.Rect(windowX, windowY, windowX+windowWidth, windowY+windowHeight)

	prompt := ""
	if g.State == GameStateMessage {
		prompt = "クリックして続行..."
	}
	DrawMessagePanel(screen, windowRect, g.message, prompt, MplusFont, &g.Config.UI)

	if g.State == GameStateOver {
		btnRect := g.getResetButtonRect()
		DrawButton(screen, btnRect, "リセット", MplusFont,
			g.Config.UI.Colors.Gray, g.Config.UI.Colors.White, g.Config.UI.Colors.White)
	}
}

// ★★★ [修正点] availableParts を都度取得するように変更 ★★★
func (g *Game) drawActionModal(screen *ebiten.Image, medarot *Medarot) {
	overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(g.Config.UI.Screen.Width), float32(g.Config.UI.Screen.Height), overlayColor, false)

	boxW, boxH := 320, 200
	boxX := (g.Config.UI.Screen.Width - boxW) / 2
	boxY := (g.Config.UI.Screen.Height - boxH) / 2
	windowRect := image.Rect(boxX, boxY, boxX+boxW, boxY+boxH)

	DrawWindow(screen, windowRect, g.Config.UI.Colors.Background, g.Config.UI.Colors.Team1)

	titleStr := fmt.Sprintf("%s の行動を選択", medarot.Name)
	if MplusFont != nil {
		bounds, _ := font.BoundString(MplusFont, titleStr)
		titleWidth := (bounds.Max.X - bounds.Min.X).Ceil()
		text.Draw(screen, titleStr, MplusFont, g.Config.UI.Screen.Width/2-titleWidth/2, boxY+30, g.Config.UI.Colors.White)
	}

	availableParts := medarot.GetAvailableAttackParts() // ★ここで直接取得
	for i, part := range availableParts {
		btnW := g.Config.UI.ActionModal.ButtonWidth
		btnH := g.Config.UI.ActionModal.ButtonHeight
		btnSpacing := g.Config.UI.ActionModal.ButtonSpacing
		buttonX := g.Config.UI.Screen.Width/2 - int(btnW/2)
		buttonY := g.Config.UI.Screen.Height/2 - 50 + (int(btnH)+int(btnSpacing))*i
		buttonRect := image.Rect(buttonX, buttonY, buttonX+int(btnW), buttonY+int(btnH))

		partStr := fmt.Sprintf("%s (%s)", part.PartName, part.Type)
		if part.Category == CategoryShoot {
			if g.playerActionTarget != nil {
				partStr += fmt.Sprintf(" -> %s", g.playerActionTarget.Name)
			} else {
				partStr += " (ターゲットなし)"
			}
		} else if part.Category == CategoryFight {
			partStr += " -> 最寄りの敵"
		}

		DrawButton(screen, buttonRect, partStr, MplusFont,
			g.Config.UI.Colors.Background, g.Config.UI.Colors.White, g.Config.UI.Colors.White)
	}
}

func (g *Game) drawBattlefield(screen *ebiten.Image) {
	vector.StrokeRect(screen, 0, 0, float32(g.Config.UI.Screen.Width), g.Config.UI.Battlefield.Height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.White, false)
	playersPerTeam := len(g.Medarots) / 2
	for i := 0; i < playersPerTeam; i++ {
		yPos := g.Config.UI.Battlefield.MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, g.Config.UI.Battlefield.Team1HomeX, yPos, g.Config.UI.Battlefield.HomeMarkerRadius, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, g.Config.UI.Battlefield.Team2HomeX, yPos, g.Config.UI.Battlefield.HomeMarkerRadius, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, g.Config.UI.Battlefield.Team1ExecutionLineX, 0, g.Config.UI.Battlefield.Team1ExecutionLineX, g.Config.UI.Battlefield.Height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, false)
	vector.StrokeLine(screen, g.Config.UI.Battlefield.Team2ExecutionLineX, 0, g.Config.UI.Battlefield.Team2ExecutionLineX, g.Config.UI.Battlefield.Height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, false)
}

func (g *Game) drawMedarotIcons(screen *ebiten.Image) {
	for _, medarot := range g.Medarots {
		baseYPos := g.Config.UI.Battlefield.MedarotVerticalSpacing * float32(medarot.DrawIndex+1)
		currentX := g.calculateIconX(medarot)
		if currentX < g.Config.UI.Battlefield.IconRadius {
			currentX = g.Config.UI.Battlefield.IconRadius
		}
		if currentX > float32(g.Config.UI.Screen.Width)-g.Config.UI.Battlefield.IconRadius {
			currentX = float32(g.Config.UI.Screen.Width) - g.Config.UI.Battlefield.IconRadius
		}
		iconColor := g.Config.UI.Colors.Team1
		if medarot.Team == Team2 {
			iconColor = g.Config.UI.Colors.Team2
		}
		if medarot.State == StateBroken {
			iconColor = g.Config.UI.Colors.Broken
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, g.Config.UI.Battlefield.IconRadius, iconColor, true)
		if medarot.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, g.Config.UI.Battlefield.IconRadius+2, 2, g.Config.UI.Colors.Leader, true)
		}
	}
}

func (g *Game) drawInfoPanels(screen *ebiten.Image) {
	team1InfoCount, team2InfoCount := 0, 0
	for _, medarot := range g.Medarots {
		var panelX, panelY float32
		if medarot.Team == Team1 {
			panelX = g.Config.UI.InfoPanel.Padding
			panelY = g.Config.UI.InfoPanel.StartY + g.Config.UI.InfoPanel.Padding + float32(team1InfoCount)*(g.Config.UI.InfoPanel.BlockHeight+g.Config.UI.InfoPanel.Padding)
			team1InfoCount++
		} else {
			panelX = g.Config.UI.InfoPanel.Padding*2 + g.Config.UI.InfoPanel.BlockWidth
			panelY = g.Config.UI.InfoPanel.StartY + g.Config.UI.InfoPanel.Padding + float32(team2InfoCount)*(g.Config.UI.InfoPanel.BlockHeight+g.Config.UI.InfoPanel.Padding)
			team2InfoCount++
		}
		g.drawMedarotInfo(screen, medarot, panelX, panelY)
	}
}

func (g *Game) calculateIconX(medarot *Medarot) float32 {
	progress := medarot.Gauge / 100.0
	homeX, execX := g.Config.UI.Battlefield.Team1HomeX, g.Config.UI.Battlefield.Team1ExecutionLineX
	if medarot.Team == Team2 {
		homeX, execX = g.Config.UI.Battlefield.Team2HomeX, g.Config.UI.Battlefield.Team2ExecutionLineX
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

func (g *Game) drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY float32) {
	if MplusFont == nil {
		return
	}
	var nameColor color.Color = g.Config.UI.Colors.White
	if medarot.State == StateBroken {
		nameColor = g.Config.UI.Colors.Broken
	}
	text.Draw(screen, medarot.Name, MplusFont, int(startX), int(startY)+int(g.Config.UI.InfoPanel.TextLineHeight), nameColor)
	if g.DebugMode {
		stateStr := fmt.Sprintf("St: %s", medarot.State)
		text.Draw(screen, stateStr, MplusFont, int(startX+70), int(startY)+int(g.Config.UI.InfoPanel.TextLineHeight), g.Config.UI.Colors.Yellow)
	}
	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	partSlotDisplayNames := map[string]string{"head": "頭部", "rightArm": "右腕", "leftArm": "左腕", "legs": "脚部"}
	currentInfoY := startY + g.Config.UI.InfoPanel.TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+g.Config.UI.InfoPanel.TextLineHeight > startY+g.Config.UI.InfoPanel.BlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		part, exists := medarot.Parts[slotKey]
		if exists && part != nil {
			currentArmor := part.Armor
			if part.IsBroken {
				currentArmor = 0
			}
			hpText = fmt.Sprintf("%s: %d/%d", displayName, currentArmor, part.MaxArmor)
			if part.MaxArmor > 0 {
				hpPercentage := float64(part.Armor) / float64(part.MaxArmor)
				gaugeX := startX + g.Config.UI.InfoPanel.PartHPGaugeOffsetX
				gaugeY := currentInfoY - g.Config.UI.InfoPanel.TextLineHeight/2 - g.Config.UI.InfoPanel.PartHPGaugeHeight/2
				vector.DrawFilledRect(screen, gaugeX, gaugeY, g.Config.UI.InfoPanel.PartHPGaugeWidth, g.Config.UI.InfoPanel.PartHPGaugeHeight, color.NRGBA{50, 50, 50, 255}, true)
				barFillColor := g.Config.UI.Colors.HP
				if part.IsBroken {
					barFillColor = g.Config.UI.Colors.Broken
				} else if hpPercentage < 0.3 {
					barFillColor = g.Config.UI.Colors.Red
				}
				vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(g.Config.UI.InfoPanel.PartHPGaugeWidth)*hpPercentage), g.Config.UI.InfoPanel.PartHPGaugeHeight, barFillColor, true)
			}
		} else {
			hpText = fmt.Sprintf("%s: N/A", displayName)
		}
		var textColor color.Color = g.Config.UI.Colors.White
		if exists && part != nil && part.IsBroken {
			textColor = g.Config.UI.Colors.Broken
		}
		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY), textColor)
		if exists && part != nil && part.MaxArmor > 0 {
			gaugeX := startX + g.Config.UI.InfoPanel.PartHPGaugeOffsetX
			partNameX := gaugeX + g.Config.UI.InfoPanel.PartHPGaugeWidth + 5
			text.Draw(screen, part.PartName, MplusFont, int(partNameX), int(currentInfoY), textColor)
		}
		currentInfoY += g.Config.UI.InfoPanel.TextLineHeight + 4
	}
}

// Layout は画面レイアウトを定義します
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.Config.UI.Screen.Width, g.Config.UI.Screen.Height
}
