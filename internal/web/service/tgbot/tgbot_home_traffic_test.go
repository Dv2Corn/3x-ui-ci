package tgbot

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// The admin home screen put the inbounds' whole traffic under upload and 0
// under download.
func TestHomeSummarySplitsUploadAndDownload(t *testing.T) {
	cases := []struct {
		name     string
		up, down []int64
		want     string
	}{
		{"one inbound", []int64{3 << 30}, []int64{5 << 30}, "🚦 Traffic: 8.00GB (↑3.00GB,↓5.00GB)"},
		{"two inbounds", []int64{1 << 30, 2 << 30}, []int64{4 << 30, 0}, "🚦 Traffic: 7.00GB (↑3.00GB,↓4.00GB)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tb := initReportDB(t)
			localizeWithRealBundle(t)
			for i := range tc.up {
				ib := &model.Inbound{UserId: 1, Tag: "home-" + string(rune('a'+i)), Port: 9443 + i,
					Protocol: model.VLESS, Enable: true, Settings: `{"clients":[]}`, Up: tc.up[i], Down: tc.down[i]}
				if err := database.GetDB().Create(ib).Error; err != nil {
					t.Fatal(err)
				}
			}

			if got := tb.homeSummary(); !strings.Contains(got, tc.want) {
				t.Fatalf("home summary has no %q:\n%s", tc.want, got)
			}
		})
	}
}
