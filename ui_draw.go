package main

import (
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

// DrawButton は、テキスト付きのボタンを描画します。
// テキストはボタンの中央に配置されます。
func DrawButton(screen *ebiten.Image, rect image.Rectangle, label string, face font.Face, bgColor, textColor, borderColor color.Color) {
	// ボタン背景と枠線
	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), bgColor, true)
	vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), 1, borderColor, true)

	// ボタンテキスト（中央揃え）
	if face != nil && label != "" {
		bounds, _ := font.BoundString(face, label)
		textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
		textHeight := (bounds.Max.Y - bounds.Min.Y).Ceil()

		textX := rect.Min.X + (rect.Dx()-textWidth)/2
		textY := rect.Min.Y + (rect.Dy()+textHeight)/2

		text.Draw(screen, label, face, textX, textY, textColor)
	}
}

// DrawMessagePanel は、メッセージとオプションのプロンプトテキストを持つパネルを描画します。
func DrawMessagePanel(screen *ebiten.Image, rect image.Rectangle, message, prompt string, face font.Face, config *UIConfig) {
	// ウィンドウ背景
	DrawWindow(screen, rect, color.NRGBA{0, 0, 0, 200}, config.Colors.Orange)

	if face == nil {
		return
	}

	// メインメッセージ (中央揃え)
	if message != "" {
		bounds, _ := font.BoundString(face, message)
		msgW := (bounds.Max.X - bounds.Min.X).Ceil()
		msgX := rect.Min.X + (rect.Dx()-msgW)/2
		msgY := rect.Min.Y + rect.Dy()/2 // 簡略化のため垂直中央
		text.Draw(screen, message, face, msgX, msgY, config.Colors.White)
	}

	// プロンプトメッセージ (右下)
	if prompt != "" {
		bounds, _ := font.BoundString(face, prompt)
		promptW := (bounds.Max.X - bounds.Min.X).Ceil()
		promptX := rect.Max.X - promptW - 20
		promptY := rect.Max.Y - 20
		text.Draw(screen, prompt, face, promptX, promptY, config.Colors.White)
	}
}
