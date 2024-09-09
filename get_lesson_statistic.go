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
	StatisticTs string `json:"statistic_ts"`
}

type GetLessonStatisticRsp struct {
	Code                      int                    `json:"code"`
	ErrorMsg                  string                 `json:"errorMsg,omitempty"`
	TotalCoursePurchasers     int64                  `json:"total_course_purchasers"`      // 总购课用户数
	TotalCoursePackages       int64                  `json:"total_course_packages"`        // 总购买课包数
	TotalCoursePackageRevenue int64                  `json:"total_course_package_revenue"` // 总课包支付金额
	TotalRedemptionAmount     int64                  `json:"total_redemption_amount"`      // 总核销金额
	TotalClassesAttended      int64                  `json:"total_classes_attended"`       // 总上课节数
	NewCoursePurchasersToday  int64                  `json:"new_course_purchasers_today"`  // 今日新增购课用户数
	TodayBookedClasses        int64                  `json:"today_booked_classes"`         // 今日预约课程数
	TodayCompletedClasses     int64                  `json:"today_completed_classes"`      // 今日完成课程数
	PackageStatisticItemList  []PackageStatisticItem `json:"package_statistic_item_list"`
	LessonStatisticItemList   []LessonStatisticItem  `json:"lesson_statistic_item_list"`
}

// 课包统计信息
type PackageStatisticItem struct {
	OrderTime         string  `json:"order_time"`           // 订单时间
	PackageOrderID    string  `json:"package_order_id"`     // 课包订单号
	PackageName       string  `json:"package_name"`         // 课包名称
	CoachName         string  `json:"coach_name"`           // 授课教练
	GymName           string  `json:"gym_name"`             // 场地
	UnitPrice         int     `json:"unit_price"`           // 课单价
	TotalAmount       float64 `json:"total_amount"`         // 总金额
	TotalLessonCnt    int     `json:"total_lesson_cnt"`     // 总课时
	WriteOffLessonCnt int     `json:"write_off_lesson_cnt"` // 核销课程数
	RemainCnt         int     `json:"remain_cnt"`           // 剩余课时
	WriteOffAmount    int     `json:"write_off_amount"`     // 核销金额
	LastLessonTime    string  `json:"last_class_time"`      // 上次上课时间
}

// 课程统计信息
type LessonStatisticItem struct {
	BookingTime    string `json:"booking_time"`    // 预约时间
	LessonID       string `json:"lesson_id"`       // 课程编号
	LessonType     string `json:"lesson_type"`     // 正式课or体验课
	LessonStatus   string `json:"lesson_status"`   // 课程状态
	ScheduleBegTs  string `json:"schedule_beg_ts"` // 上课时间
	WriteOffTs     string `json:"write_off_ts"`    // 核销时间
	CommentContent string `json:"comment_content"` // 课程评价
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
	if len(req.StatisticTs) == 0 {
		dayBegTs = comm.GetTodayBegTsByTs(time.Now().Unix())
	} else {
		t, _ := time.Parse("20060102", req.StatisticTs)
		dayBegTs = comm.GetTodayBegTsByTs(t.Unix())
	}
	mapCourse, err := comm.GetAllCouse()
	if err != nil {
		rsp.Code = -9111
		rsp.ErrorMsg = err.Error()
		return
	}
	mapCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -9111
		rsp.ErrorMsg = err.Error()
		return
	}
	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -9111
		rsp.ErrorMsg = err.Error()
		return
	}

	var vecAllPackageModel []model.CoursePackageModel
	var turnPageTs int64
	for i := 0; i <= 5000; i++ {
		tmpVecAllUserModel, err := dao.ImpCoursePackage.GetAllCoursePackageList(turnPageTs)
		if err != nil {
			rsp.Code = -911
			rsp.ErrorMsg = err.Error()
			Printf("GetAllCoursePackageList err, StatisticTs:%d err:%+v\n", req.StatisticTs, err)
			return
		}
		if len(tmpVecAllUserModel) == 0 {
			Printf("GetAllCoursePackageList empty, StatisticTs:%d vecAllPackageModel.len:%d\n", req.StatisticTs, len(vecAllPackageModel))
			break
		}
		turnPageTs = tmpVecAllUserModel[len(tmpVecAllUserModel)-1].Ts
		vecAllPackageModel = append(vecAllPackageModel, tmpVecAllUserModel...)
	}

	mapPayPackageUser := make(map[int64]bool)
	for _, v := range vecAllPackageModel {
		if v.PackageType == model.Enum_PackageType_PaidPackage {
			rsp.TotalCoursePackages += 1
			mapPayPackageUser[v.Uid] = true
			rsp.TotalCoursePackageRevenue += int64(v.Price)
			if v.Ts > dayBegTs {
				rsp.NewCoursePurchasersToday += 1
			}
		}
	}
	rsp.TotalCoursePurchasers = int64(len(mapPayPackageUser))

	var vecAllSingleLesson []model.CoursePackageSingleLessonModel
	turnPageTs = 0
	for i := 0; i <= 5000; i++ {
		tmpVecAllSingleLesson, err := dao.ImpCoursePackageSingleLesson.GetAllSingleLessonList(turnPageTs)
		if err != nil {
			rsp.Code = -911
			rsp.ErrorMsg = err.Error()
			Printf("GetAllSingleLessonList err, StatisticTs:%d err:%+v\n", req.StatisticTs, err)
			return
		}
		if len(tmpVecAllSingleLesson) == 0 {
			Printf("GetAllSingleLessonList empty, StatisticTs:%d vecAllSingleLesson.len:%d\n", req.StatisticTs, len(vecAllSingleLesson))
			break
		}
		turnPageTs = tmpVecAllSingleLesson[len(tmpVecAllSingleLesson)-1].CreateTs
		vecAllSingleLesson = append(vecAllSingleLesson, tmpVecAllSingleLesson...)
	}

	for _, v := range vecAllSingleLesson {
		if v.Status == model.En_LessonStatusCompleted {
			rsp.TotalClassesAttended += 1
			rsp.TotalRedemptionAmount += int64(mapCourse[v.CourseID].Price)
		}
		if v.ScheduleBegTs > dayBegTs {
			if v.Status == model.En_LessonStatusCompleted {
				rsp.TodayCompletedClasses += 1
			} else {
				rsp.TodayBookedClasses += 1
			}
		}
	}

	//表单字段统计
	for _, v := range vecAllPackageModel {
		var stPackageStatisticItem PackageStatisticItem
		//stCoachAppointmentModel, err := dao.ImpPaymentOrder.GetOrderList(v.AppointmentID)
		//if err != nil {
		//	Printf("GetAppointmentById err, err:%+v AppointmentID:%d\n", err, v.AppointmentID)
		//}

		stPackageStatisticItem.GymName = mapGym[v.GymId].LocName
		stPackageStatisticItem.CoachName = mapCoach[v.CoachId].CoachName
		stPackageStatisticItem.PackageName = mapCourse[v.CourseId].Name
		stPackageStatisticItem.UnitPrice = mapCourse[v.CourseId].Price
		stPackageStatisticItem.TotalAmount = 1
		stPackageStatisticItem.TotalLessonCnt = v.TotalCnt
		stPackageStatisticItem.RemainCnt = v.RemainCnt
		stPackageStatisticItem.WriteOffLessonCnt = v.TotalCnt - v.RemainCnt
		stPackageStatisticItem.WriteOffAmount = stPackageStatisticItem.WriteOffLessonCnt * stPackageStatisticItem.UnitPrice
		t := time.Unix(v.LastLessonTs, 0)
		stPackageStatisticItem.LastLessonTime = "上次上课时间 " + t.Format("2006年01月02日 15:04")

		rsp.PackageStatisticItemList = append(rsp.PackageStatisticItemList, stPackageStatisticItem)
	}

	for _, v := range vecAllSingleLesson {
		if v.ScheduleBegTs < dayBegTs {
			continue
		}
		var stLessonStatisticItem LessonStatisticItem
		_, _, packageType := comm.ParseCoursePackageId(v.PackageID)
		stCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentById(v.AppointmentID)
		if err != nil {
			Printf("GetAppointmentById err, err:%+v AppointmentID:%d\n", err, v.AppointmentID)
		}
		if stCoachAppointmentModel != nil {
			t := time.Unix(stCoachAppointmentModel.CreateTs, 0)
			stLessonStatisticItem.BookingTime = "课程发起预约的时间 " + t.Format("2006年01月02日 15:04")
		}

		stLessonStatisticItem.LessonID = v.LessonID
		if packageType == model.Enum_PackageType_PaidPackage {
			stLessonStatisticItem.LessonType = "正式课"
		} else {
			stLessonStatisticItem.LessonType = "体验课"
		}

		if v.Status == model.En_LessonStatus_Scheduled {
			stLessonStatisticItem.LessonStatus = "已预约"
		} else if v.Status == model.En_LessonStatusCompleted {
			stLessonStatisticItem.LessonStatus = "已完成"
		} else if v.Status == model.En_LessonStatusCanceled {
			stLessonStatisticItem.LessonStatus = "已取消"
		} else if v.Status == model.En_LessonStatusMissed {
			stLessonStatisticItem.LessonStatus = "已旷课"
		}
		t := time.Unix(v.ScheduleBegTs, 0)
		stLessonStatisticItem.ScheduleBegTs = "课程开始时间 " + t.Format("2006年01月02日 15:04")
		t = time.Unix(v.WriteOffTs, 0)
		stLessonStatisticItem.WriteOffTs = "课程核销时间 " + t.Format("2006年01月02日 15:04")

		stLessonStatisticItem.CommentContent = v.CommentContent
		rsp.LessonStatisticItemList = append(rsp.LessonStatisticItemList, stLessonStatisticItem)
	}
	return
}
