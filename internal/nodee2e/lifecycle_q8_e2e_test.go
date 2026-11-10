//go:build lifecyclee2e

package nodee2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// "Start after first use" must start on the client's traffic, not on the next
// poll after it was re-armed while its counter already exists in the core.
func TestE2EDelayedStartWaitsForTraffic(t *testing.T) {
	p := startCorePanel(t)
	const email = "q8-rearmed"
	ibPort := freePort(t)
	var ib inboundView
	if err := json.Unmarshal(p.call(http.MethodPost, "/panel/api/inbounds/add", vlessInbound("q8", ibPort, nil, vlessClient(email))), &ib); err != nil {
		t.Fatal(err)
	}
	socks := startVlessClient(t, ibPort, newUUID(email))

	proxy, _ := url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", socks))
	hc := &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}}
	use := func() {
		t.Helper()
		eventually(t, settleTimeout, "a request through the client", func() (bool, string) {
			resp, err := hc.Get("https://www.gstatic.com/generate_204")
			if err != nil {
				return false, err.Error()
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return true, ""
		})
	}
	// The first stats poll only takes a baseline, so keep using until it counts.
	eventually(t, 60*time.Second, "the core counted the client's traffic", func() (bool, string) {
		use()
		tr := p.trafficState(email)
		return tr.Up+tr.Down > 0, fmt.Sprintf("up=%d down=%d", tr.Up, tr.Down)
	})

	// Let the polls account for every byte already sent before re-arming.
	hc.CloseIdleConnections()
	waitTicks()

	idle := p.trafficState(email)
	const sevenDays = int64(7 * 24 * time.Hour / time.Millisecond)
	p.call(http.MethodPost, "/panel/api/clients/update/"+email, map[string]any{"expiryTime": -sevenDays})
	waitTicks()
	waitTicks()
	st, tr := p.clientState(email), p.trafficState(email)
	t.Logf("re-armed, idle for 4+ polls: up/down %d/%d -> %d/%d, expiry=%d (traffic row %d), want %d",
		idle.Up, idle.Down, tr.Up, tr.Down, st.ExpiryTime, tr.ExpiryTime, -sevenDays)
	if tr.Up != idle.Up || tr.Down != idle.Down {
		t.Fatalf("traffic moved while idle: %d/%d -> %d/%d", idle.Up, idle.Down, tr.Up, tr.Down)
	}
	if st.ExpiryTime != -sevenDays || tr.ExpiryTime != -sevenDays {
		t.Errorf("an idle poll started the period: expiry=%d traffic expiry=%d, want %d", st.ExpiryTime, tr.ExpiryTime, -sevenDays)
	}

	before := time.Now().UnixMilli()
	use()
	eventually(t, settleTimeout, "first use starts the period", func() (bool, string) {
		st := p.clientState(email)
		return st.ExpiryTime > 0, fmt.Sprintf("expiry=%d", st.ExpiryTime)
	})
	st = p.clientState(email)
	t.Logf("after first use: expiry=%d (now+%.2f days)", st.ExpiryTime, float64(st.ExpiryTime-before)/float64(24*time.Hour/time.Millisecond))
	if st.ExpiryTime < before+sevenDays || st.ExpiryTime > before+sevenDays+60_000 {
		t.Errorf("expiry after first use = %d, want ~%d", st.ExpiryTime, before+sevenDays)
	}
}

// startVlessClient runs a second core as a VLESS client with a local SOCKS port.
func startVlessClient(t *testing.T, serverPort int, id string) int {
	t.Helper()
	socks := freePort(t)
	inbound := map[string]any{"listen": "127.0.0.1", "port": socks, "protocol": "socks"}
	user := map[string]any{"id": id, "encryption": "none"}
	server := map[string]any{"address": "127.0.0.1", "port": serverPort, "users": []any{user}}
	outbound := map[string]any{"protocol": "vless", "settings": map[string]any{"vnext": []any{server}}}
	b, _ := json.Marshal(map[string]any{"inbounds": []any{inbound}, "outbounds": []any{outbound}})
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(os.Getenv("XUI_E2E_XRAY_DIR"), "xray"), "run", "-c", path)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	return socks
}
