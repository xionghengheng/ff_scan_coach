package main

import (
	"encoding/json"
	"fmt"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"net/http"
	"time"
)

type GetCoachStatisticReq struct {
	StatisticTs string `json:"statistic_ts"`
}

type GetCoachStatisticRsp struct {
	Code                   int    `json:"code"`
	ErrorMsg               string `json:"errorMsg,omitempty"`
	TotalCoacheCnt         int    `json:"total_coache_cnt"`           // 总教练人数
	NewCoacheCntToday      int64  `json:"new_coach_cnt_today"`        // 今日新增教练数
	TotalWriteOffLessonCnt int64  `json:"total_write_off_lesson_cnt"` // 核销课程总数
	TotalSales             int64  `json:"total_sales"`                // 教练总销售额
}

func getGetCoachStatiticHandlerReq(r *http.Request) (GetCoachStatisticReq, error) {
	req := GetCoachStatisticReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetCoachStatiticHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetCoachStatiticHandlerReq(r)
	rsp := &GetCoachStatisticRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetCoachStatiticHandler start, openid:%s req:%+v\n", strOpenId, req)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	var dayBegTs int64
	if len(req.StatisticTs) == 0 {
		dayBegTs = comm.GetTodayBegTsByTs(time.Now().Unix())
	} else {
		t, _ := time.Parse("20060102", req.StatisticTs)
		dayBegTs = comm.GetTodayBegTsByTs(t.Unix())
	}

	mapCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -9111
		rsp.ErrorMsg = err.Error()
		return
	}

	vecCoachMonthlyStatisticModel, err := dao.ImpCoachClientMonthlyStatistic.GetAllItem()
	if err != nil {
		rsp.Code = -9000
		rsp.ErrorMsg = err.Error()
		return
	}

	rsp.TotalCoacheCnt = len(mapCoach)
	rsp.NewCoacheCntToday = 0
	for _, v := range mapCoach {
		if v.JoinTs >= dayBegTs {
			rsp.NewCoacheCntToday += 1
		}
	}

	for _, v := range vecCoachMonthlyStatisticModel {
		rsp.TotalWriteOffLessonCnt += int64(v.LessonCnt)
		rsp.TotalSales += int64(v.SaleRevenue)
	}
	return
}
