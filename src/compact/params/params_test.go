package params

import (
	"testing"

	"github.com/WillMorrison/JouleQuestCardGame/params"
)

func TestFromLegacyStartingFossils(t *testing.T) {
	c, err := FromLegacy(params.Default)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.StartingFossils(4); got != 5 {
		t.Fatalf("StartingFossils(4) = %d, want 5", got)
	}
	if got := c.StartingFossils(99); got != 0 {
		t.Fatalf("StartingFossils(99) = %d, want 0", got)
	}
}
