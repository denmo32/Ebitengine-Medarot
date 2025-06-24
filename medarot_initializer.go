package main

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/yohamta/donburi"
	//"github.com/yohamta/donburi/features/math"
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

// findPartByID はパーツIDでパーツを検索し、コピーを返します。
// ECSではコンポーネントのデータはエンティティごとにユニークであるべきなので、
// 共有されるマスターデータからコピーを作成します。
func findPartByID(allParts map[string]*Part, id string) *Part {
	originalPart, exists := allParts[id]
	if !exists {
		return nil
	}
	// パーツデータのコピーを作成
	newPart := *originalPart
	return &newPart
}

// findMedalByID はメダルIDでメダルを検索し、コピーを返します。
// パーツと同様に、メダルデータもエンティティごとにユニークであるべきです。
func findMedalByID(medals []Medal, id string) *Medal {
	for i := range medals {
		if medals[i].ID == id {
			// メダルデータのコピーを作成
			m := medals[i]
			return &m
		}
	}
	return nil
}

func createMedarotEntity(w donburi.World, teamID TeamID, medarotNumber int, isLeader bool, gameData *GameData, drawIndex int) donburi.Entity {
	medarotDisplayID := fmt.Sprintf("p%d", medarotNumber)
	medarotName := fmt.Sprintf("機体 %d", medarotNumber)

	entity := w.Create(IdentityComponentType, CMedal, PartsComponentType, StatusComponentType, ActionComponentType, RenderComponentType)

	// IdentityComponent
	IdentityComponentType.SetValue(w.Entry(entity), IdentityComponent{
		ID:       medarotDisplayID,
		Name:     medarotName,
		Team:     teamID,
		IsLeader: isLeader,
	})

	// MedalComponent & PartsComponent
	var selectedMedal *Medal // This Medal is models.Medal
	var partsConfig DefaultLoadout = defaultLoadouts[rand.Intn(len(defaultLoadouts))]

	if teamID == Team1 && isLeader {
		metabeeMedal := findMedalByID(gameData.Medals, "M001")
		if metabeeMedal != nil {
			selectedMedal = metabeeMedal
		}
		partsConfig = defaultLoadouts[0]
	} else {
		if len(gameData.Medals) > 0 {
			medalIndex := rand.Intn(len(gameData.Medals))
			selectedMedal = &gameData.Medals[medalIndex]
		}
	}

	if selectedMedal == nil {
		log.Printf("Warning: No suitable medal found. Creating a fallback medal for %s.\n", medarotDisplayID)
		selectedMedal = &Medal{ID: "M_FALLBACK", Name: "Fallback", SkillShoot: 5, SkillFight: 5}
	}
	CMedal.SetValue(w.Entry(entity), MedalComponent{Medal: selectedMedal})

	partsMap := make(map[PartSlotKey]*Part)
	partIDMap := map[PartSlotKey]string{
		PartSlotHead:     partsConfig.Head,
		PartSlotRightArm: partsConfig.RightArm,
		PartSlotLeftArm:  partsConfig.LeftArm,
		PartSlotLegs:     partsConfig.Legs,
	}

	// dummyOwnerMedarotForPart と p.Owner の参照を削除
	for slot, partID := range partIDMap {
		if p := findPartByID(gameData.AllParts, partID); p != nil {
			partsMap[slot] = p
		} else {
			log.Printf("Warning: Part %s for slot %s not found for %s. Equipping placeholder.\n", partID, slot, medarotDisplayID)
			placeholderPart := &Part{ID: "placeholder", PartName: "Missing", Type: PartType(slot), IsBroken: true, MaxArmor: 1, Armor: 1}
			partsMap[slot] = placeholderPart
		}
	}
	PartsComponentType.SetValue(w.Entry(entity), PartsComponent{Parts: partsMap})

	// StatusComponent
	StatusComponentType.SetValue(w.Entry(entity), StatusComponent{
		State:             StateReadyToSelectAction,
		Gauge:             100.0,
		IsEvasionDisabled: false,
		IsDefenseDisabled: false,
	})

	// ActionComponent
	ActionComponentType.SetValue(w.Entry(entity), ActionComponent{})

	// RenderComponent
	RenderComponentType.SetValue(w.Entry(entity), RenderComponent{DrawIndex: drawIndex})

	// AIControlled / PlayerControlled
	// w.AddComponentではなく、w.Entry(entity).AddComponent を使用する
	medarotEntry := w.Entry(entity) // Entryを一度取得
	if teamID == Team1 {
		medarotEntry.AddComponent(PlayerControlledComponentType)
	} else {
		medarotEntry.AddComponent(AIControlledComponentType)
	}

	logMedarotInitialization(entity, w)
	return entity
}

func InitializeAllMedarotEntities(w donburi.World, gameData *GameData) {
	team1Count := 0
	team2Count := 0

	for i := 0; i < PlayersPerTeam; i++ {
		medarotIDNumberTeam1 := i + 1
		isLeaderTeam1 := (i == 0)
		createMedarotEntity(w, Team1, medarotIDNumberTeam1, isLeaderTeam1, gameData, team1Count)
		team1Count++

		medarotIDNumberTeam2 := PlayersPerTeam + i + 1
		isLeaderTeam2 := (i == 0)
		createMedarotEntity(w, Team2, medarotIDNumberTeam2, isLeaderTeam2, gameData, team2Count)
		team2Count++
	}
	log.Printf("Initialized %d medarot entities in total.", team1Count+team2Count)
}

func logMedarotInitialization(entity donburi.Entity, w donburi.World) {
	entry := w.Entry(entity)
	identity := IdentityComponentType.Get(entry)
	medalComp := CMedal.Get(entry) // CMedal を使用
	partsComp := PartsComponentType.Get(entry)

	teamStr := "Team1"
	if identity.Team == Team2 {
		teamStr = "Team2"
	}
	log.Printf("  - EntityID: %d, Name: %s (%s), Leader: %t, Medal: %s", entity.Id(), identity.Name, teamStr, identity.IsLeader, medalComp.Medal.Name)
	for slot, part := range partsComp.Parts {
		if part != nil {
			log.Printf("    %s: %s (Armor: %d/%d, Pow: %d)", string(slot), part.PartName, part.Armor, part.MaxArmor, part.Power)
		} else {
			log.Printf("    %s: <NONE>", string(slot))
		}
	}
}
