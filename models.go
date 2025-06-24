package main

// PartType はパーツの部位を定義します。
type PartType string

const (
	PartTypeHead PartType = "HEAD"
	PartTypeRArm PartType = "R_ARM"
	PartTypeLArm PartType = "L_ARM"
	PartTypeLegs PartType = "LEG"
)

// ActionCategory は行動の大区分を定義します。
type ActionCategory string

const (
	CategoryShoot ActionCategory = "SHOOT"
	CategoryFight ActionCategory = "FIGHT"
	CategoryNone  ActionCategory = "NONE"
)

// ActionTrait は行動の中区分（特性）を定義します。
type ActionTrait string

const (
	TraitNormal  ActionTrait = "NORMAL"
	TraitAim     ActionTrait = "AIM"
	TraitStrike  ActionTrait = "STRIKE"
	TraitBerserk ActionTrait = "BERSERK"
	TraitNone    ActionTrait = "NONE"
)

// MedarotState はメダロットの状態を定義します。
type MedarotState string

const (
	StateReadyToSelectAction  MedarotState = "ReadyToSelectAction"
	StateActionCharging       MedarotState = "ActionCharging"
	StateReadyToExecuteAction MedarotState = "ReadyToExecuteAction"
	StateActionCooldown       MedarotState = "ActionCooldown"
	StateBroken               MedarotState = "Broken"
)

// Medal はメダルのデータ構造です。
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

// Part はパーツのデータ構造です。
type Part struct {
	ID         string
	PartName   string
	Type       PartType
	Category   ActionCategory
	Trait      ActionTrait
	WeaponType string
	Armor      int
	MaxArmor   int
	Power      int
	Charge     int
	Cooldown   int
	Defense    int
	Accuracy   int
	Mobility   int
	Propulsion int
	IsBroken   bool
	SetID      string
	Owner      *Medarot
}

// Medarot はメダロットのデータ構造です。
type Medarot struct {
	ID                string
	Name              string
	Team              TeamID
	Medal             *Medal
	Parts             map[string]*Part
	IsLeader          bool
	State             MedarotState
	Gauge             float64
	SelectedPartKey   string
	TargetedMedarot   *Medarot
	LastActionLog     string
	IsEvasionDisabled bool
	IsDefenseDisabled bool
	DrawIndex         int
}

// GameState はゲームの進行状態を定義します。
type GameState int

const (
	StatePlaying GameState = iota
	StatePlayerActionSelect
	GameStateMessage
	GameStateOver
)

// TeamID はチームを識別します。
type TeamID int

const (
	Team1 TeamID = iota
	Team2
)

// Game はゲーム全体の状態を保持する構造体です。
type Game struct {
	Medarots              []*Medarot
	GameData              *GameData
	Config                Config
	TickCount             int
	DebugMode             bool
	State                 GameState
	PlayerTeam            TeamID
	actionQueue           []*Medarot
	message               string
	postMessageCallback   func()
	winner                TeamID
	playerActionTarget    *Medarot
	restartRequested      bool
	sortedMedarotsForDraw []*Medarot
	// actionablePartsForModal []*Part // ★★★ [修正点] この行を削除 ★★★
	team1Leader *Medarot
	team2Leader *Medarot
}