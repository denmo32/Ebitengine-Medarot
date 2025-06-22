package main

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
	ID          string
	Name        string
	Personality string
	Medaforce   string
	Attribute   string
	SkillShoot  int
	SkillFight  int
	SkillScan   int
	SkillSupport int
}

// Part represents a Medarot part.
type Part struct {
	ID            string
	Name          string
	Category      string
	SubCategory   string
	Slot          string
	HP            int
	MaxHP         int
	Charge        int
	Cooldown      int
	IsBroken      bool
	MovementType  string
	Accuracy      int
	Mobility      int
	Propulsion    int
	DefenseParam  int
	SetID         string
	ActionType    string // "shoot", "fight", "other"
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
		State:                 StateIdleCharging,
		Gauge:                 0,
		MaxGauge:              100.0,
		CurrentActionCharge:   0,
		CurrentActionCooldown: 0,
		SelectedPartKey:       "",
		TargetedMedarot:       nil,
	}
}

const (
	GaugeChargeRateMultiplier = 1.0
)

// Update processes a single turn for the Medarot.
func (m *Medarot) Update() {
	if m.State == StateBroken {
		return
	}

	if head, ok := m.Parts["head"]; ok && head.IsBroken {
		m.State = StateBroken
		m.Gauge = 0
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
				m.State = StateReadyToSelectAction // Or StateActionCooldown if preferred
				m.Gauge = 0
				m.SelectedPartKey = ""
				m.CurrentActionCharge = 0
				// m.CurrentActionCooldown = 0; // Decide if cooldown applies on cancellation
				m.TargetedMedarot = nil
				// log.Printf("%s's shoot target %s is broken. Action cancelled.", m.Name, m.TargetedMedarot.Name) // Requires log import
				return // Exit update for this tick
			}
			if selectedPartForCharge.IsBroken {
				// Own action part is broken
				m.State = StateReadyToSelectAction // Or StateActionCooldown
				m.Gauge = 0
				m.SelectedPartKey = ""
				m.CurrentActionCharge = 0
				// m.CurrentActionCooldown = 0;
				m.TargetedMedarot = nil // Clear target as action is cancelled
				// log.Printf("%s's action part %s is broken. Action cancelled.", m.Name, selectedPartForCharge.Name)
				return // Exit update for this tick
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
			// --- ★★★ ここが修正点 ★★★ ---
			// Cooldown finished, go directly to selecting the next action.
			m.State = StateReadyToSelectAction
			m.Gauge = 0
			m.SelectedPartKey = ""
			m.CurrentActionCharge = 0
			m.CurrentActionCooldown = 0
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
		return false
	}

	m.SelectedPartKey = partSlotKey
	m.CurrentActionCharge = float64(partToUse.Charge)
	m.CurrentActionCooldown = float64(partToUse.Cooldown)
	m.State = StateActionCharging
	m.Gauge = 0
	return true
}

// ExecuteAction transitions the Medarot to the cooldown phase.
// It now accepts the game instance to allow for context-aware actions like finding the closest opponent.
func (m *Medarot) ExecuteAction(g *Game) bool {
	if m.State != StateReadyToExecuteAction {
		return false
	}
	
	selectedPart := m.GetPart(m.SelectedPartKey)
	if selectedPart == nil {
		m.State = StateReadyToSelectAction // Go back to select if part is now broken.
		m.Gauge = 0
		m.SelectedPartKey = ""
		m.CurrentActionCharge = 0
		m.CurrentActionCooldown = 0
		return false
	}
	
	m.State = StateActionCooldown
	m.Gauge = 0
	return true
}