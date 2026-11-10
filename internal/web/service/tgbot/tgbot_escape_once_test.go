package tgbot

import (
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// Bot text is escaped once, where a value enters it; screens that wrapped the
// finished text escaped it again, so "R&D" reached Telegram as "R&amp;amp;D".
func TestClientTextIsEscapedOnce(t *testing.T) {
	const remark, email = "R&D <eu>", "o'neil&co"
	cases := []struct {
		name   string
		render func(tb *Tgbot) string
		want   []string
	}{
		{"client card", func(tb *Tgbot) string {
			tb.renderPickerOnClient(1, email, nil)
			sc, ok := tb.screens().get(1)
			if !ok {
				t.Fatal("no client card was rendered")
			}
			return sc.text
		}, []string{"R&amp;D &lt;eu&gt;", "o&#39;neil&amp;co"}},
		{"qr title", func(tb *Tgbot) string {
			return tb.clientScreenBody(email, tb.I18nBot("tgbot.answers.qrCodeForClient", "Email=="+email))
		}, []string{"QRCode for client o&#39;neil&amp;co:"}},
		{"deplete report", func(tb *Tgbot) string {
			return tb.depleteReport()
		}, []string{"R&amp;D &lt;eu&gt;", "Email: o&#39;neil&amp;co"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tb, _ := newScreenTgbot(t, false)
			initReportDB(t)
			localizeWithRealBundle(t)
			seedReportClients(t, remark, []string{email})
			// Expired, so the deplete report lists the client.
			expired := time.Now().Add(-time.Hour).UnixMilli()
			if err := database.GetDB().Model(&xray.ClientTraffic{}).Where("email = ?", email).
				Update("expiry_time", expired).Error; err != nil {
				t.Fatal(err)
			}

			got := tc.render(tb)

			if strings.Contains(got, "&amp;amp;") || strings.Contains(got, "&amp;lt;") || strings.Contains(got, "&amp;#39;") {
				t.Fatalf("text escaped twice:\n%s", got)
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("text does not contain %q:\n%s", want, got)
				}
			}
		})
	}
}
