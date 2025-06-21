package main

import (
	"fmt" // Correctly imported once
	"image/color"
	"log"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	ScreenWidth  = 800
	ScreenHeight = 600

	// GUI constants
	IconRadius        = 15
	IconDiameter      = IconRadius * 2
	Team1HomeX        = 100
	Team2HomeX        = ScreenWidth - 100
	ExecutionLineX    = ScreenWidth / 2
	MedarotVerticalSpacing = ScreenHeight / (PlayersPerTeam + 1)

	ChargeBarHeight = 5
	HPBarHeight     = 5
	BarWidth        = IconDiameter * 2
)

var (
	ColorWhite   = color.White
	ColorBlack   = color.Black
	ColorRed     = color.RGBA{R: 255, G: 100, B: 100, A: 255}
	ColorBlue    = color.RGBA{R: 100, G: 100, B: 255, A: 255}
	ColorGreen   = color.RGBA{R: 100, G: 255, B: 100, A: 255}
	ColorYellow  = color.RGBA{R: 255, G: 255, B: 100, A: 255}
	ColorGray    = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	Team1Color   = ColorBlue
	Team2Color   = ColorRed
	LeaderColor  = ColorYellow
	BrokenColor  = ColorGray
	HPColor      = ColorGreen
	ChargeColor  = ColorYellow
	CooldownColor= color.RGBA{R: 180, G: 180, B: 255, A: 255}
	FontColor    = ColorWhite
	BGColor      = color.NRGBA{R: 0x1a, G: 0x20, B: 0x2c, A: 0xff}
)

type Game struct {
	Medarots   []*Medarot
	GameData   *GameData
	TickCount  int
	DebugMode  bool
}

func NewGame(gameData *GameData) *Game {
	medarots := InitializeAllMedarots(gameData)
	if len(medarots) == 0 {
		log.Fatal("No medarots were initialized. Exiting.")
	}

	sort.Slice(medarots, func(i, j int) bool {
		return medarots[i].ID < medarots[j].ID
	})

	return &Game{
		Medarots:   medarots,
		GameData:   gameData,
		TickCount:  0,
		DebugMode:  true,
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

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(BGColor)

	team1Count := 0
	team2Count := 0

	for _, medarot := range g.Medarots {
		var baseYPos float32
		if medarot.Team == Team1 {
			baseYPos = float32(MedarotVerticalSpacing * (team1Count + 1))
			team1Count++
		} else {
			baseYPos = float32(MedarotVerticalSpacing * (team2Count + 1))
			team2Count++
		}

		var currentX float32
		progress := 0.0
		if medarot.State == StateActionCharging && medarot.CurrentActionCharge > 0 {
			progress = medarot.Gauge / medarot.CurrentActionCharge
		} else if medarot.State == StateActionCooldown && medarot.CurrentActionCooldown > 0 {
			progress = medarot.Gauge / medarot.CurrentActionCooldown
		}

		switch medarot.State {
		case StateIdleCharging, StateReadyToSelectAction:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		case StateActionCharging:
			if medarot.Team == Team1 {
				currentX = float32(Team1HomeX + progress*(ExecutionLineX-Team1HomeX))
			} else {
				currentX = float32(Team2HomeX - progress*(Team2HomeX-ExecutionLineX))
			}
		case StateReadyToExecuteAction:
			currentX = float32(ExecutionLineX)
		case StateActionCooldown:
			if medarot.Team == Team1 {
				currentX = float32(ExecutionLineX - progress*(ExecutionLineX-Team1HomeX))
			} else {
				currentX = float32(ExecutionLineX + progress*(Team2HomeX-ExecutionLineX))
			}
		case StateBroken:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		default:
			currentX = float32(Team1HomeX)
			if medarot.Team == Team2 {
				currentX = float32(Team2HomeX)
			}
		}

		if currentX < IconRadius { currentX = IconRadius }
		if currentX > ScreenWidth - IconRadius { currentX = ScreenWidth - IconRadius}

		iconColor := Team1Color
		if medarot.Team == Team2 {
			iconColor = Team2Color
		}
		if medarot.State == StateBroken {
			iconColor = BrokenColor
		}

		// Draw Medarot Icon
		if medarot.IsLeader {
			vector.DrawFilledCircle(screen, currentX, baseYPos, float32(IconRadius+2), LeaderColor, true) // Border
		}
		vector.DrawFilledCircle(screen, currentX, baseYPos, float32(IconRadius), iconColor, true)   // Actual icon

		nameStr := medarot.Name
		var hpStr string
		if head, ok := medarot.Parts["head"]; ok && head != nil {
		    hpStr = fmt.Sprintf("HP: %d/%d", head.HP, head.MaxHP)
		} else {
		    hpStr = "HP: N/A"
		}

		nameTextYOffset := baseYPos - float32(IconRadius) - 15
		nameTextX := int(currentX - float32(len(nameStr)*basicfont.Face7x13.Advance)/2)
		text.Draw(screen, nameStr, basicfont.Face7x13, nameTextX, int(nameTextYOffset), FontColor)

		hpTextYOffset := baseYPos - float32(IconRadius) - 5
		hpTextX := int(currentX - float32(len(hpStr)*basicfont.Face7x13.Advance)/2)
		text.Draw(screen, hpStr, basicfont.Face7x13, hpTextX, int(hpTextYOffset), FontColor) // hpStr is used here

		hpBarY := baseYPos + float32(IconRadius) + 3
		hpPercentage := 0.0
		if headPart := medarot.GetPart("head"); headPart != nil && headPart.MaxHP > 0 {
			hpPercentage = float64(headPart.HP) / float64(headPart.MaxHP)
		}
		vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), hpBarY, float32(BarWidth), ChargeBarHeight, color.RGBA{50,50,50,255}, true)
		vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), hpBarY, float32(BarWidth*hpPercentage), ChargeBarHeight, HPColor, true)

		chargeBarY := hpBarY + HPBarHeight + 2
		barColor := ChargeColor
		currentGaugeVal := medarot.Gauge
		maxGaugeForBar := medarot.MaxGauge

		switch medarot.State {
		case StateActionCharging:
			maxGaugeForBar = medarot.CurrentActionCharge
			barColor = ChargeColor
		case StateActionCooldown:
			maxGaugeForBar = medarot.CurrentActionCooldown
			barColor = CooldownColor
		case StateIdleCharging, StateReadyToSelectAction:
			barColor = ColorGreen
		case StateBroken, StateReadyToExecuteAction:
			currentGaugeVal = 0
			maxGaugeForBar = 0
		}

		gaugePercentage := 0.0
		if maxGaugeForBar > 0 {
			gaugePercentage = currentGaugeVal / maxGaugeForBar
			if gaugePercentage > 1.0 { gaugePercentage = 1.0 }
		} else if medarot.State == StateReadyToExecuteAction {
			gaugePercentage = 1.0
			barColor = ChargeColor
		}

		if medarot.State != StateBroken {
			vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), chargeBarY, float32(BarWidth), ChargeBarHeight, color.RGBA{50,50,50,255}, true)
			vector.DrawFilledRect(screen, currentX-float32(BarWidth/2), chargeBarY, float32(BarWidth*gaugePercentage), ChargeBarHeight, barColor, true)
		}

		if g.DebugMode {
			stateStr := string(medarot.State)
			if medarot.State == StateActionCharging || medarot.State == StateActionCooldown {
				if part := medarot.GetPart(medarot.SelectedPartKey); part != nil {
					stateStr += fmt.Sprintf(" (%s)", part.Name)
				}
			}
			stateTextYOffset := baseYPos + float32(IconRadius) + HPBarHeight + ChargeBarHeight + 5
			stateTextX := int(currentX - float32(len(stateStr)*basicfont.Face7x13.Advance)/2)
			text.Draw(screen, stateStr, basicfont.Face7x13, stateTextX, int(stateTextYOffset), FontColor)
		}
	}

	if g.DebugMode {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tick: %d", g.TickCount), 10, ScreenHeight-20)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}
