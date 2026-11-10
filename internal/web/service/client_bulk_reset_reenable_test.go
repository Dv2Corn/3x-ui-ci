package service

import (
	"testing"
	"time"
)

// TestBulkResetTraffic_ReenablesDisabledClient covers the re-enable branch of
// BulkResetTraffic: a disabled client whose traffic is bulk-reset must come back
// enabled with its counters zeroed, in all three enable locations. This is the
// path whose s.Update failure was previously swallowed silently.
func TestBulkResetTraffic_ReenablesDisabledClient(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	email := "reset-reenable@x"
	ib := seedLocalDisabledClient(t, svc, 52010, "", email, 1000, 0, 600, 500)

	affected, err := svc.BulkResetTraffic(inboundSvc, []string{email})
	if err != nil {
		t.Fatalf("BulkResetTraffic: %v", err)
	}
	if affected < 1 {
		t.Fatalf("affected = %d, want >= 1", affected)
	}

	assertEnableEverywhere(t, svc, inboundSvc, ib.Id, email, true)
	if tr := trafficOf(t, email); tr.Up != 0 || tr.Down != 0 {
		t.Fatalf("%s: traffic after reset up=%d down=%d, want 0/0", email, tr.Up, tr.Down)
	}
}

func TestResetAllTraffics_ReenablesDepletedClientEverywhere(t *testing.T) {
	setupBulkDB(t)
	svc := &ClientService{}
	inboundSvc := &InboundService{}

	depleted := seedLocalDisabledClient(t, svc, 52011, "", "spent-all@x", 1000, 0, 600, 500)
	banned := seedLocalDisabledClient(t, svc, 52012, "", "banned-all@x", 100000, 0, 600, 500)

	if _, err := svc.ResetAllTraffics(); err != nil {
		t.Fatalf("ResetAllTraffics: %v", err)
	}

	assertEnableEverywhere(t, svc, inboundSvc, depleted.Id, "spent-all@x", true)
	if recordEnableOf(t, svc, "banned-all@x") {
		t.Fatal("banned-all@x: an operator-disabled client was switched back on by Reset All Client Traffic")
	}
	if jsonClientEnable(t, inboundSvc, banned.Id, "banned-all@x") {
		t.Fatal("banned-all@x: an operator-disabled client was enabled in the inbound JSON")
	}
}

// A client that expired and spent its quota is kept off by the expiry alone, so
// no traffic reset may switch it back on; the counters are still zeroed.
func TestTrafficReset_KeepsExpiredDepletedClientOff(t *testing.T) {
	resets := []struct {
		name string
		run  func(svc *ClientService, inboundSvc *InboundService, inboundId int, email string) error
	}{
		{"client", func(svc *ClientService, inboundSvc *InboundService, _ int, email string) error {
			_, err := svc.ResetTrafficByEmail(inboundSvc, email)
			return err
		}},
		{"client on one inbound", func(_ *ClientService, inboundSvc *InboundService, inboundId int, email string) error {
			_, err := inboundSvc.ResetClientTraffic(inboundId, email)
			return err
		}},
		{"bulk", func(svc *ClientService, inboundSvc *InboundService, _ int, email string) error {
			_, err := svc.BulkResetTraffic(inboundSvc, []string{email})
			return err
		}},
		{"inbound", func(svc *ClientService, inboundSvc *InboundService, inboundId int, _ string) error {
			_, err := svc.ResetAllClientTraffics(inboundSvc, inboundId)
			return err
		}},
		{"all", func(svc *ClientService, _ *InboundService, _ int, _ string) error {
			_, err := svc.ResetAllTraffics()
			return err
		}},
	}
	for i, reset := range resets {
		t.Run(reset.name, func(t *testing.T) {
			setupBulkDB(t)
			svc := &ClientService{}
			inboundSvc := &InboundService{}

			email := "expired-spent@x"
			expired := time.Now().Add(-time.Hour).UnixMilli()
			ib := seedLocalDisabledClient(t, svc, 52020+i, "", email, 1000, expired, 600, 500)

			if err := reset.run(svc, inboundSvc, ib.Id, email); err != nil {
				t.Fatalf("reset: %v", err)
			}
			if tr := trafficOf(t, email); tr.Up != 0 || tr.Down != 0 {
				t.Fatalf("%s: traffic after reset up=%d down=%d, want 0/0", email, tr.Up, tr.Down)
			}
			assertEnableEverywhere(t, svc, inboundSvc, ib.Id, email, false)
		})
	}
}
