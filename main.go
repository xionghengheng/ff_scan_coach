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

	autoScanAllCoursePackageSingleLesson()

	//测试接口，清空用户信息
	http.HandleFunc("/api/getUserStatistic", GetUserStatiticHandler)

	log.Fatal(http.ListenAndServe(":80", nil))
}

//扫描订单表和课程表，生成教练单月的营收数据统计（每17分钟扫描一次）
func autoScanCoachPersonalPageData() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(1020))
		for range ticker.C {
			ScanCoachPersonalPageData()
		}
	}()
}

//扫描所有单次课程，处理旷课以及旷课退回的情况（每5分钟扫描一次）
func autoScanAllCoursePackageSingleLesson() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(300))
		for range ticker.C {
			ScanAllCoursePackageSingleLesson()
		}
	}()
}

