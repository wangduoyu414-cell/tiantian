//go:build windows

package wxkey

import (
	"strconv"
	"strings"
	"time"
)

// Per-route scan budgets. Every capture route gets its own slice of the total
// scan budget so one slow route can never starve the rest (a route that burns
// its share reports "budget-timeout" and the chain continues; only the global
// deadline stops the run). Env override per route:
// WECHAT_CLI_BUDGET_<NAME> (name uppercased, '-' -> '_'), seconds or Go duration.
var windowsRouteShares = map[string]time.Duration{
	"capture-file":     10 * time.Second,
	"env-passphrase":   15 * time.Second,
	"primary-literal":  60 * time.Second,
	"signatures":       30 * time.Second,
	"wxkey-dll":        45 * time.Second,
	"restart-capture":  10 * time.Minute, // waits for the user to log in
	"hook":             60 * time.Second,
	"d0-object":        30 * time.Second,
	"d1-salt":          45 * time.Second,
	"d2-heap":          60 * time.Second,
	"a-poll":           45 * time.Second,
	"brute-passphrase": 30 * time.Second,
}

// windowsRouteShare returns the budget for one route.
func windowsRouteShare(name string) time.Duration {
	env := "WECHAT_CLI_BUDGET_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if raw := strings.TrimSpace(firstEnv(env)); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	if d, ok := windowsRouteShares[name]; ok {
		return d
	}
	return 30 * time.Second
}

// windowsRouteDeadline computes a route's deadline: now+share, capped at the
// global deadline. A zero global deadline disables capping.
func windowsRouteDeadline(globalDeadline time.Time, share time.Duration) time.Time {
	d := time.Now().Add(share)
	if !globalDeadline.IsZero() && d.After(globalDeadline) {
		return globalDeadline
	}
	return d
}

// windowsRouteStat is one route's recorded outcome in the setup stats.
type windowsRouteStat struct {
	Name        string `json:"name"`
	DurationMs  int64  `json:"duration_ms"`
	NewCoverage int    `json:"new_coverage"`
	// Exit: completed | error | budget-timeout | global-timeout |
	// skipped-not-applicable | skipped-global-timeout
	Exit string `json:"exit"`
}
