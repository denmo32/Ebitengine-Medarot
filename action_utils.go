package main

import (
	"math/rand"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
)

// showGameMessage はメッセージ表示状態に移行し、コールバックを設定します。
// action_execution_system など、複数のシステムから利用されるユーティリティです。
func showGameMessage(ecs *ecs.ECS, msg string, callback func()) {
	entry, ok := GameStateComponentType.First(ecs.World)
	if !ok {
		// ゲーム状態が取得できない場合でも、コールバックは実行を試みる
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

// calculateHit は命中判定とクリティカル判定を行います。
func calculateHit(attackerMedal *MedalComponent, attackerPart *Part, targetStatus *StatusComponent, targetLegs *Part, cfg BalanceConfig) (bool, bool) {
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
	// 回避不可状態なら機動力を0にする
	if targetStatus.IsEvasionDisabled {
		targetMobility = 0
	}

	hitChance := cfg.Hit.BaseChance + finalAccuracy - targetMobility
	if hitChance < 0 {
		hitChance = 0
	}

	isHit := rand.Intn(100) < hitChance
	isCritical := false
	// 命中率が100を超えた分がクリティカル率になる
	if isHit && hitChance > 100 {
		if rand.Intn(100) < (hitChance - 100) {
			isCritical = true
		}
	}

	return isHit, isCritical
}

// selectRandomPartToDamage は攻撃対象のパーツをランダムに1つ選択します。
func selectRandomPartToDamage(targetParts *PartsComponent) *Part {
	vulnerable := []*Part{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if p, ok := targetParts.Parts[s]; ok && !p.IsBroken {
			vulnerable = append(vulnerable, p)
		}
	}
	if len(vulnerable) == 0 {
		return nil
	}
	return vulnerable[rand.Intn(len(vulnerable))]
}

// calculateDamage は最終的なダメージ量を計算します。
func calculateDamage(attackerEntry *donburi.Entry, attackerMedal *MedalComponent, attackingPart *Part,
	targetPart *Part, targetLegs *Part,
	isCritical bool, cfg BalanceConfig, isTargetDefenseDisabled bool) int {

	// 威力計算
	basePower := float64(attackingPart.Power)
	if attackingPart.Category == CategoryFight {
		basePower += float64(attackerMedal.Medal.SkillFight * cfg.Damage.MedalSkillFactor)
	} else if attackingPart.Category == CategoryShoot {
		basePower += float64(attackerMedal.Medal.SkillShoot * cfg.Damage.MedalSkillFactor)
	}
	if attackingPart.Trait == TraitBerserk {
		// if attackerPartsComp, ok := donburi.GetComponent[*PartsComponent](attackerEntry); ok {
		if attackerEntry.HasComponent(PartsComponentType) { // ←存在チェック
			attackerPartsComp := PartsComponentType.Get(attackerEntry) // ←取得
			if attackerLegs, legsOk := attackerPartsComp.Parts[PartSlotLegs]; legsOk && !attackerLegs.IsBroken {
				basePower += float64(attackerLegs.Propulsion)
			}
		}
	}

	// 防御計算
	defenseValue := float64(targetPart.Defense)
	if targetLegs != nil && !targetLegs.IsBroken {
		defenseValue += float64(targetLegs.Defense)
	}
	// 防御不可状態なら防御力を0にする
	if isTargetDefenseDisabled {
		defenseValue = 0
	}

	rawDamage := basePower - defenseValue
	if rawDamage < 1.0 {
		rawDamage = 1.0
	}

	// クリティカル補正
	if isCritical {
		rawDamage *= cfg.Damage.CriticalMultiplier
	}

	return int(rawDamage)
}
