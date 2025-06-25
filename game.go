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

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

// NewGame はゲームを初期化します
func NewGame(gameData *GameData, config Config) *Game {
	rootContainer := widget.NewContainer() // EbitenUIのルートコンテナを作成
	ui := &ebitenui.UI{
		Container: rootContainer,
	}

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
		ui:                    ui, // UIマネージャーをセット
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

	// Initialize and add static UI elements like info panels to the root container
	infoPanelsWidget := createInfoPanelsUI(g)
	g.ui.Container.AddChild(infoPanelsWidget) // Add to the main UI tree

	return g
}

// Update はゲームのメインループです
func (g *Game) Update() error {
	// Handle restart triggered by UI button
	if g.restartRequested {
		// When restarting, the old UI is discarded with the old game state.
		// NewGame() will set up a new UI with new info panels.
		newG := NewGame(g.GameData, g.Config)
		*g = *newG
		// Ensure UI state is correct after restart
		if g.State == GameStateMessage || g.State == GameStateOver {
			showUIMessage(g)
		} else {
			hideUIMessage(g)
		}
		return nil
	}

	// If game is over, only UI updates (which include reset button) and debug toggle
	if g.State == GameStateOver {
		if inpututil.IsKeyJustPressed(ebiten.KeyD) {
			g.DebugMode = !g.DebugMode
		}
		g.ui.Update()
		return nil
	}

	// Regular updates
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.DebugMode = !g.DebugMode
	}
	g.TickCount++
	previousState := g.State

	switch g.State {
	case StatePlaying:
		g.updatePlaying()
	case StatePlayerActionSelect:
		g.updatePlayerActionSelect() // This will eventually be UI driven too
	case GameStateMessage:
		g.updateMessage() // Handles click-to-continue for UI message
	}

	// Manage visibility of the message window based on state changes
	if previousState != g.State {
		if g.State == GameStateMessage || g.State == GameStateOver {
			// If moving TO a message/over state, ensure any old one is gone, then show new.
			hideUIMessage(g)
			showUIMessage(g)
		} else {
			// If moving FROM a message/over state to something else, hide it.
			hideUIMessage(g)
		}
	} else if g.State == GameStateMessage || g.State == GameStateOver {
		// If state hasn't changed but we are in a message state,
		// it implies the content of the message might need updating (e.g. new log message).
		// For now, createMessageWindowUI reads g.message directly.
		// A more robust way would be to explicitly call an update function for the window if its content changes.
		// However, showUIMessage already removes and re-adds, effectively updating.
		// This line ensures it's visible if it was hidden for some reason (though current logic shouldn't allow that).
		// Re-evaluate if this specific `else if` is truly needed after action modal.
		// showUIMessage(g) // This might cause flicker if called every frame.

		// Manage visibility of the action modal based on state changes
		if g.State == StatePlayerActionSelect {
			// If moving TO action select state, show modal.
			// (hide is implicitly handled if state changes AWAY from ActionSelect by the 'else' below)
			showUIActionModal(g)
		} else if previousState == StatePlayerActionSelect && g.State != StatePlayerActionSelect {
			// If moving FROM action select state, hide modal.
			hideUIActionModal(g)
		}
	} else if g.State == StatePlayerActionSelect {
		// If state hasn't changed but we are in action select,
		// the modal might need an update (e.g. target changed).
		// showUIActionModal removes and re-adds, effectively updating.
		showUIActionModal(g)
	}

	// Update all info panels unconditionally for now.
	// Later, this could be optimized to update only when data changes.
	updateAllInfoPanels(g)

	g.ui.Update() // EbitenUIのUpdateを呼び出す
	return nil
}

// showMessage はメッセージ表示状態に移行します
func (g *Game) showMessage(msg string, callback func()) {
	g.message = msg
	g.postMessageCallback = callback
	previousState := g.State
	g.State = GameStateMessage

	// If we are transitioning to a message state, ensure the UI reflects this.
	// hideUIMessage will remove any existing message window (e.g. from a previous message).
	// showUIMessage will create and add the new one.
	if previousState != GameStateMessage && previousState != GameStateOver {
		hideUIMessage(g) // Good practice to hide before showing new, avoids overlap if logic changes
	}
	showUIMessage(g)
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
	if g.State == StatePlaying && len(g.actionQueue) > 0 && g.actionQueue[0].Team == g.PlayerTeam {
		actingMedarot := g.actionQueue[0]
		// Ensure a default target is selected if applicable (for shooting parts)
		// This target might be changed by player input later (not implemented yet)
		if g.playerActionTarget == nil || g.playerActionTarget.State == StateBroken {
			g.playerActionTarget = g.selectAutomaticTarget(actingMedarot)
		}
		// If still no valid target for a shooter, AI might need to pick a different action or wait.
		// For player, the UI will show "no target" or similar.

		g.State = StatePlayerActionSelect
		// The call to showUIActionModal() is handled by the state change detection in Update()
	}
}

// updatePlayerActionSelect はプレイヤーの行動選択を処理します
// With EbitenUI, button clicks are handled by their respective ClickedHandler.
// So, this function becomes much simpler or might even be removed if all logic moves to handlers.
// For now, it can be used to check if the action selection phase should end (e.g., if actionQueue becomes empty unexpectedly).
func (g *Game) updatePlayerActionSelect() {
	if len(g.actionQueue) == 0 || g.actionQueue[0].Team != g.PlayerTeam {
		// If the queue is empty or the turn has passed to AI somehow, transition out.
		g.State = StatePlaying
		g.playerActionTarget = nil
		hideUIActionModal(g) // Ensure modal is hidden
		return
	}

	// Potentially, add logic here for changing targets with keyboard/mouse if not done via UI buttons.
	// For example, pressing a key to cycle targets.
	// If target changes, we need to refresh the action modal:
	// oldTarget := g.playerActionTarget
	// if newTargetSelected { g.playerActionTarget = newTarget }
	// if oldTarget != g.playerActionTarget { showUIActionModal(g) }
}

// updateMessage はメッセージ表示中のクリックを処理します
// For EbitenUI, the "click to continue" for messages (that don't have a button)
// needs to be handled slightly differently. The UI window itself won't have a click handler.
// So, we keep the global mouse click check for this specific case.
// The reset button on Game Over is handled by its own ClickedHandler.
func (g *Game) updateMessage() {
	// Only process click-to-continue if not in game over (which has a button)
	// and if there's no specific UI element handling the click (like a dedicated "Next" button).
	if g.State == GameStateMessage && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.postMessageCallback != nil {
			g.postMessageCallback() // This might change state
		} else {
			// Default action if no callback: return to playing
			g.State = StatePlaying
		}
		// After processing the click, the state might change,
		// so the main Update loop's state management will hide the message window.
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
	// InfoPanels are now handled by EbitenUI.
	// The old drawInfoPanels call is removed.
	// Action Modal is now handled by EbitenUI.
	// The old drawActionModal call is removed.
	// EbitenUI's Draw method (called later) will handle drawing active UI elements.

	// Message window is now handled by EbitenUI, so drawMessageWindow(screen, g) is removed.
	// EbitenUI's Draw method (called later) will handle drawing active UI elements like the message window.

	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d | State: %v", g.TickCount, g.State), 10, g.Config.UI.Screen.Height-15)
	}
	g.ui.Draw(screen) // EbitenUIのDrawを呼び出す
}

// Layout は画面レイアウトを定義します
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.Config.UI.Screen.Width, g.Config.UI.Screen.Height
}
