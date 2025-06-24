package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func drawBattlefield(screen *ebiten.Image, g *Game) {
	vector.StrokeRect(screen, 0, 0, float32(g.Config.UI.Screen.Width), g.Config.UI.Battlefield.Height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.White, false)
	playersPerTeam := len(g.Medarots) / 2
	for i := 0; i < playersPerTeam; i++ {
		yPos := g.Config.UI.Battlefield.MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, g.Config.UI.Battlefield.Team1HomeX, yPos, g.Config.UI.Battlefield.HomeMarkerRadius, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, true)
		vector.StrokeCircle(screen, g.Config.UI.Battlefield.Team2HomeX, yPos, g.Config.UI.Battlefield.HomeMarkerRadius, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, true)
	}
	vector.StrokeLine(screen, g.Config.UI.Battlefield.Team1ExecutionLineX, 0, g.Config.UI.Battlefield.Team1ExecutionLineX, g.Config.UI.Battlefield.Height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, false)
	vector.StrokeLine(screen, g.Config.UI.Battlefield.Team2ExecutionLineX, 0, g.Config.UI.Battlefield.Team2ExecutionLineX, g.Config.UI.Battlefield.Height, g.Config.UI.Battlefield.LineWidth, g.Config.UI.Colors.Gray, false)
}

func drawMedarotIcons(screen *ebiten.Image, g *Game) {
	for _, medarot := range g.Medarots {
		baseYPos := g.Config.UI.Battlefield.MedarotVerticalSpacing * float32(medarot.DrawIndex+1)
		currentX := calculateIconX(medarot, &g.Config)
		if currentX < g.Config.UI.Battlefield.IconRadius {
			currentX = g.Config.UI.Battlefield.IconRadius
		}
		if currentX > float32(g.Config.UI.Screen.Width)-g.Config.UI.Battlefield.IconRadius {
			currentX = float32(g.Config.UI.Screen.Width) - g.Config.UI.Battlefield.IconRadius
		}
		iconColor := g.Config.UI.Colors.Team1
		if medarot.Team == Team2 {
			iconColor = g.Config.UI.Colors.Team2
		}
		if medarot.State == StateBroken {
			iconColor = g.Config.UI.Colors.Broken
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, g.Config.UI.Battlefield.IconRadius, iconColor, true)
		if medarot.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, g.Config.UI.Battlefield.IconRadius+2, 2, g.Config.UI.Colors.Leader, true)
		}
	}
}

func drawInfoPanels(screen *ebiten.Image, g *Game) {
	// g.sortedMedarotsForDraw を使うことで、表示順を固定
	for _, medarot := range g.sortedMedarotsForDraw {
		var panelX, panelY float32
		if medarot.Team == Team1 {
			panelX = g.Config.UI.InfoPanel.Padding
			panelY = g.Config.UI.InfoPanel.StartY + g.Config.UI.InfoPanel.Padding + float32(medarot.DrawIndex)*(g.Config.UI.InfoPanel.BlockHeight+g.Config.UI.InfoPanel.Padding)
		} else {
			panelX = g.Config.UI.InfoPanel.Padding*2 + g.Config.UI.InfoPanel.BlockWidth
			panelY = g.Config.UI.InfoPanel.StartY + g.Config.UI.InfoPanel.Padding + float32(medarot.DrawIndex)*(g.Config.UI.InfoPanel.BlockHeight+g.Config.UI.InfoPanel.Padding)
		}
		drawMedarotInfo(screen, medarot, panelX, panelY, &g.Config, g.DebugMode)
	}
}

func calculateIconX(medarot *Medarot, config *Config) float32 {
	progress := medarot.Gauge / 100.0
	homeX, execX := config.UI.Battlefield.Team1HomeX, config.UI.Battlefield.Team1ExecutionLineX
	if medarot.Team == Team2 {
		homeX, execX = config.UI.Battlefield.Team2HomeX, config.UI.Battlefield.Team2ExecutionLineX
	}
	switch medarot.State {
	case StateActionCharging:
		return homeX + float32(progress)*(execX-homeX)
	case StateReadyToExecuteAction:
		return execX
	case StateActionCooldown:
		return execX - float32(progress)*(execX-homeX)
	default:
		return homeX
	}
}

func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY float32, config *Config, debugMode bool) {
	if MplusFont == nil {
		return
	}
	var nameColor color.Color = config.UI.Colors.White
	if medarot.State == StateBroken {
		nameColor = config.UI.Colors.Broken
	}
	text.Draw(screen, medarot.Name, MplusFont, int(startX), int(startY)+int(config.UI.InfoPanel.TextLineHeight), nameColor)
	if debugMode {
		stateStr := fmt.Sprintf("St: %s", medarot.State)
		text.Draw(screen, stateStr, MplusFont, int(startX+70), int(startY)+int(config.UI.InfoPanel.TextLineHeight), config.UI.Colors.Yellow)
	}
	// ★定数を使用
	partSlots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	partSlotDisplayNames := map[PartSlotKey]string{PartSlotHead: "頭部", PartSlotRightArm: "右腕", PartSlotLeftArm: "左腕", PartSlotLegs: "脚部"}
	currentInfoY := startY + config.UI.InfoPanel.TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+config.UI.InfoPanel.TextLineHeight > startY+config.UI.InfoPanel.BlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		part, exists := medarot.Parts[slotKey]
		if exists && part != nil {
			currentArmor := part.Armor
			if part.IsBroken {
				currentArmor = 0
			}
			hpText = fmt.Sprintf("%s: %d/%d", displayName, currentArmor, part.MaxArmor)
			if part.MaxArmor > 0 {
				hpPercentage := float64(part.Armor) / float64(part.MaxArmor)
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
			hpText = fmt.Sprintf("%s: N/A", displayName)
		}
		var textColor color.Color = config.UI.Colors.White
		if exists && part != nil && part.IsBroken {
			textColor = config.UI.Colors.Broken
		}
		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY), textColor)
		if exists && part != nil && part.MaxArmor > 0 {
			gaugeX := startX + config.UI.InfoPanel.PartHPGaugeOffsetX
			partNameX := gaugeX + config.UI.InfoPanel.PartHPGaugeWidth + 5
			text.Draw(screen, part.PartName, MplusFont, int(partNameX), int(currentInfoY), textColor)
		}
		currentInfoY += config.UI.InfoPanel.TextLineHeight + 4
	}
}

func drawActionModal(screen *ebiten.Image, g *Game) {
	// ★ g から行動中のメダロットを取得
	medarot := g.actionQueue[0]

	overlayColor := color.NRGBA{R: 0, G: 0, B: 0, A: 180}
	vector.DrawFilledRect(screen, 0, 0, float32(g.Config.UI.Screen.Width), float32(g.Config.UI.Screen.Height), overlayColor, false)

	boxW, boxH := 320, 200
	boxX := (g.Config.UI.Screen.Width - boxW) / 2
	boxY := (g.Config.UI.Screen.Height - boxH) / 2
	windowRect := image.Rect(boxX, boxY, boxX+boxW, boxY+boxH)

	DrawWindow(screen, windowRect, g.Config.UI.Colors.Background, g.Config.UI.Colors.Team1)

	titleStr := fmt.Sprintf("%s の行動を選択", medarot.Name)
	if MplusFont != nil {
		// ▼▼▼ ここを修正 ▼▼▼
		bounds := text.BoundString(MplusFont, titleStr)
		// ▲▲▲ ここを修正 ▲▲▲
		titleWidth := (bounds.Max.X - bounds.Min.X) // .Ceil() は image.Point にはないため削除（必要ならintに変換）
		text.Draw(screen, titleStr, MplusFont, g.Config.UI.Screen.Width/2-titleWidth/2, boxY+30, g.Config.UI.Colors.White)
	}

	availableParts := medarot.GetAvailableAttackParts()
	for i, part := range availableParts {
		btnW := g.Config.UI.ActionModal.ButtonWidth
		btnH := g.Config.UI.ActionModal.ButtonHeight
		btnSpacing := g.Config.UI.ActionModal.ButtonSpacing
		buttonX := g.Config.UI.Screen.Width/2 - int(btnW/2)
		buttonY := g.Config.UI.Screen.Height/2 - 50 + (int(btnH)+int(btnSpacing))*i
		buttonRect := image.Rect(buttonX, buttonY, buttonX+int(btnW), buttonY+int(btnH))

		partStr := fmt.Sprintf("%s (%s)", part.PartName, part.Type)
		if part.Category == CategoryShoot {
			// ★ g からターゲット情報を取得
			if g.playerActionTarget != nil {
				partStr += fmt.Sprintf(" -> %s", g.playerActionTarget.Name)
			} else {
				partStr += " (ターゲットなし)"
			}
		}

		DrawButton(screen, buttonRect, partStr, MplusFont,
			g.Config.UI.Colors.Background, g.Config.UI.Colors.White, g.Config.UI.Colors.White)
	}
}

func drawMessageWindow(screen *ebiten.Image, g *Game) {
	windowWidth := int(float32(g.Config.UI.Screen.Width) * 0.7)
	windowHeight := int(float32(g.Config.UI.Screen.Height) * 0.25)
	windowX := (g.Config.UI.Screen.Width - windowWidth) / 2
	windowY := int(g.Config.UI.Battlefield.Height) - windowHeight/2
	windowRect := image.Rect(windowX, windowY, windowX+windowWidth, windowY+windowHeight)

	prompt := ""
	if g.State == GameStateMessage {
		prompt = "クリックして続行..."
	}
	DrawMessagePanel(screen, windowRect, g.message, prompt, MplusFont, &g.Config.UI)

	if g.State == GameStateOver {
		btnRect := getResetButtonRect(g)
		DrawButton(screen, btnRect, "リセット", MplusFont,
			g.Config.UI.Colors.Gray, g.Config.UI.Colors.White, g.Config.UI.Colors.White)
	}
}

func getResetButtonRect(g *Game) image.Rectangle {
	btnW, btnH := 100, 40
	btnX := (g.Config.UI.Screen.Width - btnW) / 2
	btnY := g.Config.UI.Screen.Height - btnH - 20
	return image.Rect(btnX, btnY, btnX+btnW, btnY+btnH)
}
