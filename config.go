package main

import "image/color"

// ★★★ [修正] BalanceConfigを独立した型として定義 ★★★
type BalanceConfig struct {
	Time struct {
		PropulsionEffectRate float64
		OverallTimeDivisor   float64
	}
	Hit struct {
		BaseChance         int
		TraitAimBonus      int
		TraitStrikeBonus   int
		TraitBerserkDebuff int
	}
	Damage struct {
		CriticalMultiplier float64
		MedalSkillFactor   int
	}
}

// UIConfig はUIのレイアウトや色に関する設定を管理します。
type UIConfig struct {
	Screen struct {
		Width  int
		Height int
	}
	Battlefield struct {
		Height                 float32
		Team1HomeX             float32
		Team2HomeX             float32
		Team1ExecutionLineX    float32
		Team2ExecutionLineX    float32
		IconRadius             float32
		HomeMarkerRadius       float32
		LineWidth              float32
		MedarotVerticalSpacing float32
	}
	InfoPanel struct {
		StartY             float32
		Padding            float32
		BlockWidth         float32
		BlockHeight        float32
		PartHPGaugeWidth   float32
		PartHPGaugeHeight  float32
		TextLineHeight     float32
		PartHPGaugeOffsetX float32
	}
	ActionModal struct {
		ButtonWidth   float32
		ButtonHeight  float32
		ButtonSpacing float32
	}
	Colors struct {
		White      color.Color
		Red        color.Color
		Blue       color.Color
		Yellow     color.Color
		Gray       color.Color
		Orange     color.Color
		Team1      color.Color
		Team2      color.Color
		Leader     color.Color
		Broken     color.Color
		HP         color.Color
		Background color.Color
	}
}

// Config はゲーム全体のすべての設定を保持します。
type Config struct {
	Balance BalanceConfig
	UI      UIConfig
}

// LoadConfig はデフォルトの全設定を生成して返します。
func LoadConfig() Config {
	// 先にUIの基本定数を計算
	screenWidth := 960
	screenHeight := 540
	playersPerTeam := 3
	battlefieldHeight := float32(screenHeight) * 0.4
	infoPanelHeight := float32(screenHeight) * 0.6
	infoPanelPadding := float32(10)
	iconRadius := float32(15)

	return Config{
		Balance: BalanceConfig{
			Time: struct {
				PropulsionEffectRate float64
				OverallTimeDivisor   float64
			}{
				PropulsionEffectRate: 0.5,
				OverallTimeDivisor:   50.0,
			},
			Hit: struct {
				BaseChance         int
				TraitAimBonus      int
				TraitStrikeBonus   int
				TraitBerserkDebuff int
			}{
				BaseChance:         75,
				TraitAimBonus:      50,
				TraitStrikeBonus:   20,
				TraitBerserkDebuff: -10,
			},
			Damage: struct {
				CriticalMultiplier float64
				MedalSkillFactor   int
			}{
				CriticalMultiplier: 1.5,
				MedalSkillFactor:   2,
			},
		},
		UI: UIConfig{
			Screen: struct {
				Width  int
				Height int
			}{
				Width:  screenWidth,
				Height: screenHeight,
			},
			Battlefield: struct {
				Height                 float32
				Team1HomeX             float32
				Team2HomeX             float32
				Team1ExecutionLineX    float32
				Team2ExecutionLineX    float32
				IconRadius             float32
				HomeMarkerRadius       float32
				LineWidth              float32
				MedarotVerticalSpacing float32
			}{
				Height:                 battlefieldHeight,
				Team1HomeX:             100,
				Team2HomeX:             float32(screenWidth - 100),
				Team1ExecutionLineX:    float32(screenWidth/2) - (iconRadius + 5),
				Team2ExecutionLineX:    float32(screenWidth/2) + (iconRadius + 5),
				IconRadius:             iconRadius,
				HomeMarkerRadius:       iconRadius / 3,
				LineWidth:              1,
				MedarotVerticalSpacing: battlefieldHeight / (float32(playersPerTeam) + 1),
			},
			InfoPanel: struct {
				StartY             float32
				Padding            float32
				BlockWidth         float32
				BlockHeight        float32
				PartHPGaugeWidth   float32
				PartHPGaugeHeight  float32
				TextLineHeight     float32
				PartHPGaugeOffsetX float32
			}{
				StartY:             battlefieldHeight,
				Padding:            infoPanelPadding,
				BlockWidth:         (float32(screenWidth) - infoPanelPadding*3) / 2,
				BlockHeight:        (infoPanelHeight - (infoPanelPadding * (float32(playersPerTeam) + 1))) / float32(playersPerTeam),
				PartHPGaugeWidth:   100,
				PartHPGaugeHeight:  7,
				TextLineHeight:     12,
				PartHPGaugeOffsetX: 80,
			},
			ActionModal: struct {
				ButtonWidth   float32
				ButtonHeight  float32
				ButtonSpacing float32
			}{
				ButtonWidth:   300,
				ButtonHeight:  35,
				ButtonSpacing: 5,
			},
			Colors: struct {
				White      color.Color
				Red        color.Color
				Blue       color.Color
				Yellow     color.Color
				Gray       color.Color
				Orange     color.Color
				Team1      color.Color
				Team2      color.Color
				Leader     color.Color
				Broken     color.Color
				HP         color.Color
				Background color.Color
			}{
				White:      color.White,
				Red:        color.RGBA{R: 255, G: 100, B: 100, A: 255},
				Blue:       color.RGBA{R: 100, G: 100, B: 255, A: 255},
				Yellow:     color.RGBA{R: 255, G: 255, B: 100, A: 255},
				Gray:       color.RGBA{R: 128, G: 128, B: 128, A: 255},
				Orange:     color.RGBA{R: 255, G: 165, B: 0, A: 255},
				Team1:      color.RGBA{R: 100, G: 100, B: 255, A: 255},
				Team2:      color.RGBA{R: 255, G: 100, B: 100, A: 255},
				Leader:     color.RGBA{R: 255, G: 255, B: 100, A: 255},
				Broken:     color.RGBA{R: 128, G: 128, B: 128, A: 255},
				HP:         color.RGBA{R: 100, G: 255, B: 100, A: 255},
				Background: color.NRGBA{R: 0x1a, G: 0x20, B: 0x2c, A: 0xff},
			},
		},
	}
}
