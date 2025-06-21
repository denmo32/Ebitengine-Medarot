package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ★リファクタリング3: マジックナンバーを定数化
const (
	// --- Screen ---
	ScreenWidth  = 960
	ScreenHeight = 540

	// --- Game Logic ---
	PlayersPerTeam = 3

	// --- Battlefield Layout ---
	IconRadius                 = 15
	Team1HomeX                 = 100
	Team2HomeX                 = ScreenWidth - 100
	MedarotIconExecutionOffset = float32(IconRadius + 5)
	Team1ExecutionLineX        = float32(ScreenWidth/2) - MedarotIconExecutionOffset
	Team2ExecutionLineX        = float32(ScreenWidth/2) + MedarotIconExecutionOffset
	BattlefieldHeight          = float32(ScreenHeight * 0.4)
	MedarotVerticalSpacing     = BattlefieldHeight / (PlayersPerTeam + 1)
	HomeMarkerRadius           = float32(IconRadius / 3)
	LineWidth                  = float32(1)

	// --- Info Panel Layout ---
	InfoPanelStartY        = BattlefieldHeight
	InfoPanelHeight        = float32(ScreenHeight * 0.6)
	InfoPanelPadding       = float32(10)
	MedarotInfoBlockWidth  = (float32(ScreenWidth) - InfoPanelPadding*3) / 2
	MedarotInfoBlockHeight = (InfoPanelHeight - (InfoPanelPadding * (float32(PlayersPerTeam) + 1))) / float32(PlayersPerTeam)
	PartHPGaugeWidth       = float32(100)
	PartHPGaugeHeight      = float32(7)
	TextLineHeight         = float32(12)
	PartHPGaugeOffsetX     = float32(80) // 「80」という数字を定数にした
)

// (Color definitions and Game struct are unchanged)
var (
	ColorWhite    = color.White
	ColorBlack    = color.Black
	ColorRed      = color.RGBA{R: 255, G: 100, B: 100, A: 255}
	ColorBlue     = color.RGBA{R: 100, G: 100, B: 255, A: 255}
	ColorGreen    = color.RGBA{R: 100, G: 255, B: 100, A: 255}
	ColorYellow   = color.RGBA{R: 255, G: 255, B: 100, A: 255}
	ColorGray     = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	Team1Color    = ColorBlue
	Team2Color    = ColorRed
	LeaderColor   = ColorYellow
	BrokenColor   = ColorGray
	HPColor       = ColorGreen
	ChargeColor   = ColorYellow
	CooldownColor = color.RGBA{R: 180, G: 180, B: 255, A: 255}
	FontColor     = ColorWhite
	BGColor       = color.NRGBA{R: 0x1a, G: 0x20, B: 0x2c, A: 0xff}
)

type Game struct {
	Medarots  []*Medarot
	GameData  *GameData
	TickCount int
	DebugMode bool
}

// (NewGame and Update are unchanged)
func NewGame(gameData *GameData) *Game {
	medarots := InitializeAllMedarots(gameData)
	if len(medarots) == 0 {
		log.Fatal("No medarots were initialized. Exiting.")
	}
	sort.Slice(medarots, func(i, j int) bool {
		return medarots[i].ID < medarots[j].ID
	})
	return &Game{
		Medarots:  medarots,
		GameData:  gameData,
		TickCount: 0,
		DebugMode: true,
	}
}

func (g *Game) Update() error {
	g.TickCount++
	for _, medarot := range g.Medarots {
		switch medarot.State {
		case StateReadyToSelectAction:
			if medarot.State == StateBroken {
				continue
			}
			availableParts := medarot.GetAvailableAttackParts()
			if len(availableParts) > 0 {
				selectedPart := availableParts[rand.Intn(len(availableParts))]
				if g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) is selecting action. Attempting to use %s (Slot: %s).\n", medarot.Name, medarot.ID, selectedPart.Name, selectedPart.Slot)
				}
				success := medarot.SelectAction(selectedPart.Slot)
				if success && g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) successfully selected %s. Now %s.\n", medarot.Name, medarot.ID, selectedPart.Name, medarot.State)
				} else if !success && g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) failed to select %s.\n", medarot.Name, medarot.ID, selectedPart.Name)
				}
			} else {
				if g.DebugMode && g.TickCount%60 == 0 {
					log.Printf("Game Update: %s (%s) is ReadyToSelectAction but has no available attack parts.\n", medarot.Name, medarot.ID)
				}
			}
		case StateReadyToExecuteAction:
			if medarot.State == StateBroken {
				continue
			}
			if g.DebugMode && g.TickCount%60 == 0 {
				log.Printf("Game Update: %s (%s) is executing action: %s.\n", medarot.Name, medarot.ID, medarot.Parts[medarot.SelectedPartKey].Name)
			}
			success := medarot.ExecuteAction()
			if success && g.DebugMode && g.TickCount%60 == 0 {
				log.Printf("Game Update: %s (%s) successfully executed. Now %s.\n", medarot.Name, medarot.ID, medarot.State)
			} else if !success && g.DebugMode && g.TickCount%60 == 0 {
				log.Printf("Game Update: %s (%s) failed to execute action.\n", medarot.Name, medarot.ID)
			}
		}
		medarot.Update()
	}
	return nil
}

// ★リファクタリング1: Draw関数を責務ごとに分割
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(BGColor)

	g.drawBattlefield(screen)
	g.drawMedarotIcons(screen)
	g.drawInfoPanels(screen)

	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d", g.TickCount), 10, ScreenHeight-15)
	}
}

// drawBattlefieldは静的な背景要素を描画する
func (g *Game) drawBattlefield(screen *ebiten.Image) {
	// Borders
	vector.StrokeRect(screen, 0, 0, float32(ScreenWidth), BattlefieldHeight, LineWidth, FontColor, false)

	// Home Markers
	for i := 0; i < PlayersPerTeam; i++ {
		yPos := MedarotVerticalSpacing * (float32(i) + 1)
		vector.StrokeCircle(screen, float32(Team1HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
		vector.StrokeCircle(screen, float32(Team2HomeX), yPos, HomeMarkerRadius, LineWidth, ColorGray, true)
	}

	// Execution Lines
	vector.StrokeLine(screen, Team1ExecutionLineX, 0, Team1ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
	vector.StrokeLine(screen, Team2ExecutionLineX, 0, Team2ExecutionLineX, BattlefieldHeight, LineWidth, ColorGray, false)
}

// drawMedarotIconsはバトルフィールド上の動くアイコンを描画する
func (g *Game) drawMedarotIcons(screen *ebiten.Image) {
	team1Count := 0
	team2Count := 0

	for _, medarot := range g.Medarots {
		var yIndex int
		if medarot.Team == Team1 {
			yIndex = team1Count
			team1Count++
		} else {
			yIndex = team2Count
			team2Count++
		}
		baseYPos := MedarotVerticalSpacing * float32(yIndex+1)
		currentX := g.calculateIconX(medarot)

		// Clamp X to screen bounds
		if currentX < float32(IconRadius) {
			currentX = float32(IconRadius)
		}
		if currentX > float32(ScreenWidth-IconRadius) {
			currentX = float32(ScreenWidth - IconRadius)
		}

		iconColor := Team1Color
		if medarot.Team == Team2 {
			iconColor = Team2Color
		}
		if medarot.State == StateBroken {
			iconColor = BrokenColor
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, float32(IconRadius), iconColor, true)

		if medarot.IsLeader {
			vector.StrokeCircle(screen, currentX, baseYPos, float32(IconRadius+2), float32(2), LeaderColor, true)
		}
	}
}

// drawInfoPanelsは下部の情報パネルを描画する
func (g *Game) drawInfoPanels(screen *ebiten.Image) {
	team1InfoCount := 0
	team2InfoCount := 0

	for _, medarot := range g.Medarots {
		var panelX, panelY float32
		if medarot.Team == Team1 {
			panelX = InfoPanelPadding
			panelY = InfoPanelStartY + InfoPanelPadding + float32(team1InfoCount)*(MedarotInfoBlockHeight+InfoPanelPadding)
			team1InfoCount++
		} else { // Team2
			panelX = InfoPanelPadding*2 + MedarotInfoBlockWidth
			panelY = InfoPanelStartY + InfoPanelPadding + float32(team2InfoCount)*(MedarotInfoBlockHeight+InfoPanelPadding)
			team2InfoCount++
		}
		drawMedarotInfo(screen, medarot, panelX, panelY)
	}
}

// ★リファクタリング2: 複雑な計算ロジックをヘルパー関数に切り出し
func (g *Game) calculateIconX(medarot *Medarot) float32 {
	progress := 0.0
	if medarot.State == StateActionCharging && medarot.CurrentActionCharge > 0 {
		progress = medarot.Gauge / medarot.CurrentActionCharge
	} else if medarot.State == StateActionCooldown && medarot.CurrentActionCooldown > 0 {
		progress = medarot.Gauge / medarot.CurrentActionCooldown
	}

	homeX, execX := float32(Team1HomeX), Team1ExecutionLineX
	if medarot.Team == Team2 {
		homeX, execX = float32(Team2HomeX), Team2ExecutionLineX
	}

	switch medarot.State {
	case StateActionCharging:
		return homeX + float32(progress)*(execX-homeX)
	case StateReadyToExecuteAction:
		return execX
	case StateActionCooldown:
		return execX - float32(progress)*(execX-homeX)
	case StateIdleCharging, StateReadyToSelectAction, StateBroken:
		fallthrough
	default:
		return homeX
	}
}

// drawMedarotInfoは個々のメダロットの情報を描画する
func drawMedarotInfo(screen *ebiten.Image, medarot *Medarot, startX, startY float32) {
	text.Draw(screen, medarot.Name, MplusFont, int(startX), int(startY)+int(TextLineHeight), FontColor)
	partSlots := []string{"head", "rightArm", "leftArm", "legs"}
	partSlotDisplayNames := map[string]string{
		"head":     "頭部", "rightArm": "右腕", "leftArm": "左腕", "legs": "脚部",
	}

	currentInfoY := startY + TextLineHeight*2
	for _, slotKey := range partSlots {
		if currentInfoY+TextLineHeight > startY+MedarotInfoBlockHeight {
			break
		}
		displayName := partSlotDisplayNames[slotKey]
		var hpText string
		var hpPercentage float64
		if part, ok := medarot.Parts[slotKey]; ok && part != nil {
			hpText = fmt.Sprintf("%s: %d/%d", displayName, part.HP, part.MaxHP)
			if part.MaxHP > 0 {
				hpPercentage = float64(part.HP) / float64(part.MaxHP)
			}
			// Draw HP Gauge
			gaugeX := startX + PartHPGaugeOffsetX
			gaugeY := currentInfoY + TextLineHeight - PartHPGaugeHeight
			vector.DrawFilledRect(screen, gaugeX, gaugeY, PartHPGaugeWidth, PartHPGaugeHeight, color.RGBA{50, 50, 50, 255}, true)
			barFillColor := HPColor
			if part.HP == 0 {
				barFillColor = BrokenColor
			} else if hpPercentage < 0.3 {
				barFillColor = ColorRed
			} else if hpPercentage < 0.6 {
				barFillColor = ColorYellow
			}
			vector.DrawFilledRect(screen, gaugeX, gaugeY, float32(float64(PartHPGaugeWidth)*hpPercentage), PartHPGaugeHeight, barFillColor, true)
		} else {
			hpText = fmt.Sprintf("%s: N/A", displayName)
		}

		text.Draw(screen, hpText, MplusFont, int(startX), int(currentInfoY)+int(TextLineHeight), FontColor)
		currentInfoY += TextLineHeight + 2
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}