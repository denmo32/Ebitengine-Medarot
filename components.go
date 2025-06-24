package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/features/math"
)

// --- Entity Components ---

// IdentityComponent はエンティティの基本的な識別情報を保持します。
type IdentityComponent struct {
	ID       string
	Name     string
	Team     TeamID
	IsLeader bool
}

var IdentityComponentType = donburi.NewComponentType[IdentityComponent]()

// MedalComponent はメダルの情報を保持します。
type MedalComponent struct {
	Medal *Medal // models.Medal を参照
}

var CMedal = donburi.NewComponentType[MedalComponent]()

// PartsComponent はパーツの情報を保持します。
type PartsComponent struct {
	Parts map[PartSlotKey]*Part // models.Part を参照
}

var PartsComponentType = donburi.NewComponentType[PartsComponent]()

// StatusComponent はメダロットの現在の動的な状態を保持します。
type StatusComponent struct {
	State             MedarotState
	Gauge             float64
	IsEvasionDisabled bool // 「狙い撃ち」などで回避ができない状態
	IsDefenseDisabled bool // 「がむしゃら」などで防御ができない状態
}

var StatusComponentType = donburi.NewComponentType[StatusComponent]()

// ActionComponent は選択されたアクションとターゲットの情報を保持します。
type ActionComponent struct {
	SelectedPartKey PartSlotKey
	TargetedMedarot donburi.Entity // ターゲットエンティティ
	LastActionLog   string
}

var ActionComponentType = donburi.NewComponentType[ActionComponent]()

// RenderComponent は描画に関する情報を保持します。
type RenderComponent struct {
	DrawIndex int       // 情報パネルなどでの描画順
	Position  math.Vec2 // 将来的な拡張用の座標
}

var RenderComponentType = donburi.NewComponentType[RenderComponent]()

// AIControlledComponent はAIによって制御されることを示すタグコンポーネントです。
type AIControlledComponent struct{}

var AIControlledComponentType = donburi.NewComponentType[AIControlledComponent]()

// PlayerControlledComponent はプレイヤーによって制御されることを示すタグコンポーネントです。
type PlayerControlledComponent struct{}

var PlayerControlledComponentType = donburi.NewComponentType[PlayerControlledComponent]()

// --- Singleton Components ---

// GameStateComponent はゲーム全体のグローバルな状態を保持します。
type GameStateComponent struct {
	TickCount           int
	CurrentState        GameState
	Message             string
	PostMessageCallback func()
	Winner              TeamID
	RestartRequested    bool
	DebugMode           bool
}

var GameStateComponentType = donburi.NewComponentType[GameStateComponent]()

// PlayerActionSelectComponent はプレイヤーの行動選択UIの状態を保持します。
type PlayerActionSelectComponent struct {
	CurrentTarget    donburi.Entity
	AvailableActions []PartSlotKey
	ActionQueue      []donburi.Entity // 行動選択待ちのエンティティのキュー
}

var PlayerActionSelectComponentType = donburi.NewComponentType[PlayerActionSelectComponent]()

// ConfigComponent はロードされた設定とゲームデータを保持します。
type ConfigComponent struct {
	GameConfig *Config
	GameData   *GameData
}

var ConfigComponentType = donburi.NewComponentType[ConfigComponent]()

// --- Tags ---

var ReadyToExecuteActionTag = donburi.NewTag().SetName("ReadyToExecuteAction")
var BrokenTag = donburi.NewTag().SetName("Broken")
var ActionChargingTag = donburi.NewTag().SetName("ActionCharging")
var ActionCooldownTag = donburi.NewTag().SetName("ActionCooldown")

// --- Component Helper Methods ---

// IsBroken はStatusComponentが破壊状態かどうかを返します。
// これにより、各システムで状態をチェックするロジックが統一されます。
func (s *StatusComponent) IsBroken() bool {
	return s.State == StateBroken
}
