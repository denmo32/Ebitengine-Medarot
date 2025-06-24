package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// parseInt は変更ありません
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

// LoadMedals は変更ありません
func LoadMedals(filePath string) ([]Medal, error) {
	// (元のコードをそのままコピー)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open medal csv file %s: %w", filePath, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
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
			continue
		}
		if len(record) < len(headers) {
			continue
		}
		data := make(map[string]string)
		for i, header := range headers {
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
	return medals, nil
}

// ★★★ [変更点] 部位別LoadPartsを廃止し、単一のLoadAllPartsを新設 ★★★
func LoadAllParts(filePath string) (map[string]*Part, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open parts csv file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// ヘッダー行を読み飛ばす
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header row: %w", err)
	}

	partsMap := make(map[string]*Part)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		// CSVの各列を構造体にマッピング
		armor := parseInt(record[6])
		part := &Part{
			ID:         record[0],
			PartName:   record[1],
			Type:       PartType(record[2]),
			Category:   ActionCategory(record[3]),
			Trait:      ActionTrait(record[4]),
			WeaponType: record[5],
			Armor:      armor,
			MaxArmor:   armor, // Max値も初期化
			Power:      parseInt(record[7]),
			Charge:     parseInt(record[8]),
			Cooldown:   parseInt(record[9]),
			Defense:    parseInt(record[10]),
			Accuracy:   parseInt(record[11]),
			Mobility:   parseInt(record[12]),
			Propulsion: parseInt(record[13]),
			IsBroken:   false,
		}

		partsMap[part.ID] = part
	}

	if len(partsMap) == 0 {
		return nil, fmt.Errorf("no parts loaded from %s", filePath)
	}
	return partsMap, nil
}

// ★★★ [変更点] GameData構造体とLoadAllGameDataをシンプルに ★★★
type GameData struct {
	Medals   []Medal
	AllParts map[string]*Part // 全てのパーツをIDをキーにして保持
}

func LoadAllGameData() (*GameData, error) {
	var err error
	gameData := &GameData{}

	gameData.Medals, err = LoadMedals("medals.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to load medals: %w", err)
	}

	gameData.AllParts, err = LoadAllParts("parts.csv") // 修正したparts.csvを読み込む
	if err != nil {
		return nil, fmt.Errorf("failed to load parts: %w", err)
	}

	if len(gameData.Medals) == 0 {
		fmt.Println("Warning: No medals were loaded.")
	}
	if len(gameData.AllParts) == 0 {
		fmt.Println("Warning: No parts were loaded.")
	}

	return gameData, nil
}
