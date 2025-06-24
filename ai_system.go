// --- ai_system.go ---
package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
	"math/rand"
)

// AISystem はAIメダロットの行動を決定します。
type AISystem struct {
	aiQuery         *donburi.Query
	targetableQuery *donburi.Query
}

func NewAISystem() *AISystem {
	return &AISystem{
		aiQuery: donburi.NewQuery(filter.And(
			filter.Contains(AIControlledComponentType), filter.Contains(StatusComponentType),
			filter.Contains(PartsComponentType), filter.Contains(ActionComponentType),
			filter.Contains(IdentityComponentType), filter.Not(filter.Contains(BrokenTag)),
		)),
		targetableQuery: donburi.NewQuery(filter.And(
			filter.Contains(IdentityComponentType), filter.Contains(StatusComponentType),
			filter.Not(filter.Contains(BrokenTag)),
		)),
	}
}

func (sys *AISystem) Update(ecs *ecs.ECS) {
	gs, ok := GameStateComponentType.First(ecs.World)
	if !ok || GameStateComponentType.Get(gs).CurrentState != StatePlaying {
		return
	}

	sys.aiQuery.Each(ecs.World, func(entry *donburi.Entry) {
		status := StatusComponentType.Get(entry)
		if status.State != StateReadyToSelectAction {
			return
		}

		partsComp := PartsComponentType.Get(entry)
		actionComp := ActionComponentType.Get(entry)
		aiIdentity := IdentityComponentType.Get(entry)

		// 1. 使用可能なパーツを選ぶ
		availablePartSlots := []PartSlotKey{}
		slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
		rand.Shuffle(len(slots), func(i, j int) { slots[i], slots[j] = slots[j], slots[i] })
		for _, slotKey := range slots {
			part, exists := partsComp.Parts[slotKey]
			if exists && !part.IsBroken && part.Charge > 0 {
				availablePartSlots = append(availablePartSlots, slotKey)
			}
		}
		if len(availablePartSlots) == 0 {
			return
		}
		selectedSlotKey := availablePartSlots[0]
		selectedPart := partsComp.Parts[selectedSlotKey]

		// 2. ターゲットを選ぶ
		var opponentTeam TeamID = Team1
		if aiIdentity.Team == Team1 {
			opponentTeam = Team2
		}
		candidates := []donburi.Entity{}
		sys.targetableQuery.Each(ecs.World, func(targetEntry *donburi.Entry) {
			// ★★★ 修正箇所: IsBroken() メソッドを使用 ★★★
			if IdentityComponentType.Get(targetEntry).Team == opponentTeam && !StatusComponentType.Get(targetEntry).IsBroken() {
				candidates = append(candidates, targetEntry.Entity())
			}
		})

		if selectedPart.Category == CategoryShoot || selectedPart.Category == CategoryFight {
			if len(candidates) == 0 {
				return
			} // 攻撃対象がいなければ行動しない
			actionComp.TargetedMedarot = candidates[rand.Intn(len(candidates))]
		}

		// 3. アクションを確定
		actionComp.SelectedPartKey = selectedSlotKey
		status.State = StateActionCharging
		status.Gauge = 0
		switch selectedPart.Trait {
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
	})
}
