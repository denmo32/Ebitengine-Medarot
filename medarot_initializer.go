package main

import (
	"fmt"
	"log"
	"math/rand"
)

// ★★★ ここに定数を追加 ★★★
const PlayersPerTeam = 3

// DefaultLoadout は変更ありません
type DefaultLoadout struct {
	Head     string
	RightArm string
	LeftArm  string
	Legs     string
}

// ★★★ [変更点] パーツIDを新しいCSVに合わせて修正 ★★★
// あなたが作成したparts.csvのIDに合わせて、ここを修正してください。
var defaultLoadouts = []DefaultLoadout{
	{Head: "H-001", RightArm: "RA-001", LeftArm: "LA-001", Legs: "L-001"}, // マグナムセット
	{Head: "H-002", RightArm: "RA-002", LeftArm: "LA-002", Legs: "L-002"}, // ソードセット
	{Head: "H-003", RightArm: "RA-003", LeftArm: "LA-003", Legs: "L-003"}, // ショットガンセット
	{Head: "H-004", RightArm: "RA-004", LeftArm: "LA-004", Legs: "L-004"}, // ハンマーセット
	{Head: "H-005", RightArm: "RA-005", LeftArm: "LA-005", Legs: "L-005"}, // レーザーセット
	{Head: "H-006", RightArm: "RA-006", LeftArm: "LA-006", Legs: "L-006"}, // クロウセット
}

// ★★★ [変更点] findPartByID関数をマップ検索用に全面改修 ★★★
// ★★★ 重要コメント ★★★
// findPartByID は、全パーツデータの中から指定されたIDのパーツを探し、そのコピーを返します。
//
// なぜコピーを返すのか？:
// 戦闘中、各メダロットが装備するパーツは、装甲値(Armor)や破壊状態(IsBroken)といった
// 固有の状態を持ちます。もし元のデータのポインタを共有してしまうと、
// ある一機のパーツがダメージを受けた際に、同じパーツを装備する他の全機体に影響が及んでしまいます。
// それを防ぐため、各メダロットにはパーツデータの独立したコピーを装備させる必要があります。
func findPartByID(allParts map[string]*Part, id string) *Part {
	// マップからIDでパーツのポインタを取得
	originalPart, exists := allParts[id]
	if !exists {
		return nil // パーツが見つからなければnilを返す
	}

	// 元のデータを変更しないように、新しいPartインスタンス（コピー）を作成してそのポインタを返す
	newPart := *originalPart
	return &newPart
}

// findMedalByID は変更ありません
func findMedalByID(medals []Medal, id string) *Medal {
	for i := range medals {
		if medals[i].ID == id {
			m := medals[i]
			return &m
		}
	}
	return nil
}

// ★★★ [変更点] createMedarotTeamを新しいデータ構造に合わせて修正 ★★★
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

		// チーム1のリーダー機は特別なロードアウトとメダルにする（例）
		var selectedMedal *Medal
		var partsConfig DefaultLoadout
		if teamID == Team1 && isLeader {
			// ★★★ メダルIDをあなたのmedals.csvに合わせてください ★★★
			metabeeMedal := findMedalByID(gameData.Medals, "M001") // 例: カブトメダル
			if metabeeMedal != nil {
				selectedMedal = metabeeMedal
			}
			partsConfig = defaultLoadouts[0] // 最初のロードアウトを割り当て
		} else {
			// その他の機体はランダム（または順番）に割り当て
			loadoutIndex := rand.Intn(len(defaultLoadouts))
			partsConfig = defaultLoadouts[loadoutIndex]

			medalIndex := rand.Intn(len(gameData.Medals))
			selectedMedal = &gameData.Medals[medalIndex]
		}

		if selectedMedal == nil {
			log.Printf("Warning: No medals loaded. Creating a fallback medal for %s.\n", medarotDisplayID)
			selectedMedal = &Medal{ID: "M_FALLBACK", Name: "Fallback", SkillShoot: 5, SkillFight: 5}
		}

		// NewMedarotの呼び出しは変更なし
		medarot := NewMedarot(medarotDisplayID, medarotName, teamID, selectedMedal, isLeader)

		// パーツIDのマップ
		partIDMap := map[string]string{
			"head":     partsConfig.Head,
			"rightArm": partsConfig.RightArm,
			"leftArm":  partsConfig.LeftArm,
			"legs":     partsConfig.Legs,
		}

		for slot, partID := range partIDMap {
			// ★★★ findPartByIDの呼び出し方を変更 ★★★
			if p := findPartByID(gameData.AllParts, partID); p != nil {
				// ▼▼▼ この1行を追加 ▼▼▼
				p.Owner = medarot // パーツに持ち主の情報を設定
				medarot.Parts[slot] = p
			} else {
				log.Printf("Warning: Part %s for slot %s not found for %s. Equipping placeholder.\n", partID, slot, medarot.ID)
				placeholderPart := &Part{ID: "placeholder", PartName: "Missing", Type: PartType(slot), IsBroken: true, MaxArmor: 1, Armor: 1}
				// ▼▼▼ こちらも忘れずに追加 ▼▼▼
				placeholderPart.Owner = medarot // プレースホルダーにも持ち主情報を設定
				medarot.Parts[slot] = placeholderPart
			}
		}

		teamMedarots = append(teamMedarots, medarot)
	}
	return teamMedarots
}

// ★★★ [変更点] InitializeAllMedarotsのログ出力を新Part構造体に合わせて修正 ★★★
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
		// SpeedはもうMedarot構造体にないのでログから削除
		log.Printf("  - %s (%s), Leader: %t, Medal: %s", m.Name, teamStr, m.IsLeader, m.Medal.Name)
		for slot, part := range m.Parts {
			if part != nil {
				log.Printf("    %s: %s (Armor: %d/%d, Pow: %d)", slot, part.PartName, part.Armor, part.MaxArmor, part.Power)
			} else {
				log.Printf("    %s: <NONE>", slot)
			}
		}
	}

	return allMedarots
}
