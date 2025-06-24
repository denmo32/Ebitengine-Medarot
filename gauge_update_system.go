package main

import (
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

// GaugeUpdateSystem はメダロットのゲージを更新します。
type GaugeUpdateSystem struct {
	query *donburi.Query
}

func NewGaugeUpdateSystem() *GaugeUpdateSystem {
	return &GaugeUpdateSystem{
		query: donburi.NewQuery(
			filter.And(
				filter.Contains(StatusComponentType),
				filter.Contains(PartsComponentType),
				filter.Contains(CMedal), // CMedal を使用
				filter.Contains(ActionComponentType),
				filter.Not(filter.Contains(BrokenTag)),
			)),
	}
}

func (sys *GaugeUpdateSystem) Update(ecs *ecs.ECS) {
	configEntry, ok := ConfigComponentType.First(ecs.World)
	if !ok {
		return
	}
	gameConfig := ConfigComponentType.Get(configEntry).GameConfig
	if gameConfig == nil {
		return
	}

	sys.query.Each(ecs.World, func(entry *donburi.Entry) {
		status := StatusComponentType.Get(entry)
		parts := PartsComponentType.Get(entry)
		actionComp := ActionComponentType.Get(entry)

		headPart, headExists := parts.Parts[PartSlotHead]
		if headExists && headPart.IsBroken {
			if !entry.HasComponent(BrokenTag) {
				entry.AddComponent(BrokenTag)
				status.State = StateBroken
				status.Gauge = 0
				StatusComponentType.Set(entry, status)
			}
			return
		}

		legsPart := parts.Parts[PartSlotLegs]
		legPropulsion := 0
		if legsPart != nil && !legsPart.IsBroken {
			legPropulsion = legsPart.Propulsion
		}

		var selectedPart *Part
		if status.State == StateActionCharging || status.State == StateActionCooldown {
			if actionComp.SelectedPartKey != "" {
				part, exists := parts.Parts[actionComp.SelectedPartKey]
				if exists && !part.IsBroken {
					selectedPart = part
				}
			}
		}

		if selectedPart == nil {
			if status.State == StateActionCharging || status.State == StateActionCooldown {
				status.State = StateReadyToSelectAction
				status.Gauge = 100.0
				actionComp.SelectedPartKey = ""
				status.IsEvasionDisabled = false
				status.IsDefenseDisabled = false
				entry.RemoveComponent(ActionChargingTag)
				entry.RemoveComponent(ActionCooldownTag)
				StatusComponentType.Set(entry, status)
				ActionComponentType.Set(entry, actionComp)
			}
			return
		}

		stat := 0
		if status.State == StateActionCharging {
			stat = selectedPart.Charge
		} else if status.State == StateActionCooldown {
			stat = selectedPart.Cooldown
		} else {
			return
		}

		cfgBalance := gameConfig.Balance
		moveSpeed := (float64(stat) + float64(legPropulsion)*cfgBalance.Time.PropulsionEffectRate) / cfgBalance.Time.OverallTimeDivisor
		status.Gauge += moveSpeed

		if status.Gauge >= 100.0 {
			status.Gauge = 100.0
			if status.State == StateActionCharging {
				status.State = StateReadyToExecuteAction
				entry.RemoveComponent(ActionChargingTag)
				entry.AddComponent(ReadyToExecuteActionTag)
			} else if status.State == StateActionCooldown {
				status.State = StateReadyToSelectAction
				entry.RemoveComponent(ActionCooldownTag)
				status.IsEvasionDisabled = false
				status.IsDefenseDisabled = false
				actionComp.TargetedMedarot = donburi.Entity(0) // ★ donburi.Entity{} を donburi.Entity(0) に修正
				actionComp.SelectedPartKey = ""
				ActionComponentType.Set(entry, actionComp)
			}
		}
		StatusComponentType.Set(entry, status)
	})
}
