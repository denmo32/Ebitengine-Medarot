package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
)

// Game はECSのワールドとシステムを保持する、ゲーム全体の管理構造体です。
type Game struct {
	World          donburi.World
	ECS            *ecs.ECS
	systems        []System
	renderSystems  []DrawSystem
	gameStateEntry *donburi.Entry // グローバルな状態を持つシングルトンエンティティへの参照
}

// System はUpdateメソッドを持つすべてのシステムのインターフェースです。
type System interface {
	Update(ecs *ecs.ECS)
}

// DrawSystem はDrawメソッドを持つ描画システムのインターフェースです。
type DrawSystem interface {
	Draw(ecs *ecs.ECS, screen *ebiten.Image)
}

// NewGame はゲームを初期化します。
func NewGame(gameData *GameData, appConfig Config) *Game {
	world := donburi.NewWorld()
	gameECS := ecs.NewECS(world)
	// --- グローバルな状態を保持するシングルトンエンティティを作成 ---
	gameStateEntity := world.Create(GameStateComponentType, ConfigComponentType, PlayerActionSelectComponentType)
	gameStateEntry := world.Entry(gameStateEntity)
	// 各グローバルコンポーネントを初期化
	GameStateComponentType.SetValue(gameStateEntry, GameStateComponent{
		CurrentState: StatePlaying,
		// PlayerTeam:   Team1,
		DebugMode: true,
	})
	ConfigComponentType.SetValue(gameStateEntry, ConfigComponent{
		GameConfig: &appConfig,
		GameData:   gameData,
	})
	PlayerActionSelectComponentType.SetValue(gameStateEntry, PlayerActionSelectComponent{})
	// --- メダロットのエンティティを初期化 ---
	InitializeAllMedarotEntities(world, gameData)
	g := &Game{
		World:          world,
		ECS:            gameECS,
		gameStateEntry: gameStateEntry,
	}
	// --- システムを登録 ---
	// Updateされるシステム
	g.AddSystem(NewPlayerInputSystem())
	g.AddSystem(NewAISystem())
	g.AddSystem(NewGaugeUpdateSystem())
	g.AddSystem(NewActionExecutionSystem())
	g.AddSystem(NewGameRuleSystem())
	g.AddSystem(NewMessageSystem())
	// Drawされるシステム
	g.AddDrawSystem(NewRenderSystem())
	log.Println("Game instance created with ECS and systems registered.")
	return g
}

// AddSystem は更新ロジックを持つシステムをゲームに追加します。
func (g *Game) AddSystem(system System) {
	g.systems = append(g.systems, system)
}

// AddDrawSystem は描画ロジックを持つシステムをゲームに追加します。
func (g *Game) AddDrawSystem(system DrawSystem) {
	g.renderSystems = append(g.renderSystems, system)
}

// getSystem ヘルパーメソッド：特定の型のシステムを取得
func (g *Game) getSystem(target System) System {
	for _, s := range g.systems {
		if fmt.Sprintf("%T", s) == fmt.Sprintf("%T", target) {
			return s
		}
	}
	return nil
}

// Update はゲームのメインループです。
func (g *Game) Update() error {
	gs := GameStateComponentType.Get(g.gameStateEntry)

	// デバッグモードの切り替え
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		gs.DebugMode = !gs.DebugMode
	}

	// ゲームの状態に応じて実行するシステムを切り替える
	switch gs.CurrentState {
	case StatePlaying:
		// StatePlayingの時に動かすべきシステムだけを呼ぶ
		g.getSystem(&GameRuleSystem{}).Update(g.ECS)
		if GameStateComponentType.Get(g.gameStateEntry).CurrentState != StatePlaying {
			return nil
		} // 状態が変わったら即終了

		g.getSystem(&ActionExecutionSystem{}).Update(g.ECS)
		if GameStateComponentType.Get(g.gameStateEntry).CurrentState != StatePlaying {
			return nil
		}

		g.getSystem(&AISystem{}).Update(g.ECS)
		g.getSystem(&PlayerInputSystem{}).Update(g.ECS) // PlayerActionSelectへの遷移を担当
		if GameStateComponentType.Get(g.gameStateEntry).CurrentState != StatePlaying {
			return nil
		}

		g.getSystem(&GaugeUpdateSystem{}).Update(g.ECS)

	case StatePlayerActionSelect:
		// この状態ではPlayerInputSystemだけを動かす
		g.getSystem(&PlayerInputSystem{}).Update(g.ECS)

	case GameStateMessage:
		g.getSystem(&MessageSystem{}).Update(g.ECS)

	case GameStateOver:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			gs.RestartRequested = true
		}
	}

	// ティックカウントを更新
	gs.TickCount++

	// 変更された可能性のあるGameStateComponentを書き戻す
	GameStateComponentType.Set(g.gameStateEntry, gs)

	// リスタート要求があれば、特別なエラーを返してmain側で処理させる想定だったが、
	// ここで直接リセットする方がシンプル。
	if gs.RestartRequested {
		log.Println("Restarting game...")
		// NewGameを呼び出して自身をリセットする
		// 元のポインタが指す先のメモリを新しいゲームインスタンスで上書き
		config := ConfigComponentType.Get(g.gameStateEntry).GameConfig
		gameData := ConfigComponentType.Get(g.gameStateEntry).GameData
		*g = *NewGame(gameData, *config)
	}
	return nil
}

// Draw はゲーム画面を描画します。
func (g *Game) Draw(screen *ebiten.Image) {
	for _, s := range g.renderSystems {
		s.Draw(g.ECS, screen)
	}
}

// Layout はEbitengineにウィンドウサイズを伝えます。
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	config := ConfigComponentType.Get(g.gameStateEntry).GameConfig
	return config.UI.Screen.Width, config.UI.Screen.Height
}
