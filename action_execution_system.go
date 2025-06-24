package main

import (
	"fmt"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
	"math/rand"
)

// ActionExecutionSystem は選択されたアクションの実行を担当します。
type ActionExecutionSystem struct{ query *donburi.Query }

// NewActionExecutionSystem はActionExecutionSystemを初期化します。
func NewActionExecutionSystem() *ActionExecutionSystem {
	return &ActionExecutionSystem{
		query: donburi.NewQuery(filter.And(
			filter.Contains(ReadyToExecuteActionTag), filter.Contains(ActionComponentType),
			filter.Contains(StatusComponentType), filter.Contains(PartsComponentType),
			filter.Contains(CMedal), filter.Contains(IdentityComponentType),
			filter.Not(filter.Contains(BrokenTag)),
		)),
	}
}

// Update はActionExecutionSystemのメインロジックです。
func (sys *ActionExecutionSystem) Update(ecs *ecs.ECS) {
	configEntry, configOk := ConfigComponentType.First(ecs.World)
	if !configOk {
		return
	}
	balanceConfig := ConfigComponentType.Get(configEntry).GameConfig.Balance

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		// 行動を実行するエンティティのコンポーネントを取得
		actionComp := ActionComponentType.Get(entry)
		statusComp := StatusComponentType.Get(entry)
		partsComp := PartsComponentType.Get(entry)
		identityComp := IdentityComponentType.Get(entry)
		// medalComp := CMedal.Get(entry)
		selectedPart := partsComp.Parts[actionComp.SelectedPartKey]

		// 選択されたパーツが壊れている場合はアクション失敗
		if selectedPart == nil || selectedPart.IsBroken {
			handleActionFailure(ecs, entry, actionComp, statusComp, identityComp.Name+": パーツが壊れていて失敗")
			return
		}

		// 攻撃対象を決定・検証する
		targetEntry, targetIsValid := sys.determineTarget(ecs, entry, actionComp.TargetedMedarot, selectedPart.Category)

		// アクション実行のメッセージをまず表示し、コールバックで実際の処理を行う
		initialMessage := sys.createInitialMessage(identityComp, selectedPart, targetEntry)
		showGameMessage(ecs, initialMessage, func() {
			sys.executeAction(ecs, entry, targetEntry, targetIsValid, balanceConfig)
		})
	})
}

// determineTarget はアクションの最終的なターゲットを決定します。
func (sys *ActionExecutionSystem) determineTarget(ecs *ecs.ECS, attackerEntry *donburi.Entry, intendedTarget donburi.Entity, category ActionCategory) (*donburi.Entry, bool) {
	// 射撃・格闘の場合、ターゲットを検証
	if category == CategoryShoot || category == CategoryFight {
		if ecs.World.Valid(intendedTarget) {
			targetEntry := ecs.World.Entry(intendedTarget)
			// IsBroken() を使用して破壊状態をチェック
			if targetEntry.Valid() && !StatusComponentType.Get(targetEntry).IsBroken() {
				return targetEntry, true
			}
		}

		// 格闘でターゲットが無効な場合、ランダムな敵を再ターゲット
		if category == CategoryFight {
			return sys.findRandomOpponent(ecs, attackerEntry)
		}

		// 射撃でターゲットが無効な場合は失敗
		return nil, false
	}
	// 攻撃以外はターゲット不要
	return nil, true
}

// findRandomOpponent はランダムな敵を探します。
func (sys *ActionExecutionSystem) findRandomOpponent(ecs *ecs.ECS, attackerEntry *donburi.Entry) (*donburi.Entry, bool) {
	attackerID := IdentityComponentType.Get(attackerEntry)
	var opponentTeam TeamID = Team2
	if attackerID.Team == Team2 {
		opponentTeam = Team1
	}

	candidates := []*donburi.Entry{}
	query := donburi.NewQuery(filter.And(filter.Contains(IdentityComponentType), filter.Not(filter.Contains(BrokenTag))))
	query.Each(ecs.World, func(entry *donburi.Entry) {
		if IdentityComponentType.Get(entry).Team == opponentTeam && !StatusComponentType.Get(entry).IsBroken() {
			candidates = append(candidates, entry)
		}
	})

	if len(candidates) > 0 {
		return candidates[rand.Intn(len(candidates))], true
	}
	return nil, false
}

// executeAction はアクションの主効果（命中判定、ダメージ計算など）を実行します。
func (sys *ActionExecutionSystem) executeAction(ecs *ecs.ECS, attackerEntry, targetEntry *donburi.Entry, targetIsValid bool, balanceConfig BalanceConfig) {
	actionComp := ActionComponentType.Get(attackerEntry)
	selectedPart := PartsComponentType.Get(attackerEntry).Parts[actionComp.SelectedPartKey]

	logMsg := ""

	if !targetIsValid {
		logMsg = fmt.Sprintf("%sは失敗した", selectedPart.PartName)
	} else if selectedPart.Category == CategoryShoot || selectedPart.Category == CategoryFight {
		// 攻撃アクション
		if targetEntry == nil {
			logMsg = fmt.Sprintf("%sのターゲットが見つからない", selectedPart.PartName)
		} else {
			logMsg = sys.performAttack(attackerEntry, targetEntry, balanceConfig)
		}
	} else {
		// 補助など、その他のアクション
		logMsg = fmt.Sprintf("%sは%sを使用した", IdentityComponentType.Get(attackerEntry).Name, selectedPart.PartName)
	}

	actionComp.LastActionLog = logMsg
	ActionComponentType.Set(attackerEntry, actionComp)

	// アクション後の状態遷移と最終メッセージ表示
	transitionToCooldown(ecs, attackerEntry, actionComp, logMsg)
}

// performAttack は一連の攻撃処理を行います。
func (sys *ActionExecutionSystem) performAttack(attackerEntry, targetEntry *donburi.Entry, cfg BalanceConfig) string {
	// ... コンポーネント取得 ...
	_, attackerMedal, attackerPart := getAttackerData(attackerEntry)
	targetID, targetStatus, targetParts := getTargetData(targetEntry)

	// isHit, isCritical := calculateHit(attackerID, attackerMedal, attackerPart, targetID, targetStatus, targetParts.Parts[PartSlotLegs], cfg)
	isHit, isCritical := calculateHit(attackerMedal, attackerPart, targetStatus, targetParts.Parts[PartSlotLegs], cfg)
	if !isHit {
		return fmt.Sprintf("%sへの攻撃は回避された！", targetID.Name)
	}

	partToDamage := selectRandomPartToDamage(targetParts)
	if partToDamage == nil {
		return fmt.Sprintf("%sには攻撃できる部位がない！", targetID.Name)
	}

	damage := calculateDamage(attackerEntry, attackerMedal, attackerPart, partToDamage, targetParts.Parts[PartSlotLegs], isCritical, cfg, targetStatus.IsDefenseDisabled)

	// ... ダメージ適用とログ生成 ...
	origArmor := partToDamage.Armor
	partToDamage.Armor -= damage
	if partToDamage.Armor <= 0 {
		partToDamage.Armor = 0
		if !partToDamage.IsBroken {
			partToDamage.IsBroken = true
			if partToDamage.Type == PartTypeHead {
				handleHeadDestruction(targetEntry)
			}
		}
	}
	PartsComponentType.Set(targetEntry, targetParts)

	logMsg := fmt.Sprintf("%sの%sに%dダメージ！ (%d -> %d)", targetID.Name, partToDamage.PartName, damage, origArmor, partToDamage.Armor)
	if isCritical {
		logMsg = "クリティカル！ " + logMsg
	}
	if partToDamage.IsBroken && origArmor > 0 {
		logMsg += " [破壊！]"
	}

	return logMsg
}

// --- Helper functions ---

func (sys *ActionExecutionSystem) createInitialMessage(attackerID *IdentityComponent, part *Part, targetEntry *donburi.Entry) string {
	targetInfo := ""
	if (part.Category == CategoryShoot || part.Category == CategoryFight) && targetEntry != nil {
		targetInfo = fmt.Sprintf(" -> %s", IdentityComponentType.Get(targetEntry).Name)
	}
	return fmt.Sprintf("%s: %s%s！", attackerID.Name, part.PartName, targetInfo)
}

func handleActionFailure(ecs *ecs.ECS, entry *donburi.Entry, action *ActionComponent, status *StatusComponent, logMsg string) {
	status.State = StateReadyToSelectAction
	status.Gauge = 100
	entry.RemoveComponent(ReadyToExecuteActionTag)
	StatusComponentType.Set(entry, status)

	action.LastActionLog = logMsg
	ActionComponentType.Set(entry, action)

	showGameMessage(ecs, logMsg, func() {
		// メッセージを閉じたら即座に次の行動選択に移る
		gsEntry, _ := GameStateComponentType.First(ecs.World)
		gs := GameStateComponentType.Get(gsEntry)
		gs.CurrentState = StatePlaying
		GameStateComponentType.Set(gsEntry, gs)
	})
}

func transitionToCooldown(ecs *ecs.ECS, entry *donburi.Entry, action *ActionComponent, logMsg string) {
	status := StatusComponentType.Get(entry)
	status.State = StateActionCooldown
	status.Gauge = 0
	entry.RemoveComponent(ReadyToExecuteActionTag)
	entry.AddComponent(ActionCooldownTag)
	StatusComponentType.Set(entry, status)

	showGameMessage(ecs, logMsg, func() {
		gsEntry, _ := GameStateComponentType.First(ecs.World)
		gs := GameStateComponentType.Get(gsEntry)
		if gs.CurrentState == GameStateMessage { // 他の要因で状態が変わっていなければ
			gs.CurrentState = StatePlaying
			GameStateComponentType.Set(gsEntry, gs)
		}
	})
}

func handleHeadDestruction(targetEntry *donburi.Entry) {
	if !targetEntry.HasComponent(BrokenTag) {
		targetEntry.AddComponent(BrokenTag)
		status := StatusComponentType.Get(targetEntry)
		status.State = StateBroken
		status.Gauge = 0
		StatusComponentType.Set(targetEntry, status)
	}
}

// Utility functions to get component data
func getAttackerData(entry *donburi.Entry) (*IdentityComponent, *MedalComponent, *Part) {
	id := IdentityComponentType.Get(entry)
	medal := CMedal.Get(entry)
	action := ActionComponentType.Get(entry)
	part := PartsComponentType.Get(entry).Parts[action.SelectedPartKey]
	return id, medal, part
}
func getTargetData(entry *donburi.Entry) (*IdentityComponent, *StatusComponent, *PartsComponent) {
	id := IdentityComponentType.Get(entry)
	status := StatusComponentType.Get(entry)
	parts := PartsComponentType.Get(entry)
	return id, status, parts
}
