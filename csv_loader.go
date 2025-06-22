package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// parseInt safely converts a string to an integer, returning 0 on error.
func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// LoadMedals loads medal data from a CSV file.
func LoadMedals(filePath string) ([]Medal, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open medal csv file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	headers, err := reader.Read() // Read header row
	if err != nil {
		return nil, fmt.Errorf("failed to read headers from medal csv %s: %w", filePath, err)
	}

	var medals []Medal
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Warning: error reading record from medal csv %s: %v\n", filePath, err)
			continue
		}

		if len(record) < len(headers) {
			continue
		}

		data := make(map[string]string)
			for i, header := range headers {
			// ヘッダーを小文字に統一し、前後の空白も除去して堅牢性を高める
			normalizedHeader := strings.ToLower(strings.TrimSpace(header))
			data[normalizedHeader] = record[i]
		}

		if data["id"] == "" {
			continue
		}

		medal := Medal{
			ID:           data["id"],
			Name:         data["name_jp"],
			Personality:  data["personality_jp"],
			Medaforce:    data["medaforce_jp"],
			Attribute:    data["attribute_jp"],
			SkillShoot:   parseInt(data["skill_shoot"]),
			SkillFight:   parseInt(data["skill_fight"]),
			SkillScan:    parseInt(data["skill_scan"]),
			SkillSupport: parseInt(data["skill_support"]),
		}
		medals = append(medals, medal)
	}
	if len(medals) == 0 && err != io.EOF {
		return nil, fmt.Errorf("no medals loaded from %s, last error: %w", filePath, err)
	}
	return medals, nil
}

// ★★★ 修正点2: LoadParts関数の修正 ★★★
//パーツの種類（スロット名）に応じて、読み込むべきデータだけを読み込むように修正し、堅牢性を高めます。
//これにより、例えば頭パーツのCSVに脚部専用の列がなくても問題なく動作します。
func LoadParts(filePath string, slotName string) ([]Part, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open part csv file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read headers from part csv %s: %w", filePath, err)
	}

	var parts []Part
	const defaultPartHPBase = 50
	const defaultLegsHPBonus = 10

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Warning: error reading record from part csv %s: %v\n", filePath, err)
			continue
		}

		if len(record) < len(headers) {
			continue
		}

		data := make(map[string]string)
			for i, header := range headers {
			// ヘッダーを小文字に統一し、前後の空白も除去して堅牢性を高める
			normalizedHeader := strings.ToLower(strings.TrimSpace(header))
			data[normalizedHeader] = record[i]
		}

		if data["id"] == "" {
			continue
		}

		hp := parseInt(data["base_hp"])
		if hp == 0 {
			hp = defaultPartHPBase
		}
		if slotName == "legs" {
			hp += defaultLegsHPBonus
		}

		part := Part{
			ID:           data["id"],
			Name:         data["name_jp"],
			Category:     data["category_jp"],
			SubCategory:  data["sub_category_jp"],
			Slot:         slotName,
			HP:           hp,
			MaxHP:        hp,
			Power:        parseInt(data["power"]),
			Charge:       parseInt(data["charge"]),
			Cooldown:     parseInt(data["cooldown"]),
			IsBroken:     false,
			DefenseParam: parseInt(data["defense_param"]),
			SetID:        data["set_id"],
		}

		// Parts-specific attributes
		if slotName == "legs" {
			part.MovementType = data["movement_type_jp"]
			part.Mobility = parseInt(data["mobility"])
			part.Propulsion = parseInt(data["propulsion"])
		} else { // Head, RightArm, LeftArm
			part.Accuracy = parseInt(data["accuracy"])
		}

		switch part.Category {
		case "射撃":
			part.ActionType = "shoot"
		case "格闘":
			part.ActionType = "fight"
		default:
			part.ActionType = "other"
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 && err != io.EOF {
		return nil, fmt.Errorf("no parts loaded from %s, last error: %w", filePath, err)
	}
	return parts, nil
}

// GameData holds all loaded game data.
type GameData struct {
	Medals        []Medal
	HeadParts     []Part
	RightArmParts []Part
	LeftArmParts  []Part
	LegsParts     []Part
	AllParts      map[string][]Part
}

// LoadAllGameData loads all necessary CSV files.
func LoadAllGameData() (*GameData, error) {
	var err error
	gameData := &GameData{
		AllParts: make(map[string][]Part),
	}

	gameData.Medals, err = LoadMedals("medals.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to load medals: %w", err)
	}

	gameData.HeadParts, err = LoadParts("head_parts.csv", "head")
	if err != nil {
		return nil, fmt.Errorf("failed to load head parts: %w", err)
	}
	gameData.AllParts["head"] = gameData.HeadParts

	gameData.RightArmParts, err = LoadParts("right_arm_parts.csv", "rightArm")
	if err != nil {
		return nil, fmt.Errorf("failed to load right arm parts: %w", err)
	}
	gameData.AllParts["rightArm"] = gameData.RightArmParts

	gameData.LeftArmParts, err = LoadParts("left_arm_parts.csv", "leftArm")
	if err != nil {
		return nil, fmt.Errorf("failed to load left arm parts: %w", err)
	}
	gameData.AllParts["leftArm"] = gameData.LeftArmParts

	gameData.LegsParts, err = LoadParts("legs_parts.csv", "legs")
	if err != nil {
		return nil, fmt.Errorf("failed to load legs parts: %w", err)
	}
	gameData.AllParts["legs"] = gameData.LegsParts

	if len(gameData.Medals) == 0 {
		fmt.Println("Warning: No medals were loaded.")
	}
	if len(gameData.HeadParts) == 0 && len(gameData.RightArmParts) == 0 && len(gameData.LeftArmParts) == 0 && len(gameData.LegsParts) == 0 {
		fmt.Println("Warning: No parts were loaded for any slot.")
	}

	return gameData, nil
}
