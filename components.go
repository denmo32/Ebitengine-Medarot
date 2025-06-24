package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/features/math"
)

// 基本的な識別情報
type IdentityComponent struct {
	ID       string
	Name     string
	Team     TeamID
	IsLeader bool
}

var IdentityComponentType = donburi.NewComponentType[IdentityComponent]()

// メダル情報
type MedalComponent struct {
	Medal *Medal // models.Medal を再利用
}

var CMedal = donburi.NewComponentType[MedalComponent]() // さらに名前を変更して衝突を避ける

// パーツ情報
type PartsComponent struct {
	Parts map[PartSlotKey]*Part // models.Part を再利用
}

var PartsComponentType = donburi.NewComponentType[PartsComponent]()

// メダロットの現在の状態
type StatusComponent struct {
	State             MedarotState
	Gauge             float64
	IsEvasionDisabled bool // 狙い撃ちなどで回避不可
	IsDefenseDisabled bool // 打撃などで防御不可
}

var StatusComponentType = donburi.NewComponentType[StatusComponent]()

// 選択されたアクションとターゲット
type ActionComponent struct {
	SelectedPartKey     PartSlotKey
	TargetedMedarot     donburi.Entity // ターゲットエンティティを保持
	LastActionLog       string
	PendingActionEffect func(w donburi.World, actingEntity donburi.Entity) // 実行待ちのアクション効果
}

var ActionComponentType = donburi.NewComponentType[ActionComponent]()

// 描画に関する情報
type RenderComponent struct {
	DrawIndex int        // リスト内での描画順 (0から)
	Position  math.Vec2  // バトルフィールド上の現在位置 (将来的に詳細化)
	Color     [4]float32 // RGBA
}

var RenderComponentType = donburi.NewComponentType[RenderComponent]()

// AI制御用コンポーネント（今はシンプルにタグとして機能）
type AIControlledComponent struct{}

var AIControlledComponentType = donburi.NewComponentType[AIControlledComponent]()

// プレイヤー制御用コンポーネント（今はシンプルにタグとして機能）
type PlayerControlledComponent struct{}

var PlayerControlledComponentType = donburi.NewComponentType[PlayerControlledComponent]()

// ゲーム全体の状態を保持するコンポーネント (シングルトンエンティティ用)
type GameStateComponent struct {
	TickCount           int
	CurrentState        GameState // models.GameState
	PlayerTeam          TeamID
	Message             string
	PostMessageCallback func()
	Winner              TeamID
	RestartRequested    bool
	DebugMode           bool
	// actionQueueはシステム内で処理するか、別の方法で管理
}

var GameStateComponentType = donburi.NewComponentType[GameStateComponent]()

// プレイヤーの行動選択UIに関連するコンポーネント (シングルトンまたはUIエンティティ用)
type PlayerActionSelectComponent struct {
	ActingMedarot    donburi.Entity // 現在行動選択中のプレイヤーメダロット
	CurrentTarget    donburi.Entity // 現在選択されている仮ターゲット
	AvailableActions []PartSlotKey  // 選択可能なパーツスロット
}

var PlayerActionSelectComponentType = donburi.NewComponentType[PlayerActionSelectComponent]()

// タグコンポーネントの例
var NeedsActionSelectionTag = donburi.NewTag().SetName("NeedsActionSelection")
var ReadyToExecuteActionTag = donburi.NewTag().SetName("ReadyToExecuteAction")
var BrokenTag = donburi.NewTag().SetName("Broken")
var ActionChargingTag = donburi.NewTag().SetName("ActionCharging")
var ActionCooldownTag = donburi.NewTag().SetName("ActionCooldown")

// グローバル設定やデータを保持するコンポーネント (シングルトンエンティティ用)
type ConfigComponent struct {
	GameConfig *Config   // main.Config
	GameData   *GameData // main.GameData
}

var ConfigComponentType = donburi.NewComponentType[ConfigComponent]()
