package common

import "testing"

func TestEnsureURLScheme(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"bare telegram handle", "t.me/xui_support", "https://t.me/xui_support"},
		{"bare domain with path", "example.com/help", "https://example.com/help"},
		{"already https", "https://t.me/xui_support", "https://t.me/xui_support"},
		{"already http", "http://example.com", "http://example.com"},
		{"telegram deep link", "tg://resolve?domain=xui_support", "tg://resolve?domain=xui_support"},
		{"mailto", "mailto:support@example.com", "mailto:support@example.com"},
		{"tel", "tel:+1234567890", "tel:+1234567890"},
		{"trims whitespace", "  t.me/xui_support  ", "https://t.me/xui_support"},
		{"capitalized mailto", "Mailto:support@example.com", "Mailto:support@example.com"},
		{"uppercase tel", "TEL:+1234567890", "TEL:+1234567890"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnsureURLScheme(tt.in); got != tt.want {
				t.Errorf("EnsureURLScheme(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseRemoteRoutingURLKeepsInlineCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSource string
		wantRemote bool
		wantErr    bool
	}{
		{name: "deeplink stays inline", input: "happ://routing/onadd/abc"},
		{name: "plain HTTP stays inline", input: "http://example.com/rules"},
		{name: "multiline stays inline", input: "https://example.com/rules\nMATCH,PROXY"},
		{name: "HTTPS source", input: "  https://example.com/rules#ignored  ", wantSource: "https://example.com/rules", wantRemote: true},
		{name: "uppercase scheme", input: "HTTPS://example.com/rules", wantSource: "https://example.com/rules", wantRemote: true},
		{name: "credentials rejected", input: "https://user:pass@example.com/rules", wantRemote: true, wantErr: true},
		{name: "missing host rejected", input: "https:///rules", wantRemote: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, remote, err := ParseRemoteRoutingURL(tt.input)
			if got != tt.wantSource || remote != tt.wantRemote || (err != nil) != tt.wantErr {
				t.Fatalf("got=%q remote=%v err=%v", got, remote, err)
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"blank", "   ", ""},
		{"plain", "panel.example.com", "panel.example.com"},
		{"upper case", "Panel.Example.COM", "panel.example.com"},
		{"trailing dot", "panel.example.com.", "panel.example.com"},
		{"spaces and port", " panel.example.com:2053 ", "panel.example.com"},
		{"IDN", "BÜCHER.example", "xn--bcher-kva.example"},
		{"underscore kept", "my_panel.example.com", "my_panel.example.com"},
		{"IPv4 with port", "203.0.113.10:2053", "203.0.113.10"},
		{"IPv6 with port", "[2001:DB8::1]:2053", "2001:db8::1"},
		{"IPv6 bracketed", "[2001:db8::1]", "2001:db8::1"},
		{"IPv6 bare", "2001:db8::1", "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeHost(tt.in); got != tt.want {
				t.Errorf("NormalizeHost(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
