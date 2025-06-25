package main

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames" // Using for placeholder colors
)

// createMessageWindowUI creates and returns the UI for the message window.
// It's designed to be added to the game's root UI container when a message needs to be displayed.
func createMessageWindowUI(game *Game) widget.PreferredSizeLocateableWidget {
	// Load resources (font, images for button)
	// For simplicity, using NineSliceColor for backgrounds initially.
	// In a real scenario, you might load these from image files.
	buttonImage, _ := loadButtonImage(game) // Assuming you have a helper for this

	// --- Main Container for the Message Window ---
	windowWidth := int(float32(game.Config.UI.Screen.Width) * 0.7)
	windowHeight := int(float32(game.Config.UI.Screen.Height) * 0.25)

	messageWindowContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 200})), // Semi-transparent black
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.Insets{
				Top:    20,
				Bottom: 20,
				Left:   20,
				Right:  20,
			}),
			widget.RowLayoutOpts.Spacing(10),
		)),
		// Setting a minimum size for the container.
		// The actual size will be determined by its content or LayoutData if part of a larger layout.
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(windowWidth, windowHeight)),
	)

	// --- Message Text ---
	messageTextWidget := widget.NewText(
		widget.TextOpts.Text(game.message, MplusFont, game.Config.UI.Colors.White),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter), // Center align text
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true, // Stretch horizontally
		})),
	)
	messageWindowContainer.AddChild(messageTextWidget)

	// --- Prompt Text (if any) ---
	promptText := ""
	if game.State == GameStateMessage {
		promptText = "クリックして続行..."
	}

	if promptText != "" {
		promptTextWidget := widget.NewText(
			widget.TextOpts.Text(promptText, MplusFont, game.Config.UI.Colors.White),
			widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionCenter), // Align to the right
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			})),
		)
		messageWindowContainer.AddChild(promptTextWidget)
	}

	// --- Reset Button (if game over) ---
	if game.State == GameStateOver {
		resetButton := widget.NewButton(
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter, // Center button in the row
			})),
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text("リセット", MplusFont, &widget.ButtonTextColor{
				Idle: game.Config.UI.Colors.White,
			}),
			widget.ButtonOpts.TextPadding(widget.Insets{Left: 20, Right: 20, Top: 5, Bottom: 5}),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				game.restartRequested = true
			}),
		)
		messageWindowContainer.AddChild(resetButton)
	}

	// To position this window in the center of the screen, or a specific location,
	// you would typically add it to a root container that uses an AnchorLayout or similar.
	// For now, we return the container itself. The calling code will handle positioning.
	// We'll wrap it in another container to control its overall placement if needed.
	outerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(widget.Insets{
			// Example: Center it. The exact values might need adjustment.
			// This assumes the message window has a defined preferred size.
		}))),
		widget.ContainerOpts.WidgetOpts(
			// This widget will be added to the root UI.
			// We want it to be centered.
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				// StretchHorizontal: true, // If you want it to span screen width
				// StretchVertical:   true, // If you want it to span screen height
			}),
			// Ensure this outer container itself doesn't have a background unless intended.
			// widget.WidgetOpts.GetWidget().Visibility = widget.Visibility_None, // This is not the right way to make it transparent
		),
	)
	outerContainer.AddChild(messageWindowContainer)

	return outerContainer
}

// loadButtonImage is a placeholder for loading button graphics.
// EbitenUI buttons are typically styled with NineSlice images.
func loadButtonImage(game *Game) (*widget.ButtonImage, error) {
	idle := image.NewNineSliceColor(game.Config.UI.Colors.Gray)
	hover := image.NewNineSliceColor(colornames.Darkgray) // Placeholder
	pressed := image.NewNineSliceColor(colornames.Gray)   // Placeholder

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}, nil
}

// Temporary struct to hold the current message window widget, so we can remove it.
var currentMessageWindow widget.PreferredSizeLocateableWidget

// showUIMessage displays the message window.
func showUIMessage(g *Game) {
	// Remove previous message window if any
	if currentMessageWindow != nil {
		g.ui.Container.RemoveChild(currentMessageWindow)
		currentMessageWindow = nil
	}

	if g.State == GameStateMessage || g.State == GameStateOver {
		msgWindow := createMessageWindowUI(g)
		currentMessageWindow = msgWindow
		g.ui.Container.AddChild(currentMessageWindow)
	}
}

// hideUIMessage removes the message window.
func hideUIMessage(g *Game) {
	if currentMessageWindow != nil {
		g.ui.Container.RemoveChild(currentMessageWindow)
		currentMessageWindow = nil
	}
}
