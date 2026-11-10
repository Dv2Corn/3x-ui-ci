// e2e helper for queue item 5 (not part of the patch).
// "seed DB DIR": a WireGuard (Xray) and an AmneziaWG inbound, each with one good
// client and one client whose public key is malformed; keys go to DIR.
// "handshake ADDR SERVERPUB CLIENTPRIV LOCALIP AWG(0|1)": a real client handshake.
package main

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amnezia-vpn/amneziawg-go/v3/conn"
	"github.com/amnezia-vpn/amneziawg-go/v3/device"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

const awgObf = "jc=4\njmin=40\njmax=70\ns1=20\ns2=30\n"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func keypair() (string, string) {
	priv, pub, err := wireguard.GenerateWireguardKeypair()
	must(err)
	return priv, pub
}

func seed(dbPath, dir string) {
	must(database.InitDB(dbPath))
	db := database.GetDB()
	write := func(name, v string) { must(os.WriteFile(filepath.Join(dir, name), []byte(v), 0o600)) }

	wgServerPriv, wgServerPub := keypair()
	wgClientPriv, wgClientPub := keypair()
	wgClients := []model.Client{
		{Email: "good-wg@e2e", Enable: true, PublicKey: wgClientPub, AllowedIPs: []string{"10.0.0.2/32"}},
		{Email: "bad-wg@e2e", Enable: true, PublicKey: "not-a-key", AllowedIPs: []string{"10.0.0.3/32"}},
	}
	wgSettings, _ := json.Marshal(map[string]any{"secretKey": wgServerPriv, "mtu": 1420, "clients": wgClients})
	wg := &model.Inbound{Tag: "wg-e2e", Enable: true, Port: 51820, Listen: "127.0.0.1", Protocol: model.WireGuard, Settings: string(wgSettings)}
	must(db.Create(wg).Error)
	must((&service.ClientService{}).SyncInbound(nil, wg.Id, wgClients))
	write("wg.server.pub", wgServerPub)
	write("wg.client.priv", wgClientPriv)

	awgServerPriv, awgServerPub := keypair()
	awgClientPriv, awgClientPub := keypair()
	awgClients := []model.Client{
		{Email: "good-awg@e2e", Enable: true, PublicKey: awgClientPub, AllowedIPs: []string{"10.8.1.2/32"}},
		{Email: "bad-awg@e2e", Enable: true, PublicKey: "not-a-key", AllowedIPs: []string{"10.8.1.3/32"}},
	}
	awgSettings, _ := json.Marshal(map[string]any{
		"server": map[string]any{
			"privateKey": awgServerPriv, "publicKey": awgServerPub, "subnetIp": "10.8.1.0", "subnetCidr": 24,
			"jc": 4, "jmin": 40, "jmax": 70, "s1": 20, "s2": 30,
		},
		"clients": awgClients,
	})
	awg := &model.Inbound{Tag: "awg-e2e", Enable: true, Port: 51830, Listen: "127.0.0.1", Protocol: model.AmneziaWG, Settings: string(awgSettings)}
	must(db.Create(awg).Error)
	must((&service.ClientService{}).SyncInbound(nil, awg.Id, awgClients))
	write("awg.server.pub", awgServerPub)
	write("awg.client.priv", awgClientPriv)
}

func handshake(addr, serverPub, clientPriv, localIP string, awg bool) error {
	tun, tnet, err := netstack.CreateNetTUN([]netip.Addr{netip.MustParseAddr(localIP)}, nil, 1380)
	if err != nil {
		return err
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, "client: "))
	defer dev.Close()
	privHex, err := wireguard.KeyToHex(clientPriv)
	if err != nil {
		return err
	}
	pubHex, err := wireguard.KeyToHex(serverPub)
	if err != nil {
		return err
	}
	conf := "private_key=" + privHex + "\n"
	if awg {
		conf += awgObf
	}
	conf += "public_key=" + pubHex + "\nendpoint=" + addr + "\nallowed_ip=0.0.0.0/0\npersistent_keepalive_interval=1\n"
	if err := dev.IpcSet(conf); err != nil {
		return err
	}
	if err := dev.Up(); err != nil {
		return err
	}
	if c, err := tnet.Dial("udp", "10.255.0.1:53"); err == nil {
		_, _ = c.Write([]byte("x"))
		c.Close()
	}
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(500 * time.Millisecond) {
		state, err := dev.IpcGet()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(state, "\n") {
			if strings.HasPrefix(line, "last_handshake_time_sec=") && line != "last_handshake_time_sec=0" {
				return nil
			}
		}
	}
	return fmt.Errorf("no handshake with %s within 15 s", addr)
}

func main() {
	switch os.Args[1] {
	case "seed":
		seed(os.Args[2], os.Args[3])
	case "handshake":
		if err := handshake(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6] == "1"); err != nil {
			fmt.Println("FAIL:", err)
			os.Exit(1)
		}
		fmt.Println("handshake ok:", os.Args[2])
	}
}
