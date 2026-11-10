//go:build lifecyclee2e

package nodee2e

import (
	"net/http"
	"testing"
	"time"
)

// A renewal turns back on a client the lifecycle job switched off, but not one
// the operator switched off while it was live and whose deadline passed later.
func TestE2ERenewalKeepsOperatorDisable(t *testing.T) {
	p := startCorePanel(t)
	deadline := time.Now().Add(20 * time.Second).UnixMilli()
	emails := []string{"q6-by-job", "q6-by-hand", "q6-by-hand-bulk"}
	clients := make([]map[string]any, 0, len(emails))
	for _, e := range emails {
		clients = append(clients, map[string]any{"email": e, "enable": true, "expiryTime": deadline})
	}
	p.addInboundWithClients(clients...)

	p.call(http.MethodPost, "/panel/api/clients/update/q6-by-hand", map[string]any{"enable": false})
	p.call(http.MethodPost, "/panel/api/clients/update/q6-by-hand-bulk", map[string]any{"enable": false})
	eventually(t, 60*time.Second, "the job switches q6-by-job off after its deadline", func() (bool, string) {
		st := p.clientState("q6-by-job")
		return !st.Enable && time.Now().UnixMilli() > deadline, ""
	})
	waitTicks()
	for _, e := range emails {
		t.Logf("expired, before renewal: %s enable=%v", e, p.clientState(e).Enable)
	}

	renewed := time.Now().Add(30 * 24 * time.Hour).UnixMilli()
	p.call(http.MethodPost, "/panel/api/clients/update/q6-by-job", map[string]any{"expiryTime": renewed})
	p.call(http.MethodPost, "/panel/api/clients/update/q6-by-hand", map[string]any{"expiryTime": renewed})
	p.call(http.MethodPost, "/panel/api/clients/bulkAdjust", map[string]any{"emails": []string{"q6-by-hand-bulk"}, "addDays": 30})
	waitTicks()

	want := map[string]bool{"q6-by-job": true, "q6-by-hand": false, "q6-by-hand-bulk": false}
	for _, e := range emails {
		st, tr := p.clientState(e), p.trafficState(e)
		t.Logf("after renewal: %s enable=%v (traffic row enable=%v) expiry=%d", e, st.Enable, tr.Enable, st.ExpiryTime)
		if st.Enable != want[e] || tr.Enable != want[e] {
			t.Errorf("%s after renewal: enable=%v traffic enable=%v, want %v", e, st.Enable, tr.Enable, want[e])
		}
	}
}
