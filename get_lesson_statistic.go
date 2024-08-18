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

type GetLessonStatisticReq struct {
	StatisticTs int64 `json:"statistic_ts"`
}

type GetLessonStatisticRsp struct {
	Code                      int    `json:"code"`
	ErrorMsg                  string `json:"errorMsg,omitempty"`
	TotalCoursePurchasers     int64  // 总购课用户数
	TotalCoursePackages       int64  // 总购买课包数
	TotalCoursePackageRevenue int64  // 总课包支付金额
	TotalRedemptionAmount     int64  // 总核销金额
	TotalClassesAttended      int64  // 总上课节数
	NewCoursePurchasersToday  int64  // 今日新增购课用户数
	TodayBookedClasses        int64  // 今日预约课程数
	TodayCompletedClasses     int64  // 今日完成课程数

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

	var dayBegTs int64
	if req.StatisticTs == 0 {
		dayBegTs = comm.GetTodayBegTsByTs(time.Now().Unix())
	} else {
		dayBegTs = comm.GetTodayBegTsByTs(req.StatisticTs)
	}

	var vecAllUserModel []model.CoursePackageModel
	var turnPageTs int64
	for i := 0; i <= 5000; i++ {
		tmpVecAllUserModel, err := dao.ImpCoursePackage.GetAllCoursePackageList(turnPageTs)
		if err != nil {
			rsp.Code = -911
			rsp.ErrorMsg = err.Error()
			Printf("GetAllCoursePackageList err, strOpenId:%s StatisticTs:%d err:%+v\n", req.StatisticTs, err)
			return
		}
		if len(tmpVecAllUserModel) == 0 {
			Printf("GetAllCoursePackageList empty, StatisticTs:%d vecAllUserModel.len:%d\n", req.StatisticTs, len(vecAllUserModel))
			break
		}
		turnPageTs = tmpVecAllUserModel[len(tmpVecAllUserModel)-1].Ts
		vecAllUserModel = append(vecAllUserModel, tmpVecAllUserModel...)
	}

	mapPayPackageUser := make(map[int64]bool)
	for _, v := range vecAllUserModel {
		if v.PackageType == model.Enum_PackageType_PaidPackage {
			rsp.TotalCoursePackages += 1
			mapPayPackageUser[v.Uid] = true
			rsp.TotalCoursePackageRevenue += int64(v.Price)
			if v.Ts > dayBegTs{
				rsp.NewCoursePurchasersToday += 1
			}
		}
	}
	rsp.TotalCoursePurchasers = int64(len(mapPayPackageUser))
	rsp.TotalRedemptionAmount = 1

	return
}
