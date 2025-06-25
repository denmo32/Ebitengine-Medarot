package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"golang.org/x/image/colornames" // Using for placeholder colors
)

// currentActionModal holds the widget for the currently displayed action modal.
var currentActionModal widget.PreferredSizeLocateableWidget

// createActionModalUI creates and returns the UI for the action selection modal.
func createActionModalUI(game *Game) widget.PreferredSizeLocateableWidget {
	if len(game.actionQueue) == 0 {
		return nil // Should not happen if called correctly
	}
	actingMedarot := game.actionQueue[0]

	// Resources
	buttonImage, _ := loadButtonImage(game) // Reusing the button image loading logic

	// --- Root Container for the Modal ---
	// This container will be centered on the screen.
	modalRootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(widget.Insets{}))),
		// No background for the root, the actual modal panel will have it.
	)

	// --- Panel Container (the visible modal) ---
	panelWidth := 320
	panelHeight := 200 // Adjust as needed based on content (especially button count)

	panelContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{R: 20, G: 20, B: 40, A: 230})), // Darker background
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.Insets{Top: 15, Bottom: 15, Left: 20, Right: 20}),
			widget.RowLayoutOpts.Spacing(10),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(panelWidth, panelHeight), // Give it a minimum size
		),
	)
	modalRootContainer.AddChild(panelContainer)

	// --- Title ---
	titleText := fmt.Sprintf("%s の行動を選択", actingMedarot.Name)
	titleWidget := widget.NewText(
		widget.TextOpts.Text(titleText, MplusFont, game.Config.UI.Colors.White),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		})),
	)
	panelContainer.AddChild(titleWidget)

	// --- Action Buttons ---
	availableParts := actingMedarot.GetAvailableAttackParts()
	if len(availableParts) == 0 {
		// Add a text indicating no actions available, or handle this state appropriately
		noActionsText := widget.NewText(
			widget.TextOpts.Text("使用可能なパーツがありません", MplusFont, game.Config.UI.Colors.Red),
			widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		)
		panelContainer.AddChild(noActionsText)
	}

	for _, part := range availableParts {
		partStr := fmt.Sprintf("%s (%s)", part.PartName, part.Type)
		if part.Category == CategoryShoot {
			if game.playerActionTarget != nil {
				partStr += fmt.Sprintf(" -> %s", game.playerActionTarget.Name)
			} else {
				partStr += " (ターゲットなし)"
			}
		}

		// Need to capture loop variable `part` for the handler
		capturedPart := part
		actionButton := widget.NewButton(
			widget.ButtonOpts.Image(buttonImage),
			widget.ButtonOpts.Text(partStr, MplusFont, &widget.ButtonTextColor{Idle: game.Config.UI.Colors.White}),
			widget.ButtonOpts.TextPadding(widget.Insets{Left: 15, Right: 15, Top: 8, Bottom: 8}),
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch:  true, // Stretch button horizontally
				Position: widget.RowLayoutPositionCenter,
			})),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if game.playerActionTarget == nil && capturedPart.Category == CategoryShoot {
					log.Println("Action Modal: Target is required for shooting part.")
					// Optionally show a UI message to the player here
					return
				}
				if game.playerActionTarget != nil && game.playerActionTarget.State == StateBroken && capturedPart.Category == CategoryShoot {
					log.Println("Action Modal: Target is already broken.")
					// Optionally show a UI message
					// Potentially auto-clear target or re-evaluate state
					return
				}

				actingMedarot.TargetedMedarot = game.playerActionTarget // Set target
				var slotKey PartSlotKey
				switch capturedPart.Type {
				case PartTypeHead:
					slotKey = PartSlotHead
				case PartTypeRArm:
					slotKey = PartSlotRightArm
				case PartTypeLArm:
					slotKey = PartSlotLeftArm
				default:
					log.Printf("Action Modal: Unknown part type %s for action selection", capturedPart.Type)
					return
				}

				if actingMedarot.SelectAction(slotKey) {
					game.actionQueue = game.actionQueue[1:] // Dequeue
				}

				if len(game.actionQueue) == 0 {
					game.State = StatePlaying
					game.playerActionTarget = nil // Clear target after action sequence for player is done
					hideUIActionModal(game)       // Hide modal explicitly
				} else {
					// If there are more actions in queue for the player,
					// refresh the modal for the next Medarot.
					// This might involve hiding and showing, or updating content.
					// For simplicity, current logic in game.Update will handle state transition.
					// If the next in queue is still player controlled, game.Update will set
					// StatePlayerActionSelect again, and showUIActionModal will be called.
					hideUIActionModal(game) // Hide current one
					// game.Update will then call tryEnterActionSelect, which sets the state,
					// then the state change detection in Update calls showUIActionModal.
				}
			}),
		)
		panelContainer.AddChild(actionButton)
	}
	// Add an overlay container to dim the background
	overlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{0, 0, 0, 180})), // Semi-transparent black
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		// This overlay should be behind the modal panel but in front of other game elements.
		// We achieve this by adding the modalRootContainer (which contains the panel) to this overlay.
	)
	overlay.AddChild(modalRootContainer)

	return overlay // Return the overlay which contains the centered modal
}

// showUIActionModal displays the action selection modal.
func showUIActionModal(g *Game) {
	if g.State != StatePlayerActionSelect || len(g.actionQueue) == 0 {
		return // Only show if in correct state and an action is pending
	}

	if currentActionModal != nil {
		g.ui.Container.RemoveChild(currentActionModal)
		currentActionModal = nil
	}

	modal := createActionModalUI(g)
	if modal != nil {
		currentActionModal = modal
		g.ui.Container.AddChild(currentActionModal)
	}
}

// hideUIActionModal removes the action selection modal from the UI.
func hideUIActionModal(g *Game) {
	if currentActionModal != nil {
		g.ui.Container.RemoveChild(currentActionModal)
		currentActionModal = nil
	}
}

// loadButtonImage is a placeholder, assumed to be in ui_message_window.go or similar shared file.
// If not, it needs to be defined or imported.
// For now, let's copy a simplified version here if it's not automatically available.
/*
func loadButtonImage(game *Game) (*widget.ButtonImage, error) {
	idle := image.NewNineSliceColor(game.Config.UI.Colors.Gray)
	hover := image.NewNineSliceColor(colornames.Darkgray)
	pressed := image.NewNineSliceColor(colornames.Gray)

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}, nil
}
*/
