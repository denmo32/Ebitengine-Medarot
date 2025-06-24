package main

import (
	"fmt"
	"image"
	"log"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
		team1Leader:           nil,
		team2Leader:           nil,
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
	g.sortedMedarotsForDraw = sortedMedarots
	return g
}

// Update はゲームのメインループです
func (g *Game) Update() error {
	if g.State == GameStateOver {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cursorPos := image.Pt(ebiten.CursorPosition())
			// ★ getResetButtonRect は renderer.go に移動したので、g を渡す
			if cursorPos.In(getResetButtonRect(g)) {
				g.restartRequested = true
			}
		}
		if g.restartRequested {
			newG := NewGame(g.GameData, g.Config)
			*g = *newG
		}
		return nil
	}
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
		// ▼▼▼ このチェックを追加 ▼▼▼
		if medarot.State == StateBroken {
			continue // 機能停止している機体は以降の処理をスキップ
		}
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

// selectAutomaticTarget は、指定されたメダロットの攻撃対象を自動で1体選定します。
// AIのターゲット選定と、プレイヤーの仮ターゲット選定の両方で使われます。
func (g *Game) selectAutomaticTarget(actingMedarot *Medarot) *Medarot {
	candidates := g.getTargetCandidates(actingMedarot)
	if len(candidates) == 0 {
		return nil
	}

	// === 将来のAI拡張はここを修正 ===
	// 例: リーダーを優先的に狙うAIロジック
	/*
	   for _, c := range candidates {
	       if c.IsLeader {
	           return c // リーダーがいれば最優先で返す
	       }
	   }
	*/
	// === AI拡張ここまで ===

	// デフォルトの動作: 候補の中からランダムに1体を選ぶ
	return candidates[rand.Intn(len(candidates))]
}

// tryEnterActionSelect はプレイヤーの行動選択状態に移行できるか試みます
func (g *Game) tryEnterActionSelect() {
	if g.State == StatePlaying && len(g.actionQueue) > 0 {
		// ▼▼▼ この部分を修正 ▼▼▼
		actingMedarot := g.actionQueue[0]
		g.playerActionTarget = g.selectAutomaticTarget(actingMedarot)
		g.State = StatePlayerActionSelect
		// ▲▲▲ ここまで ▲▲▲
	}
}

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
	availableParts := currentMedarot.GetAvailableAttackParts()
	mx, my := ebiten.CursorPosition()
	for i, part := range availableParts {
		btnW := g.Config.UI.ActionModal.ButtonWidth
		btnH := g.Config.UI.ActionModal.ButtonHeight
		btnSpacing := g.Config.UI.ActionModal.ButtonSpacing
		buttonX := g.Config.UI.Screen.Width/2 - int(btnW/2)
		buttonY := g.Config.UI.Screen.Height/2 - 50 + (int(btnH)+int(btnSpacing))*i
		buttonRect := image.Rect(buttonX, buttonY, buttonX+int(btnW), buttonY+int(btnH))
		if (image.Point{X: mx, Y: my}).In(buttonRect) {
			// ▼▼▼ このチェックを追加 ▼▼▼
			if g.playerActionTarget == nil || g.playerActionTarget.State == StateBroken {
				log.Println("Action cancelled: Target is invalid or already broken.")
				// ここでゲームを StatePlaying に戻すか、メッセージを出すかはお好みで
				g.State = StatePlaying
				return
			}
			// ▲▲▲ ここまで ▲▲▲

			// ▼▼▼ 先にターゲットを設定する ▼▼▼
			currentMedarot.TargetedMedarot = g.playerActionTarget
			if currentMedarot.TargetedMedarot == nil && part.Category == CategoryShoot {
				// ターゲットがいない場合は何もしないか、フィードバックを出す
				log.Println("Cannot select shooting part without a target.")
				// ターゲットをnilに戻しておく（任意だが、安全のため）
				currentMedarot.TargetedMedarot = nil
				return
			}
			var slotKey PartSlotKey
			switch part.Type {
			case PartTypeHead:
				slotKey = PartSlotHead
			case PartTypeRArm:
				slotKey = PartSlotRightArm
			case PartTypeLArm:
				slotKey = PartSlotLeftArm
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
		medarot.ChangeState(StateReadyToSelectAction)
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

	// ▼▼▼ この部分を修正 ▼▼▼
	medarot.TargetedMedarot = g.selectAutomaticTarget(medarot)
	if medarot.TargetedMedarot == nil && selectedPart.Category == CategoryShoot {
		// 射撃パーツなのにターゲットがいない場合は何もしない
		return
	}
	// ▲▲▲ ここまで ▲▲▲

	var slotKey PartSlotKey
	switch selectedPart.Type {
	case PartTypeHead:
		slotKey = PartSlotHead
	case PartTypeRArm:
		slotKey = PartSlotRightArm
	case PartTypeLArm:
		slotKey = PartSlotLeftArm
	}
	medarot.SelectAction(slotKey)
}

// getTargetCandidates は攻撃対象の候補を返します
func (g *Game) getTargetCandidates(actingMedarot *Medarot) []*Medarot {
	candidates := []*Medarot{}
	var opponentTeamID TeamID = Team2
	if actingMedarot.Team == Team2 {
		opponentTeamID = Team1
	}
	for _, m := range g.Medarots {
		if m.Team == opponentTeamID && m.State != StateBroken {
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
	drawBattlefield(screen, g)
	drawMedarotIcons(screen, g)
	drawInfoPanels(screen, g)
	if g.State == StatePlayerActionSelect && len(g.actionQueue) > 0 {
		drawActionModal(screen, g) // g.actionQueue[0] は renderer 側で参照
	} else if g.State == GameStateMessage || g.State == GameStateOver {
		drawMessageWindow(screen, g)
	}
	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d | State: %v", g.TickCount, g.State), 10, g.Config.UI.Screen.Height-15)
	}
}

// Layout は画面レイアウトを定義します
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.Config.UI.Screen.Width, g.Config.UI.Screen.Height
}
