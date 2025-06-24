package main

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// RenderSystem はゲームの描画を担当します。
type RenderSystem struct {
	medarotQuery *donburi.Query
}

// NewRenderSystem はRenderSystemを初期化します。
func NewRenderSystem() *RenderSystem {
	return &RenderSystem{
		medarotQuery: donburi.NewQuery(filter.And(
			filter.Contains(IdentityComponentType), filter.Contains(StatusComponentType),
			filter.Contains(RenderComponentType), filter.Contains(PartsComponentType),
		)),
	}
}

// Update はSystemインターフェースを満たすためのメソッドです。
// RenderSystemではUpdateロジックは不要なため空実装とします。
func (sys *RenderSystem) Update(ecs *ecs.ECS) {}

// Draw はRenderSystemのメイン描画ロジックです。
func (sys *RenderSystem) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	// グローバルなコンポーネントを取得
	gameStateEntry, gsOk := GameStateComponentType.First(ecs.World)
	configEntry, cfgOk := ConfigComponentType.First(ecs.World)
	pasEntry, pasOk := PlayerActionSelectComponentType.First(ecs.World)
	if !gsOk || !cfgOk || !pasOk {
		return // 必要なコンポーネントがなければ描画しない
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	appConfig := ConfigComponentType.Get(configEntry).GameConfig
	pasComp := PlayerActionSelectComponentType.Get(pasEntry)

	sys.drawBattlefield(screen, ecs, appConfig)
	sys.drawAllMedarots(screen, ecs, appConfig)
	sys.drawUI(screen, ecs, gs, pasComp, appConfig)
	sys.drawDebugInfo(screen, ecs, gs, pasComp, appConfig)
}

// drawBattlefield は背景と戦場を描画します。
func (sys *RenderSystem) drawBattlefield(screen *ebiten.Image, ecs *ecs.ECS, config *Config) {
	screen.Fill(config.UI.Colors.Background)
	bf := config.UI.Battlefield
	vector.StrokeRect(screen, 0, 0, float32(config.UI.Screen.Width), bf.Height, bf.LineWidth, config.UI.Colors.White, false)

	medarotCount := 0
	donburi.NewQuery(filter.Contains(IdentityComponentType)).Each(ecs.World, func(_ *donburi.Entry) { medarotCount++ })
	playersPerTeam := medarotCount / 2

	for i := 0; i < playersPerTeam; i++ {
		yPos := bf.MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, bf.Team1HomeX, yPos, bf.HomeMarkerRadius, bf.LineWidth, config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, bf.Team2HomeX, yPos, bf.HomeMarkerRadius, bf.LineWidth, config.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, bf.Team1ExecutionLineX, 0, bf.Team1ExecutionLineX, bf.Height, bf.LineWidth, config.UI.Colors.Gray, false)
	vector.StrokeLine(screen, bf.Team2ExecutionLineX, 0, bf.Team2ExecutionLineX, bf.Height, bf.LineWidth, config.UI.Colors.Gray, false)
}

// MedarotDrawInfo は描画用にソートするための一時的な構造体です。
type MedarotDrawInfo struct {
	Identity *IdentityComponent
	Status   *StatusComponent
	Render   *RenderComponent
	Parts    *PartsComponent
}

// drawAllMedarots は全てのメダロットのアイコンと情報パネルを描画します。
func (sys *RenderSystem) drawAllMedarots(screen *ebiten.Image, ecs *ecs.ECS, config *Config) {
	allMedarotsToDraw := []MedarotDrawInfo{}
	sys.medarotQuery.Each(ecs.World, func(entry *donburi.Entry) {
		allMedarotsToDraw = append(allMedarotsToDraw, MedarotDrawInfo{
			Identity: IdentityComponentType.Get(entry),
			Status:   StatusComponentType.Get(entry),
			Render:   RenderComponentType.Get(entry),
			Parts:    PartsComponentType.Get(entry),
		})
	})
	// チームと描画インデックスでソート
	sort.Slice(allMedarotsToDraw, func(i, j int) bool {
		if allMedarotsToDraw[i].Identity.Team != allMedarotsToDraw[j].Identity.Team {
			return allMedarotsToDraw[i].Identity.Team < allMedarotsToDraw[j].Identity.Team
		}
		return allMedarotsToDraw[i].Render.DrawIndex < allMedarotsToDraw[j].Render.DrawIndex
	})

	for _, mdi := range allMedarotsToDraw {
		sys.drawMedarotIcon(screen, mdi.Identity, mdi.Status, mdi.Render, config)
		sys.drawMedarotInfo(screen, mdi.Identity, mdi.Status, mdi.Parts, mdi.Render, config, GameStateComponentType.Get(GameStateComponentType.MustFirst(ecs.World)).DebugMode)
	}
}

// drawMedarotIcon はメダロットのアイコンをバトルフィールドに描画します。
func (sys *RenderSystem) drawMedarotIcon(screen *ebiten.Image, identity *IdentityComponent, status *StatusComponent, render *RenderComponent, config *Config) {
	bf := config.UI.Battlefield
	baseYPos := bf.MedarotVerticalSpacing * float32(render.DrawIndex+1)
	progress := status.Gauge / 100.0
	homeX, execX := bf.Team1HomeX, bf.Team1ExecutionLineX
	if identity.Team == Team2 {
		homeX, execX = bf.Team2HomeX, bf.Team2ExecutionLineX
	}

	var currentX float32
	switch status.State {
	case StateActionCharging:
		currentX = homeX + float32(progress)*(execX-homeX)
	case StateReadyToExecuteAction:
		currentX = execX
	case StateActionCooldown:
		currentX = execX - float32(progress)*(execX-homeX)
	default:
		currentX = homeX
	}

	iconColor := config.UI.Colors.Team1
	if identity.Team == Team2 {
		iconColor = config.UI.Colors.Team2
	}
	if status.IsBroken() {
		iconColor = config.UI.Colors.Broken
	}
	vector.DrawFilledCircle(screen, currentX, baseYPos, bf.IconRadius, iconColor, true)
	if identity.IsLeader {
		vector.StrokeCircle(screen, currentX, baseYPos, bf.IconRadius+2, 2, config.UI.Colors.Leader, true)
	}
}

// drawMedarotInfo は情報パネルを描画します。ui_draw.goのヘルパーを呼び出します。
func (sys *RenderSystem) drawMedarotInfo(screen *ebiten.Image, identity *IdentityComponent, status *StatusComponent, parts *PartsComponent, render *RenderComponent, config *Config, debug bool) {
	ip := config.UI.InfoPanel
	var panelX, panelY float32
	if identity.Team == Team1 {
		panelX = ip.Padding
		panelY = ip.StartY + ip.Padding + float32(render.DrawIndex)*(ip.BlockHeight+ip.Padding)
	} else {
		panelX = ip.Padding*2 + ip.BlockWidth
		panelY = ip.StartY + ip.Padding + float32(render.DrawIndex)*(ip.BlockHeight+ip.Padding)
	}
	drawMedarotInfoPanel(screen, identity, status, parts, panelX, panelY, config, debug)
}

// drawUI はゲームの状態に応じたUI（行動選択モーダル、メッセージパネルなど）を描画します。
func (sys *RenderSystem) drawUI(screen *ebiten.Image, ecs *ecs.ECS, gs *GameStateComponent, pasComp *PlayerActionSelectComponent, config *Config) {
	switch gs.CurrentState {
	case StatePlayerActionSelect:
		sys.drawActionSelectModal(screen, ecs, pasComp, config)
	case GameStateMessage, GameStateOver:
		sys.drawGameMessagePanel(screen, gs, config)
	}
}

// drawActionSelectModal は行動選択モーダルを描画します。
func (sys *RenderSystem) drawActionSelectModal(screen *ebiten.Image, ecs *ecs.ECS, pasComp *PlayerActionSelectComponent, config *Config) {
	if len(pasComp.ActionQueue) == 0 {
		return
	}
	actingMedarotEntry := ecs.World.Entry(pasComp.ActionQueue[0])
	if !actingMedarotEntry.Valid() {
		return
	}

	identity := IdentityComponentType.Get(actingMedarotEntry)
	actingPartsComp := PartsComponentType.Get(actingMedarotEntry)
	ui := config.UI

	// 背景オーバーレイ
	overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(ui.Screen.Width), float32(ui.Screen.Height), overlayColor, false)

	// ウィンドウ
	boxW, boxH := 320, 200
	boxX := (ui.Screen.Width - boxW) / 2
	boxY := (ui.Screen.Height - boxH) / 2
	windowRect := image.Rect(boxX, boxY, boxX+boxW, boxY+boxH)
	DrawWindow(screen, windowRect, ui.Colors.Background, ui.Colors.Team1)

	// タイトル
	titleStr := fmt.Sprintf("%s の行動を選択", identity.Name)
	if MplusFont != nil {
		bounds := text.BoundString(MplusFont, titleStr)
		titleWidth := (bounds.Max.X - bounds.Min.X)
		text.Draw(screen, titleStr, MplusFont, ui.Screen.Width/2-titleWidth/2, boxY+30, ui.Colors.White)
	}

	// アクションボタン
	for i, slotKey := range pasComp.AvailableActions {
		partData := actingPartsComp.Parts[slotKey]
		btnW, btnH, btnS := ui.ActionModal.ButtonWidth, ui.ActionModal.ButtonHeight, ui.ActionModal.ButtonSpacing
		btnX := ui.Screen.Width/2 - int(btnW/2)
		btnY := ui.Screen.Height/2 - 50 + (int(btnH)+int(btnS))*i
		btnRect := image.Rect(btnX, btnY, btnX+int(btnW), btnY+int(btnH))

		partStr := fmt.Sprintf("%s (%s)", partData.PartName, partData.Type)
		if partData.Category == CategoryShoot || partData.Category == CategoryFight {
			if ecs.World.Valid(pasComp.CurrentTarget) {
				if targetEntry := ecs.World.Entry(pasComp.CurrentTarget); targetEntry.Valid() {
					partStr += fmt.Sprintf(" -> %s", IdentityComponentType.Get(targetEntry).Name)
				}
			}
		}
		DrawButton(screen, btnRect, partStr, MplusFont, ui.Colors.Background, ui.Colors.White, ui.Colors.White)
	}
}

// drawGameMessagePanel はメッセージやゲームオーバー表示を描画します。
func (sys *RenderSystem) drawGameMessagePanel(screen *ebiten.Image, gs *GameStateComponent, config *Config) {
	ui := config.UI
	width := int(float32(ui.Screen.Width) * 0.7)
	height := int(float32(ui.Screen.Height) * 0.25)
	x := (ui.Screen.Width - width) / 2
	y := int(ui.Battlefield.Height) - height/2
	rect := image.Rect(x, y, x+width, y+height)

	prompt := ""
	if gs.CurrentState == GameStateMessage {
		prompt = "クリックして続行..."
	} else if gs.CurrentState == GameStateOver {
		prompt = "クリックでリスタート"
	}
	DrawMessagePanel(screen, rect, gs.Message, prompt, MplusFont, &ui)
}

// drawDebugInfo はデバッグ情報を描画します。
func (sys *RenderSystem) drawDebugInfo(screen *ebiten.Image, ecs *ecs.ECS, gs *GameStateComponent, pasComp *PlayerActionSelectComponent, config *Config) {
	if !gs.DebugMode {
		return
	}
	var queueIds []int
	for _, e := range pasComp.ActionQueue {
		queueIds = append(queueIds, int(e.Id()))
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick:%d St:%s Ent:%d Q:%v",
		gs.TickCount, gs.CurrentState, ecs.World.Len(), queueIds),
		10, config.UI.Screen.Height-15)
}
