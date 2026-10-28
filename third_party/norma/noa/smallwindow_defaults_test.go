// Added by ScopeWeaver on 2026-10-25: small-window default floor regression tests.
package noa

import "testing"

// DefaultConfig's floors were tuned for a 200K window. Below that reference the
// two default floors scale with the window; at or above it, and when the window
// is unknown, the tuned values are kept. GrowthCap never scales.
func TestDefaultConfigScalesSmallWindowFloors(t *testing.T) {
	cases := []struct {
		window             int
		growthFloor, minGF int
	}{
		{40_000, 10_000, 4_000},
		{128_000, 32_000, 12_800},
		{200_000, 50_000, 20_000},
		{400_000, 50_000, 20_000},
		// Unknown windows keep the tuned values: the adapter substitutes 200K
		// after construction, so scaling here would leave zero floors.
		{0, 50_000, 20_000},
		{-1, 50_000, 20_000},
	}
	for _, c := range cases {
		n := DefaultConfig(c.window).Nudge
		if n.GrowthFloor != c.growthFloor || n.MinGrowthFloor != c.minGF || n.GrowthCap != 50_000 {
			t.Errorf("DefaultConfig(%d): GrowthFloor=%d MinGrowthFloor=%d GrowthCap=%d, want %d/%d/50000",
				c.window, n.GrowthFloor, n.MinGrowthFloor, n.GrowthCap, c.growthFloor, c.minGF)
		}
	}

	// 40K worked example: adaptive = min(50000, max(10000, 2000)) = 10000,
	// cadence = max(4000, 0.45*10000) = 4500.
	cfg := DefaultConfig(40_000)
	if got := resolveAdaptiveGrowth(cfg.ModelContextLimit, cfg.Nudge); got != 10_000 {
		t.Errorf("40K adaptive growth = %d, want 10000", got)
	}
	if got := NudgeGrowthFloor(cfg); got != 4_500 {
		t.Errorf("40K cadence floor = %d, want 4500", got)
	}
}

// Explicitly configured floors are literal: only DefaultConfig scales, never the
// calculations, so a caller who sets 50000 on a 40K window gets 50000.
func TestExplicitNudgeFloorsStayLiteral(t *testing.T) {
	cfg := DefaultConfig(200_000)
	cfg.ModelContextLimit = 40_000
	if got := resolveAdaptiveGrowth(cfg.ModelContextLimit, cfg.Nudge); got != 50_000 {
		t.Errorf("explicit GrowthFloor 50000 on a 40K window resolved to %d, want 50000", got)
	}
	if got := NudgeGrowthFloor(cfg); got != 22_500 {
		t.Errorf("explicit floors on a 40K window gave cadence %d, want 22500", got)
	}
}
