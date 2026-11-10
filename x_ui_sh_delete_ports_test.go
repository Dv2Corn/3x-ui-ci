package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeUFW keeps one rule per line and renumbers on delete, as ufw does.
const fakeUFW = `ufw() {
    case "$1 ${2:-}" in
        "status numbered") nl -w1 -s' ' "$TEST_DIR/rules" ;;
        delete*) sed -i "${2}d" "$TEST_DIR/rules" ;;
    esac
}
`

func TestDeletePortsByRuleNumbers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("x-ui.sh regressions require a POSIX environment")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is required for the x-ui.sh regression tests")
	}
	cases := []struct{ input, want string }{
		{"1,3", "80/tcp\n2053/tcp\n"},
		{"3,1", "80/tcp\n2053/tcp\n"},
		{"2,2", "22/tcp\n443/tcp\n2053/tcp\n"},
		{"4", "22/tcp\n80/tcp\n443/tcp\n"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			dir := t.TempDir()
			rules := filepath.Join(dir, "rules")
			if err := os.WriteFile(rules, []byte("22/tcp\n80/tcp\n443/tcp\n2053/tcp\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(bash, "-c", fakeUFW+scriptFunction(t, "x-ui.sh", "delete_ports")+"delete_ports\n")
			cmd.Env = append(os.Environ(), "TEST_DIR="+dir)
			cmd.Stdin = strings.NewReader("1\n" + tc.input + "\n")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("delete_ports: %v\n%s", err, out)
			}
			got, err := os.ReadFile(rules)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("deleting rules %s left %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
