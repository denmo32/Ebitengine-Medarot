package main

import (
	"fmt" // Keep fmt for Sprintf
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
	"math/rand" // Keep rand for random target selection in fight if primary target is invalid
)

// ActionExecutionSystem は選択されたアクションを実行します。
type ActionExecutionSystem struct{ query *donburi.Query }

func NewActionExecutionSystem() *ActionExecutionSystem {
	return &ActionExecutionSystem{
		query: donburi.NewQuery(filter.And(
			filter.Contains(ReadyToExecuteActionTag), filter.Contains(ActionComponentType), filter.Contains(StatusComponentType),
			filter.Contains(PartsComponentType), filter.Contains(CMedal), filter.Contains(IdentityComponentType), filter.Not(filter.Contains(BrokenTag)),
		)),
	}
}

func (sys *ActionExecutionSystem) Update(ecs *ecs.ECS) {
	configEntry, configOk := ConfigComponentType.First(ecs.World)
	if !configOk {
		return
	}
	gameConfig := ConfigComponentType.Get(configEntry).GameConfig
	balanceConfig := gameConfig.Balance

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		actionComp := ActionComponentType.Get(entry)
		statusComp := StatusComponentType.Get(entry)
		partsComp := PartsComponentType.Get(entry)
		identityComp := IdentityComponentType.Get(entry)
		medalComp := CMedal.Get(entry)
		selectedPart := partsComp.Parts[actionComp.SelectedPartKey]

		if selectedPart == nil || selectedPart.IsBroken {
			statusComp.State = StateReadyToSelectAction
			statusComp.Gauge = 100
			entry.RemoveComponent(ReadyToExecuteActionTag)
			StatusComponentType.Set(entry, statusComp)
			actionComp.LastActionLog = fmt.Sprintf("%s:パーツ失敗", identityComp.Name)
			showGameMessage(ecs, actionComp.LastActionLog, func() {
				gsEntry, _ := GameStateComponentType.First(ecs.World)
				gs := GameStateComponentType.Get(gsEntry)
				gs.CurrentState = StatePlaying
				GameStateComponentType.Set(gsEntry, gs)
			})
			ActionComponentType.Set(entry, actionComp)
			return
		}

		executeActualAction := func() {
			actualTargetEntity := actionComp.TargetedMedarot
			var actualTargetEntry *donburi.Entry
			targetIsValid := false
			if ecs.World.Valid(actualTargetEntity) {
				targetCandEntry := ecs.World.Entry(actualTargetEntity)
				if targetCandEntry.Valid() && !StatusComponentType.Get(targetCandEntry).State_is_broken_internal() {
					actualTargetEntry = targetCandEntry
					targetIsValid = true
				}
			}
			if selectedPart.Category == CategoryFight && !targetIsValid {
				var opponentTeam TeamID
				if identityComp.Team == Team1 {
					opponentTeam = Team2
				} else {
					opponentTeam = Team1
				}
				candidates := []donburi.Entity{}
				tgtQuery := donburi.NewQuery(filter.And(filter.Contains(IdentityComponentType), filter.Not(filter.Contains(BrokenTag))))
				tgtQuery.Each(ecs.World, func(pTargetEntry *donburi.Entry) {
					if IdentityComponentType.Get(pTargetEntry).Team == opponentTeam && !StatusComponentType.Get(pTargetEntry).State_is_broken_internal() {
						candidates = append(candidates, pTargetEntry.Entity())
					}
				})
				if len(candidates) > 0 {
					actualTargetEntity = candidates[rand.Intn(len(candidates))]
					actualTargetEntry = ecs.World.Entry(actualTargetEntity)
					targetIsValid = true
					actionComp.TargetedMedarot = actualTargetEntity
				}
			}

			if selectedPart.Category == CategoryShoot && !targetIsValid {
				actionComp.LastActionLog = fmt.Sprintf("%s:射撃失敗", identityComp.Name)
			} else if (selectedPart.Category == CategoryFight) && !targetIsValid {
				actionComp.LastActionLog = fmt.Sprintf("%s:ターゲットなし", identityComp.Name)
			} else if targetIsValid && actualTargetEntry != nil {
				targetIdentityComp := IdentityComponentType.Get(actualTargetEntry)
				targetStatusComp := StatusComponentType.Get(actualTargetEntry)
				targetPartsComp := PartsComponentType.Get(actualTargetEntry)

				isHit, isCritical := calculateHit(identityComp, medalComp, selectedPart, targetIdentityComp, targetStatusComp, targetPartsComp.Parts[PartSlotLegs], balanceConfig)
				if isHit {
					targetPartToDamage := selectRandomPartToDamage(actualTargetEntry, targetPartsComp)
					if targetPartToDamage != nil {
						damage := calculateDamage_refactored(entry, medalComp, selectedPart, targetPartToDamage, targetPartsComp.Parts[PartSlotLegs], isCritical, balanceConfig, targetStatusComp.IsDefenseDisabled)
						origArmor := targetPartToDamage.Armor
						targetPartToDamage.Armor -= damage
						if targetPartToDamage.Armor < 0 {
							targetPartToDamage.Armor = 0
						}
						logMsg := fmt.Sprintf("%s %s %d dmg (%d->%d)", targetIdentityComp.Name, targetPartToDamage.PartName, damage, origArmor, targetPartToDamage.Armor)
						if isCritical {
							logMsg = "CRIT! " + logMsg
						}
						if targetPartToDamage.Armor == 0 && !targetPartToDamage.IsBroken {
							targetPartToDamage.IsBroken = true
							logMsg += "破壊!"
							if targetPartToDamage.Type == PartTypeHead {
								if !actualTargetEntry.HasComponent(BrokenTag) {
									actualTargetEntry.AddComponent(BrokenTag)
									targetStatusComp.State = StateBroken
									targetStatusComp.Gauge = 0
									StatusComponentType.Set(actualTargetEntry, targetStatusComp)
								}
							}
						}
						actionComp.LastActionLog = logMsg
						PartsComponentType.Set(actualTargetEntry, targetPartsComp)
					} else {
						actionComp.LastActionLog = fmt.Sprintf("%s対象部位なし", targetIdentityComp.Name)
					}
				} else {
					actionComp.LastActionLog = fmt.Sprintf("%sへの%s攻撃回避", targetIdentityComp.Name, identityComp.Name)
				}
			} else {
				actionComp.LastActionLog = fmt.Sprintf("%sは%s使用", identityComp.Name, selectedPart.PartName)
			}

			statusComp.State = StateActionCooldown
			statusComp.Gauge = 0
			entry.RemoveComponent(ReadyToExecuteActionTag)
			entry.AddComponent(ActionCooldownTag)
			StatusComponentType.Set(entry, statusComp)
			ActionComponentType.Set(entry, actionComp)
			showGameMessage(ecs, actionComp.LastActionLog, func() {
				gsEntry, _ := GameStateComponentType.First(ecs.World)
				gs := GameStateComponentType.Get(gsEntry)
				if gs.CurrentState == GameStateMessage {
					gs.CurrentState = StatePlaying
					GameStateComponentType.Set(gsEntry, gs)
				}
			})
		}
		actionVerb := string(selectedPart.Category)
		targetInfo := ""
		if selectedPart.Category == CategoryShoot && ecs.World.Valid(actionComp.TargetedMedarot) {
			if tgtEntry := ecs.World.Entry(actionComp.TargetedMedarot); tgtEntry.Valid() {
				targetInfo = fmt.Sprintf(" -> %s", IdentityComponentType.Get(tgtEntry).Name)
			}
		}
		initialMessage := fmt.Sprintf("%s:%s(%s)%s！", identityComp.Name, selectedPart.PartName, actionVerb, targetInfo)
		showGameMessage(ecs, initialMessage, executeActualAction)
	})
}

// --- Helper functions for ActionExecutionSystem ---
func (s *StatusComponent) State_is_broken_internal() bool { return s.State == StateBroken }
