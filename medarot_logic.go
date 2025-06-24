package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
)

// NewMedarot はMedarotインスタンスを生成します。
func NewMedarot(id, name string, team TeamID, medal *Medal, isLeader bool) *Medarot {
	return &Medarot{
		ID:       id,
		Name:     name,
		Team:     team,
		Medal:    medal,
		Parts:    make(map[PartSlotKey]*Part),
		IsLeader: isLeader,
		State:    StateReadyToSelectAction,
		Gauge:    100.0,
	}
}

// ChangeState は状態遷移とそれに伴う初期化処理を行います。
func (m *Medarot) ChangeState(newState MedarotState) {
	m.State = newState
	switch newState {
	case StateReadyToSelectAction:
		m.Gauge = 100.0
		m.SelectedPartKey = ""
		m.IsEvasionDisabled = false
		m.IsDefenseDisabled = false
	case StateActionCharging, StateActionCooldown:
		m.Gauge = 0
	case StateBroken:
		m.Gauge = 0
	}
}

// GetPart はパーツを取得します。
func (m *Medarot) GetPart(slotKey PartSlotKey) *Part {
	part, exists := m.Parts[slotKey]
	if !exists || part == nil || part.IsBroken {
		return nil
	}
	return part
}

// GetAvailableAttackParts は利用可能な攻撃パーツを返します。
func (m *Medarot) GetAvailableAttackParts() []*Part {
	available := []*Part{}
	attackSlots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm}
	for _, slotKey := range attackSlots {
		part := m.GetPart(slotKey)
		if part != nil && part.Charge > 0 {
			available = append(available, part)
		}
	}
	return available
}

// Update はメダロットのゲージを更新します。
func (m *Medarot) Update(cfg BalanceConfig) {
	if m.State == StateBroken {
		return
	}

	headPart, exists := m.Parts[PartSlotHead]
	if exists && headPart.IsBroken {
		m.ChangeState(StateBroken)
		return
	}

	var moveSpeed float64
	legs := m.GetPart(PartSlotLegs)
	legPropulsion := 0
	if legs != nil {
		legPropulsion = legs.Propulsion
	}
	var part *Part
	if m.SelectedPartKey != "" {
		part = m.GetPart(m.SelectedPartKey)
	}
	if part == nil {
		if m.State == StateActionCharging || m.State == StateActionCooldown {
			m.ChangeState(StateReadyToSelectAction)
		}
		return
	}
	stat := 0
	if m.State == StateActionCharging {
		stat = part.Charge
	} else if m.State == StateActionCooldown {
		stat = part.Cooldown
	}
	moveSpeed = (float64(stat) + float64(legPropulsion)*cfg.Time.PropulsionEffectRate) / cfg.Time.OverallTimeDivisor
	m.Gauge += moveSpeed
	if m.Gauge >= 100.0 {
		m.Gauge = 100.0
		if m.State == StateActionCharging {
			m.ChangeState(StateReadyToExecuteAction)
		} else if m.State == StateActionCooldown {
			m.ChangeState(StateReadyToSelectAction)
		}
	}
}

// SelectAction は行動を選択します。
func (m *Medarot) SelectAction(partSlotKey PartSlotKey) bool {
	if m.State != StateReadyToSelectAction {
		return false
	}
	partToUse := m.GetPart(partSlotKey)
	if partToUse == nil || partToUse.Charge <= 0 {
		return false
	}
	m.SelectedPartKey = partSlotKey
	m.ChangeState(StateActionCharging)
	m.Gauge = 0
	m.applyActionConstraints(partToUse.Trait, true)
	return true
}

// ExecuteAction は行動を実行します。引数にConfigと攻撃対象リストを取ります。
func (m *Medarot) ExecuteAction(cfg BalanceConfig, opponents []*Medarot) {
	defer func() {
		m.ChangeState(StateActionCooldown)
		m.Gauge = 0
		if m.SelectedPartKey != "" {
			if selectedPart := m.GetPart(m.SelectedPartKey); selectedPart != nil {
				m.applyActionConstraints(selectedPart.Trait, false)
			}
		}
	}()
	var selectedPart *Part
	if m.SelectedPartKey != "" {
		selectedPart = m.GetPart(m.SelectedPartKey)
	}
	if selectedPart == nil {
		m.LastActionLog = "パーツが破壊されていて失敗！"
		return
	}
	target := m.determineTarget(opponents)
	if target == nil {
		m.LastActionLog = "ターゲットが見つからず失敗！"
		return
	}
	isHit, isCritical := m.calculateHit(target, selectedPart, cfg)
	if !isHit {
		m.LastActionLog = fmt.Sprintf("%sへの攻撃は回避された！", target.Name)
		return
	}
	targetPart := target.selectRandomPartToDamage()
	if targetPart == nil {
		m.LastActionLog = "攻撃対象部位がなかった！"
		return
	}
	damage := m.calculateDamage(targetPart, selectedPart, isCritical, cfg)
	m.applyDamage(targetPart, damage)
	m.LastActionLog = m.generateActionLog(target, targetPart, damage, isCritical)
	target.handlePostAttack()
}

// determineTarget は攻撃対象を決定します。
func (m *Medarot) determineTarget(opponents []*Medarot) *Medarot {
	// この時点で m.TargetedMedarot は有効なはずなので、それをそのまま返す
	return m.TargetedMedarot
}

// calculateHit は命中判定を計算します。
func (m *Medarot) calculateHit(target *Medarot, part *Part, cfg BalanceConfig) (bool, bool) {
	skillValue := 0
	if part.Category == CategoryShoot {
		skillValue = m.Medal.SkillShoot
	} else {
		skillValue = m.Medal.SkillFight
	}
	traitBonus := 0
	switch part.Trait {
	case TraitAim:
		traitBonus = cfg.Hit.TraitAimBonus
	case TraitStrike:
		traitBonus = cfg.Hit.TraitStrikeBonus
	case TraitBerserk:
		traitBonus = cfg.Hit.TraitBerserkDebuff
	}
	finalAccuracy := part.Accuracy + skillValue + traitBonus
	targetMobility := 0
	if targetLegs := target.GetPart(PartSlotLegs); targetLegs != nil {
		targetMobility = targetLegs.Mobility
	}
	if target.IsEvasionDisabled {
		targetMobility = 0
	}
	hitChance := cfg.Hit.BaseChance + finalAccuracy - targetMobility
	isHit := hitChance > 0 && rand.Intn(100) < hitChance
	isCritical := isHit && hitChance > 100 && rand.Intn(100) < (hitChance-100)
	return isHit, isCritical
}

// calculateDamage はダメージを計算します。
func (m *Medarot) calculateDamage(targetPart, attackingPart *Part, isCritical bool, cfg BalanceConfig) int {
	basePower := attackingPart.Power + (m.Medal.SkillFight * cfg.Damage.MedalSkillFactor)
	if attackingPart.Trait == TraitBerserk {
		if legs := m.GetPart(PartSlotLegs); legs != nil {
			basePower += legs.Propulsion
		}
	}
	defenseValue := targetPart.Defense
	if targetLegs := targetPart.Owner.GetPart(PartSlotLegs); targetLegs != nil {
		defenseValue += targetLegs.Defense
	}
	if targetPart.Owner.IsDefenseDisabled {
		defenseValue = 0
	}
	rawDamage := float64(basePower - defenseValue)
	if isCritical {
		rawDamage *= cfg.Damage.CriticalMultiplier
	}
	return int(math.Max(1, math.Floor(rawDamage)))
}

// applyDamage はダメージを適用し、装甲値を減らします。
func (m *Medarot) applyDamage(part *Part, damage int) {
	part.Armor -= damage
	if part.Armor <= 0 {
		part.Armor = 0
		part.IsBroken = true
	}
}

// generateActionLog は攻撃結果のログを生成します。
func (m *Medarot) generateActionLog(target *Medarot, part *Part, damage int, isCritical bool) string {
	logMsg := fmt.Sprintf("%sの%sに%dダメージ！", target.Name, part.PartName, damage)
	if isCritical {
		logMsg = fmt.Sprintf("%sの%sにクリティカル！ %dダメージ！", target.Name, part.PartName, damage)
	}
	if part.IsBroken {
		logMsg += " パーツを破壊した！"
		log.Printf("!!! %sのパーツ「%s」が破壊されました !!!", target.Name, part.PartName)
	}
	return logMsg
}

// handlePostAttack は攻撃後の状態変化を処理します。
func (m *Medarot) handlePostAttack() {
	if head := m.GetPart(PartSlotHead); head != nil && head.IsBroken {
		m.ChangeState(StateBroken)
	}
}

// selectRandomPartToDamage は攻撃対象部位をランダムに選択します。
func (m *Medarot) selectRandomPartToDamage() *Part {
	vulnerable := []*Part{}
	slots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	for _, s := range slots {
		if p := m.GetPart(s); p != nil {
			vulnerable = append(vulnerable, p)
		}
	}
	if len(vulnerable) == 0 {
		return nil
	}
	return vulnerable[rand.Intn(len(vulnerable))]
}

// applyActionConstraints は行動制限を適用します。
func (m *Medarot) applyActionConstraints(trait ActionTrait, enable bool) {
	switch trait {
	case TraitAim:
		m.IsEvasionDisabled = enable
	case TraitStrike:
		m.IsDefenseDisabled = enable
	case TraitBerserk:
		m.IsEvasionDisabled = enable
		m.IsDefenseDisabled = enable
	}
}
