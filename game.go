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
	// GameStateシングルトンエンティティを取得
	if g.gameStateEntry == nil || !g.gameStateEntry.Valid() {
		return fmt.Errorf("gameStateEntry is not initialized")
	}
	gs := GameStateComponentType.Get(g.gameStateEntry)

	// デバッグモードの切り替えは常時受け付ける
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		gs.DebugMode = !gs.DebugMode
	}

	// ゲームの状態に応じて、実行するロジックを完全に分離する
	switch gs.CurrentState {
	case StatePlaying:
		// === ゲーム進行中の処理 ===
		// 1. ルールチェック (勝敗判定)
		g.getSystem(&GameRuleSystem{}).Update(g.ECS)
		// GameRuleSystemが状態をOverに変えたら、即座にこのフレームの処理を中断
		if GameStateComponentType.Get(g.gameStateEntry).CurrentState == GameStateOver {
			return nil
		}

		// 2. アクションの実行
		g.getSystem(&ActionExecutionSystem{}).Update(g.ECS)

		// 3. AIとプレイヤーの行動選択準備
		g.getSystem(&AISystem{}).Update(g.ECS)
		g.getSystem(&PlayerInputSystem{}).Update(g.ECS)

		// 4. 最後にゲージを更新
		g.getSystem(&GaugeUpdateSystem{}).Update(g.ECS)

	case StatePlayerActionSelect:
		// === プレイヤー行動選択中の処理 ===
		// プレイヤー入力システムのみを実行
		g.getSystem(&PlayerInputSystem{}).Update(g.ECS)

	case GameStateMessage:
		// === メッセージ表示中の処理 ===
		// メッセージを進めるシステムのみを実行
		g.getSystem(&MessageSystem{}).Update(g.ECS)

	case GameStateOver:
		// === ゲームオーバー時の処理 ===
		// クリックでリスタート要求フラグを立てる
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			gs.RestartRequested = true
		}
	}

	// ティックカウントは状態に関わらず更新
	gs.TickCount++

	// 最後に、変更された可能性のあるコンポーネントデータを書き戻す
	GameStateComponentType.Set(g.gameStateEntry, gs)

	// リスタート要求があれば、Terminationの代わりに特別なエラーを返す
	if gs.RestartRequested {
		// NewGameを呼び出して自身をリセットする
		log.Println("Restarting game...")
		*g = *NewGame(g.GameData, *g.Config)
		// gs.RestartRequestedをfalseに戻す
		gs = GameStateComponentType.Get(g.gameStateEntry)
		gs.RestartRequested = false
		GameStateComponentType.Set(g.gameStateEntry, gs)
	}

	return nil
}

// g.systemsスライスから特定の型のシステムを取得するヘルパーメソッド
func (g *Game) getSystem(target System) System {
	for _, s := range g.systems {
		// 型を比較して一致するシステムを返す
		if fmt.Sprintf("%T", s) == fmt.Sprintf("%T", target) {
			return s
		}
	}
	return nil // 見つからなかった場合
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
