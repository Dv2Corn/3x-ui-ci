// e2e helper for queue item 12 (not part of the patch).
// "seed DB": a client whose only subscription entry is an external provider that never answers.
// "hang ADDR": that provider — accepts connections and never replies.
package main

import (
	"net"
	"os"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	switch os.Args[1] {
	case "hang":
		ln, err := net.Listen("tcp", os.Args[2])
		must(err)
		for {
			c, err := ln.Accept()
			must(err)
			go func() { time.Sleep(time.Hour); c.Close() }()
		}
	case "seed":
		must(database.InitDB(os.Args[2]))
		db := database.GetDB()
		for k, v := range map[string]string{"subEnable": "true", "subPath": "/sub/", "subPort": "2096", "subListen": "127.0.0.1"} {
			db.Where("key = ?", k).Delete(&model.Setting{})
			must(db.Create(&model.Setting{Key: k, Value: v}).Error)
		}
		rec := model.ClientRecord{Email: "e2e@x", SubID: "e2esub", UUID: "0b8a2c3e-1111-4222-8333-944455556666", Enable: true}
		must(db.Create(&rec).Error)
		must(db.Create(&model.ClientExternalLink{ClientId: rec.Id, Kind: model.ExternalLinkKindSubscription, Value: "http://127.0.0.1:18080/dead"}).Error)
		must(db.Create(&model.ClientExternalLink{ClientId: rec.Id, Kind: model.ExternalLinkKindSubscription, Value: "http://127.0.0.1:18080/dead2"}).Error)
		must(db.Create(&model.ClientExternalLink{ClientId: rec.Id, Kind: "link", Value: "vless://0b8a2c3e-1111-4222-8333-944455556666@example.com:443#static"}).Error)
	}
}
