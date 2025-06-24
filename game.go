package main

import (
	"fmt"
	//"image"
	"log"
	//"math/rand"
	// "sort" // ECSでは直接ソートされたリストをGame構造体で持たない想定

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter" // ★ filter をインポート
	"image/color"                       // ★ 追加
)

// Game はECS Worldとシステムを保持します。
type Game struct {
	World          donburi.World
	ECS            *ecs.ECS
	GameData       *GameData    // CSVからロードしたデータ
	Config         *Config      // 設定ファイルからロードしたデータ
	systems        []System     // 実行するシステムのリスト
	renderSystems  []DrawSystem // ★ 型を []DrawSystem に変更
	gameStateEntry *donburi.Entry
}

// System はUpdateメソッドを持つインターフェースです。
type System interface {
	Update(ecs *ecs.ECS)
}

// DrawSystem はDrawメソッドを持つインターフェースです。
type DrawSystem interface {
	Draw(ecs *ecs.ECS, screen *ebiten.Image)
}

// NewGame はゲームを初期化します
func NewGame(gameData *GameData, appConfig Config) *Game {
	world := donburi.NewWorld()
	gameECS := ecs.NewECS(world)

	// グローバルなゲーム状態と設定を保持するシングルトンエンティティを作成
	gameStateEntity := world.Create(GameStateComponentType, ConfigComponentType, PlayerActionSelectComponentType)
	gameStateEntry := world.Entry(gameStateEntity)

	GameStateComponentType.SetValue(gameStateEntry, GameStateComponent{
		TickCount:    0,
		CurrentState: StatePlaying,
		PlayerTeam:   Team1,
		DebugMode:    true,
	})
	ConfigComponentType.SetValue(gameStateEntry, ConfigComponent{
		GameConfig: &appConfig,
		GameData:   gameData,
	})
	PlayerActionSelectComponentType.SetValue(gameStateEntry, PlayerActionSelectComponent{})

	// メダロットエンティティを初期化
	InitializeAllMedarotEntities(world, gameData)

	// TODO: この時点でMedarotsのスライスは不要になるはず。
	// sortedMedarotsForDraw や team1Leader/team2Leader もECSのクエリで代替する。
	// actionQueueもPlayerActionSelectComponentやシステム内で管理する。

	g := &Game{
		World:          world,
		ECS:            gameECS,
		GameData:       gameData,
		Config:         &appConfig,
		gameStateEntry: gameStateEntry,
	}

	// システムを登録
	g.AddSystem(NewPlayerInputSystem())
	g.AddSystem(NewAISystem())
	g.AddSystem(NewGaugeUpdateSystem())
	g.AddSystem(NewActionExecutionSystem())
	// Note: NewMovementSystem, NewDamageSystem, NewStateTransitionSystem are not yet implemented
	// or their logic is partially integrated into other systems for now.
	g.AddSystem(NewGameRuleSystem())
	g.AddSystem(NewMessageSystem())
	g.AddSystem(NewRenderSystem())

	log.Println("Game instance created with ECS and systems registered.")
	if countMedarotEntities(world) == 0 {
		log.Fatal("Game initialized with no Medarot entities.")
	}

	return g
}

func countMedarotEntities(w donburi.World) int {
	query := donburi.NewQuery(filter.Contains(IdentityComponentType)) // ★ filter.Contains を使用
	count := 0
	query.Each(w, func(entry *donburi.Entry) {
		count++
	})
	return count
}

// AddSystem はゲームにシステムを追加します。
// DrawSystemインターフェースを実装していれば、renderSystemsにも追加します。
func (g *Game) AddSystem(system System) {
	g.systems = append(g.systems, system)
	if drawSystem, ok := system.(DrawSystem); ok {
		if g.renderSystems == nil {
			g.renderSystems = make([]DrawSystem, 0) // ★ []DrawSystem に変更
		}
		g.renderSystems = append(g.renderSystems, drawSystem)
		log.Printf("Added DrawSystem: %T", drawSystem)
	}
}

// AddRenderSystem は描画専用システムを登録します (AddSystemと役割が被るなら不要)
// 今回はAddSystemにDrawSystemの判定を入れたので、このメソッドは不要。
// func (g *Game) AddRenderSystem(system DrawSystem) {
// 	if g.renderSystems == nil {
// 		g.renderSystems = make([]DrawSystem, 0)
// 	}
// 	g.renderSystems = append(g.renderSystems, system)
// 	log.Printf("Added RenderSystem: %T", system)
// }

// Update はゲームのメインループです
func (g *Game) Update() error {
	// gameStateEntryがnilでないことを確認
	if g.gameStateEntry == nil || !g.gameStateEntry.Valid() {
		log.Println("Error: gameStateEntry is nil or invalid in Game.Update")
		return fmt.Errorf("gameStateEntry is not initialized")
	}
	gs := GameStateComponentType.Get(g.gameStateEntry)

	if gs.CurrentState == GameStateOver {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			gs.RestartRequested = true
			log.Println("GameStateOver: Click detected, requesting restart.")
		}
		if gs.RestartRequested {
			log.Println("Restarting game...")
			// Proper game restart would involve re-initializing world, entities, systems.
			// For now, simple termination.
			return ebiten.Termination
		}
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		gs.DebugMode = !gs.DebugMode
		GameStateComponentType.Set(g.gameStateEntry, gs)
	}

	gs.TickCount++
	GameStateComponentType.Set(g.gameStateEntry, gs)

	// 登録された各更新システムを実行
	for _, system := range g.systems {
		// DrawSystemインターフェースを実装しているシステムはUpdateロジックを持たない場合があるので、
		// SystemインターフェースのUpdateメソッドのみを呼び出す。
		// RenderSystemのようにUpdateを持たないものは、g.systemsには追加されるが、
		// DrawSystemとしてg.renderSystemsに追加され、DrawメソッドのみがGame.Drawから呼ばれる。
		// もしSystemインターフェースを実装しつつUpdateが不要な描画専用システムがある場合、
		// そのUpdateメソッドは空実装にする。
		// 今回のRenderSystemはSystemインターフェースを実装していないので、このループでは実行されない。
		// (もしRenderSystemもSystemインターフェースを実装しUpdateを持つならここで呼ばれる)
		system.Update(g.ECS)
	}

	return nil
}

// showMessage はメッセージ表示状態に移行します (GameStateComponentを更新)
// この関数は外部 (main.goなど) や、ECSのシステムに直接アクセスできない古いコードから
// ゲームの状態を変更するために残すこともできますが、理想的にはシステム内で完結すべきです。
// 今回はActionExecutionSystem内のshowGameMessageヘルパーに類似ロジックを移譲しています。
// グローバルなshowMessageは不要になるかもしれません。
/*
func (g *Game) showMessage(msg string, callback func()) {
	gs := GameStateData.Get(g.gameStateEntry)
	gs.Message = msg
	gs.PostMessageCallback = callback
	gs.CurrentState = GameStateMessage
	GameStateData.Set(g.gameStateEntry, gs)
}
*/

// 古い updateXXX 関数群はsystemsに移行したため不要。

// Draw はゲーム画面を描画します
func (g *Game) Draw(screen *ebiten.Image) {
	// RenderSystemが登録されていれば、それが全ての描画を担当します。
	if len(g.renderSystems) > 0 {
		for _, system := range g.renderSystems {
			// AddSystemでDrawSystem型アサーション済みなので、再度チェックは理論上不要だが念のため。
			if drawSystem, ok := system.(DrawSystem); ok {
				drawSystem.Draw(g.ECS, screen)
			}
		}
	} else {
		// RenderSystemが登録されていない場合のフォールバック/エラー表示
		screen.Fill(color.RGBA{R: 255, G: 0, B: 255, A: 255}) // Magenta to indicate error
		if MplusFont != nil {                                 // MplusFontがロードされていればエラーメッセージ表示
			ebitenutil.DebugPrintAt(screen, "Error: RenderSystem not found!", 10, 10)
		}
		log.Println("Error in Game.Draw: No render systems available.")
	}
}

// Layout は画面レイアウトを定義します
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// gameStateEntryがnilチェックも追加（NewGame完了前に呼ばれる可能性は低いが念のため）
	if g.gameStateEntry == nil {
		log.Println("Warning: gameStateEntry is nil in Layout. Using default size.")
		return 640, 480
	}
	configComp := ConfigComponentType.Get(g.gameStateEntry) // ConfigComponentType を使用
	if configComp != nil && configComp.GameConfig != nil {
		return configComp.GameConfig.UI.Screen.Width, configComp.GameConfig.UI.Screen.Height
	}
	// フォールバック値（あるいはエラーハンドリング）
	log.Println("Warning: ConfigComponent or GameConfig not fully initialized for Layout. Using default size.")
	return 640, 480
}
