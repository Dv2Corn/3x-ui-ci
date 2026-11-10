//go:build lifecyclee2e

package nodee2e

import (
	"net/http"
	"testing"
	"time"
)

// A traffic reset turns back on a client its quota switched off, but not one
// whose deadline has also passed.
func TestE2ETrafficResetKeepsExpiredClientOff(t *testing.T) {
	p := startCorePanel(t)
	deadline := time.Now().Add(20 * time.Second).UnixMilli()
	p.addInboundWithClients(
		map[string]any{"email": "q7-expired", "enable": true, "totalGB": 1000, "expiryTime": deadline},
		map[string]any{"email": "q7-live", "enable": true, "totalGB": 1000, "expiryTime": time.Now().Add(30 * 24 * time.Hour).UnixMilli()},
	)
	for _, e := range []string{"q7-expired", "q7-live"} {
		p.call(http.MethodPost, "/panel/api/clients/updateTraffic/"+e, map[string]any{"upload": 600, "download": 500})
	}
	eventually(t, 60*time.Second, "the job switches both off and q7-expired's deadline passes", func() (bool, string) {
		return !p.clientState("q7-expired").Enable && !p.clientState("q7-live").Enable && time.Now().UnixMilli() > deadline, ""
	})
	waitTicks()

	// The next traffic tick switches a wrongly restored client off again, so
	// watch the state between the reset and that tick, not after it.
	want := map[string]bool{"q7-live": true, "q7-expired": false}
	for e, w := range want {
		t.Logf("before reset: %s enable=%v", e, p.clientState(e).Enable)
		p.call(http.MethodPost, "/panel/api/clients/resetTraffic/"+e, nil)
		seenOn := false
		for end := time.Now().Add(7 * time.Second); time.Now().Before(end); time.Sleep(200 * time.Millisecond) {
			seenOn = seenOn || p.clientState(e).Enable
		}
		st, tr := p.clientState(e), p.trafficState(e)
		t.Logf("after reset: %s enabled at any point in 7s=%v, now enable=%v (traffic row enable=%v) up=%d down=%d", e, seenOn, st.Enable, tr.Enable, tr.Up, tr.Down)
		if tr.Up != 0 || tr.Down != 0 {
			t.Errorf("%s: counters not zeroed: up=%d down=%d", e, tr.Up, tr.Down)
		}
		if seenOn != w || st.Enable != w || tr.Enable != w {
			t.Errorf("%s after reset: enabled at any point=%v, now enable=%v traffic enable=%v, want %v", e, seenOn, st.Enable, tr.Enable, w)
		}
	}
}
