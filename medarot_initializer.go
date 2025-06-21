package main

import (
	"fmt"
	"math/rand"
)

const (
	// PlayersPerTeam = 3
	Team1          = "Team1"
	Team2          = "Team2"
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
			// Return a copy to avoid modifying the original slice data if the part is modified later
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

// createMedarotTeam creates a team of Medarots.
func createMedarotTeam(teamID string, teamBaseSpeed float64, gameData *GameData, isTeam1 bool) []*Medarot {
	var teamMedarots []*Medarot

	for i := 0; i < PlayersPerTeam; i++ {
		medarotIDNumber := 0
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
		if teamID == Team1 && i == 0 {
			metabeeSetID := "METABEE_SET"
			metabeePartID := "P001" // In JS, this was the same for all slots of Metabee set.
			
			head := findPartBySetIDAndID(gameData.AllParts["head"], metabeeSetID, metabeePartID)
			ra := findPartBySetIDAndID(gameData.AllParts["rightArm"], metabeeSetID, metabeePartID)
			la := findPartBySetIDAndID(gameData.AllParts["leftArm"], metabeeSetID, metabeePartID)
			legs := findPartBySetIDAndID(gameData.AllParts["legs"], metabeeSetID, metabeePartID)

			if head != nil && ra != nil && la != nil && legs != nil {
				partsConfig = DefaultLoadout{Head: head.ID, RightArm: ra.ID, LeftArm: la.ID, Legs: legs.ID}
				selectedMedal = findMedalByID(gameData.Medals, "M001") // Metabee's medal
				if selectedMedal == nil {
					fmt.Printf("Warning: Metabee's conventional medal (M001) not found for %s. Falling back.\n", medarotDisplayID)
				}
			} else {
				fmt.Printf("Warning: METABEE_SET for %s is incomplete. Falling back to default loadout.\n", medarotDisplayID)
				// Fallback to default loadout if set is not complete
                if len(defaultLoadouts) > 0 {
				    partsConfig = defaultLoadouts[0]
                }
			}
		}
		
		// Fallback or standard selection
		if partsConfig.Head == "" { // If not Metabee or Metabee setup failed
			loadoutIndex := 0
			if teamID == Team1 {
				loadoutIndex = i
			} else {
				loadoutIndex = PlayersPerTeam + i
			}
			partsConfig = defaultLoadouts[loadoutIndex%len(defaultLoadouts)]
		}

		if selectedMedal == nil { // If not Metabee's medal or it wasn't found
			medalIndex := 0
			if teamID == Team1 {
				medalIndex = i
			} else {
				medalIndex = PlayersPerTeam + i
			}
			if len(gameData.Medals) > 0 {
				selectedMedal = &gameData.Medals[medalIndex%len(gameData.Medals)]
			} else {
				// Create a fallback medal if no medals are loaded
				fmt.Printf("Warning: No medals loaded. Creating a fallback medal for %s.\n", medarotDisplayID)
				selectedMedal = &Medal{ID: "M_FALLBACK", Name: "Fallback Medal", SkillShoot: 1, SkillFight: 1} // Simplified
			}
		}
		
		// Ensure a medal is always assigned
		if selectedMedal == nil {
			selectedMedal = &Medal{ID: "M_DEFAULT", Name: "Default Medal", SkillShoot: 1, SkillFight: 1}
		}


		// Create the Medarot instance
		// Speed variation from JS: + (Math.random() * 0.2)
		medarotSpeed := teamBaseSpeed + (rand.Float64() * 0.2)
		medarot := NewMedarot(medarotDisplayID, medarotName, teamID, medarotSpeed, selectedMedal, isLeader)

		// Equip parts
		if p := findPartByID(gameData.AllParts["head"], partsConfig.Head); p != nil {
			medarot.Parts["head"] = p
		} else {
			fmt.Printf("Warning: Head part %s not found for %s. Equipping placeholder.\n", partsConfig.Head, medarot.ID)
			medarot.Parts["head"] = &Part{ID: "placeholder", Name: "Missing Head", Slot: "head", IsBroken: true, MaxHP: 1, HP: 1}
		}
		if p := findPartByID(gameData.AllParts["rightArm"], partsConfig.RightArm); p != nil {
			medarot.Parts["rightArm"] = p
		} else {
			fmt.Printf("Warning: RightArm part %s not found for %s. Equipping placeholder.\n", partsConfig.RightArm, medarot.ID)
			medarot.Parts["rightArm"] = &Part{ID: "placeholder", Name: "Missing RARM", Slot: "rightArm", IsBroken: true, MaxHP: 1, HP: 1}
		}
		if p := findPartByID(gameData.AllParts["leftArm"], partsConfig.LeftArm); p != nil {
			medarot.Parts["leftArm"] = p
		} else {
			fmt.Printf("Warning: LeftArm part %s not found for %s. Equipping placeholder.\n", partsConfig.LeftArm, medarot.ID)
			medarot.Parts["leftArm"] = &Part{ID: "placeholder", Name: "Missing LARM", Slot: "leftArm", IsBroken: true, MaxHP: 1, HP: 1}
		}
		if p := findPartByID(gameData.AllParts["legs"], partsConfig.Legs); p != nil {
			medarot.Parts["legs"] = p
		} else {
			fmt.Printf("Warning: Legs part %s not found for %s. Equipping placeholder.\n", partsConfig.Legs, medarot.ID)
			medarot.Parts["legs"] = &Part{ID: "placeholder", Name: "Missing Legs", Slot: "legs", IsBroken: true, MaxHP: 1, HP: 1}
		}
		teamMedarots = append(teamMedarots, medarot)
	}
	return teamMedarots
}

// InitializeAllMedarots creates all Medarots for the game.
func InitializeAllMedarots(gameData *GameData) []*Medarot {
	var allMedarots []*Medarot

	// Base speeds from JS CONFIG (can be constants or configurable later)
	const team1BaseSpeed = 1.0
	const team2BaseSpeed = 0.9

	team1Medarots := createMedarotTeam(Team1, team1BaseSpeed, gameData, true)
	allMedarots = append(allMedarots, team1Medarots...)

	team2Medarots := createMedarotTeam(Team2, team2BaseSpeed, gameData, false)
	allMedarots = append(allMedarots, team2Medarots...)
	
	fmt.Printf("Initialized %d medarots in total.\n", len(allMedarots))
	for _, m := range allMedarots {
		fmt.Printf("  - %s (%s), Leader: %t, Speed: %.2f, Medal: %s\n", m.Name, m.Team, m.IsLeader, m.Speed, m.Medal.Name)
		for slot, part := range m.Parts {
			if part != nil {
				fmt.Printf("    %s: %s (HP: %d/%d, CHG: %d, CD: %d)\n", slot, part.Name, part.HP, part.MaxHP, part.Charge, part.Cooldown)
			} else {
				fmt.Printf("    %s: <NONE>\n", slot)
			}
		}
	}

	return allMedarots
}
