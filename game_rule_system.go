// --- game_rule_system.go ---
package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// GameRuleSystem はゲームの勝敗判定を行います。
type GameRuleSystem struct{ query *donburi.Query }

func NewGameRuleSystem() *GameRuleSystem {
	return &GameRuleSystem{
		query: donburi.NewQuery(filter.And(
			filter.Contains(IdentityComponentType),
			filter.Contains(StatusComponentType),
		)),
	}
}

func (sys *GameRuleSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, ok := GameStateComponentType.First(ecs.World)
	if !ok {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState == GameStateOver {
		return
	}

	team1LeaderAlive := false
	team2LeaderAlive := false
	team1LeaderExists := false
	team2LeaderExists := false

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		identity := IdentityComponentType.Get(entry)
		// ★★★ 修正箇所: IsBroken() メソッドを使用 ★★★
		isAlive := !StatusComponentType.Get(entry).IsBroken()

		if identity.IsLeader {
			if identity.Team == Team1 {
				team1LeaderExists = true
				if isAlive {
					team1LeaderAlive = true
				}
			} else {
				team2LeaderExists = true
				if isAlive {
					team2LeaderAlive = true
				}
			}
		}
	})

	winnerFound := false
	if team1LeaderExists && !team1LeaderAlive {
		gs.Winner = Team2
		gs.Message = "チーム2の勝利！"
		winnerFound = true
	} else if team2LeaderExists && !team2LeaderAlive {
		gs.Winner = Team1
		gs.Message = "チーム1の勝利！"
		winnerFound = true
	}

	if winnerFound {
		gs.CurrentState = GameStateOver
		GameStateComponentType.Set(gameStateEntry, gs)
	}
}
