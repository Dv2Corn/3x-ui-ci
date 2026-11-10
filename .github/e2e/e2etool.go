// e2e helper: TOTP codes and JSON field edits for the sandbox run (not part of any patch).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xlzd/gotp"
)

func main() {
	switch os.Args[1] {
	case "code":
		fmt.Print(gotp.NewDefaultTOTP(os.Args[2]).Now())
	case "set":
		raw, _ := os.ReadFile(os.Args[2])
		var env map[string]json.RawMessage
		if err := json.Unmarshal(raw, &env); err != nil {
			panic(err)
		}
		obj := map[string]any{}
		src := raw
		if o, ok := env["obj"]; ok {
			src = o
		}
		if err := json.Unmarshal(src, &obj); err != nil {
			panic(err)
		}
		for _, kv := range os.Args[3:] {
			k, v, _ := strings.Cut(kv, "=")
			var parsed any
			if json.Unmarshal([]byte(v), &parsed) == nil {
				obj[k] = parsed
			} else {
				obj[k] = v
			}
		}
		out, _ := json.Marshal(obj)
		fmt.Print(string(out))
	case "get":
		raw, _ := os.ReadFile(os.Args[2])
		var env struct{ Obj map[string]any }
		_ = json.Unmarshal(raw, &env)
		fmt.Printf("%v", env.Obj[os.Args[3]])
	}
}
