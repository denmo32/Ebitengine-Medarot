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
	{Head: "P002", RightArm: "P003", LeftArm: "P003", Legs: "P002"},
	{Head: "P002", RightArm: "P002", LeftArm: "P001", Legs: "P003"},
	{Head: "P005", RightArm: "P001", LeftArm: "P004", Legs: "P004"},
	{Head: "P001", RightArm: "P007", LeftArm: "P003", Legs: "P005"},
	{Head: "P004", RightArm: "P004", LeftArm: "P001", Legs: "P006"},
	{Head: "P005", RightArm: "P006", LeftArm: "P006", Legs: "P007"},
}

// findPartByID searches for a part by ID within a slice of parts.
func findPartByID(parts []Part, id string) *Part {
	for i := range parts {
		if parts[i].ID == id {
			p := parts[i]
			return &p
		}
	}
	return nil
}

// findPartBySetIDAndID searches for a part by SetID and ID.
func findPartBySetIDAndID(parts []Part, setID, partID string) *Part {
	for i := range parts {
		if parts[i].SetID == setID && parts[i].ID == partID {
			p := parts[i]
			return &p
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

// ★★★ 修正点1: 引数の型を `TeamID` に変更 ★★★
func createMedarotTeam(teamID TeamID, teamBaseSpeed float64, gameData *GameData, isTeam1 bool) []*Medarot {
	var teamMedarots []*Medarot

	for i := 0; i < PlayersPerTeam; i++ {
		medarotIDNumber := 0
		// ★★★ 修正点2: 比較を `TeamID` 型で行う ★★★
		if teamID == Team1 {
			medarotIDNumber = i + 1
		} else {
			medarotIDNumber = PlayersPerTeam + i + 1
		}
		medarotDisplayID := fmt.Sprintf("p%d", medarotIDNumber)
		medarotName := fmt.Sprintf("Medarot %d", medarotIDNumber)
		isLeader := (i == 0)

		var selectedMedal *Medal
		var partsConfig DefaultLoadout

		// Special setup for p1 (Metabee)
		// ★★★ 修正点2: 比較を `TeamID` 型で行う ★★★
		if teamID == Team1 && i == 0 {
			metabeeSetID := "METABEE_SET"
			metabeePartID := "P001"

			head := findPartBySetIDAndID(gameData.AllParts["head"], metabeeSetID, metabeePartID)
			ra := findPartBySetIDAndID(gameData.AllParts["rightArm"], metabeeSetID, metabeePartID)
			la := findPartBySetIDAndID(gameData.AllParts["leftArm"], metabeeSetID, metabeePartID)
			legs := findPartBySetIDAndID(gameData.AllParts["legs"], metabeeSetID, metabeePartID)

			if head != nil && ra != nil && la != nil && legs != nil {
				partsConfig = DefaultLoadout{Head: head.ID, RightArm: ra.ID, LeftArm: la.ID, Legs: legs.ID}
				selectedMedal = findMedalByID(gameData.Medals, "M001")
				if selectedMedal == nil {
					log.Printf("Warning: Metabee's conventional medal (M001) not found for %s. Falling back.\n", medarotDisplayID)
				}
			} else {
				log.Printf("Warning: METABEE_SET for %s is incomplete. Falling back to default loadout.\n", medarotDisplayID)
				if len(defaultLoadouts) > 0 {
					partsConfig = defaultLoadouts[0]
				}
			}
		}

		if partsConfig.Head == "" {
			loadoutIndex := 0
			// ★★★ 修正点2: 比較を `TeamID` 型で行う ★★★
			if teamID == Team1 {
				loadoutIndex = i
			} else {
				loadoutIndex = PlayersPerTeam + i
			}
			partsConfig = defaultLoadouts[loadoutIndex%len(defaultLoadouts)]
		}

		if selectedMedal == nil {
			medalIndex := 0
			// ★★★ 修正点2: 比較を `TeamID` 型で行う ★★★
			if teamID == Team1 {
				medalIndex = i
			} else {
				medalIndex = PlayersPerTeam + i
			}
			if len(gameData.Medals) > 0 {
				selectedMedal = &gameData.Medals[medalIndex%len(gameData.Medals)]
			} else {
				log.Printf("Warning: No medals loaded. Creating a fallback medal for %s.\n", medarotDisplayID)
				selectedMedal = &Medal{ID: "M_FALLBACK", Name: "Fallback Medal", SkillShoot: 1, SkillFight: 1}
			}
		}

		if selectedMedal == nil {
			selectedMedal = &Medal{ID: "M_DEFAULT", Name: "Default Medal", SkillShoot: 1, SkillFight: 1}
		}

		medarotSpeed := teamBaseSpeed + (rand.Float64() * 0.2)
		// ★★★ 修正点3: `NewMedarot`に `TeamID` 型を渡す ★★★
		medarot := NewMedarot(medarotDisplayID, medarotName, teamID, medarotSpeed, selectedMedal, isLeader)

		if p := findPartByID(gameData.AllParts["head"], partsConfig.Head); p != nil {
			medarot.Parts["head"] = p
		} else {
			log.Printf("Warning: Head part %s not found for %s. Equipping placeholder.\n", partsConfig.Head, medarot.ID)
			medarot.Parts["head"] = &Part{ID: "placeholder", Name: "Missing Head", Slot: "head", IsBroken: true, MaxHP: 1, HP: 1}
		}
		if p := findPartByID(gameData.AllParts["rightArm"], partsConfig.RightArm); p != nil {
			medarot.Parts["rightArm"] = p
		} else {
			log.Printf("Warning: RightArm part %s not found for %s. Equipping placeholder.\n", partsConfig.RightArm, medarot.ID)
			medarot.Parts["rightArm"] = &Part{ID: "placeholder", Name: "Missing RARM", Slot: "rightArm", IsBroken: true, MaxHP: 1, HP: 1}
		}
		if p := findPartByID(gameData.AllParts["leftArm"], partsConfig.LeftArm); p != nil {
			medarot.Parts["leftArm"] = p
		} else {
			log.Printf("Warning: LeftArm part %s not found for %s. Equipping placeholder.\n", partsConfig.LeftArm, medarot.ID)
			medarot.Parts["leftArm"] = &Part{ID: "placeholder", Name: "Missing LARM", Slot: "leftArm", IsBroken: true, MaxHP: 1, HP: 1}
		}
		if p := findPartByID(gameData.AllParts["legs"], partsConfig.Legs); p != nil {
			medarot.Parts["legs"] = p
		} else {
			log.Printf("Warning: Legs part %s not found for %s. Equipping placeholder.\n", partsConfig.Legs, medarot.ID)
			medarot.Parts["legs"] = &Part{ID: "placeholder", Name: "Missing Legs", Slot: "legs", IsBroken: true, MaxHP: 1, HP: 1}
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

	// ★★★ 修正点4: `TeamID` 型の定数を渡す ★★★
	team1Medarots := createMedarotTeam(Team1, team1BaseSpeed, gameData, true)
	allMedarots = append(allMedarots, team1Medarots...)

	// ★★★ 修正点4: `TeamID` 型の定数を渡す ★★★
	team2Medarots := createMedarotTeam(Team2, team2BaseSpeed, gameData, false)
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
				log.Printf("    %s: %s (HP: %d/%d, CHG: %d, CD: %d)", slot, part.Name, part.HP, part.MaxHP, part.Charge, part.Cooldown)
			} else {
				log.Printf("    %s: <NONE>", slot)
			}
		}
	}

	return allMedarots
}