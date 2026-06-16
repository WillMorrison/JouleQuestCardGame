package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/WillMorrison/JouleQuestCardGame/core"
)

const (
	numPlayers = 4
	rngSeed    = 0
)

var actionNames = [...]string{
	"BuildRenewable",
	"BuildBattery",
	"BuildFossil",
	"ScrapRenewable",
	"ScrapBattery",
	"ScrapFossil",
	"TakeoverRenewable",
	"TakeoverBattery",
	"TakeoverFossil",
	"TakeoverScrapRenewable",
	"TakeoverScrapBattery",
	"TakeoverScrapFossil",
	"PledgeBattery",
	"PledgeFossil",
	"Finished",
}

func main() {
	wasmPath := flag.String("wasm", "", "path to joulequest.wasm reactor binary")
	flag.Parse()
	if *wasmPath == "" {
		fmt.Fprintln(os.Stderr, "usage: joulequest_wasm_execute -wasm <path>")
		flag.PrintDefaults()
		os.Exit(2)
	}

	ctx := context.Background()
	mod, err := loadWasmModule(ctx, *wasmPath)
	if err != nil {
		log.Fatalf("load wasm: %v", err)
	}

	if code, err := mod.Reset(ctx, numPlayers); err != nil {
		log.Fatalf("Reset: %v", err)
	} else if code != 0 {
		log.Fatalf("Reset: error code %d", code)
	}
	if code, err := mod.SetRNGSeed(ctx, rngSeed); err != nil {
		log.Fatalf("SetRNGSeed: %v", err)
	} else if code != 0 {
		log.Fatalf("SetRNGSeed: error code %d", code)
	}

	maxAction := mustI32(ctx, mod.MaxAction)
	rng := rand.New(rand.NewSource(rngSeed))
	lastRound := int32(-1)

	for {
		status, err := mod.GameStatus(ctx)
		if err != nil {
			log.Fatalf("GameStatus: %v", err)
		}
		if status != int32(core.GameStatusOngoing) {
			printGameState(ctx, mod)
			fmt.Printf("game over: status=%s reason=%s\n",
				core.GameStatus(status).String(),
				core.LossCondition(mustI32(ctx, mod.GameReason)).String())
			return
		}

		acted := false
		n, err := mod.NumPlayers(ctx)
		if err != nil {
			log.Fatalf("NumPlayers: %v", err)
		}
		for player := int32(0); player < n; player++ {
			mask, err := mod.PossibleActionsMask(ctx, player)
			if err != nil {
				log.Fatalf("PossibleActionsMask(%d): %v", player, err)
			}
			if mask == 0 {
				continue
			}

			action := randomActionFromMask(mask, maxAction, rng)
			code, err := mod.ApplyAction(ctx, player, action)
			if err != nil {
				log.Fatalf("ApplyAction(%d, %s): %v", player, actionNames[action], err)
			}
			if code != 0 {
				log.Fatalf("ApplyAction(%d, %s): error code %d", player, actionNames[action], code)
			}
			acted = true

			fmt.Printf("player %d applied %s\n", player, actionNames[action])
			printPlayerState(ctx, mod, player)

			round, err := mod.Round(ctx)
			if err != nil {
				log.Fatalf("Round: %v", err)
			}
			if round > lastRound {
				lastRound = round
				printGameState(ctx, mod)
				printSnapshot(ctx, mod)
			}
		}

		if !acted {
			log.Fatal("no legal actions while game is ongoing")
		}
	}
}

func mustI32(ctx context.Context, fn func(context.Context) (int32, error)) int32 {
	v, err := fn(ctx)
	if err != nil {
		log.Fatalf("wasm call: %v", err)
	}
	return v
}

func randomActionFromMask(mask uint32, maxAction int32, rng *rand.Rand) int32 {
	var legal []int32
	for action := int32(0); action <= maxAction; action++ {
		if mask&(1<<action) != 0 {
			legal = append(legal, action)
		}
	}
	return legal[rng.Intn(len(legal))]
}

func printPlayerState(ctx context.Context, mod *wasmModule, player int32) {
	fmt.Printf("  player %d: status=%s reason=%s money=%d assets=[ren=%d batA=%d batC=%d fosW=%d fosC=%d]\n",
		player,
		core.PlayerStatus(mustI32(ctx, func(ctx context.Context) (int32, error) {
			return mod.PlayerStatus(ctx, player)
		})).String(),
		core.LossCondition(mustI32(ctx, func(ctx context.Context) (int32, error) {
			return mod.PlayerLossReason(ctx, player)
		})).String(),
		mustI32(ctx, func(ctx context.Context) (int32, error) { return mod.PlayerMoney(ctx, player) }),
		mustI32(ctx, func(ctx context.Context) (int32, error) { return mod.PlayerRenewableAssets(ctx, player) }),
		mustI32(ctx, func(ctx context.Context) (int32, error) {
			return mod.PlayerBatteriesArbitrageAssets(ctx, player)
		}),
		mustI32(ctx, func(ctx context.Context) (int32, error) {
			return mod.PlayerBatteriesCapacityAssets(ctx, player)
		}),
		mustI32(ctx, func(ctx context.Context) (int32, error) {
			return mod.PlayerFossilsWholesaleAssets(ctx, player)
		}),
		mustI32(ctx, func(ctx context.Context) (int32, error) {
			return mod.PlayerFossilsCapacityAssets(ctx, player)
		}),
	)
}

func printGameState(ctx context.Context, mod *wasmModule) {
	fmt.Printf("game: status=%s reason=%s round=%d emissions=%d takeover=[ren=%d batA=%d batC=%d fosW=%d fosC=%d]\n",
		core.GameStatus(mustI32(ctx, mod.GameStatus)).String(),
		core.LossCondition(mustI32(ctx, mod.GameReason)).String(),
		mustI32(ctx, mod.Round),
		mustI32(ctx, mod.CarbonEmissions),
		mustI32(ctx, mod.TakeoverRenewableAssets),
		mustI32(ctx, mod.TakeoverBatteriesArbitrageAssets),
		mustI32(ctx, mod.TakeoverBatteriesCapacityAssets),
		mustI32(ctx, mod.TakeoverFossilsWholesaleAssets),
		mustI32(ctx, mod.TakeoverFossilsCapacityAssets),
	)
}

func printSnapshot(ctx context.Context, mod *wasmModule) {
	fmt.Printf("snapshot: priceVol=%d gridStab=%d assets=[ren=%d batA=%d batC=%d fosW=%d fosC=%d]\n",
		mustI32(ctx, mod.LastSnapshotPriceVolatility),
		mustI32(ctx, mod.LastSnapshotGridStability),
		mustI32(ctx, mod.LastSnapshotRenewableAssets),
		mustI32(ctx, mod.LastSnapshotBatteriesArbitrageAssets),
		mustI32(ctx, mod.LastSnapshotBatteriesCapacityAssets),
		mustI32(ctx, mod.LastSnapshotFossilsWholesaleAssets),
		mustI32(ctx, mod.LastSnapshotFossilsCapacityAssets),
	)
}
