// Added by ScopeWeaver on 2026-10-25: verify worthwhile count-triggered tier consolidation.
package noa

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func countBenefitInput(tier Tier, pending int) DecideNudgeInput {
	cfg := DefaultConfig(128000)
	st := CreateInitialState("count-benefit", "/unused")
	st.Nudge.LastPerMessageNudgeTokens = 20000
	n := cfg.Tiers.Tier2Trigger
	if tier == 3 {
		n = cfg.Tiers.Tier3Trigger
	}
	for i := range n {
		b := block(fmt.Sprintf("source-%d", i), tier-1, "gone")
		tokens := pending / n
		if i < pending%n {
			tokens++
		}
		b.Summary = strings.Repeat("x", tokens*4)
		st.Blocks = append(st.Blocks, b)
	}
	ignored := block("inactive", tier-1, "gone")
	ignored.Active = false
	ignored.Summary = strings.Repeat("x", 100000)
	st.Blocks = append(st.Blocks, ignored)
	return DecideNudgeInput{Config: cfg, State: st, TokenCount: 40000}
}

func TestNudgeCountConsolidatesWorthwhileLowUsageSummaries(t *testing.T) {
	for _, tier := range []Tier{2, 3} {
		t.Run(fmt.Sprintf("tier_%d", tier), func(t *testing.T) {
			in := countBenefitInput(tier, 6000)
			d := DecideNudge(in)
			if !d.ShouldInject || d.Tier != tier {
				t.Fatalf("want tier %d below usage floor with worthwhile summaries: %+v; %s", tier, d.Breakdown, d.Reason)
			}
			if d.Breakdown.Usage != 0.3125 || d.Breakdown.Growth != 20000 ||
				d.Breakdown.GrowthFloor != 14400 || d.Breakdown.AdaptiveGrowth != 32000 ||
				d.Breakdown.MinPressureBenefit != 5000 || d.Breakdown.PendingT1 != 0 {
				t.Fatalf("unexpected decision picture: %+v", d.Breakdown)
			}
			pending, count := d.Breakdown.PendingT2, d.Breakdown.CountT1
			if tier == 3 {
				pending, count = d.Breakdown.PendingT3, d.Breakdown.CountT2
			}
			want := in.State.Blocks[:len(in.State.Blocks)-1]
			if pending != 6000 || count != len(want) || !reflect.DeepEqual(d.TierTargetBlocks, want) {
				t.Fatalf("pending=%d count=%d targets=%+v, want 6000/%d and exactly active source blocks", pending, count, d.TierTargetBlocks, len(want))
			}
			t.Logf("tier=%d breakdown=%+v reason=%s", d.Tier, d.Breakdown, d.Reason)
		})
	}
}

func TestNudgeCountBenefitKeepsCadenceAndConfiguration(t *testing.T) {
	for _, tier := range []Tier{2, 3} {
		for _, tc := range []struct {
			name    string
			pending int
			mutate  func(*DecideNudgeInput)
			want    Tier
		}{
			{name: "below benefit", pending: 4999},
			{name: "below global cadence", pending: 6000, mutate: func(in *DecideNudgeInput) {
				in.State.Nudge.LastPerMessageNudgeTokens = 30000
			}},
			{name: "below per tier cadence", pending: 6000, mutate: func(in *DecideNudgeInput) {
				in.State.Nudge.LastShownByTier[tier] = 30000
			}},
			{name: "tiers disabled", pending: 6000, mutate: func(in *DecideNudgeInput) {
				in.Config.Tiers.Enabled = false
			}},
			{name: "custom benefit not met", pending: 6000, mutate: func(in *DecideNudgeInput) {
				benefit := 7000
				in.Config.Nudge.MinPressureBenefitTokens = &benefit
			}},
			{name: "custom benefit met", pending: 7000, want: tier, mutate: func(in *DecideNudgeInput) {
				benefit := 7000
				in.Config.Nudge.MinPressureBenefitTokens = &benefit
			}},
			{name: "literal zero custom benefit", want: tier, mutate: func(in *DecideNudgeInput) {
				benefit := 0
				in.Config.Nudge.MinPressureBenefitTokens = &benefit
			}},
			{name: "original usage path", pending: 1000, want: tier, mutate: func(in *DecideNudgeInput) {
				in.TokenCount = 60000
			}},
			{name: "tier one priority", pending: 6000, want: 1, mutate: func(in *DecideNudgeInput) {
				in.Recommendation.CompressibleRanges = []RangeInfo{{Tokens: 32000, Chars: 128000}}
			}},
		} {
			t.Run(fmt.Sprintf("tier_%d/%s", tier, tc.name), func(t *testing.T) {
				in := countBenefitInput(tier, tc.pending)
				if tc.mutate != nil {
					tc.mutate(&in)
				}
				d := DecideNudge(in)
				if d.ShouldInject != (tc.want != 0) || d.Tier != tc.want {
					t.Fatalf("inject=%v tier=%d, want tier=%d: %+v; %s", d.ShouldInject, d.Tier, tc.want, d.Breakdown, d.Reason)
				}
			})
		}
	}
}
