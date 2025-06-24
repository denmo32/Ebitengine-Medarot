package main

// Action related helper functions will be moved here
import (
	"fmt"
	"math/rand"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

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
