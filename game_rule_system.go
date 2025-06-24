package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// GameRuleSystem はゲームの勝敗判定などを行います。
type GameRuleSystem struct{ query *donburi.Query }

func NewGameRuleSystem() *GameRuleSystem {
	return &GameRuleSystem{
		query: donburi.NewQuery(filter.And(filter.Contains(IdentityComponentType), filter.Contains(StatusComponentType))),
	}
}
func (sys *GameRuleSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, gameStateOk := GameStateComponentType.First(ecs.World)
	if !gameStateOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState == GameStateOver {
		return
	}

	team1LeaderAlive := false
	team2LeaderAlive := false

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		identity := IdentityComponentType.Get(entry)
		status := StatusComponentType.Get(entry)
		isAlive := !status.State_is_broken_internal()
		if identity.Team == Team1 {
			if identity.IsLeader && isAlive {
				team1LeaderAlive = true
			}
		} else if identity.Team == Team2 {
			if identity.IsLeader && isAlive {
				team2LeaderAlive = true
			}
		}
	})

	var team1LeaderExists, team2LeaderExists bool
	leaderQuery := donburi.NewQuery(filter.Contains(IdentityComponentType))
	leaderQuery.Each(ecs.World, func(entry *donburi.Entry) {
		id := IdentityComponentType.Get(entry)
		if id.IsLeader {
			if id.Team == Team1 {
				team1LeaderExists = true
			}
			if id.Team == Team2 {
				team2LeaderExists = true
			}
		}
	})

	if team1LeaderExists && !team1LeaderAlive {
		gs.Winner = Team2
		gs.CurrentState = GameStateOver
		gs.Message = "チーム2の勝利！"
		// showGameMessage(ecs, gs.Message, nil) // ★ この行をコメントアウトまたは削除
		GameStateComponentType.Set(gameStateEntry, gs) // ★ 状態を保存するために追加
		return
	}
	if team2LeaderExists && !team2LeaderAlive {
		gs.Winner = Team1
		gs.CurrentState = GameStateOver
		gs.Message = "チーム1の勝利！"
		// showGameMessage(ecs, gs.Message, nil) // ★ この行をコメントアウトまたは削除
		GameStateComponentType.Set(gameStateEntry, gs) // ★ 状態を保存するために追加
		return
	}
}
