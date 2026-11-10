//go:build lifecyclee2e

package nodee2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// startCorePanel runs one panel with the real Xray core from XUI_E2E_XRAY_DIR;
// the traffic job, which disables, renews and activates clients, needs a running core.
func startCorePanel(t *testing.T) *panel {
	t.Helper()
	bin := panelBinary(t)
	xrayDir := os.Getenv("XUI_E2E_XRAY_DIR")
	if xrayDir == "" {
		t.Skip("XUI_E2E_XRAY_DIR not set")
	}
	p := newPanel(t, bin, "panel")
	for src, dst := range map[string]string{"xray": "xray-linux-amd64", "geoip.dat": "geoip.dat", "geosite.dat": "geosite.dat"} {
		b, err := os.ReadFile(filepath.Join(xrayDir, src))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, "bin", dst), b, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p.start()
	eventually(t, settleTimeout, "xray running", func() (bool, string) {
		var st struct {
			Xray struct {
				State string `json:"state"`
			} `json:"xray"`
		}
		_ = json.Unmarshal(p.call(http.MethodGet, "/panel/api/server/status", nil), &st)
		return st.Xray.State == "running", st.Xray.State
	})
	return p
}

type clientState struct {
	Enable     bool  `json:"enable"`
	ExpiryTime int64 `json:"expiryTime"`
}

type trafficState struct {
	Enable     bool  `json:"enable"`
	Up         int64 `json:"up"`
	Down       int64 `json:"down"`
	ExpiryTime int64 `json:"expiryTime"`
}

func (p *panel) clientState(email string) clientState {
	p.t.Helper()
	var v struct {
		Client clientState `json:"client"`
	}
	if err := json.Unmarshal(p.call(http.MethodGet, "/panel/api/clients/get/"+email, nil), &v); err != nil {
		p.t.Fatalf("decode client %s: %v", email, err)
	}
	return v.Client
}

func (p *panel) trafficState(email string) trafficState {
	p.t.Helper()
	var v trafficState
	if err := json.Unmarshal(p.call(http.MethodGet, "/panel/api/clients/traffic/"+email, nil), &v); err != nil {
		p.t.Fatalf("decode traffic %s: %v", email, err)
	}
	return v
}

// addInboundWithClients adds a local VLESS inbound and one client per entry.
func (p *panel) addInboundWithClients(clients ...map[string]any) int {
	p.t.Helper()
	var ib inboundView
	if err := json.Unmarshal(p.call(http.MethodPost, "/panel/api/inbounds/add", vlessInbound("lifecycle", freePort(p.t), nil)), &ib); err != nil || ib.Id == 0 {
		p.t.Fatalf("decode inbound add: %v", err)
	}
	for _, c := range clients {
		p.call(http.MethodPost, "/panel/api/clients/add", map[string]any{"client": c, "inboundIds": []int{ib.Id}})
	}
	return ib.Id
}

// waitTicks lets the 5s traffic job run at least twice.
func waitTicks() { time.Sleep(12 * time.Second) }
