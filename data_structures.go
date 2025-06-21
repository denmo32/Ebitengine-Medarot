package main

// MedarotState defines the possible states of a Medarot.
type MedarotState string

const (
	StateIdleCharging        MedarotState = "IdleCharging"        // 初期チャージ中
	StateReadyToSelectAction MedarotState = "ReadyToSelectAction" // 行動選択可能
	StateActionCharging      MedarotState = "ActionCharging"      // 行動チャージ中
	StateReadyToExecuteAction MedarotState = "ReadyToExecuteAction" // 行動実行可能
	StateActionCooldown      MedarotState = "ActionCooldown"      // クールダウン中
	StateBroken              MedarotState = "Broken"              // 破壊状態
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
	Category      string // e.g., "射撃", "格闘"
	SubCategory   string // e.g., "ライフル", "ソード"
	Slot          string // "head", "rightArm", "leftArm", "legs"
	HP            int
	MaxHP         int
	Charge        int // チャージに必要な時間 (ティック数やフレーム数で表現)
	Cooldown      int // クールダウンに必要な時間
	IsBroken      bool
	// Legs specific stats (if applicable, can be a separate struct if more complex)
	MovementType string
	Accuracy     int
	Mobility     int
	Propulsion   int
	DefenseParam int
	SetID         string // For identifying parts belonging to a set like "METABEE_SET"
}

// Medarot represents a single Medarot unit.
type Medarot struct {
	ID                  string
	Name                string
	Team                string
	Speed               float64 // Base speed, influences gauge charge rate
	Medal               *Medal
	Parts               map[string]*Part // Keyed by slot: "head", "rightArm", "leftArm", "legs"
	IsLeader            bool
	Color               string // Placeholder for team color or individual color

	State               MedarotState
	Gauge               float64 // Current gauge value
	MaxGauge            float64 // Typically 100 for idle charging
	CurrentActionCharge float64 // Charge time for the selected action
	CurrentActionCooldown float64 // Cooldown time for the selected action
	SelectedPartKey     string  // Key of the part selected for the current action

	// Target related fields (will be used later)
	// CurrentTargetedEnemy *Medarot
	// CurrentTargetedPartKey string
}

// Helper function to get a specific part, returns nil if not found or broken
func (m *Medarot) GetPart(slotKey string) *Part {
	part, exists := m.Parts[slotKey]
	if !exists || part == nil || part.IsBroken {
		return nil
	}
	return part
}

// Helper function to get available (non-broken) attack parts
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
// Parts will be assigned later after CSV loading.
func NewMedarot(id, name, team string, speed float64, medal *Medal, isLeader bool) *Medarot {
	return &Medarot{
		ID:                  id,
		Name:                name,
		Team:                team,
		Speed:               speed,
		Medal:               medal,
		Parts:               make(map[string]*Part),
		IsLeader:            isLeader,
		State:               StateIdleCharging,
		Gauge:               0,
		MaxGauge:            100.0, // Default max gauge for idle charging
		CurrentActionCharge: 0,
		CurrentActionCooldown:0,
		SelectedPartKey:     "",
	}
}

const (
	GaugeChargeRateMultiplier = 1.0 // General multiplier for gauge charging speed. Can be adjusted.
	// Corresponds to CONFIG.MAX_GAUGE in JS for idle charging.
	// Action charge and cooldown will use part-specific values.
)

// Update processes a single turn for the Medarot.
// It updates the gauge based on the current state and handles state transitions.
func (m *Medarot) Update() {
	if m.State == StateBroken {
		return // Broken Medarots do nothing
	}

	// Check if head is broken, if so, Medarot is broken
	if head, ok := m.Parts["head"]; ok && head.IsBroken {
		m.State = StateBroken
		m.Gauge = 0
		return
	}

	chargeRate := m.Speed * GaugeChargeRateMultiplier
	// Propulsion bonus from legs (similar to JS: (this.legPropulsion || 0) * 0.05;)
	if legs, ok := m.Parts["legs"]; ok && !legs.IsBroken {
		chargeRate += float64(legs.Propulsion) * 0.05 // Make sure Propulsion is scaled appropriately
	}
	if chargeRate <= 0 { // Ensure some charge rate if speed/propulsion is too low
		chargeRate = 0.1
	}


	switch m.State {
	case StateIdleCharging:
		m.Gauge += chargeRate
		if m.Gauge >= m.MaxGauge {
			m.Gauge = m.MaxGauge
			m.State = StateReadyToSelectAction
			// fmt.Printf("%s (%s) is now ReadyToSelectAction (Idle Charge Complete)\n", m.Name, m.ID)
		}
	case StateActionCharging:
		if m.CurrentActionCharge <= 0 { // Should not happen if a part is selected
			// fmt.Printf("Warning: %s (%s) is ActionCharging but CurrentActionCharge is 0. Resetting to Idle.\n", m.Name, m.ID)
			m.State = StateIdleCharging
			m.Gauge = 0
			return
		}
		m.Gauge += chargeRate
		if m.Gauge >= m.CurrentActionCharge {
			m.Gauge = m.CurrentActionCharge
			m.State = StateReadyToExecuteAction
			// fmt.Printf("%s (%s) is now ReadyToExecuteAction (Action Charge Complete for %s)\n", m.Name, m.ID, m.SelectedPartKey)
		}
	case StateActionCooldown:
		if m.CurrentActionCooldown <= 0 { // Should not happen if an action was executed
			// fmt.Printf("Warning: %s (%s) is ActionCooldown but CurrentActionCooldown is 0. Resetting to Idle.\n", m.Name, m.ID)
			m.State = StateIdleCharging
			m.Gauge = 0
			return
		}
		m.Gauge += chargeRate
		if m.Gauge >= m.CurrentActionCooldown {
			m.Gauge = m.CurrentActionCooldown // Keep gauge at max for this phase for clarity if needed
			m.State = StateIdleCharging      // Back to idle charging after cooldown
			m.Gauge = 0                      // Reset gauge for idle charging
			m.SelectedPartKey = ""
			m.CurrentActionCharge = 0
			m.CurrentActionCooldown = 0
			// fmt.Printf("%s (%s) is now IdleCharging (Cooldown Complete)\n", m.Name, m.ID)
		}
	case StateReadyToSelectAction, StateReadyToExecuteAction, StateBroken:
		// No gauge change in these states by default, actions are pending.
		return
	}
}

// SelectAction sets the Medarot to charge a specific part's action.
func (m *Medarot) SelectAction(partSlotKey string) bool {
	if m.State != StateReadyToSelectAction {
		// fmt.Printf("Warning: %s (%s) cannot select action, not in ReadyToSelectAction state (current: %s)\n", m.Name, m.ID, m.State)
		return false
	}
	
	partToUse, exists := m.Parts[partSlotKey]
	if !exists || partToUse == nil || partToUse.IsBroken {
		// fmt.Printf("Warning: %s (%s) cannot select action, part %s not available or broken.\n", m.Name, m.ID, partSlotKey)
		return false
	}

	if partToUse.Charge <= 0 { // Actions must have a charge time
		// fmt.Printf("Warning: %s (%s) selected part %s has no charge time. Action cannot be initiated.\n", m.Name, m.ID, partToUse.Name)
		return false
	}

	m.SelectedPartKey = partSlotKey
	m.CurrentActionCharge = float64(partToUse.Charge)
	m.CurrentActionCooldown = float64(partToUse.Cooldown) // Store for after execution
	m.State = StateActionCharging
	m.Gauge = 0 // Reset gauge for action charging
	// fmt.Printf("%s (%s) selected action: %s (Charge: %.0f, Cooldown: %.0f)\n", m.Name, m.ID, partToUse.Name, m.CurrentActionCharge, m.CurrentActionCooldown)
	return true
}

// ExecuteAction transitions the Medarot to the cooldown phase after an action.
// Actual action effects (damage, etc.) are not handled here yet.
func (m *Medarot) ExecuteAction() bool {
	if m.State != StateReadyToExecuteAction {
		// fmt.Printf("Warning: %s (%s) cannot execute action, not in ReadyToExecuteAction state (current: %s)\n", m.Name, m.ID, m.State)
		return false
	}
	
	selectedPart := m.GetPart(m.SelectedPartKey)
	if selectedPart == nil {
		// This case should ideally be prevented by checks before ExecuteAction is called
		// or by ensuring SelectedPartKey is valid.
		// fmt.Printf("Error: %s (%s) tried to execute action but selected part (%s) is broken or missing. Resetting to Idle.\n", m.Name, m.ID, m.SelectedPartKey)
		m.State = StateIdleCharging
		m.Gauge = 0
		m.SelectedPartKey = ""
		m.CurrentActionCharge = 0
		m.CurrentActionCooldown = 0
		return false
	}
	
	// fmt.Printf("%s (%s) executed action: %s!\n", m.Name, m.ID, selectedPart.Name)

	// Transition to cooldown
	m.State = StateActionCooldown
	m.Gauge = 0 // Reset gauge for cooldown
	// CurrentActionCooldown was already set during SelectAction.
	// SelectedPartKey remains for context during cooldown if needed, cleared when cooldown finishes.
	return true
}
