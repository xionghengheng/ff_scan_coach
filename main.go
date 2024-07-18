package main

import (
	"fmt"
	"github.com/xionghengheng/ff_plib/db"
	"log"
	"net/http"
	"time"
)

func main() {
	if err := db.Init(); err != nil {
		panic(fmt.Sprintf("mysql init failed with %+v", err))
	}

	autoScanCoachPersonalPageData()

	//测试接口，清空用户信息
	http.HandleFunc("/api/test", ForTestHandler)

	log.Fatal(http.ListenAndServe(":80", nil))
}

func autoScanCoachPersonalPageData() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(60))
		for range ticker.C {
			ScanCoachPersonalPageData()
		}
	}()
}

// ForTestHandler
func ForTestHandler(w http.ResponseWriter, r *http.Request) {
	return
}
