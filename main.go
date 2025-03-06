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

	mux.HandleFunc("/api/getCoachStatistic", GetCoachStatiticHandler)

	mux.HandleFunc("/api/getUvPvStatistic", GetUvPvStatisticHandler)

	mux.HandleFunc("/api/getAllUserWithBindPhone", GetAllUserWithBindPhoneHandler)

	autoScanCoachPersonalPageData()

	autoScanAllCoursePackageSingleLesson()

	autoScanAllPackage()

	// 调用函数，设置每天晚上 11 点执行任务
	autoScanAllAppointments()

	if err := http.ListenAndServe(":80", handler); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// 扫描订单表和课程表，生成教练单月的营收数据统计（每17分钟扫描一次）
func autoScanCoachPersonalPageData() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(1020))
		for range ticker.C {
			ScanCoachPersonalPageData()
		}
	}()
}

// 扫描所有单次课程，处理旷课以及旷课退回的情况（每5分钟扫描一次）
func autoScanAllCoursePackageSingleLesson() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(300))
		for range ticker.C {
			ScanAllCoursePackageSingleLesson()
		}
	}()
}

// 每小时扫描一次
func autoScanAllPackage() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(600))
		for range ticker.C {
			ScanAllPackage()
		}
	}()
}

// 扫描所有预约，如果有教练连续两天没有设置课程，则触发微信通知告诉教练试课
func autoScanAllAppointments() {
	// 计算下一个执行时间
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, now.Location())

	// 如果当前时间已经过了晚上 11 点，则设置下一个执行时间为明天的 11 点
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// 计算到下一个执行时间的间隔
	durationUntilNextRun := nextRun.Sub(now)

	// 创建一个 ticker，间隔为 24 小时
	ticker := time.NewTicker(24 * time.Hour)

	// 启动一个 goroutine 在下一个执行时间执行任务
	go func() {
		time.Sleep(durationUntilNextRun) // 等待到下一个执行时间
		for {
			ScanAllAppointments() // 执行任务
			<-ticker.C            // 等待下一个周期
		}
	}()
}
