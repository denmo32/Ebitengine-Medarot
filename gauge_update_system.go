// --- gauge_update_system.go ---
package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

type GaugeUpdateSystem struct {
	query *donburi.Query
}

func NewGaugeUpdateSystem() *GaugeUpdateSystem {
	return &GaugeUpdateSystem{
		query: donburi.NewQuery(filter.And(
			filter.Contains(StatusComponentType, PartsComponentType, ActionComponentType),
			filter.Not(filter.Contains(BrokenTag)),
		)),
	}
}

func (sys *GaugeUpdateSystem) Update(ecs *ecs.ECS) {
	gs, gsOk := GameStateComponentType.First(ecs.World)
	config, cfgOk := ConfigComponentType.First(ecs.World)
	if !gsOk || !cfgOk || GameStateComponentType.Get(gs).CurrentState == GameStateMessage {
		return // メッセージ表示中はゲージを更新しない
	}
	balanceCfg := ConfigComponentType.Get(config).GameConfig.Balance

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		status := StatusComponentType.Get(entry)
		parts := PartsComponentType.Get(entry)
		action := ActionComponentType.Get(entry)

		// 頭部が破壊されている場合は本体も破壊
		if head, ok := parts.Parts[PartSlotHead]; ok && head.IsBroken {
			entry.AddComponent(BrokenTag)
			status.State = StateBroken
			status.Gauge = 0
			StatusComponentType.Set(entry, status)
			return
		}

		// ゲージを進めるための基礎値（パーツのチャージ/クールダウンと脚部の推進）
		var baseStat, legPropulsion int
		if legs, ok := parts.Parts[PartSlotLegs]; ok && !legs.IsBroken {
			legPropulsion = legs.Propulsion
		}

		selectedPart, partExists := parts.Parts[action.SelectedPartKey]
		isValidPartSelected := partExists && !selectedPart.IsBroken

		switch status.State {
		case StateActionCharging:
			if !isValidPartSelected { // 充填中にパーツが壊れた
				resetToActionSelect(entry, status, action)
				return
			}
			baseStat = selectedPart.Charge
		case StateActionCooldown:
			if !isValidPartSelected { // 冷却中にパーツが壊れた
				resetToActionSelect(entry, status, action)
				return
			}
			baseStat = selectedPart.Cooldown
		default:
			return // 他の状態ではゲージは進まない
		}

		// ゲージ更新
		moveSpeed := (float64(baseStat) + float64(legPropulsion)*balanceCfg.Time.PropulsionEffectRate) / balanceCfg.Time.OverallTimeDivisor
		status.Gauge += moveSpeed

		// ゲージ満タン時の処理
		if status.Gauge >= 100.0 {
			status.Gauge = 100.0 // 100を超えないように
			if status.State == StateActionCharging {
				status.State = StateReadyToExecuteAction
				entry.RemoveComponent(ActionChargingTag)
				entry.AddComponent(ReadyToExecuteActionTag)
			} else if status.State == StateActionCooldown {
				resetToActionSelect(entry, status, action)
			}
		}
		StatusComponentType.Set(entry, status)
	})
}

// resetToActionSelect はメダロットの状態を行動選択可能に戻します。
func resetToActionSelect(entry *donburi.Entry, status *StatusComponent, action *ActionComponent) {
	status.State = StateReadyToSelectAction
	status.Gauge = 100.0 // 行動選択は即時可能
	status.IsEvasionDisabled = false
	status.IsDefenseDisabled = false
	entry.RemoveComponent(ActionChargingTag)
	entry.RemoveComponent(ActionCooldownTag)

	action.TargetedMedarot = donburi.Entity(0)
	action.SelectedPartKey = ""
	ActionComponentType.Set(entry, action)
}
