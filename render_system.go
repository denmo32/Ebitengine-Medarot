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

// Update は System インターフェースを満たすために追加（現在は空）。
func (sys *RenderSystem) Update(ecs *ecs.ECS) {
	// 描画システムは通常Updateロジックを持たないことが多いが、
	// Gameのsystemsスライスに追加するために必要。
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

// MedarotDrawInfo は描画用のメダロット情報をまとめた構造体です。
type MedarotDrawInfo struct {
	Entry    *donburi.Entry
	Identity *IdentityComponent
	Status   *StatusComponent
	Render   *RenderComponent
	Parts    *PartsComponent
}

// Draw はRenderSystemのメイン描画ロジックです。
func (sys *RenderSystem) Draw(ecs *ecs.ECS, screen *ebiten.Image) {
	gameStateEntry, gsOk := GameStateComponentType.First(ecs.World)
	configEntry, cfgOk := ConfigComponentType.First(ecs.World)
	if !gsOk || !cfgOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	appConfig := ConfigComponentType.Get(configEntry).GameConfig
	pasComp := PlayerActionSelectComponentType.Get(gameStateEntry)

	// 背景とバトルフィールドの描画
	screen.Fill(appConfig.UI.Colors.Background)
	vector.StrokeRect(screen, 0, 0, float32(appConfig.UI.Screen.Width), appConfig.UI.Battlefield.Height, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.White, false)

	medarotCount := 0
	countQuery := donburi.NewQuery(filter.Contains(IdentityComponentType))
	countQuery.Each(ecs.World, func(_ *donburi.Entry) { medarotCount++ })
	playersPerTeam := medarotCount / 2
	if playersPerTeam == 0 && medarotCount > 0 {
		playersPerTeam = 1
	}

	for i := 0; i < playersPerTeam; i++ {
		yPos := appConfig.UI.Battlefield.MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, appConfig.UI.Battlefield.Team1HomeX, yPos, appConfig.UI.Battlefield.HomeMarkerRadius, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, appConfig.UI.Battlefield.Team2HomeX, yPos, appConfig.UI.Battlefield.HomeMarkerRadius, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, appConfig.UI.Battlefield.Team1ExecutionLineX, 0, appConfig.UI.Battlefield.Team1ExecutionLineX, appConfig.UI.Battlefield.Height, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, false)
	vector.StrokeLine(screen, appConfig.UI.Battlefield.Team2ExecutionLineX, 0, appConfig.UI.Battlefield.Team2ExecutionLineX, appConfig.UI.Battlefield.Height, appConfig.UI.Battlefield.LineWidth, appConfig.UI.Colors.Gray, false)

	// メダロット情報の収集とソート
	allMedarotsToDraw := []MedarotDrawInfo{}
	sys.medarotQuery.Each(ecs.World, func(entry *donburi.Entry) {
		allMedarotsToDraw = append(allMedarotsToDraw, MedarotDrawInfo{
			Entry: entry, Identity: IdentityComponentType.Get(entry), Status: StatusComponentType.Get(entry),
			Render: RenderComponentType.Get(entry), Parts: PartsComponentType.Get(entry),
		})
	})
	sort.Slice(allMedarotsToDraw, func(i, j int) bool {
		if allMedarotsToDraw[i].Identity.Team != allMedarotsToDraw[j].Identity.Team {
			return allMedarotsToDraw[i].Identity.Team < allMedarotsToDraw[j].Identity.Team
		}
		return allMedarotsToDraw[i].Render.DrawIndex < allMedarotsToDraw[j].Render.DrawIndex
	})

	// メダロットアイコンの描画
	for _, mdi := range allMedarotsToDraw {
		statusComp := mdi.Status
		identityComp := mdi.Identity
		renderComp := mdi.Render
		baseYPos := appConfig.UI.Battlefield.MedarotVerticalSpacing * float32(renderComp.DrawIndex+1)
		progress := statusComp.Gauge / 100.0
		homeX, execX := appConfig.UI.Battlefield.Team1HomeX, appConfig.UI.Battlefield.Team1ExecutionLineX
		if identityComp.Team == Team2 {
			homeX, execX = appConfig.UI.Battlefield.Team2HomeX, appConfig.UI.Battlefield.Team2ExecutionLineX
		}
		var currentX float32
		switch statusComp.State {
		case StateActionCharging:
			currentX = homeX + float32(progress)*(execX-homeX)
		case StateReadyToExecuteAction:
			currentX = execX
		case StateActionCooldown:
			currentX = execX - float32(progress)*(execX-homeX)
		default:
			currentX = homeX
		}
		if currentX < appConfig.UI.Battlefield.IconRadius {
			currentX = appConfig.UI.Battlefield.IconRadius
		}
		if currentX > float32(appConfig.UI.Screen.Width)-appConfig.UI.Battlefield.IconRadius {
			currentX = float32(appConfig.UI.Screen.Width) - appConfig.UI.Battlefield.IconRadius
		}

		iconColor := appConfig.UI.Colors.Team1
		if identityComp.Team == Team2 {
			iconColor = appConfig.UI.Colors.Team2
		}
		if statusComp.State == StateBroken {
			iconColor = appConfig.UI.Colors.Broken
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, appConfig.UI.Battlefield.IconRadius, iconColor, true)
		if identityComp.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, appConfig.UI.Battlefield.IconRadius+2, 2, appConfig.UI.Colors.Leader, true)
		}
	}

	// 情報パネルの描画
	for _, mdi := range allMedarotsToDraw {
		identityComp := mdi.Identity
		statusComp := mdi.Status
		renderComp := mdi.Render
		partsComp := mdi.Parts
		var panelX, panelY float32
		if identityComp.Team == Team1 {
			panelX = appConfig.UI.InfoPanel.Padding
			panelY = appConfig.UI.InfoPanel.StartY + appConfig.UI.InfoPanel.Padding + float32(renderComp.DrawIndex)*(appConfig.UI.InfoPanel.BlockHeight+appConfig.UI.InfoPanel.Padding)
		} else {
			panelX = appConfig.UI.InfoPanel.Padding*2 + appConfig.UI.InfoPanel.BlockWidth
			panelY = appConfig.UI.InfoPanel.StartY + appConfig.UI.InfoPanel.Padding + float32(renderComp.DrawIndex)*(appConfig.UI.InfoPanel.BlockHeight+appConfig.UI.InfoPanel.Padding)
		}
		drawMedarotInfoECS(screen, identityComp, statusComp, partsComp, panelX, panelY, appConfig, gs.DebugMode)
	}

	// --- ★★★ 修正箇所 ★★★ ---
	// 行動選択UIの描画
	if gs.CurrentState == StatePlayerActionSelect && len(pasComp.ActionQueue) > 0 {
		actingMedarotEntry := ecs.World.Entry(pasComp.ActionQueue[0])
		if actingMedarotEntry.Valid() {
			identity := IdentityComponentType.Get(actingMedarotEntry)

			// 背景オーバーレイ
			overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
			vector.DrawFilledRect(screen, 0, 0, float32(appConfig.UI.Screen.Width), float32(appConfig.UI.Screen.Height), overlayColor, false)

			// ウィンドウ本体
			boxW, boxH := 320, 200
			boxX := (appConfig.UI.Screen.Width - boxW) / 2
			boxY := (appConfig.UI.Screen.Height - boxH) / 2
			windowRect := image.Rect(boxX, boxY, boxX+boxW, boxY+boxH)
			DrawWindow(screen, windowRect, appConfig.UI.Colors.Background, appConfig.UI.Colors.Team1)

			// タイトル
			titleStr := fmt.Sprintf("%s の行動を選択", identity.Name)
			if MplusFont != nil {
				bounds := text.BoundString(MplusFont, titleStr)
				titleWidth := (bounds.Max.X - bounds.Min.X)
				text.Draw(screen, titleStr, MplusFont, appConfig.UI.Screen.Width/2-titleWidth/2, boxY+30, appConfig.UI.Colors.White)
			}

			// アクションボタン
			actingPartsComp := PartsComponentType.Get(actingMedarotEntry)
			for i, slotKey := range pasComp.AvailableActions {
				partData, exists := actingPartsComp.Parts[slotKey]
				if !exists {
					continue
				}
				btnW_modal := appConfig.UI.ActionModal.ButtonWidth
				btnH_modal := appConfig.UI.ActionModal.ButtonHeight
				btnSpacing_modal := appConfig.UI.ActionModal.ButtonSpacing
				buttonX_modal := appConfig.UI.Screen.Width/2 - int(btnW_modal/2)
				buttonY_modal := appConfig.UI.Screen.Height/2 - 50 + (int(btnH_modal)+int(btnSpacing_modal))*i
				buttonRect_modal := image.Rect(buttonX_modal, buttonY_modal, buttonX_modal+int(btnW_modal), buttonY_modal+int(btnH_modal))

				partStr := fmt.Sprintf("%s (%s)", partData.PartName, partData.Type)
				if partData.Category == CategoryShoot {
					if ecs.World.Valid(pasComp.CurrentTarget) {
						if targetEntry := ecs.World.Entry(pasComp.CurrentTarget); targetEntry.Valid() {
							partStr += fmt.Sprintf(" -> %s", IdentityComponentType.Get(targetEntry).Name)
						} else {
							partStr += " (ターゲット消失)"
						}
					} else {
						partStr += " (ターゲットなし)"
					}
				}
				DrawButton(screen, buttonRect_modal, partStr, MplusFont, appConfig.UI.Colors.Background, appConfig.UI.Colors.White, appConfig.UI.Colors.White)
			}
		} else {
			// 行動選択中のキャラが無効になった場合、システム側でキューから除外されるはずだが、
			// 念のため描画は何もしない
		}
	} else if gs.CurrentState == GameStateMessage || gs.CurrentState == GameStateOver {
		// メッセージ/ゲームオーバーパネルの描画
		windowWidth := int(float32(appConfig.UI.Screen.Width) * 0.7)
		windowHeight := int(float32(appConfig.UI.Screen.Height) * 0.25)
		windowX := (appConfig.UI.Screen.Width - windowWidth) / 2
		windowY := int(appConfig.UI.Battlefield.Height) - windowHeight/2
		windowRect := image.Rect(windowX, windowY, windowX+windowWidth, windowY+windowHeight)
		prompt := ""
		if gs.CurrentState == GameStateMessage {
			prompt = "クリックして続行..."
		}
		DrawMessagePanel(screen, windowRect, gs.Message, prompt, MplusFont, &appConfig.UI)
		if gs.CurrentState == GameStateOver && MplusFont != nil {
			resetMsg := "クリックでリスタート"
			bounds := text.BoundString(MplusFont, resetMsg)
			msgX := windowX + (windowWidth-(bounds.Max.X-bounds.Min.X))/2
			msgY := windowY + windowHeight - (bounds.Max.Y - bounds.Min.Y) - 10
			text.Draw(screen, resetMsg, MplusFont, msgX, msgY, appConfig.UI.Colors.White)
		}
	}

	// デバッグ情報の描画
	if gs.DebugMode {
		var queueIds []int
		for _, e := range pasComp.ActionQueue {
			queueIds = append(queueIds, int(e.Id()))
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick:%d St:%s Ent:%d Q:%v",
			gs.TickCount, gs.CurrentState, ecs.World.Len(), queueIds),
			10, appConfig.UI.Screen.Height-15)
	}
}

func drawMedarotInfoECS(screen *ebiten.Image, identity *IdentityComponent, status *StatusComponent, parts *PartsComponent, startX, startY float32, config *Config, debugMode bool) {
	if MplusFont == nil {
		return
	}
	nameColor := config.UI.Colors.White
	if status.State_is_broken_internal() { // Assumes State_is_broken_internal is available
		nameColor = config.UI.Colors.Broken
	}
	text.Draw(screen, identity.Name, MplusFont, int(startX), int(startY)+int(config.UI.InfoPanel.TextLineHeight), nameColor)
	if debugMode {
		stateStr := fmt.Sprintf("St:%s(G:%.0f)", status.State, status.Gauge)
		text.Draw(screen, stateStr, MplusFont, int(startX+70), int(startY)+int(config.UI.InfoPanel.TextLineHeight), config.UI.Colors.Yellow)
	}
	partSlots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	partSlotDisplayNames := map[PartSlotKey]string{PartSlotHead: "頭", PartSlotRightArm: "右", PartSlotLeftArm: "左", PartSlotLegs: "脚"}
	currentInfoY := startY + config.UI.InfoPanel.TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+config.UI.InfoPanel.TextLineHeight > startY+config.UI.InfoPanel.BlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		part, exists := parts.Parts[slotKey]
		if exists && part != nil {
			currentArmor := part.Armor
			if part.IsBroken {
				currentArmor = 0
			}
			hpText = fmt.Sprintf("%s:%d/%d", displayName, currentArmor, part.MaxArmor)
			if part.MaxArmor > 0 {
				hpPercentage := 0.0
				if part.MaxArmor > 0 {
					hpPercentage = float64(currentArmor) / float64(part.MaxArmor)
				}
				gaugeX := startX + config.UI.InfoPanel.PartHPGaugeOffsetX
				gaugeY := currentInfoY - config.UI.InfoPanel.TextLineHeight/2 - config.UI.InfoPanel.PartHPGaugeHeight/2
				vector.DrawFilledRect(screen, gaugeX, gaugeY, config.UI.InfoPanel.PartHPGaugeWidth, config.UI.InfoPanel.PartHPGaugeHeight, color.NRGBA{50, 50, 50, 255}, true)
				barFillColor := config.UI.Colors.HP
				if part.IsBroken {
					barFillColor = config.UI.Colors.Broken
				} else if hpPercentage < 0.3 {
					barFillColor = config.UI.Colors.Red
				}
				vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(config.UI.InfoPanel.PartHPGaugeWidth)*hpPercentage), config.UI.InfoPanel.PartHPGaugeHeight, barFillColor, true)
			}
		} else {
			hpText = fmt.Sprintf("%s:N/A", displayName)
		}
		textColor := config.UI.Colors.White
		if exists && part != nil && part.IsBroken {
			textColor = config.UI.Colors.Broken
		}
		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY), textColor)
		if exists && part != nil && part.MaxArmor > 0 {
			partNameX := startX + config.UI.InfoPanel.PartHPGaugeOffsetX + config.UI.InfoPanel.PartHPGaugeWidth + 5
			text.Draw(screen, part.PartName, MplusFont, int(partNameX), int(currentInfoY), textColor)
		}
		currentInfoY += config.UI.InfoPanel.TextLineHeight + 4
	}
}
