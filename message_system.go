package main

import (
	"github.com/hajimehoshi/ebiten/v2" // Import ebiten package
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yohamta/donburi/ecs"
)

// MessageSystem はメッセージ表示中のクリックを処理します。
type MessageSystem struct{}

func NewMessageSystem() *MessageSystem { return &MessageSystem{} }
func (sys *MessageSystem) Update(ecs *ecs.ECS) {
	gameStateEntry, gameStateOk := GameStateComponentType.First(ecs.World)
	if !gameStateOk {
		return
	}
	gs := GameStateComponentType.Get(gameStateEntry)
	if gs.CurrentState == GameStateMessage {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			callback := gs.PostMessageCallback
			gs.PostMessageCallback = nil
			GameStateComponentType.Set(gameStateEntry, gs)
			if callback != nil {
				callback()
			} else {
				gsAfterCb := GameStateComponentType.Get(gameStateEntry)
				if gsAfterCb.CurrentState == GameStateMessage {
					gsAfterCb.CurrentState = StatePlaying
					GameStateComponentType.Set(gameStateEntry, gsAfterCb)
				}
			}
		}
	}
}
