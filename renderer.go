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

// drawInfoPanels is now handled by EbitenUI (ui_info_panels.go)
// func drawInfoPanels(screen *ebiten.Image, g *Game) { ... }

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

// drawMedarotInfo is now handled by EbitenUI (ui_info_panels.go)
// func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY float32, config *Config, debugMode bool) { ... }

// drawActionModal is now handled by EbitenUI (ui_action_modal.go)
// func drawActionModal(screen *ebiten.Image, g *Game) { ... }

// drawMessageWindow is now handled by EbitenUI (ui_message_window.go)
// func drawMessageWindow(screen *ebiten.Image, g *Game) { ... }

// getResetButtonRect is no longer needed as the reset button is an EbitenUI widget.
// func getResetButtonRect(g *Game) image.Rectangle { ... }
