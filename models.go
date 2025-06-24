package main

// PartSlotKey はパーツのスロットを一意に識別するための型です。
type PartSlotKey string

// PartSlot... はパーツのスロットキーを表す定数です。
const (
	PartSlotHead     PartSlotKey = "head"
	PartSlotRightArm PartSlotKey = "rightArm"
	PartSlotLeftArm  PartSlotKey = "leftArm"
	PartSlotLegs     PartSlotKey = "legs"
)

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
}

// Medarot はメダロットのデータ構造です。
// ECS化に伴い、この構造体はエンティティとコンポーネントに分割されるため、基本的には不要になります。
// PartsComponent内のPart構造体がOwnerとしてこの型を参照しているため、完全削除は後回し。
// type Medarot struct {
// 	ID                string
// 	Name              string
// 	Team              TeamID
// 	Medal             *Medal
// 	Parts             map[PartSlotKey]*Part
// 	IsLeader          bool
// 	State             MedarotState
// 	Gauge             float64
// 	SelectedPartKey   PartSlotKey
// 	TargetedMedarot   *Medarot
// 	LastActionLog     string
// 	IsEvasionDisabled bool
// 	IsDefenseDisabled bool
// 	DrawIndex         int
// }

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

// 古いGame構造体は削除。ECSベースのGame構造体はgame.goにあります。
// type Game struct {
// 	Medarots              []*Medarot
// 	GameData              *GameData
// 	Config                Config
// 	TickCount             int
// 	DebugMode             bool
// 	State                 GameState
// 	PlayerTeam            TeamID
// 	actionQueue           []*Medarot
// 	message               string
// 	postMessageCallback   func()
// 	winner                TeamID
// 	playerActionTarget    *Medarot
// 	restartRequested      bool
// 	sortedMedarotsForDraw []*Medarot
// 	team1Leader           *Medarot
// 	team2Leader           *Medarot
// }
