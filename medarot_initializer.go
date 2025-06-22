package main

import (
	"fmt"
	"log"
	"math/rand"
)

// DefaultLoadout defines a set of part IDs for a Medarot.
type DefaultLoadout struct {
	Head     string
	RightArm string
	LeftArm  string
	Legs     string
}

var defaultLoadouts = []DefaultLoadout{
	// CSVファイルのID体系 (P001, P002など) に合わせて修正
	{Head: "P001", RightArm: "P001", LeftArm: "P001", Legs: "P001"}, // Metabee-like (各パーツCSVのP001を使用)
	{Head: "P002", RightArm: "P002", LeftArm: "P002", Legs: "P002"}, // Rokusho-like (各パーツCSVのP002を使用)
	{Head: "P003", RightArm: "P003", LeftArm: "P003", Legs: "P003"}, // 汎用セット
	{Head: "P001", RightArm: "P002", LeftArm: "P003", Legs: "P004"}, // 組み合わせ1
	{Head: "P002", RightArm: "P003", LeftArm: "P001", Legs: "P005"}, // 組み合わせ2
	{Head: "P003", RightArm: "P001", LeftArm: "P002", Legs: "P006"}, // 組み合わせ3
}

// ★★★ 修正点1: findPartByID関数の修正 ★★★
//パーツが見つかった際に、安全に新しいインスタンス（コピー）を作成してそのポインタを返します。
//これにより、複数のメダロットが同じパーツデータを共有してしまうことを防ぎ、
//ポインタに関する潜在的な問題を回避します。
func findPartByID(parts []Part, id string) *Part {
	for i := range parts {
		if parts[i].ID == id {
			// 元のデータを変更しないように、新しいPartインスタンスを作成して返す
			newPart := parts[i]
			return &newPart
		}
	}
	return nil
}

// findMedalByID searches for a medal by ID.
func findMedalByID(medals []Medal, id string) *Medal {
	for i := range medals {
		if medals[i].ID == id {
			m := medals[i]
			return &m
		}
	}
	return nil
}

// createMedarotTeam creates a team of Medarots.
func createMedarotTeam(teamID TeamID, teamBaseSpeed float64, gameData *GameData) []*Medarot {
	var teamMedarots []*Medarot

	for i := 0; i < PlayersPerTeam; i++ {
		medarotIDNumber := 0
		if teamID == Team1 {
			medarotIDNumber = i + 1
		} else {
			medarotIDNumber = PlayersPerTeam + i + 1
		}
		medarotDisplayID := fmt.Sprintf("p%d", medarotIDNumber)
		medarotName := fmt.Sprintf("機体 %d", medarotIDNumber)
		isLeader := (i == 0)

		var selectedMedal *Medal
		var partsConfig DefaultLoadout

		if teamID == Team1 && i == 0 {
			metabeeMedal := findMedalByID(gameData.Medals, "M001")
			if metabeeMedal != nil {
				selectedMedal = metabeeMedal
			}
			if len(defaultLoadouts) > 0 {
				partsConfig = defaultLoadouts[0]
			}
		}

		if partsConfig.Head == "" {
			loadoutIndex := medarotIDNumber - 1
			if len(defaultLoadouts) > 0 {
				partsConfig = defaultLoadouts[loadoutIndex%len(defaultLoadouts)]
			}
		}

		if selectedMedal == nil {
			medalIndex := medarotIDNumber - 1
			if len(gameData.Medals) > 0 {
				selectedMedal = &gameData.Medals[medalIndex%len(gameData.Medals)]
			} else {
				log.Printf("Warning: No medals loaded. Creating a fallback medal for %s.\n", medarotDisplayID)
				selectedMedal = &Medal{ID: "M_FALLBACK", Name: "Fallback", SkillShoot: 5, SkillFight: 5}
			}
		}

		medarotSpeed := teamBaseSpeed + (rand.Float64() * 0.2)
		medarot := NewMedarot(medarotDisplayID, medarotName, teamID, medarotSpeed, selectedMedal, isLeader)

		partMap := map[string]string{
			"head":     partsConfig.Head,
			"rightArm": partsConfig.RightArm,
			"leftArm":  partsConfig.LeftArm,
			"legs":     partsConfig.Legs,
		}

		for slot, partID := range partMap {
			if p := findPartByID(gameData.AllParts[slot], partID); p != nil {
				medarot.Parts[slot] = p
			} else {
				log.Printf("Warning: Part %s for slot %s not found for %s. Equipping placeholder.\n", partID, slot, medarot.ID)
				medarot.Parts[slot] = &Part{ID: "placeholder", Name: "Missing", Slot: slot, IsBroken: true, MaxHP: 1, HP: 1}
			}
		}

		teamMedarots = append(teamMedarots, medarot)
	}
	return teamMedarots
}

// InitializeAllMedarots creates all Medarots for the game.
func InitializeAllMedarots(gameData *GameData) []*Medarot {
	var allMedarots []*Medarot

	const team1BaseSpeed = 1.0
	const team2BaseSpeed = 0.9

	team1Medarots := createMedarotTeam(Team1, team1BaseSpeed, gameData)
	allMedarots = append(allMedarots, team1Medarots...)

	team2Medarots := createMedarotTeam(Team2, team2BaseSpeed, gameData)
	allMedarots = append(allMedarots, team2Medarots...)

	log.Printf("Initialized %d medarots in total.", len(allMedarots))
	for _, m := range allMedarots {
		teamStr := "Team1"
		if m.Team == Team2 {
			teamStr = "Team2"
		}
		log.Printf("  - %s (%s), Leader: %t, Speed: %.2f, Medal: %s", m.Name, teamStr, m.IsLeader, m.Speed, m.Medal.Name)
		for slot, part := range m.Parts {
			if part != nil {
				log.Printf("    %s: %s (HP: %d/%d, IsBroken: %t, Pow: %d)", slot, part.Name, part.HP, part.MaxHP, part.IsBroken, part.Power)
			} else {
				log.Printf("    %s: <NONE>", slot)
			}
		}
	}

	return allMedarots
}
