package main

import (
	"fmt"
	"github.com/xionghengheng/ff_plib/db"
	"log"
	"net/http"
	"time"
)

// enableCORS 中间件函数，用于设置 CORS 头
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}


func main() {
	if err := db.Init(); err != nil {
		panic(fmt.Sprintf("mysql init failed with %+v", err))
	}

	mux := http.NewServeMux()
	handler := enableCORS(mux)

	mux.HandleFunc("/api/getUserStatistic", GetUserStatiticHandler)

	mux.HandleFunc("/api/getLessonStatistic", GetLessonStatiticHandler)

	mux.HandleFunc("/api/getUvPvStatistic", GetUvPvStatisticHandler)


	autoScanCoachPersonalPageData()

	autoScanAllCoursePackageSingleLesson()

	if err := http.ListenAndServe(":80", handler); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
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

