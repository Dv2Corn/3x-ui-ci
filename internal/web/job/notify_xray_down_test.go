package job

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/discord"
)

// The scheduled Discord report and its backup were skipped while Xray was
// down, exactly when the admin needs them.
func TestDiscordNotifyJobSendsReportAndBackupWhileXrayIsDown(t *testing.T) {
	// The backup is taken next to config.GetDBPath(), so the test DB must live there.
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	dbtest.InitDB(t, config.GetDBPath())
	if (&service.XrayService{}).IsXrayRunning() {
		t.Fatal("test needs Xray to be down")
	}
	settings := service.SettingService{}
	for _, err := range []error{
		settings.SetDiscordBotEnable(true),
		settings.SetDiscordBotToken("test-token"),
		settings.SetDiscordChannelId("123456789012345678"),
		settings.SetDiscordBotBackup(true),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	var mu sync.Mutex
	var report, backup bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		if r.Method == http.MethodPost && r.URL.Path == "/channels/123456789012345678/messages" {
			report = report || strings.Contains(string(body), `"embeds"`)
			backup = backup || strings.Contains(string(body), `name="files[0]"`) // the DB file
		}
		mu.Unlock()
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	t.Cleanup(srv.Close)
	ds := discord.NewDiscordService(settings)
	ds.SetBaseURL(srv.URL)
	ds.SetHTTPClient(srv.Client())

	NewDiscordNotifyJob(ds).Run()

	mu.Lock()
	defer mu.Unlock()
	if !report || !backup {
		t.Fatalf("while Xray is down: report sent = %v, backup sent = %v, want both", report, backup)
	}
}

type reportCounter struct{ n int }

func (c *reportCounter) SendReport() { c.n++ }

// The Telegram notify job returned before SendReport (report and backup) the same way.
func TestStatsNotifyJobSendsReportWhileXrayIsDown(t *testing.T) {
	if (&service.XrayService{}).IsXrayRunning() {
		t.Fatal("test needs Xray to be down")
	}
	c := &reportCounter{}
	(&StatsNotifyJob{tgbotService: c}).Run()
	if c.n != 1 {
		t.Fatalf("SendReport calls = %d, want 1", c.n)
	}
}
