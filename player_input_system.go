// --- player_input_system.go ---
package main

import (
	"image"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

type PlayerInputSystem struct {
	actionSelectQuery *donburi.Query
	targetableQuery   *donburi.Query
}

func NewPlayerInputSystem() *PlayerInputSystem {
	return &PlayerInputSystem{
		actionSelectQuery: donburi.NewQuery(filter.And(
			filter.Contains(PlayerControlledComponentType),
			filter.Contains(StatusComponentType),
			filter.Not(filter.Contains(BrokenTag)),
		)),
		targetableQuery: donburi.NewQuery(filter.And(
			filter.Contains(IdentityComponentType),
			filter.Contains(StatusComponentType),
			filter.Not(filter.Contains(BrokenTag)),
		)),
	}
}

func (sys *PlayerInputSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, _ := GameStateComponentType.First(ecs.World)
	pasCompEntry, _ := PlayerActionSelectComponentType.First(ecs.World)
	gs := GameStateComponentType.Get(gameStateEntry)
	pasComp := PlayerActionSelectComponentType.Get(pasCompEntry)

	// --- Phase 1: 行動選択が必要なキャラをキューに追加 ---
	if gs.CurrentState == StatePlaying {
		// 既に誰かがキューにいるか、メッセージ表示に移行した場合は何もしない
		if len(pasComp.ActionQueue) > 0 {
			return
		}

		sys.actionSelectQuery.Each(ecs.World, func(entry *donburi.Entry) {
			if StatusComponentType.Get(entry).State == StateReadyToSelectAction {
				pasComp.ActionQueue = append(pasComp.ActionQueue, entry.Entity())
			}
		})

		// キューに誰かが追加されたら、ゲーム状態を行動選択中に変更
		if len(pasComp.ActionQueue) > 0 {
			gs.CurrentState = StatePlayerActionSelect
		}
	}

	// --- Phase 2: 行動選択UIの操作 ---
	if gs.CurrentState == StatePlayerActionSelect {
		if len(pasComp.ActionQueue) == 0 {
			gs.CurrentState = StatePlaying
			return
		}

		actingMedarotEntry := ecs.World.Entry(pasComp.ActionQueue[0])
		// 行動者が破壊されたなどで無効になったらキューから外す
		if !actingMedarotEntry.Valid() || StatusComponentType.Get(actingMedarotEntry).IsBroken() {
			pasComp.ActionQueue = pasComp.ActionQueue[1:]
			return
		}

		// UIの初期化 (このキャラが初めて選択された場合)
		if len(pasComp.AvailableActions) == 0 {
			initializeActionUI(ecs, actingMedarotEntry, pasComp, sys.targetableQuery)
		}

		// UIのクリック処理
		handleMouseInput(ecs, actingMedarotEntry, gs, pasComp)
	}
}

// initializeActionUI は行動選択UIの初期設定を行います。
func initializeActionUI(ecs *ecs.ECS, entry *donburi.Entry, pasComp *PlayerActionSelectComponent, targetQuery *donburi.Query) {
	partsComp := PartsComponentType.Get(entry)
	pasComp.AvailableActions = []PartSlotKey{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
	for _, slotKey := range slots {
		if part, ok := partsComp.Parts[slotKey]; ok && !part.IsBroken && part.Charge > 0 {
			pasComp.AvailableActions = append(pasComp.AvailableActions, slotKey)
		}
	}

	// 行動がなければキューから外す
	if len(pasComp.AvailableActions) == 0 {
		pasComp.ActionQueue = pasComp.ActionQueue[1:]
		return
	}

	// デフォルトターゲットを選定
	actingID := IdentityComponentType.Get(entry)
	var opponentTeam TeamID = Team2
	if actingID.Team == Team2 {
		opponentTeam = Team1
	}
	candidates := []donburi.Entity{}
	targetQuery.Each(ecs.World, func(targetEntry *donburi.Entry) {
		if IdentityComponentType.Get(targetEntry).Team == opponentTeam && !StatusComponentType.Get(targetEntry).IsBroken() {
			candidates = append(candidates, targetEntry.Entity())
		}
	})
	if len(candidates) > 0 {
		pasComp.CurrentTarget = candidates[rand.Intn(len(candidates))]
	}
}

// handleMouseInput は行動選択UIでのクリックを処理します。
func handleMouseInput(ecs *ecs.ECS, entry *donburi.Entry, gs *GameStateComponent, pasComp *PlayerActionSelectComponent) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}

	config := ConfigComponentType.Get(ConfigComponentType.MustFirst(ecs.World)).GameConfig
	uiConfig := config.UI
	mx, my := ebiten.CursorPosition()

	for i, slotKey := range pasComp.AvailableActions {
		btnW, btnH, btnS := uiConfig.ActionModal.ButtonWidth, uiConfig.ActionModal.ButtonHeight, uiConfig.ActionModal.ButtonSpacing
		btnX := uiConfig.Screen.Width/2 - int(btnW/2)
		btnY := uiConfig.Screen.Height/2 - 50 + (int(btnH)+int(btnS))*i
		buttonRect := image.Rect(btnX, btnY, btnX+int(btnW), btnY+int(btnH))

		if (image.Point{X: mx, Y: my}).In(buttonRect) {
			// ボタンがクリックされた
			status := StatusComponentType.Get(entry)
			actionComp := ActionComponentType.Get(entry)
			partData := PartsComponentType.Get(entry).Parts[slotKey]

			// ターゲットの検証
			targetIsValid := false
			if targetEntry := ecs.World.Entry(pasComp.CurrentTarget); targetEntry.Valid() && !StatusComponentType.Get(targetEntry).IsBroken() {
				targetIsValid = true
			}
			if (partData.Category == CategoryShoot || partData.Category == CategoryFight) && !targetIsValid {
				return // 攻撃には有効なターゲットが必要
			}

			// アクションを確定
			actionComp.SelectedPartKey = slotKey
			actionComp.TargetedMedarot = pasComp.CurrentTarget
			status.State = StateActionCharging
			status.Gauge = 0
			switch partData.Trait {
			case TraitAim:
				status.IsEvasionDisabled = true
			case TraitStrike:
				status.IsDefenseDisabled = true
			case TraitBerserk:
				status.IsEvasionDisabled, status.IsDefenseDisabled = true, true
			}

			entry.AddComponent(ActionChargingTag)
			StatusComponentType.Set(entry, status)
			ActionComponentType.Set(entry, actionComp)

			// 状態をリセットして次へ
			pasComp.ActionQueue = pasComp.ActionQueue[1:]
			pasComp.AvailableActions = nil
			pasComp.CurrentTarget = donburi.Entity(0)
			if len(pasComp.ActionQueue) == 0 {
				gs.CurrentState = StatePlaying
			}
			return // このフレームの入力処理は終了
		}
	}
}
