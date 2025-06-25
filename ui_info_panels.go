package main

import (
	"fmt"
	"image/color" // Required for color.Color

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	// "golang.org/x/image/colornames" // For placeholder colors if needed
)

// infoPanelUI holds references to widgets within a single Medarot's info panel
// that need to be updated dynamically (e.g., HP text, HP bar).
type infoPanelUI struct {
	rootContainer *widget.Container
	nameText      *widget.Text
	stateText     *widget.Text // Optional, for debug
	partSlots     map[PartSlotKey]*infoPanelPartUI
}

type infoPanelPartUI struct {
	hpText       *widget.Text
	partNameText *widget.Text
	hpBarFill    *widget.Container // The container whose background color and width represent HP
	hpBarMax     float32           // Store max width for scaling
}

// map to store all created info panel UIs, keyed by Medarot ID
var medarotInfoPanelUIs map[string]*infoPanelUI

// createInfoPanelsUI creates the container that holds all Medarot info panels.
// This should be called once during game initialization.
func createInfoPanelsUI(game *Game) *widget.Container {
	medarotInfoPanelUIs = make(map[string]*infoPanelUI)

	// Main container to hold all info panels (e.g., one column for each team)
	// Using AnchorLayout to position two main columns (one for each team)
	rootLayout := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(widget.Insets{
			Top:    int(game.Config.UI.InfoPanel.StartY),
			Left:   int(game.Config.UI.InfoPanel.Padding),
			Right:  int(game.Config.UI.InfoPanel.Padding),
			Bottom: int(game.Config.UI.InfoPanel.Padding),
		}))),
		// This container itself is transparent; the individual panels have backgrounds.
	)

	// Team 1 column container
	team1PanelColumn := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(int(game.Config.UI.InfoPanel.Padding)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			StretchVertical:    true, // Stretch to fill height if a max height is set on root or by content
		})),
	)
	rootLayout.AddChild(team1PanelColumn)

	// Team 2 column container
	team2PanelColumn := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(int(game.Config.UI.InfoPanel.Padding)),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionEnd,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			StretchVertical:    true,
		})),
	)
	rootLayout.AddChild(team2PanelColumn)

	// Create and add individual Medarot panels
	for _, medarot := range game.sortedMedarotsForDraw {
		panel := createSingleMedarotInfoPanel(game, medarot)
		medarotInfoPanelUIs[medarot.ID] = panel // Store for updates

		if medarot.Team == Team1 {
			team1PanelColumn.AddChild(panel.rootContainer)
		} else {
			team2PanelColumn.AddChild(panel.rootContainer)
		}
	}
	return rootLayout
}

// createSingleMedarotInfoPanel creates the UI for a single Medarot's information.
func createSingleMedarotInfoPanel(game *Game, medarot *Medarot) *infoPanelUI {
	uiConfig := game.Config.UI
	panelWidth := int(uiConfig.InfoPanel.BlockWidth)
	panelHeight := int(uiConfig.InfoPanel.BlockHeight)

	panelContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{50, 50, 70, 200})), // Panel background
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(widget.Insets{Top: 5, Bottom: 5, Left: 10, Right: 10}),
			widget.RowLayoutOpts.Spacing(3), // Reduced spacing
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(panelWidth, panelHeight)),
	)

	panelUI := &infoPanelUI{
		rootContainer: panelContainer,
		partSlots:     make(map[PartSlotKey]*infoPanelPartUI),
	}

	// Medarot Name (and State if debug)
	nameRowContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(5),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	panelContainer.AddChild(nameRowContainer)

	panelUI.nameText = widget.NewText(
		widget.TextOpts.Text(medarot.Name, MplusFont, uiConfig.Colors.White),
	)
	nameRowContainer.AddChild(panelUI.nameText)

	if game.DebugMode {
		panelUI.stateText = widget.NewText(
			widget.TextOpts.Text(fmt.Sprintf("St: %s", medarot.State), MplusFont, uiConfig.Colors.Yellow),
		)
		nameRowContainer.AddChild(panelUI.stateText)
	}

	// Parts Info
	partSlots := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	partSlotDisplayNames := map[PartSlotKey]string{
		PartSlotHead:     "頭部",
		PartSlotRightArm: "右腕",
		PartSlotLeftArm:  "左腕",
		PartSlotLegs:     "脚部",
	}

	for _, slotKey := range partSlots {
		part, exists := medarot.Parts[slotKey]
		displayName := partSlotDisplayNames[slotKey]
		partUI := &infoPanelPartUI{}

		// Container for each part's info line (HP text, HP bar, Part Name)
		partRowContainer := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(5), // Spacing between elements in the row
			)),
			widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})), // Stretch this row
		)
		panelContainer.AddChild(partRowContainer)

		// HP Text (e.g., "頭部: 50/50")
		hpTextStr := fmt.Sprintf("%s: N/A", displayName)
		if exists && part != nil {
			hpTextStr = fmt.Sprintf("%s: %d/%d", displayName, part.Armor, part.MaxArmor)
		}
		partUI.hpText = widget.NewText(
			widget.TextOpts.Text(hpTextStr, MplusFont, uiConfig.Colors.White),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart, // Align to start
			})),
		)
		partRowContainer.AddChild(partUI.hpText)

		// HP Bar Container (Background and Fill)
		hpBarContainer := widget.NewContainer(
			widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(color.NRGBA{50, 50, 50, 255})), // Dark background for the bar
			widget.ContainerOpts.Layout(widget.NewAnchorLayout()),                                      // Anchor layout for the fill
			widget.ContainerOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Position: widget.RowLayoutPositionStart, // Or center, depending on desired alignment with text
				}),
				widget.WidgetOpts.MinSize(int(uiConfig.InfoPanel.PartHPGaugeWidth), int(uiConfig.InfoPanel.PartHPGaugeHeight)),
			),
		)
		partRowContainer.AddChild(hpBarContainer)

		partUI.hpBarFill = widget.NewContainer(
			widget.ContainerOpts.BackgroundImage(image.NewNineSliceColor(uiConfig.Colors.HP)), // Default HP color
			widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchVertical:    true,
			})),
			// Width is set dynamically in updateInfoPanelUI
		)
		hpBarContainer.AddChild(partUI.hpBarFill)
		partUI.hpBarMax = uiConfig.InfoPanel.PartHPGaugeWidth


		// Part Name Text
		partNameStr := ""
		if exists && part != nil {
			partNameStr = part.PartName
		}
		partUI.partNameText = widget.NewText(
			widget.TextOpts.Text(partNameStr, MplusFont, uiConfig.Colors.White),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart, // Align to start
				Stretch: true, // Allow it to take remaining space
			})),
		)
		partRowContainer.AddChild(partUI.partNameText)
		panelUI.partSlots[slotKey] = partUI
	}
	return panelUI
}

// updateInfoPanelUI updates the content of a single Medarot's info panel.
// This should be called when a Medarot's state or HP changes.
func updateInfoPanelUI(game *Game, medarot *Medarot) {
	panelComponents, exists := medarotInfoPanelUIs[medarot.ID]
	if !exists {
		return // Should not happen if initialized correctly
	}

	uiColors := game.Config.UI.Colors

	// Update Name Color (and State if debug)
	nameColor := uiColors.White
	if medarot.State == StateBroken {
		nameColor = uiColors.Broken
	}
	panelComponents.nameText.Label = medarot.Name // In case name can change (unlikely here)
	panelComponents.nameText.Color = nameColor

	if game.DebugMode && panelComponents.stateText != nil {
		panelComponents.stateText.Label = fmt.Sprintf("St: %s", medarot.State)
	} else if !game.DebugMode && panelComponents.stateText != nil {
		// Hide state text if debug mode turned off after initialization
		panelComponents.stateText.Label = "" // Or set visibility if EbitenUI supports it easily
	}


	// Update Parts Info
	partSlotsKeys := []PartSlotKey{PartSlotHead, PartSlotRightArm, PartSlotLeftArm, PartSlotLegs}
	partSlotDisplayNames := map[PartSlotKey]string{PartSlotHead: "頭部", PartSlotRightArm: "右腕", PartSlotLeftArm: "左腕", PartSlotLegs: "脚部"}

	for _, slotKey := range partSlotsKeys {
		partComponents, pcExists := panelComponents.partSlots[slotKey]
		if !pcExists {
			continue
		}

		part, partExistsInMedarot := medarot.Parts[slotKey]
		displayName := partSlotDisplayNames[slotKey]
		currentArmor := 0
		maxArmor := 0
		partName := "N/A"
		partIsBroken := true // Assume broken if not found or explicitly broken
		textColor := uiColors.Broken
		hpBarColor := uiColors.Broken
		hpPercentage := 0.0

		if partExistsInMedarot && part != nil {
			currentArmor = part.Armor
			maxArmor = part.MaxArmor
			partName = part.PartName
			partIsBroken = part.IsBroken

			if part.IsBroken {
				currentArmor = 0 // Show 0 HP if broken
			}

			if !part.IsBroken {
				textColor = uiColors.White
				hpBarColor = uiColors.HP
				if maxArmor > 0 {
					hpPercentage = float64(currentArmor) / float64(maxArmor)
				}
				if hpPercentage < 0.3 && hpPercentage > 0 { // Don't make it red if 0 and broken
					hpBarColor = uiColors.Red
				}
			}
		}

		partComponents.hpText.Label = fmt.Sprintf("%s: %d/%d", displayName, currentArmor, maxArmor)
		partComponents.hpText.Color = textColor
		partComponents.partNameText.Label = partName
		partComponents.partNameText.Color = textColor

		// Update HP Bar
		fillWidth := int(partComponents.hpBarMax * float32(hpPercentage))
		if fillWidth < 0 {
			fillWidth = 0
		}
		partComponents.hpBarFill.GetWidget().Rect.DxVal = fillWidth // This is how preferred size is set, might need SetLocation/MinSize
		partComponents.hpBarFill.GetWidget().Image = image.NewNineSliceColor(hpBarColor) // Update color

		// Request a redraw/re-layout for the bar's parent or the fill itself if properties changed.
		// EbitenUI usually handles this automatically if a widget's properties that affect layout/rendering are changed.
		// For direct manipulation like DxVal, we might need to tell its parent to re-layout.
		// However, changing the image (background color) should trigger a redraw.
		// Setting preferred size is usually done via WidgetOpts.MinSize during construction or LayoutData.
		// For dynamic width of HP bar fill:
		// The fill container is inside an AnchorLayout. Its width can be set by LayoutData if anchored to both sides,
		// or more directly, we might need to adjust its PreferredSize.
		// A common way for dynamic bars is to have a fixed-size background and a fill element whose width is changed.

		// The most reliable way to set size for a widget like this in EbitenUI is to set its MinWidth/MinHeight
		// and then ensure its parent's layout respects these minimums.
		// The hpBarFill widget is a child of hpBarContainer (AnchorLayout).
		// We gave hpBarFill AnchorLayoutData with StretchVertical = true.
		// To control its width, we will set its MinWidth.
		// Its actual rendered width will be this MinWidth, as it's anchored to the start and not stretched horizontally.

		w := partComponents.hpBarFill.GetWidget()
		w.MinWidth = fillWidth
		// We don't need to set MinHeight here if StretchVertical is true and parent has a defined height,
		// but it's good practice if the height is fixed.
		// w.MinHeight = int(game.Config.UI.InfoPanel.PartHPGaugeHeight)

		// Changing widget properties like MinWidth should ideally trigger a relayout if the layout engine is watching.
		// If not, panelComponents.rootContainer.RequestRelayout() might be needed, but try without first.
	}
	// panelComponents.rootContainer.RequestRelayout() // Request relayout for the whole panel. Try to avoid if possible.
}

// updateAllInfoPanels iterates through all medarots and updates their UI panels.
// Call this in Game.Update() or when game state changes significantly.
func updateAllInfoPanels(game *Game) {
	if medarotInfoPanelUIs == nil {
		// Not initialized yet, or called too early.
		// This can happen if Game.Update runs before Game.Draw has a chance to build the UI first.
		// Consider initializing info panels in NewGame and adding to UI tree there.
		return
	}
	for _, medarot := range game.Medarots { // Iterate over actual medarots, not just sorted list for drawing
		updateInfoPanelUI(game, medarot)
	}
}
