// --- message_system.go ---
package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi/ecs"
)

type MessageSystem struct{}

func NewMessageSystem() *MessageSystem { return &MessageSystem{} }

func (sys *MessageSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, ok := GameStateComponentType.First(ecs.World)
	if !ok {
		return
	}

	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState != GameStateMessage {
		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		callback := gs.PostMessageCallback
		gs.PostMessageCallback = nil

		// コールバックがあれば実行し、なければ状態をPlayingに戻す
		if callback != nil {
			GameStateComponentType.Set(gameStateEntry, gs) // 先にgsを保存
			callback()
		} else {
			gs.CurrentState = StatePlaying
			GameStateComponentType.Set(gameStateEntry, gs)
		}
	}
}
