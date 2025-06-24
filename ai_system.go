package main

import (
	"math/rand"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// AISystem はAIメダロットの行動を決定します。
type AISystem struct {
	aiQuery         *donburi.Query
	targetableQuery *donburi.Query
}

func NewAISystem() *AISystem {
	return &AISystem{
		aiQuery: donburi.NewQuery(
			filter.And(
				filter.Contains(AIControlledComponentType), filter.Contains(StatusComponentType), filter.Contains(PartsComponentType),
				filter.Contains(ActionComponentType), filter.Contains(IdentityComponentType), filter.Not(filter.Contains(BrokenTag)),
			),
		),
		targetableQuery: donburi.NewQuery(
			filter.And(
				filter.Contains(IdentityComponentType), filter.Contains(StatusComponentType), filter.Not(filter.Contains(BrokenTag)),
			),
		),
	}
}
func (sys *AISystem) Update(ecs *ecs.ECS) {
	gameStateEntry, gameStateOk := GameStateComponentType.First(ecs.World)
	if !gameStateOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState != StatePlaying {
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
		availablePartSlots := []PartSlotKey{}
		slotsToConsider := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
		rand.Shuffle(len(slotsToConsider), func(i, j int) { slotsToConsider[i], slotsToConsider[j] = slotsToConsider[j], slotsToConsider[i] })
		for _, slotKey := range slotsToConsider {
			part, exists := partsComp.Parts[slotKey]
			if exists && !part.IsBroken && part.Charge > 0 {
				availablePartSlots = append(availablePartSlots, slotKey)
			}
		}
		if len(availablePartSlots) == 0 {
			return
		}
		selectedSlotKey := availablePartSlots[0]
		selectedPartForAI := partsComp.Parts[selectedSlotKey]

		var opponentTeam TeamID = Team1
		if aiIdentity.Team == Team1 {
			opponentTeam = Team2
		}
		candidates := []donburi.Entity{}
		sys.targetableQuery.Each(ecs.World, func(targetEntry *donburi.Entry) {
			if IdentityComponentType.Get(targetEntry).Team == opponentTeam && !StatusComponentType.Get(targetEntry).State_is_broken_internal() { // Assuming State_is_broken_internal exists or is added to StatusComponent
				candidates = append(candidates, targetEntry.Entity())
			}
		})
		if len(candidates) == 0 {
			return
		}
		actionComp.TargetedMedarot = candidates[rand.Intn(len(candidates))]

		if selectedPartForAI.Category == CategoryShoot && !ecs.World.Valid(actionComp.TargetedMedarot) {
			actionComp.TargetedMedarot = donburi.Entity(0)
			return
		}

		actionComp.SelectedPartKey = selectedSlotKey
		status.State = StateActionCharging
		status.Gauge = 0
		if !selectedPartForAI.IsBroken {
			switch selectedPartForAI.Trait {
			case TraitAim:
				status.IsEvasionDisabled = true
			case TraitStrike:
				status.IsDefenseDisabled = true
			case TraitBerserk:
				status.IsEvasionDisabled = true
				status.IsDefenseDisabled = true
			}
		}
		entry.AddComponent(ActionChargingTag)
		StatusComponentType.Set(entry, status)
		ActionComponentType.Set(entry, actionComp)
	})
}

// Note: State_is_broken_internal is now defined in action_execution_system.go.
