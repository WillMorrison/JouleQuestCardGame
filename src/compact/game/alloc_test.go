package game

import (
	"math/bits"
	"math/rand/v2"
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/compact/params"
	"github.com/WillMorrison/JouleQuestCardGame/core"
)

func Benchmark_Game_Reset(b *testing.B) {
	g := Game{}
	for b.Loop() {
		g.Reset(5, params.Default)
	}
}

func Benchmark_Game_ApplyPlayerAction(b *testing.B) {
	var g Game
	g.Reset(4, params.Default)
	rng := rand.New(rand.NewPCG(0, 0))
	var resets, applications, playerIndex int
	for b.Loop() {
		// Reset games that are finished.
		if g.Status != core.GameStatusOngoing {
			g.Reset(4, params.Default)
			resets++
		}

		// Choose next player
		playerIndex = (playerIndex + 1) % 4
		actions := g.PossibleActionMask(int32(playerIndex))
		if actions == 0 {
			continue // This player isn't active, so skip
		}

		// Randomly select one of the possible actions and apply it.
		actionChoice := rng.IntN(bits.OnesCount32(actions))
		for i := 0; i <= ActionFinished; i++ {
			if actions&(1<<i) != 0 {
				if actionChoice == 0 {
					g.ApplyPlayerAction(int32(playerIndex), int32(i))
					applications++
					break
				}
				actionChoice--
			}
		}
	}
	// Just to make sure we actually did something
	b.ReportMetric(float64(resets), "games")
	b.ReportMetric(float64(applications), "actions")
}
