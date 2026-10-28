// Modified by ScopeWeaver on 2026-10-25: learn only unmistakable, smaller
// context windows, and never loosen a learned one.
package noaadapter

import (
	"regexp"
	"strconv"
	"sync"
)

// overflowState remembers what a context-overflow error taught us.
//
// The provider's real window is often smaller than the configured one (a proxy
// trims it, a deployment differs from the published spec). Rather than guessing
// again next turn, the error is mined for the actual number and the mechanical
// valve is armed so the next view is built to fit.
type overflowState struct {
	mu sync.Mutex
	// learnedWindow is the window size parsed out of an error, 0 if unknown.
	learnedWindow int
	// armed makes the next view build to the emergency floor.
	armed bool
}

func (o *overflowState) arm(configured int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.armed = true
	if o.learnedWindow == 0 {
		o.learnedWindow = configured
	}
}

func (o *overflowState) disarm() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.armed = false
}

// armedFloor is the token ceiling the next view must fit under, or 0 when not
// armed.
func (o *overflowState) armedFloor() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.armed || o.learnedWindow <= 0 {
		return 0
	}
	return int(float64(o.learnedWindow) * 0.95)
}

// learn records a window size parsed from an error message. Only a positive
// window below the configured one, and below anything learned before, is taken:
// a later, larger claim must not loosen a ceiling a provider already enforced.
func (o *overflowState) learn(window, configured int) {
	if window <= 0 || window >= configured {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.learnedWindow == 0 || window < o.learnedWindow {
		o.learnedWindow = window
	}
}

// windowPatterns extract the real context window from an overflow error.
// Providers phrase this several ways; each pattern captures the limit.
var windowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)maximum context length is (\d+)`),
	regexp.MustCompile(`(?i)context (?:window|length) (?:of|is) (\d+)`),
	regexp.MustCompile(`(?i)max(?:imum)?[ _-]?tokens?[^0-9]{0,20}(\d{4,})`),
	regexp.MustCompile(`(?i)limit(?:ed)? to (\d{4,}) tokens`),
}

// contextWindowPatterns are the windowPatterns that name the CONTEXT window
// outright. The max_tokens and "limited to" phrasings often name an output cap
// (an output-length error saying "too long ... max_tokens 8192" is classified as
// an overflow), so they are not evidence enough to shrink the window.
var contextWindowPatterns = windowPatterns[:2]

// learnableContextWindow extracts a window size only from an unmistakable
// context-limit statement, 0 otherwise.
func learnableContextWindow(msg string) int { return firstWindow(msg, contextWindowPatterns) }

// ParseOverflowWindow extracts a context window size from an error message.
func ParseOverflowWindow(msg string) int { return firstWindow(msg, windowPatterns) }

func firstWindow(msg string, patterns []*regexp.Regexp) int {
	for _, re := range patterns {
		if m := re.FindStringSubmatch(msg); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
