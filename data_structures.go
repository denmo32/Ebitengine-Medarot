package main

import (
	"fmt"
	"log"
	"math/rand"
)

// MedarotState defines the possible states of a Medarot.
type MedarotState string

const (
	StateIdleCharging         MedarotState = "IdleCharging"        // 初期チャージ中
	StateReadyToSelectAction  MedarotState = "ReadyToSelectAction" // 行動選択可能
	StateActionCharging       MedarotState = "ActionCharging"      // 行動チャージ中
	StateReadyToExecuteAction MedarotState = "ReadyToExecuteAction" // 行動実行可能
	StateActionCooldown       MedarotState = "ActionCooldown"      // クールダウン中
	StateBroken               MedarotState = "Broken"              // 破壊状態
)

// Medal represents a Medarot's medal.
type Medal struct {
	ID           string
	Name         string
	Personality  string
	Medaforce    string
	Attribute    string
	SkillShoot   int
	SkillFight   int
	SkillScan    int
	SkillSupport int
}

// Part represents a Medarot part.
type Part struct {
	ID           string
	Name         string
	Category     string
	SubCategory  string
	Slot         string
	HP           int
	MaxHP        int
	Power        int // ★提案に基づき威力（Power）を追加
	Charge       int
	Cooldown     int
	IsBroken     bool
	MovementType string
	Accuracy     int
	Mobility     int
	Propulsion   int
	DefenseParam int
	SetID        string
	ActionType   string // "shoot", "fight", "other"
}

// Medarot represents a single Medarot unit.
type Medarot struct {
	ID                    string
	Name                  string
	Team                  TeamID
	Speed                 float64
	Medal                 *Medal
	Parts                 map[string]*Part
	IsLeader              bool
	Color                 string
	State                 MedarotState
	Gauge                 float64
	MaxGauge              float64
	CurrentActionCharge   float64
	CurrentActionCooldown float64
	SelectedPartKey       string
	TargetedMedarot       *Medarot // Currently targeted Medarot
	LastActionLog         string   // To store the result of the last action
}

// GetPart returns a specific part, nil if not found or broken.
func (m *Medarot) GetPart(slotKey string) *Part {
	part, exists := m.Parts[slotKey]
	if !exists || part == nil || part.IsBroken {
		return nil
	}
	return part
}

// GetAvailableAttackParts returns available (non-broken) attack parts.
func (m *Medarot) GetAvailableAttackParts() []*Part {
	available := []*Part{}
	attackSlots := []string{"head", "rightArm", "leftArm"}
	for _, slotKey := range attackSlots {
		if part := m.GetPart(slotKey); part != nil {
			available = append(available, part)
		}
	}
	return available
}

// NewMedarot creates and initializes a new Medarot.
func NewMedarot(id, name string, team TeamID, speed float64, medal *Medal, isLeader bool) *Medarot {
	return &Medarot{
		ID:                    id,
		Name:                  name,
		Team:                  team,
		Speed:                 speed,
		Medal:                 medal,
		Parts:                 make(map[string]*Part),
		IsLeader:              isLeader,
		// ★★★ ここから変更 ★★★
		State:                 StateReadyToSelectAction, // 最初から行動選択可能状態にする
		Gauge:                 100.0,                    // ゲージをMAXで開始
		MaxGauge:              100.0,
		// ★★★ ここまで変更 ★★★
		CurrentActionCharge:   0,
		CurrentActionCooldown: 0,
		SelectedPartKey:       "",
		TargetedMedarot:       nil,
		LastActionLog:         "",
	}
}

const (
	GaugeChargeRateMultiplier = 1.0
)

// Update processes a single turn for the Medarot.
func (m *Medarot) Update(g *Game) {
	if m.State == StateBroken {
		return
	}

	// 頭部パーツが破壊されたら機体停止
	if head, ok := m.Parts["head"]; ok && head.IsBroken {
		isAlreadyBroken := m.State == StateBroken
		m.State = StateBroken
		m.Gauge = 0
		if !isAlreadyBroken {
			log.Printf("%sの機能が停止しました！", m.Name)
			g.checkGameEnd() // ★ ゲーム終了チェックを追加
		}
		return
	}

	chargeRate := m.Speed * GaugeChargeRateMultiplier
	if legs, ok := m.Parts["legs"]; ok && !legs.IsBroken {
		chargeRate += float64(legs.Propulsion) * 0.05
	}
	if chargeRate <= 0 {
		chargeRate = 0.1
	}

	switch m.State {
	case StateIdleCharging:
		m.Gauge += chargeRate
		if m.Gauge >= m.MaxGauge {
			m.Gauge = m.MaxGauge
			m.State = StateReadyToSelectAction
		}
	case StateActionCharging:
		if m.CurrentActionCharge <= 0 {
			m.State = StateReadyToSelectAction
			m.Gauge = 0
			return
		}
		m.Gauge += chargeRate

		// Check for action cancellation conditions for shoot attacks during charge
		selectedPartForCharge := m.GetPart(m.SelectedPartKey) // Re-fetch part to check its current status
		if selectedPartForCharge != nil && selectedPartForCharge.ActionType == "shoot" {
			if m.TargetedMedarot != nil && m.TargetedMedarot.State == StateBroken {
				// Target is broken
				m.State = StateReadyToSelectAction
				m.Gauge = 0
				m.SelectedPartKey = ""
				m.CurrentActionCharge = 0
				m.TargetedMedarot = nil
				return
			}
			if selectedPartForCharge.IsBroken {
				// Own action part is broken
				m.State = StateReadyToSelectAction
				m.Gauge = 0
				m.SelectedPartKey = ""
				m.CurrentActionCharge = 0
				m.TargetedMedarot = nil
				return
			}
		}

		if m.Gauge >= m.CurrentActionCharge {
			m.Gauge = m.CurrentActionCharge
			m.State = StateReadyToExecuteAction
		}
	case StateActionCooldown:
		if m.CurrentActionCooldown <= 0 {
			m.State = StateReadyToSelectAction
			m.Gauge = 0
			return
		}
		m.Gauge += chargeRate
		if m.Gauge >= m.CurrentActionCooldown {
			m.State = StateReadyToSelectAction
			m.Gauge = 0
			m.SelectedPartKey = ""
			m.CurrentActionCharge = 0
			m.CurrentActionCooldown = 0
			m.LastActionLog = ""
		}
	case StateReadyToSelectAction, StateReadyToExecuteAction, StateBroken:
		// No gauge change in these states.
		return
	}
}

// SelectAction sets the Medarot to charge a specific part's action.
func (m *Medarot) SelectAction(partSlotKey string) bool {
	if m.State != StateReadyToSelectAction {
		return false
	}

	partToUse, exists := m.Parts[partSlotKey]
	if !exists || partToUse == nil || partToUse.IsBroken {
		return false
	}

	if partToUse.Charge <= 0 {
		// 充填が0以下のパーツは攻撃アクションとはみなさない（仮）
		return false
	}

	m.SelectedPartKey = partSlotKey
	m.CurrentActionCharge = float64(partToUse.Charge)
	m.CurrentActionCooldown = float64(partToUse.Cooldown)
	m.State = StateActionCharging
	m.Gauge = 0
	return true
}

// ★★★ ここからが提案に基づき大幅に修正された ExecuteAction 関数 ★★★
// ExecuteAction performs the selected action, including targeting, hit check, and damage calculation.
func (m *Medarot) ExecuteAction(g *Game) bool {
	if m.State != StateReadyToExecuteAction {
		return false
	}

	selectedPart := m.GetPart(m.SelectedPartKey)
	if selectedPart == nil {
		// The part was broken during charging.
		m.State = StateActionCooldown // Go to cooldown even on failure
		m.Gauge = 0
		m.LastActionLog = "パーツが破壊されていて失敗！"
		return false
	}

	// --- Target Selection ---
	var target *Medarot
	if selectedPart.ActionType == "fight" {
		// For fighting actions, find the closest opponent now.
		target = g.findClosestOpponent(m)
		m.TargetedMedarot = target // Store for logging purposes.
	} else {
		// For shooting actions, the target was already selected.
		target = m.TargetedMedarot
	}

	// --- Validate Target ---
	if target == nil || target.State == StateBroken {
		m.State = StateActionCooldown
		m.Gauge = 0
		m.LastActionLog = "ターゲットが見つからず失敗！"
		return false // Action "fails" as there's no valid target.
	}

	// --- Hit Calculation ---
	// A simple hit check formula. This can be expanded.
	// (Part Accuracy + Medal Skill) vs (Target's Leg Mobility)
	skillValue := 0
	if selectedPart.ActionType == "shoot" {
		skillValue = m.Medal.SkillShoot
	} else if selectedPart.ActionType == "fight" {
		skillValue = m.Medal.SkillFight
	}
	targetLegs := target.GetPart("legs")
	targetMobility := 0
	if targetLegs != nil {
		targetMobility = targetLegs.Mobility
	}

	// Base hit chance: 75%. Adjust with stats.
	hitChance := 75 + (selectedPart.Accuracy + skillValue*2) - (targetMobility)
	if hitChance < 10 {
		hitChance = 10
	}
	if hitChance > 100 {
		hitChance = 100
	}

	isHit := rand.Intn(100) < hitChance
	if !isHit {
		m.State = StateActionCooldown
		m.Gauge = 0
		m.LastActionLog = fmt.Sprintf("%sへの攻撃は回避された！", target.Name)
		return false // Attack missed.
	}

	// --- Damage Calculation & Applying Damage ---
	// (Part Power + Medal Skill) - (Target Part Defense)
	baseDamage := selectedPart.Power + skillValue*3
	if baseDamage < 0 {
		baseDamage = 0
	}

	// Select a random part on the target to damage.
	targetParts := target.getVulnerableParts()
	if len(targetParts) == 0 {
		// This should not happen if the target is not broken.
		m.State = StateActionCooldown
		m.Gauge = 0
		m.LastActionLog = "ターゲットに攻撃可能な部位がなかった！"
		return false
	}
	damagedPart := targetParts[rand.Intn(len(targetParts))]
	finalDamage := baseDamage - damagedPart.DefenseParam
	if finalDamage < 5 {
		finalDamage = 5 // Minimum damage
	}

	damagedPart.HP -= finalDamage
	if damagedPart.HP <= 0 {
		damagedPart.HP = 0
		damagedPart.IsBroken = true
		m.LastActionLog = fmt.Sprintf("%sの%sに%dダメージを与え、破壊した！", target.Name, damagedPart.Name, finalDamage)
		log.Printf("!!! %sのパーツ「%s」が破壊されました !!!", target.Name, damagedPart.Name)
	} else {
		m.LastActionLog = fmt.Sprintf("%sの%sに%dダメージ！", target.Name, damagedPart.Name, finalDamage)
	}

	// --- Finalize Action ---
	m.State = StateActionCooldown
	m.Gauge = 0
	return true // Attack was successful (it hit).
}

// getVulnerableParts returns a list of non-broken parts that can be damaged.
func (m *Medarot) getVulnerableParts() []*Part {
	vulnerable := []*Part{}
	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	for _, slot := range partSlots {
		if part := m.GetPart(slot); part != nil {
			vulnerable = append(vulnerable, part)
		}
	}
	return vulnerable
}

