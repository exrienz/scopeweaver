// Modified by ScopeWeaver on 2026-10-25: replace the stale small-window tier-2
// limitation with a positive consolidation regression.
package noaadapter

import (
	"strings"
	"testing"

	"github.com/Autumn-27/norma/noa"
)

// Multi-tier compression is what keeps summaries from accumulating without
// bound: tier 1 replaces raw messages, tier 2 replaces tier-1 summaries, tier 3
// replaces tier-2 summaries.
//
// The ladder is driven entirely by the model: a tier-2 nudge names the blocks to
// consolidate and the model calls Compress on them. So "does tiering work" is
// really two questions — does the nudge fire, and can an agent act on what it
// says. These tests cover both, because a nudge nobody can parse is the same as
// no nudge at all.

// The end-to-end property: a long cooperative session advances past tier 1.
func TestLongSessionReachesTierTwo(t *testing.T) {
	// TestTierTwoIsReachableOnSmallWindows covers the 40K case.
	sess := simSession(t, 128_000)
	a := newSimAgent(t, sess, 1)
	for range 182 {
		a.work(4000)
		a.observe()
	}

	byTier := map[noa.Tier]int{}
	for _, b := range noa.ActiveBlocks(sess.State()) {
		byTier[b.Tier]++
	}
	if byTier[2] == 0 {
		t.Fatalf("a long cooperative session produced %d tier-1 blocks and no tier-2 block "+
			"(Tier2Trigger=%d). Either the tier-2 nudge stopped firing, or its target list "+
			"changed shape and tierTargets no longer reads it.",
			byTier[1], sess.Config().Tiers.Tier2Trigger)
	}
	t.Logf("active blocks by tier: %v", byTier)
}

// The tier nudge's target list is a different format from the block map
// (FormatTierTargetBlocks vs FormatBlockMap). An agent that reads the wrong one
// finds nothing and silently does nothing — the whole ladder then looks broken
// while every decision underneath it is correct.
//
// This pins the format the nudge actually emits, so a change to it fails here
// rather than showing up as "tiering doesn't work" much later.
func TestTierNudgeNamesItsTargetsInAReadableFormat(t *testing.T) {
	sess := simSession(t, 128_000)
	a := newSimAgent(t, sess, 1)

	for range 182 {
		a.work(4000)
		view := sess.View(a.history)
		nudge := findNudge(view)
		if nudge == "" {
			a.observe()
			continue
		}
		if !isTierNudge(nudge) {
			a.act(nudge)
			continue
		}
		targets := tierTargets(nudge)
		if len(targets) < 2 {
			t.Fatalf("a tier nudge named %d target blocks; an agent has nothing to act on.\n"+
				"nudge tail:\n%s", len(targets), tail(nudge, 600))
		}
		// Every named target must be a block that exists and is active, or the
		// model is being pointed at something it cannot compress.
		active := map[string]noa.Tier{}
		for _, b := range noa.ActiveBlocks(sess.State()) {
			active[b.BlockID] = b.Tier
		}
		for _, id := range targets {
			if _, ok := active[id]; !ok {
				t.Fatalf("the tier nudge names %s, which is not an active block", id)
			}
		}
		return
	}
	t.Skip("no tier nudge fired in this run")
}

// Tier 2 must keep working on a small window.
//
// ScopeWeaver: upstream pinned this as a documented limitation and asserted
// that tier 2 never fires at 40K, but upstream v0.4.3 already formed tier-2
// blocks on this exact fixture (active blocks map[1:26 2:2]) and the test asked
// to be replaced "by an assertion that it keeps working". The same 40K /
// 120-turn cooperative fixture now asserts the positive property: tier 2
// actually runs, every tier-2 block is smaller than the summaries it absorbed,
// and active summaries stay within a quarter of the window.
func TestTierTwoIsReachableOnSmallWindows(t *testing.T) {
	const window = 40_000
	sess := simSession(t, window)
	a := newSimAgent(t, sess, 1)
	for range 120 {
		a.work(4000)
		a.observe()
	}

	st := sess.State()
	byID := map[string]noa.CompressionBlock{}
	for _, b := range st.Blocks {
		byID[b.BlockID] = b
	}
	byTier := map[noa.Tier]int{}
	summaryTokens := 0
	for _, b := range noa.ActiveBlocks(st) {
		byTier[b.Tier]++
		own := noa.DefaultCountTokens(b.Summary)
		summaryTokens += own
		if b.Tier < 2 {
			continue
		}
		absorbed := 0
		for _, childID := range b.DirectBlockIDs {
			child, ok := byID[childID]
			if !ok {
				t.Fatalf("block %s names child %s, which is not in the ledger", b.BlockID, childID)
			}
			absorbed += noa.DefaultCountTokens(child.Summary)
		}
		if absorbed == 0 || own >= absorbed {
			t.Fatalf("tier-%d block %s costs %d tokens to replace %d tokens of child summaries; "+
				"consolidation reclaimed nothing", b.Tier, b.BlockID, own, absorbed)
		}
	}
	if byTier[2] == 0 {
		t.Fatalf("a %d window produced no active tier-2 block (blocks %v); small-window "+
			"consolidation regressed", window, byTier)
	}
	if summaryTokens > window/4 {
		t.Fatalf("active summaries hold %d tokens (>25%% of the %d window); consolidation is "+
			"not keeping summaries bounded", summaryTokens, window)
	}
	t.Logf("window=%d: active blocks %v holding %d summary tokens (%.0f%% of the window)",
		window, byTier, summaryTokens, float64(summaryTokens)*100/window)
}

// Consolidation must actually pay: a tier-2 block has to be smaller than the
// tier-1 summaries it absorbed, or the ladder costs turns and reclaims nothing.
func TestTierTwoConsolidationReclaimsTokens(t *testing.T) {
	sess := simSession(t, 200_000)
	a := newSimAgent(t, sess, 1)
	for range 285 {
		a.work(4000)
		a.observe()
	}
	st := sess.State()
	byID := map[string]noa.CompressionBlock{}
	for _, b := range st.Blocks {
		byID[b.BlockID] = b
	}

	checked := 0
	for _, b := range noa.ActiveBlocks(st) {
		if b.Tier < 2 {
			continue
		}
		absorbed := 0
		for _, childID := range b.DirectBlockIDs {
			child, ok := byID[childID]
			if !ok {
				t.Fatalf("block %s names child %s, which is not in the ledger", b.BlockID, childID)
			}
			absorbed += noa.DefaultCountTokens(child.Summary)
		}
		own := noa.DefaultCountTokens(b.Summary)
		if absorbed == 0 {
			t.Fatalf("tier-%d block %s absorbed no child summaries", b.Tier, b.BlockID)
		}
		if own >= absorbed {
			t.Fatalf("tier-%d block %s costs %d tokens to replace %d tokens of tier-1 summaries; "+
				"consolidation made the context bigger", b.Tier, b.BlockID, own, absorbed)
		}
		checked++
		t.Logf("%s (T%d): %d child summaries, %d → %d tokens (%.1fx)",
			b.BlockID, b.Tier, len(b.DirectBlockIDs), absorbed, own, float64(absorbed)/float64(own))
	}
	if checked == 0 {
		t.Skip("no tier-2 block formed in this run")
	}
}

func isTierNudge(nudge string) bool {
	return strings.Contains(nudge, "DISTILLATION TRIGGER") ||
		strings.Contains(nudge, "CONDENSATION TRIGGER")
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
