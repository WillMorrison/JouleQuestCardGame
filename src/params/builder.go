package params

import "github.com/WillMorrison/JouleQuestCardGame/core"

// Builder provides a way to derive Params
type Builder struct {
	p Params
}

func (pb Builder) Build() Params {
	return pb.p
}

func BuilderFrom(p Params) *Builder {
	return &Builder{p: p}
}

func (pb *Builder) Capacity(rule CapacityRule, batteryPnL, fossilPnL, poolPnL core.PnLTable) *Builder {
	pb.p.CapacityRule = rule
	pb.p.BatteryCapacityPnL = batteryPnL
	pb.p.FossilCapacityPnL = fossilPnL
	pb.p.CapacityPoolPnL = poolPnL
	return pb
}

func (pb *Builder) PnL(batteryPnL, fossilPnL, renewablePnL core.PnLTable) *Builder {
	pb.p.BatteryArbitragePnL = batteryPnL
	pb.p.FossilWholesalePnL = fossilPnL
	pb.p.RenewablePnL = renewablePnL
	return pb
}

func (pb *Builder) CarbonTax(rule CarbonTaxRule, threshold int, cost int) *Builder {
	pb.p.CarbonTaxRule = rule
	pb.p.CarbonTaxThreshold = threshold
	pb.p.CarbonTaxCost = cost
	return pb
}

func (pb *Builder) EmissionsCap(cap int) *Builder {
	pb.p.EmissionsCap = cap
	return pb
}

func (pb *Builder) GenerationConstraint(rule GenerationConstraintRule, constraint int) *Builder {
	pb.p.GenerationConstraintRule = rule
	pb.p.GenerationConstraint = constraint
	return pb
}

func (pb *Builder) WinConditionRule(rule WinConditionRule, penetration int) *Builder {
	pb.p.WinConditionRule = rule
	pb.p.RenewablePenetration = penetration
	return pb
}

func (pb *Builder) RenewableCosts(build, scrap int) *Builder {
	pb.p.RenewableBuildCost = build
	pb.p.RenewableScrapCost = scrap
	return pb
}

func (pb *Builder) FossilCosts(build, scrap int) *Builder {
	pb.p.FossilBuildCost = build
	pb.p.FossilScrapCost = scrap
	return pb
}

func (pb *Builder) BatteryCosts(build, scrap int) *Builder {
	pb.p.BatteryBuildCost = build
	pb.p.BatteryScrapCost = scrap
	return pb
}

func (pb *Builder) InitialCash(cash int) *Builder {
	pb.p.InitialCash = cash
	return pb
}

func (pb *Builder) StartingAssets(a map[int]int) *Builder {
	pb.p.StartingFossilAssetsPerPlayer = a
	return pb
}
