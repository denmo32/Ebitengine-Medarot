package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

// DrawWindow は、指定された位置とサイズで背景と枠線を持つウィンドウを描画します。
func DrawWindow(screen *ebiten.Image, rect image.Rectangle, bgColor, borderColor color.Color) {
	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), bgColor, true)
	vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), 2, borderColor, false)
}

// DrawButton は、テキスト付きのボタンを描画します。テキストはボタンの中央に配置されます。
func DrawButton(screen *ebiten.Image, rect image.Rectangle, label string, face font.Face, bgColor, textColor, borderColor color.Color) {
	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), bgColor, true)
	vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), 1, borderColor, true)

	if face != nil && label != "" {
		bounds, _ := font.BoundString(face, label)
		textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
		// Ebitenのテキスト描画はベースラインを基準にするため、中央揃えには高さの計算が必要です。
		ascent := face.Metrics().Ascent.Ceil()
		textX := rect.Min.X + (rect.Dx()-textWidth)/2
		textY := rect.Min.Y + (rect.Dy()+ascent)/2
		text.Draw(screen, label, face, textX, textY, textColor)
	}
}

// DrawMessagePanel は、メッセージとオプションのプロンプトテキストを持つパネルを描画します。
func DrawMessagePanel(screen *ebiten.Image, rect image.Rectangle, message, prompt string, face font.Face, uiConfig *UIConfig) {
	DrawWindow(screen, rect, color.NRGBA{0, 0, 0, 200}, uiConfig.Colors.White)

	if face == nil {
		return
	}
	// メインメッセージ (中央揃え)
	if message != "" {
		bounds, _ := font.BoundString(face, message)
		msgW := (bounds.Max.X - bounds.Min.X).Ceil()
		ascent := face.Metrics().Ascent.Ceil()
		msgX := rect.Min.X + (rect.Dx()-msgW)/2
		msgY := rect.Min.Y + (rect.Dy()+ascent)/2
		text.Draw(screen, message, face, msgX, msgY, uiConfig.Colors.White)
	}
	// プロンプトメッセージ (右下)
	if prompt != "" {
		bounds, _ := font.BoundString(face, prompt)
		promptW := (bounds.Max.X - bounds.Min.X).Ceil()
		ascent := face.Metrics().Ascent.Ceil()
		promptX := rect.Max.X - promptW - 20
		promptY := rect.Max.Y - 20 - ascent + face.Metrics().Height.Ceil()
		text.Draw(screen, prompt, face, promptX, promptY, uiConfig.Colors.White)
	}
}

// drawMedarotInfoPanel は個々のメダロットの情報パネルを描画します。
// render_system.goから移動し、このファイルに集約しました。
func drawMedarotInfoPanel(screen *ebiten.Image, identity *IdentityComponent, status *StatusComponent, parts *PartsComponent, startX, startY float32, config *Config, debugMode bool) {
	if MplusFont == nil {
		return
	}

	// 名前と状態
	nameColor := config.UI.Colors.White
	if status.IsBroken() {
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

	// 各パーツの情報
	for _, slotKey := range partSlots {
		part, exists := parts.Parts[slotKey]
		if !exists || part == nil {
			continue
		}

		hpText := fmt.Sprintf("%s:%d/%d", partSlotDisplayNames[slotKey], part.Armor, part.MaxArmor)
		textColor := config.UI.Colors.White
		if part.IsBroken {
			textColor = config.UI.Colors.Broken
		}

		// HPバー
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

		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY), textColor)

		// パーツ名
		partNameX := startX + config.UI.InfoPanel.PartHPGaugeOffsetX + config.UI.InfoPanel.PartHPGaugeWidth + 5
		text.Draw(screen, part.PartName, MplusFont, int(partNameX), int(currentInfoY), textColor)

		currentInfoY += config.UI.InfoPanel.TextLineHeight + 4
	}
}
