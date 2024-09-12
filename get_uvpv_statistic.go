package main

import (
	"encoding/json"
	"fmt"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"net/http"
	"time"
)

type GetUvPvStatisticReq struct {
	StatisticTs string `json:"statistic_ts"` //统计时间，比如20240908
	PageId      string `json:"page_id"`
	ButtondId   string `json:"buttond_id"`
}

type GetUvPvStatisticRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
	PageUv   int64  `json:"page_uv,omitempty"`
	PagePv   int64  `json:"page_pv,omitempty"`
	ButtonUv int64  `json:"button_uv,omitempty"`
	ButtonPv int64  `json:"button_pv,omitempty"`
}

func getGetUvPvStatisticReq(r *http.Request) (GetUvPvStatisticReq, error) {
	req := GetUvPvStatisticReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetUvPvStatisticHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetUvPvStatisticReq(r)
	rsp := &GetUvPvStatisticRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetUvPvStatisticHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	if len(req.PageId) > 0{
		vecReportData, err := dao.ImpReport.GetPageReport(req.PageId, dayBegTs, dayBegTs+86399)
		if err != nil {
			rsp.Code = -922
			rsp.ErrorMsg = err.Error()
			Printf("GetPageReport err, StatisticTs:%d err:%+v\n", req.StatisticTs, err)
			return
		}
		rsp.PagePv = int64(len(vecReportData))
		rsp.PageUv = int64(len(removeDuplicates(vecReportData)))
	}
	if len(req.ButtondId) > 0{
		vecReportData, err := dao.ImpReport.GetButtonReport(req.ButtondId, dayBegTs, dayBegTs+86399)
		if err != nil {
			rsp.Code = -922
			rsp.ErrorMsg = err.Error()
			Printf("GetButtonReport err, StatisticTs:%d err:%+v\n", req.StatisticTs, err)
			return
		}
		rsp.ButtonPv = int64(len(vecReportData))
		rsp.ButtonUv = int64(len(removeDuplicates(vecReportData)))
	}
	return
}

func removeDuplicates(input []model.ReportModel) []model.ReportModel {
	seen := make(map[int64]bool)
	var result []model.ReportModel
	for _, data := range input {
		if !seen[data.UID] {
			seen[data.UID] = true
			result = append(result, data)
		}
	}
	return result
}