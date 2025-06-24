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

// PlayerInputSystem はプレイヤーの入力を処理し、行動選択の準備をします。
type PlayerInputSystem struct {
	actionSelectQuery *donburi.Query
	targetableQuery   *donburi.Query
}

// NewPlayerInputSystem はPlayerInputSystemを初期化します。
func NewPlayerInputSystem() *PlayerInputSystem {
	return &PlayerInputSystem{
		actionSelectQuery: donburi.NewQuery(
			filter.And(
				filter.Contains(PlayerControlledComponentType),
				filter.Contains(StatusComponentType),
				filter.Not(filter.Contains(BrokenTag)),
			),
		),
		targetableQuery: donburi.NewQuery(
			filter.And(
				filter.Contains(IdentityComponentType),
				filter.Contains(StatusComponentType),
				filter.Not(filter.Contains(BrokenTag)),
			),
		),
	}
}

// Update はPlayerInputSystemのメインロジックです。
func (sys *PlayerInputSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, ok := GameStateComponentType.First(ecs.World)
	if !ok {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	pasComp := PlayerActionSelectComponentType.Get(gameStateEntry)

	// --- フェーズ1: StatePlaying時の処理 ---
	// 行動可能なプレイヤーキャラを探してキューに追加する
	if gs.CurrentState == StatePlaying {
		// 既にキューに誰かいる場合や、メッセージ表示に移行した場合は、このフレームでは新たに追加しない
		// (これにより、行動選択中にゲージが溜まったキャラが割り込むのを防ぐ)
		if len(pasComp.ActionQueue) > 0 {
			return
		}

		sys.actionSelectQuery.Each(ecs.World, func(entry *donburi.Entry) {
			status := StatusComponentType.Get(entry)
			// ゲージが100%で、まだキューに入っていないキャラを探す
			if status.State == StateReadyToSelectAction {
				// このキャラは行動選択が必要
				pasComp.ActionQueue = append(pasComp.ActionQueue, entry.Entity())
			}
		})

		// キューに誰かが追加されたら、ゲーム状態をPlayerActionSelectに変更して次の処理へ
		if len(pasComp.ActionQueue) > 0 {
			gs.CurrentState = StatePlayerActionSelect
			GameStateComponentType.Set(gameStateEntry, gs)
			PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
		}
		// StatePlayingでの仕事はここまで
		return
	}

	// --- フェーズ2: StatePlayerActionSelect時の処理 ---
	// キューの先頭のキャラの行動を処理する
	if gs.CurrentState == StatePlayerActionSelect {
		// キューが空なのにこの状態なのは異常。安全のためPlayingに戻す
		if len(pasComp.ActionQueue) == 0 {
			gs.CurrentState = StatePlaying
			// UI情報もクリア
			pasComp.AvailableActions = nil
			pasComp.CurrentTarget = donburi.Entity(0)
			GameStateComponentType.Set(gameStateEntry, gs)
			PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
			return
		}

		// 行動選択対象はキューの先頭
		actingMedarotEntity := pasComp.ActionQueue[0]
		actingMedarotEntry := ecs.World.Entry(actingMedarotEntity)

		// 対象が無効(破壊された等)ならキューから外し、次のフレームで次のキャラの処理へ
		if !actingMedarotEntry.Valid() || StatusComponentType.Get(actingMedarotEntry).State == StateBroken {
			pasComp.ActionQueue = pasComp.ActionQueue[1:]
			PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
			return
		}

		// --- UI表示のための準備 (このキャラの処理が初回の場合のみ実行) ---
		if len(pasComp.AvailableActions) == 0 {
			partsComp := PartsComponentType.Get(actingMedarotEntry)
			pasComp.AvailableActions = []PartSlotKey{}
			slotsForActionUI := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
			for _, slotKey := range slotsForActionUI {
				part, exists := partsComp.Parts[slotKey]
				if exists && !part.IsBroken && part.Charge > 0 {
					pasComp.AvailableActions = append(pasComp.AvailableActions, slotKey)
				}
			}

			// 選択可能な行動がなければ、行動できずに終了。キューから外して次へ
			if len(pasComp.AvailableActions) == 0 {
				pasComp.ActionQueue = pasComp.ActionQueue[1:]
				// 次のフレームで次のキャラを処理するために、このフレームはここで終了
				PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
				return
			}

			// 仮のターゲットを選定
			actingID := IdentityComponentType.Get(actingMedarotEntry)
			var opponentTeam TeamID = Team2
			if actingID.Team == Team2 {
				opponentTeam = Team1
			}

			candidates := []donburi.Entity{}
			sys.targetableQuery.Each(ecs.World, func(targetEntry *donburi.Entry) {
				if IdentityComponentType.Get(targetEntry).Team == opponentTeam && StatusComponentType.Get(targetEntry).State != StateBroken {
					candidates = append(candidates, targetEntry.Entity())
				}
			})
			if len(candidates) > 0 {
				pasComp.CurrentTarget = candidates[rand.Intn(len(candidates))]
			} else {
				pasComp.CurrentTarget = donburi.Entity(0)
			}
			// 準備ができたのでコンポーネントを更新
			PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
		}

		// --- クリックによる行動決定処理 ---
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			configEntry, cfgOk := ConfigComponentType.First(ecs.World)
			if !cfgOk {
				return
			}
			uiConfig := ConfigComponentType.Get(configEntry).GameConfig.UI

			status := StatusComponentType.Get(actingMedarotEntry)
			partsComp := PartsComponentType.Get(actingMedarotEntry)
			actionComp := ActionComponentType.Get(actingMedarotEntry)
			mx, my := ebiten.CursorPosition()

			for i, slotKey := range pasComp.AvailableActions {
				partData, partExists := partsComp.Parts[slotKey]
				if !partExists {
					continue
				}

				// ボタン領域の計算 (RenderSystemとロジックを合わせる)
				btnW := uiConfig.ActionModal.ButtonWidth
				btnH := uiConfig.ActionModal.ButtonHeight
				btnSpacing := uiConfig.ActionModal.ButtonSpacing
				buttonX := uiConfig.Screen.Width/2 - int(btnW/2)
				buttonY := uiConfig.Screen.Height/2 - 50 + (int(btnH)+int(btnSpacing))*i
				buttonRect := image.Rect(buttonX, buttonY, buttonX+int(btnW), buttonY+int(btnH))

				if (image.Point{X: mx, Y: my}).In(buttonRect) {
					// ボタンがクリックされた
					targetEntry := ecs.World.Entry(pasComp.CurrentTarget)
					targetIsValid := targetEntry.Valid() && StatusComponentType.Get(targetEntry).State != StateBroken

					// 射撃・格闘でターゲットが必要かチェック
					if partData.Category == CategoryShoot || partData.Category == CategoryFight {
						if !targetIsValid {
							// ターゲットがいないのに攻撃は選べない (ここでは何もしない or エラー音)
							return
						}
						actionComp.TargetedMedarot = pasComp.CurrentTarget
					}

					// アクションを確定し、コンポーネントを更新
					actionComp.SelectedPartKey = slotKey
					status.State = StateActionCharging
					status.Gauge = 0.0

					// 特性による状態変化
					switch partData.Trait {
					case TraitAim:
						status.IsEvasionDisabled = true
					case TraitStrike:
						status.IsDefenseDisabled = true
					case TraitBerserk:
						status.IsEvasionDisabled = true
						status.IsDefenseDisabled = true
					}

					actingMedarotEntry.AddComponent(ActionChargingTag)
					StatusComponentType.Set(actingMedarotEntry, status)
					ActionComponentType.Set(actingMedarotEntry, actionComp)

					// --- キューの更新と状態遷移 ---
					// 行動が決定したので、このメダロットをキューから削除
					pasComp.ActionQueue = pasComp.ActionQueue[1:]

					// 次のキャラのUI準備のため、UI情報をクリア
					pasComp.AvailableActions = nil
					pasComp.CurrentTarget = donburi.Entity(0)

					// もしキューが空になったら、その時初めてPlaying状態に戻す
					if len(pasComp.ActionQueue) == 0 {
						gs.CurrentState = StatePlaying
						GameStateComponentType.Set(gameStateEntry, gs)
					}

					// PlayerActionSelectComponentを更新して終了
					PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
					return // このフレームの処理は終わり
				}
			}
		}
	}
}
