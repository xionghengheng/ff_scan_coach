package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type GetLessonStatisticReq struct {
	StatisticTs int64 `json:"statistic_ts"`
}

type GetLessonStatisticRsp struct {
	Code                      int     `json:"code"`
	ErrorMsg                  string  `json:"errorMsg,omitempty"`
	TotalCoursePurchasers     int     // 总购课用户数
	NewCoursePurchasersToday  int     // 今日新增购课用户数
	TotalCoursePackages       int     // 总购买课包数
	TotalCoursePackageRevenue float64 // 总课包支付金额
	TotalRedemptionAmount     float64 // 总核销金额
	TotalClassesAttended      int     // 总上课节数
	TodayBookedClasses        int     // 今日预约课程数
	TodayCompletedClasses     int     // 今日完成课程数

}

func getGetLessonStatisticReq(r *http.Request) (GetLessonStatisticReq, error) {
	req := GetLessonStatisticReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetLessonStatiticHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetLessonStatisticReq(r)
	rsp := &GetLessonStatisticRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetLessonStatiticHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	//var dayBegTs int64
	//if req.StatisticTs == 0 {
	//	dayBegTs = comm.GetTodayBegTsByTs(time.Now().Unix())
	//} else {
	//	dayBegTs = comm.GetTodayBegTsByTs(req.StatisticTs)
	//}
	//
	//vecAllUserModel, err := dao.ImpCoursePackage.GetAllCoursePackageListByCoachId()
	//if err != nil {
	//	rsp.Code = -922
	//	rsp.ErrorMsg = err.Error()
	//	Printf("GetAllUser err, strOpenId:%s StatisticTs:%d err:%+v\n", strOpenId, req.StatisticTs, err)
	//	return
	//}
	return
}
