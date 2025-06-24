package main

import (
	"fmt"
	"log"
	"math/rand"
)

const PlayersPerTeam = 3

type DefaultLoadout struct {
	Head     string
	RightArm string
	LeftArm  string
	Legs     string
}

var defaultLoadouts = []DefaultLoadout{
	{Head: "H-001", RightArm: "RA-001", LeftArm: "LA-001", Legs: "L-001"},
	{Head: "H-002", RightArm: "RA-002", LeftArm: "LA-002", Legs: "L-002"},
	{Head: "H-003", RightArm: "RA-003", LeftArm: "LA-003", Legs: "L-003"},
	{Head: "H-004", RightArm: "RA-004", LeftArm: "LA-004", Legs: "L-004"},
	{Head: "H-005", RightArm: "RA-005", LeftArm: "LA-005", Legs: "L-005"},
	{Head: "H-006", RightArm: "RA-006", LeftArm: "LA-006", Legs: "L-006"},
}

func findPartByID(allParts map[string]*Part, id string) *Part {
	originalPart, exists := allParts[id]
	if !exists {
		return nil
	}
	newPart := *originalPart
	return &newPart
}

func findMedalByID(medals []Medal, id string) *Medal {
	for i := range medals {
		if medals[i].ID == id {
			m := medals[i]
			return &m
		}
	}
	return nil
}

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
		// ★ 'partsConfig' の未使用エラーを修正
		var partsConfig DefaultLoadout = defaultLoadouts[rand.Intn(len(defaultLoadouts))]
		if teamID == Team1 && isLeader {
			metabeeMedal := findMedalByID(gameData.Medals, "M001")
			if metabeeMedal != nil {
				selectedMedal = metabeeMedal
			}
			partsConfig = defaultLoadouts[0]
		} else {
			medalIndex := rand.Intn(len(gameData.Medals))
			selectedMedal = &gameData.Medals[medalIndex]
		}

		if selectedMedal == nil {
			log.Printf("Warning: No medals loaded. Creating a fallback medal for %s.\n", medarotDisplayID)
			selectedMedal = &Medal{ID: "M_FALLBACK", Name: "Fallback", SkillShoot: 5, SkillFight: 5}
		}

		medarot := NewMedarot(medarotDisplayID, medarotName, teamID, selectedMedal, isLeader)

		// ★定数を使用
		partIDMap := map[PartSlotKey]string{
			PartSlotHead:     partsConfig.Head,
			PartSlotRightArm: partsConfig.RightArm,
			PartSlotLeftArm:  partsConfig.LeftArm,
			PartSlotLegs:     partsConfig.Legs,
		}

		for slot, partID := range partIDMap {
			if p := findPartByID(gameData.AllParts, partID); p != nil {
				p.Owner = medarot
				medarot.Parts[slot] = p
			} else {
				log.Printf("Warning: Part %s for slot %s not found for %s. Equipping placeholder.\n", partID, slot, medarot.ID)
				placeholderPart := &Part{ID: "placeholder", PartName: "Missing", Type: PartType(slot), IsBroken: true, MaxArmor: 1, Armor: 1}
				placeholderPart.Owner = medarot
				medarot.Parts[slot] = placeholderPart
			}
		}

		teamMedarots = append(teamMedarots, medarot)
	}
	return teamMedarots
}

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
		log.Printf("  - %s (%s), Leader: %t, Medal: %s", m.Name, teamStr, m.IsLeader, m.Medal.Name)
		for slot, part := range m.Parts {
			if part != nil {
				log.Printf("    %s: %s (Armor: %d/%d, Pow: %d)", string(slot), part.PartName, part.Armor, part.MaxArmor, part.Power)
			} else {
				log.Printf("    %s: <NONE>", string(slot))
			}
		}
	}

	return allMedarots
}
