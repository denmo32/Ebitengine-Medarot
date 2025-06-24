package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"sort" // RenderSystemでソートに使用

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
	// "github.com/yohamta/donburi/features/math" // RenderComponentで使う場合は必要
)

// GaugeUpdateSystem はメダロットのゲージを更新します。
type GaugeUpdateSystem struct {
	query *donburi.Query
}

func NewGaugeUpdateSystem() *GaugeUpdateSystem {
	return &GaugeUpdateSystem{
		query: donburi.NewQuery(
			filter.And(
				filter.Contains(StatusComponentType),
				filter.Contains(PartsComponentType),
				filter.Contains(CMedal), // CMedal を使用
				filter.Contains(ActionComponentType),
				filter.Not(filter.Contains(BrokenTag)),
			)),
	}
}

func (sys *GaugeUpdateSystem) Update(ecs *ecs.ECS) {
	configEntry, ok := ConfigComponentType.First(ecs.World)
	if !ok {
		return
	}
	gameConfig := ConfigComponentType.Get(configEntry).GameConfig
	if gameConfig == nil {
		return
	}

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		status := StatusComponentType.Get(entry)
		parts := PartsComponentType.Get(entry)
		actionComp := ActionComponentType.Get(entry)

		headPart, headExists := parts.Parts[PartSlotHead]
		if headExists && headPart.IsBroken {
			if !entry.HasComponent(BrokenTag) {
				entry.AddComponent(BrokenTag)
				status.State = StateBroken
				status.Gauge = 0
				StatusComponentType.Set(entry, status)
			}
			return
		}

		legsPart := parts.Parts[PartSlotLegs]
		legPropulsion := 0
		if legsPart != nil && !legsPart.IsBroken {
			legPropulsion = legsPart.Propulsion
		}

		var selectedPart *Part
		if status.State == StateActionCharging || status.State == StateActionCooldown {
			if actionComp.SelectedPartKey != "" {
				part, exists := parts.Parts[actionComp.SelectedPartKey]
				if exists && !part.IsBroken {
					selectedPart = part
				}
			}
		}

		if selectedPart == nil {
			if status.State == StateActionCharging || status.State == StateActionCooldown {
				status.State = StateReadyToSelectAction
				status.Gauge = 100.0
				actionComp.SelectedPartKey = ""
				status.IsEvasionDisabled = false
				status.IsDefenseDisabled = false
				entry.RemoveComponent(ActionChargingTag)
				entry.RemoveComponent(ActionCooldownTag)
				StatusComponentType.Set(entry, status)
				ActionComponentType.Set(entry, actionComp)
			}
			return
		}

		stat := 0
		if status.State == StateActionCharging {
			stat = selectedPart.Charge
		} else if status.State == StateActionCooldown {
			stat = selectedPart.Cooldown
		} else {
			return
		}

		cfgBalance := gameConfig.Balance
		moveSpeed := (float64(stat) + float64(legPropulsion)*cfgBalance.Time.PropulsionEffectRate) / cfgBalance.Time.OverallTimeDivisor
		status.Gauge += moveSpeed

		if status.Gauge >= 100.0 {
			status.Gauge = 100.0
			if status.State == StateActionCharging {
				status.State = StateReadyToExecuteAction
				entry.RemoveComponent(ActionChargingTag)
				entry.AddComponent(ReadyToExecuteActionTag)
			} else if status.State == StateActionCooldown {
				status.State = StateReadyToSelectAction
				entry.RemoveComponent(ActionCooldownTag)
				status.IsEvasionDisabled = false
				status.IsDefenseDisabled = false
				actionComp.TargetedMedarot = donburi.Entity(0) // ★ donburi.Entity{} を donburi.Entity(0) に修正
				actionComp.SelectedPartKey = ""
				ActionComponentType.Set(entry, actionComp)
			}
		}
		StatusComponentType.Set(entry, status)
	})
}

// PlayerInputSystem はプレイヤーの入力を処理し、行動選択の準備をします。
type PlayerInputSystem struct {
	actionSelectQuery *donburi.Query
	targetableQuery   *donburi.Query
}

func NewPlayerInputSystem() *PlayerInputSystem {
	return &PlayerInputSystem{
		actionSelectQuery: donburi.NewQuery(
			filter.And(
				filter.Contains(PlayerControlledComponentType),
				filter.Contains(StatusComponentType),
				filter.Contains(PartsComponentType),
				filter.Contains(ActionComponentType),
				filter.Contains(IdentityComponentType),
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

func (sys *PlayerInputSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, ok := GameStateComponentType.First(ecs.World)
	if !ok {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	pasComp := PlayerActionSelectComponentType.Get(gameStateEntry)
	configComp := ConfigComponentType.Get(gameStateEntry)
	if configComp.GameConfig == nil {
		return
	}
	uiConfig := configComp.GameConfig.UI

	if gs.CurrentState != StatePlaying && gs.CurrentState != StatePlayerActionSelect {
		// pasComp.ActingMedarot.Valid() を ecs.World.Valid(pasComp.ActingMedarot) に修正
		if ecs.World.Valid(pasComp.ActingMedarot) && gs.CurrentState != StatePlayerActionSelect {
			pasComp.ActingMedarot = donburi.Entity(0) // ★ donburi.Entity{} を donburi.Entity(0) に修正
			pasComp.CurrentTarget = donburi.Entity(0) // ★ donburi.Entity{} を donburi.Entity(0) に修正
			pasComp.AvailableActions = nil
			PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
		}
		return
	}

	// pasComp.ActingMedarot.Valid() を ecs.World.Valid(pasComp.ActingMedarot) に修正
	if gs.CurrentState == StatePlayerActionSelect && ecs.World.Valid(pasComp.ActingMedarot) {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			actingMedarotEntry := ecs.World.Entry(pasComp.ActingMedarot)
			if !actingMedarotEntry.Valid() { // entry.Valid() はOK
				gs.CurrentState = StatePlaying
				pasComp.ActingMedarot = donburi.Entity(0)
				pasComp.CurrentTarget = donburi.Entity(0)
				pasComp.AvailableActions = nil // ★
				PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
				GameStateComponentType.Set(gameStateEntry, gs)
				return
			}

			status := StatusComponentType.Get(actingMedarotEntry)
			partsComp := PartsComponentType.Get(actingMedarotEntry)
			actionComp := ActionComponentType.Get(actingMedarotEntry)
			mx, my := ebiten.CursorPosition()

			for i, slotKey := range pasComp.AvailableActions {
				partData, partExists := partsComp.Parts[slotKey]
				if !partExists || partData.IsBroken || partData.Charge <= 0 {
					continue
				}

				btnW := uiConfig.ActionModal.ButtonWidth
				btnH := uiConfig.ActionModal.ButtonHeight
				btnSpacing := uiConfig.ActionModal.ButtonSpacing
				buttonX := uiConfig.Screen.Width/2 - int(btnW/2)
				buttonY := uiConfig.Screen.Height/2 - 50 + (int(btnH)+int(btnSpacing))*i
				buttonRect := image.Rect(buttonX, buttonY, buttonX+int(btnW), buttonY+int(btnH))

				if (image.Point{X: mx, Y: my}).In(buttonRect) {
					targetEntry := ecs.World.Entry(pasComp.CurrentTarget)
					// targetEntry.Valid() はOK
					targetIsValid := targetEntry.Valid() && !StatusComponentType.Get(targetEntry).State_is_broken_internal()

					if partData.Category == CategoryShoot {
						if !targetIsValid {
							gs.CurrentState = StatePlaying
							pasComp.ActingMedarot = donburi.Entity(0) // ★
							PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
							GameStateComponentType.Set(gameStateEntry, gs)
							return
						}
						actionComp.TargetedMedarot = pasComp.CurrentTarget
					} else if partData.Category == CategoryFight {
						if !targetIsValid {
							gs.CurrentState = StatePlaying
							pasComp.ActingMedarot = donburi.Entity(0) // ★
							PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
							GameStateComponentType.Set(gameStateEntry, gs)
							return
						}
						actionComp.TargetedMedarot = pasComp.CurrentTarget
					}

					actionComp.SelectedPartKey = slotKey
					status.State = StateActionCharging
					status.Gauge = 0
					if !partData.IsBroken {
						switch partData.Trait {
						case TraitAim:
							status.IsEvasionDisabled = true
						case TraitStrike:
							status.IsDefenseDisabled = true
						case TraitBerserk:
							status.IsEvasionDisabled = true
							status.IsDefenseDisabled = true
						}
					}
					actingMedarotEntry.AddComponent(ActionChargingTag)
					StatusComponentType.Set(actingMedarotEntry, status)
					ActionComponentType.Set(actingMedarotEntry, actionComp)

					gs.CurrentState = StatePlaying
					pasComp.ActingMedarot = donburi.Entity(0)
					pasComp.CurrentTarget = donburi.Entity(0)
					pasComp.AvailableActions = nil // ★
					PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
					GameStateComponentType.Set(gameStateEntry, gs)
					return
				}
			}
		}
		return
	}

	// !pasComp.ActingMedarot.Valid() を !ecs.World.Valid(pasComp.ActingMedarot) に修正
	if gs.CurrentState == StatePlaying && !ecs.World.Valid(pasComp.ActingMedarot) {
		var foundActingMedarotThisFrame bool = false
		sys.actionSelectQuery.Each(ecs.World, func(entry *donburi.Entry) {
			if foundActingMedarotThisFrame {
				return
			}
			status := StatusComponentType.Get(entry)
			if status.State == StateReadyToSelectAction {
				pasComp.ActingMedarot = entry.Entity()
				partsComp := PartsComponentType.Get(entry)
				pasComp.AvailableActions = []PartSlotKey{}
				slotsForActionUI := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
				for _, slotKey := range slotsForActionUI {
					part, exists := partsComp.Parts[slotKey]
					if exists && !part.IsBroken && part.Charge > 0 {
						pasComp.AvailableActions = append(pasComp.AvailableActions, slotKey)
					}
				}
				if len(pasComp.AvailableActions) == 0 {
					pasComp.ActingMedarot = donburi.Entity(0) // ★
					return
				}

				actingID := IdentityComponentType.Get(entry)
				var opponentTeam TeamID = Team2
				if actingID.Team == Team2 {
					opponentTeam = Team1
				}
				candidates := []donburi.Entity{}
				sys.targetableQuery.Each(ecs.World, func(targetEntry *donburi.Entry) {
					if IdentityComponentType.Get(targetEntry).Team == opponentTeam && !StatusComponentType.Get(targetEntry).State_is_broken_internal() {
						candidates = append(candidates, targetEntry.Entity())
					}
				})
				if len(candidates) > 0 {
					pasComp.CurrentTarget = candidates[rand.Intn(len(candidates))]
				} else {
					pasComp.CurrentTarget = donburi.Entity(0)
				} // ★

				gs.CurrentState = StatePlayerActionSelect
				PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
				GameStateComponentType.Set(gameStateEntry, gs)
				foundActingMedarotThisFrame = true
			}
		})
	}
}

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
			if IdentityComponentType.Get(targetEntry).Team == opponentTeam && !StatusComponentType.Get(targetEntry).State_is_broken_internal() {
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
func showGameMessage(ecs *ecs.ECS, msg string, callback func()) {
	entry, ok := GameStateComponentType.First(ecs.World)
	if !ok {
		if callback != nil {
			callback()
		}
		return
	}
	gs := GameStateComponentType.Get(entry)
	gs.Message = msg
	gs.PostMessageCallback = callback
	gs.CurrentState = GameStateMessage
	GameStateComponentType.Set(entry, gs)
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
func calculateHit(attackerID *IdentityComponent, attackerMedal *MedalComponent, attackerPart *Part, targetID *IdentityComponent, targetStatus *StatusComponent, targetLegs *Part, cfg BalanceConfig) (bool, bool) {
	skillValue := 0
	if attackerPart.Category == CategoryShoot {
		skillValue = attackerMedal.Medal.SkillShoot
	} else {
		skillValue = attackerMedal.Medal.SkillFight
	}
	traitBonus := 0
	switch attackerPart.Trait {
	case TraitAim:
		traitBonus = cfg.Hit.TraitAimBonus
	case TraitStrike:
		traitBonus = cfg.Hit.TraitStrikeBonus
	case TraitBerserk:
		traitBonus = cfg.Hit.TraitBerserkDebuff
	}
	finalAccuracy := attackerPart.Accuracy + skillValue + traitBonus
	targetMobility := 0
	if targetLegs != nil && !targetLegs.IsBroken {
		targetMobility = targetLegs.Mobility
	}
	if targetStatus.IsEvasionDisabled {
		targetMobility = 0
	}
	hitChance := cfg.Hit.BaseChance + finalAccuracy - targetMobility
	isHit := rand.Intn(100) < hitChance
	isCritical := false
	if isHit && hitChance > 100 {
		if rand.Intn(100) < (hitChance - 100) {
			isCritical = true
		}
	}
	return isHit, isCritical
}
func selectRandomPartToDamage(targetEntry *donburi.Entry, targetPartsComp *PartsComponent) *Part {
	vulnerable := []*Part{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if p, ok := targetPartsComp.Parts[s]; ok && !p.IsBroken {
			vulnerable = append(vulnerable, p)
		}
	}
	if len(vulnerable) == 0 {
		return nil
	}
	return vulnerable[rand.Intn(len(vulnerable))]
}

// GameRuleSystem はゲームの勝敗判定などを行います。
type GameRuleSystem struct{ query *donburi.Query }

func NewGameRuleSystem() *GameRuleSystem {
	return &GameRuleSystem{
		query: donburi.NewQuery(filter.And(filter.Contains(IdentityComponentType), filter.Contains(StatusComponentType))),
	}
}
func (sys *GameRuleSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, gameStateOk := GameStateComponentType.First(ecs.World)
	if !gameStateOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState == GameStateOver {
		return
	}

	team1LeaderAlive := false
	team2LeaderAlive := false

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		identity := IdentityComponentType.Get(entry)
		status := StatusComponentType.Get(entry)
		isAlive := !status.State_is_broken_internal()
		if identity.Team == Team1 {
			if identity.IsLeader && isAlive {
				team1LeaderAlive = true
			}
		} else if identity.Team == Team2 {
			if identity.IsLeader && isAlive {
				team2LeaderAlive = true
			}
		}
	})

	var team1LeaderExists, team2LeaderExists bool
	leaderQuery := donburi.NewQuery(filter.Contains(IdentityComponentType))
	leaderQuery.Each(ecs.World, func(entry *donburi.Entry) {
		id := IdentityComponentType.Get(entry)
		if id.IsLeader {
			if id.Team == Team1 {
				team1LeaderExists = true
			}
			if id.Team == Team2 {
				team2LeaderExists = true
			}
		}
	})

	if team1LeaderExists && !team1LeaderAlive {
		gs.Winner = Team2
		gs.CurrentState = GameStateOver
		gs.Message = "チーム2の勝利！"
		showGameMessage(ecs, gs.Message, nil)
		return
	}
	if team2LeaderExists && !team2LeaderAlive {
		gs.Winner = Team1
		gs.CurrentState = GameStateOver
		gs.Message = "チーム1の勝利！"
		showGameMessage(ecs, gs.Message, nil)
		return
	}
}

// MessageSystem はメッセージ表示中のクリックを処理します。
type MessageSystem struct{}

func NewMessageSystem() *MessageSystem { return &MessageSystem{} }
func (sys *MessageSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, gameStateOk := GameStateComponentType.First(ecs.World)
	if !gameStateOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState == GameStateMessage {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			callback := gs.PostMessageCallback
			gs.PostMessageCallback = nil
			GameStateComponentType.Set(gameStateEntry, gs)
			if callback != nil {
				callback()
			} else {
				gsAfterCb := GameStateComponentType.Get(gameStateEntry)
				if gsAfterCb.CurrentState == GameStateMessage {
					gsAfterCb.CurrentState = StatePlaying
					GameStateComponentType.Set(gameStateEntry, gsAfterCb)
				}
			}
		}
	}
}

// RenderSystem はゲームの描画を担当します。
type RenderSystem struct {
	medarotQuery *donburi.Query
}

// Update は System インターフェースを満たすために追加（現在は空）。
func (sys *RenderSystem) Update(ecs *ecs.ECS) {
	// 描画システムは通常Updateロジックを持たないことが多いが、
	// Gameのsystemsスライスに追加するために必要。
}

func NewRenderSystem() *RenderSystem {
	return &RenderSystem{
		medarotQuery: donburi.NewQuery(filter.And(
			filter.Contains(IdentityComponentType), filter.Contains(StatusComponentType),
			filter.Contains(RenderComponentType), filter.Contains(PartsComponentType),
		)),
	}
}

type MedarotDrawInfo struct {
	Entry    *donburi.Entry
	Identity *IdentityComponent
	Status   *StatusComponent
	Render   *RenderComponent
	Parts    *PartsComponent
}

func (sys *RenderSystem) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	gameStateEntry, gsOk := GameStateComponentType.First(ecs.World)
	configEntry, cfgOk := ConfigComponentType.First(ecs.World)
	if !gsOk || !cfgOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	appConfig := ConfigComponentType.Get(configEntry).GameConfig
	pasComp := PlayerActionSelectComponentType.Get(gameStateEntry)

	screen.Fill(appConfig.UI.Colors.Background)
	vector.StrokeRect(screen, 0, 0, float32(appConfig.UI.Screen.Width), appConfig.UI.Battlefield.Height, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.White, false)

	medarotCount := 0
	countQuery := donburi.NewQuery(filter.Contains(IdentityComponentType))
	countQuery.Each(ecs.World, func(_ *donburi.Entry) { medarotCount++ })
	playersPerTeam := medarotCount / 2
	if playersPerTeam == 0 && medarotCount > 0 {
		playersPerTeam = 1
	}

	for i := 0; i < playersPerTeam; i++ {
		yPos := appConfig.UI.Battlefield.MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, appConfig.UI.Battlefield.Team1HomeX, yPos, appConfig.UI.Battlefield.HomeMarkerRadius, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, appConfig.UI.Battlefield.Team2HomeX, yPos, appConfig.UI.Battlefield.HomeMarkerRadius, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, appConfig.UI.Battlefield.Team1ExecutionLineX, 0, appConfig.UI.Battlefield.Team1ExecutionLineX, appConfig.UI.Battlefield.Height, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, false)
	vector.StrokeLine(screen, appConfig.UI.Battlefield.Team2ExecutionLineX, 0, appConfig.UI.Battlefield.Team2ExecutionLineX, appConfig.UI.Battlefield.Height, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, false)

	allMedarotsToDraw := []MedarotDrawInfo{}
	sys.medarotQuery.Each(ecs.World, func(entry *donburi.Entry) {
		allMedarotsToDraw = append(allMedarotsToDraw, MedarotDrawInfo{
			Entry: entry, Identity: IdentityComponentType.Get(entry), Status: StatusComponentType.Get(entry),
			Render: RenderComponentType.Get(entry), Parts: PartsComponentType.Get(entry),
		})
	})
	sort.Slice(allMedarotsToDraw, func(i, j int) bool {
		if allMedarotsToDraw[i].Identity.Team != allMedarotsToDraw[j].Identity.Team {
			return allMedarotsToDraw[i].Identity.Team < allMedarotsToDraw[j].Identity.Team
		}
		return allMedarotsToDraw[i].Render.DrawIndex < allMedarotsToDraw[j].Render.DrawIndex
	})

	for _, mdi := range allMedarotsToDraw {
		statusComp := mdi.Status
		identityComp := mdi.Identity
		renderComp := mdi.Render
		baseYPos := appConfig.UI.Battlefield.MedarotVerticalSpacing * float32(renderComp.DrawIndex+1)
		progress := statusComp.Gauge / 100.0
		homeX, execX := appConfig.UI.Battlefield.Team1HomeX, appConfig.UI.Battlefield.Team1ExecutionLineX
		if identityComp.Team == Team2 {
			homeX, execX = appConfig.UI.Battlefield.Team2HomeX, appConfig.UI.Battlefield.Team2ExecutionLineX
		}
		var currentX float32
		switch statusComp.State {
		case StateActionCharging:
			currentX = homeX + float32(progress)*(execX-homeX)
		case StateReadyToExecuteAction:
			currentX = execX
		case StateActionCooldown:
			currentX = execX - float32(progress)*(execX-homeX)
		default:
			currentX = homeX
		}
		if currentX < appConfig.UI.Battlefield.IconRadius {
			currentX = appConfig.UI.Battlefield.IconRadius
		}
		if currentX > float32(appConfig.UI.Screen.Width)-appConfig.UI.Battlefield.IconRadius {
			currentX = float32(appConfig.UI.Screen.Width) - appConfig.UI.Battlefield.IconRadius
		}

		iconColor := appConfig.UI.Colors.Team1
		if identityComp.Team == Team2 {
			iconColor = appConfig.UI.Colors.Team2
		}
		if statusComp.State_is_broken_internal() || mdi.Entry.HasComponent(BrokenTag) {
			iconColor = appConfig.UI.Colors.Broken
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, appConfig.UI.Battlefield.IconRadius, iconColor, true)
		if identityComp.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, appConfig.UI.Battlefield.IconRadius+2, 2, appConfig.UI.Colors.Leader, true)
		}
	}

	for _, mdi := range allMedarotsToDraw {
		identityComp := mdi.Identity
		statusComp := mdi.Status
		renderComp := mdi.Render
		partsComp := mdi.Parts
		var panelX, panelY float32
		if identityComp.Team == Team1 {
			panelX = appConfig.UI.InfoPanel.Padding
			panelY = appConfig.UI.InfoPanel.StartY + appConfig.UI.InfoPanel.Padding + float32(renderComp.DrawIndex)*(appConfig.UI.InfoPanel.BlockHeight+appConfig.UI.InfoPanel.Padding)
		} else {
			panelX = appConfig.UI.InfoPanel.Padding*2 + appConfig.UI.InfoPanel.BlockWidth
			panelY = appConfig.UI.InfoPanel.StartY + appConfig.UI.InfoPanel.Padding + float32(renderComp.DrawIndex)*(appConfig.UI.InfoPanel.BlockHeight+appConfig.UI.InfoPanel.Padding)
		}
		drawMedarotInfoECS(screen, identityComp, statusComp, partsComp, panelX, panelY, appConfig, gs.DebugMode)
	}

	if gs.CurrentState == StatePlayerActionSelect && ecs.World.Valid(pasComp.ActingMedarot) {
		actingMedarotEntry := ecs.World.Entry(pasComp.ActingMedarot)
		if actingMedarotEntry.Valid() {
			identity := IdentityComponentType.Get(actingMedarotEntry)
			overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
			vector.DrawFilledRect(screen, 0, 0, float32(appConfig.UI.Screen.Width), float32(appConfig.UI.Screen.Height), overlayColor, false)
			boxW, boxH := 320, 200
			boxX := (appConfig.UI.Screen.Width - boxW) / 2
			boxY := (appConfig.UI.Screen.Height - boxH) / 2
			windowRect := image.Rect(boxX, boxY, boxX+boxW, boxY+boxH)
			DrawWindow(screen, windowRect, appConfig.UI.Colors.Background, appConfig.UI.Colors.Team1)
			titleStr := fmt.Sprintf("%s の行動を選択", identity.Name)
			if MplusFont != nil {
				bounds := text.BoundString(MplusFont, titleStr)
				titleWidth := (bounds.Max.X - bounds.Min.X)
				text.Draw(screen, titleStr, MplusFont, appConfig.UI.Screen.Width/2-titleWidth/2, boxY+30, appConfig.UI.Colors.White)
			}
			actingPartsComp := PartsComponentType.Get(actingMedarotEntry)
			for i, slotKey := range pasComp.AvailableActions {
				partData, exists := actingPartsComp.Parts[slotKey]
				if !exists {
					continue
				}
				btnW_modal := appConfig.UI.ActionModal.ButtonWidth
				btnH_modal := appConfig.UI.ActionModal.ButtonHeight
				btnSpacing_modal := appConfig.UI.ActionModal.ButtonSpacing
				buttonX_modal := appConfig.UI.Screen.Width/2 - int(btnW_modal/2)
				buttonY_modal := appConfig.UI.Screen.Height/2 - 50 + (int(btnH_modal)+int(btnSpacing_modal))*i
				buttonRect_modal := image.Rect(buttonX_modal, buttonY_modal, buttonX_modal+int(btnW_modal), buttonY_modal+int(btnH_modal))
				partStr := fmt.Sprintf("%s (%s)", partData.PartName, partData.Type)
				if partData.Category == CategoryShoot {
					if ecs.World.Valid(pasComp.CurrentTarget) {
						if targetEntry := ecs.World.Entry(pasComp.CurrentTarget); targetEntry.Valid() {
							partStr += fmt.Sprintf(" -> %s", IdentityComponentType.Get(targetEntry).Name)
						} else {
							partStr += " (ターゲット消失)"
						}
					} else {
						partStr += " (ターゲットなし)"
					}
				}
				DrawButton(screen, buttonRect_modal, partStr, MplusFont, appConfig.UI.Colors.Background, appConfig.UI.Colors.White, appConfig.UI.Colors.White)
			}
		} else {
			gs.CurrentState = StatePlaying
			pasComp.ActingMedarot = donburi.Entity(0)
			PlayerActionSelectComponentType.Set(gameStateEntry, pasComp)
			GameStateComponentType.Set(gameStateEntry, gs)
		}
	} else if gs.CurrentState == GameStateMessage || gs.CurrentState == GameStateOver {
		windowWidth := int(float32(appConfig.UI.Screen.Width) * 0.7)
		windowHeight := int(float32(appConfig.UI.Screen.Height) * 0.25)
		windowX := (appConfig.UI.Screen.Width - windowWidth) / 2
		windowY := int(appConfig.UI.Battlefield.Height) - windowHeight/2
		windowRect := image.Rect(windowX, windowY, windowX+windowWidth, windowY+windowHeight)
		prompt := ""
		if gs.CurrentState == GameStateMessage {
			prompt = "クリックして続行..."
		}
		DrawMessagePanel(screen, windowRect, gs.Message, prompt, MplusFont, &appConfig.UI)
		if gs.CurrentState == GameStateOver && MplusFont != nil {
			resetMsg := "クリックでリスタート"
			bounds := text.BoundString(MplusFont, resetMsg)
			msgX := windowX + (windowWidth-(bounds.Max.X-bounds.Min.X))/2
			msgY := windowY + windowHeight - (bounds.Max.Y - bounds.Min.Y) - 10
			text.Draw(screen, resetMsg, MplusFont, msgX, msgY, appConfig.UI.Colors.White)
		}
	}

	if gs.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick:%d St:%s Ent:%d PAS-A:%d PAS-T:%d",
			gs.TickCount, gs.CurrentState, ecs.World.Len(), pasComp.ActingMedarot.Id(), pasComp.CurrentTarget.Id()),
			10, appConfig.UI.Screen.Height-15)
	}
}

func drawMedarotInfoECS(screen *ebiten.Image, identity *IdentityComponent, status *StatusComponent, parts *PartsComponent, startX, startY float32, config *Config, debugMode bool) {
	if MplusFont == nil {
		return
	}
	nameColor := config.UI.Colors.White
	if status.State_is_broken_internal() {
		nameColor = config.UI.Colors.Broken
	}
	text.Draw(screen, identity.Name, MplusFont, int(startX), int(startY)+int(config.UI.InfoPanel.TextLineHeight), nameColor)
	if debugMode {
		stateStr := fmt.Sprintf("St:%s(G:%.0f)", status.State, status.Gauge)
		text.Draw(screen, stateStr, MplusFont, int(startX+70), int(startY)+int(config.UI.InfoPanel.TextLineHeight), config.UI.Colors.Yellow)
	}
	partSlots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	partSlotDisplayNames := map[PartSlotKey]string{PartSlotHead: "頭", PartSlotRightArm: "右", PartSlotLeftArm: "左", PartSlotLegs: "脚"}
	currentInfoY := startY + config.UI.InfoPanel.TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+config.UI.InfoPanel.TextLineHeight > startY+config.UI.InfoPanel.BlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		part, exists := parts.Parts[slotKey]
		if exists && part != nil {
			currentArmor := part.Armor
			if part.IsBroken {
				currentArmor = 0
			}
			hpText = fmt.Sprintf("%s:%d/%d", displayName, currentArmor, part.MaxArmor)
			if part.MaxArmor > 0 {
				hpPercentage := 0.0
				if part.MaxArmor > 0 {
					hpPercentage = float64(currentArmor) / float64(part.MaxArmor)
				}
				gaugeX := startX + config.UI.InfoPanel.PartHPGaugeOffsetX
				gaugeY := currentInfoY - config.UI.InfoPanel.TextLineHeight/2 - config.UI.InfoPanel.PartHPGaugeHeight/2
				vector.DrawFilledRect(screen, gaugeX, gaugeY, config.UI.InfoPanel.PartHPGaugeWidth, config.UI.InfoPanel.PartHPGaugeHeight, color.NRGBA{50, 50, 50, 255}, true)
				barFillColor := config.UI.Colors.HP
				if part.IsBroken {
					barFillColor = config.UI.Colors.Broken
				} else if hpPercentage < 0.3 {
					barFillColor = config.UI.Colors.Red
				}
				vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(config.UI.InfoPanel.PartHPGaugeWidth)*hpPercentage), config.UI.InfoPanel.PartHPGaugeHeight, barFillColor, true)
			}
		} else {
			hpText = fmt.Sprintf("%s:N/A", displayName)
		}
		textColor := config.UI.Colors.White
		if exists && part != nil && part.IsBroken {
			textColor = config.UI.Colors.Broken
		}
		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY), textColor)
		if exists && part != nil && part.MaxArmor > 0 {
			partNameX := startX + config.UI.InfoPanel.PartHPGaugeOffsetX + config.UI.InfoPanel.PartHPGaugeWidth + 5
			text.Draw(screen, part.PartName, MplusFont, int(partNameX), int(currentInfoY), textColor)
		}
		currentInfoY += config.UI.InfoPanel.TextLineHeight + 4
	}
}

// calculateDamageのBerserk部分を修正: attackerEntryを引数に追加する
func calculateDamage_refactored(attackerEntry *donburi.Entry, attackerMedal *MedalComponent, attackingPart *Part,
	targetPart *Part, targetLegs *Part, // targetLegs can be nil
	isCritical bool, cfg BalanceConfig, isTargetDefenseDisabled bool) int {

	basePower := float64(attackingPart.Power)
	if attackingPart.Category == CategoryFight {
		basePower += float64(attackerMedal.Medal.SkillFight * cfg.Damage.MedalSkillFactor)
	} else if attackingPart.Category == CategoryShoot {
		basePower += float64(attackerMedal.Medal.SkillShoot * cfg.Damage.MedalSkillFactor)
	}

	if attackingPart.Trait == TraitBerserk {
		if attackerEntry != nil && attackerEntry.Valid() {
			if attackerEntry.HasComponent(PartsComponentType) {
				attackerPartsComp := PartsComponentType.Get(attackerEntry)
				if attackerActualLegs, legsOk := attackerPartsComp.Parts[PartSlotLegs]; legsOk && !attackerActualLegs.IsBroken {
					basePower += float64(attackerActualLegs.Propulsion)
				}
			}
		}
	}

	defenseValue := float64(targetPart.Defense)
	if targetLegs != nil && !targetLegs.IsBroken {
		defenseValue += float64(targetLegs.Defense)
	}
	if isTargetDefenseDisabled {
		defenseValue = 0
	}

	rawDamage := basePower - defenseValue
	if rawDamage < 1.0 {
		rawDamage = 1.0
	}
	if isCritical {
		rawDamage *= cfg.Damage.CriticalMultiplier
	}
	return int(rawDamage)
}

// ActionExecutionSystemのUpdate内でcalculateDamageを呼び出す箇所をcalculateDamage_refactoredに置き換える必要があります。
// その際、attackerEntry (現在の'entry')を渡します。
// (上記はsystems.goのActionExecutionSystem.Update内のcalculateDamage呼び出しを修正するためのメモ)
