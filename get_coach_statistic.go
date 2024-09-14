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

type GetCoachStatisticReq struct {
	StatisticTs string `json:"statistic_ts"`
}

type GetCoachStatisticRsp struct {
	Code                   int                  `json:"code"`
	ErrorMsg               string               `json:"errorMsg,omitempty"`
	TotalCoacheCnt         int                  `json:"total_coache_cnt"`           // 总教练人数
	NewCoacheCntToday      int64                `json:"new_coach_cnt_today"`        // 今日新增教练数
	TotalWriteOffLessonCnt int64                `json:"total_write_off_lesson_cnt"` // 核销课程总数
	TotalSales             int64                `json:"total_sales"`                // 教练总销售额
	CoachStatisticItemList []CoachStatisticItem `json:"coach_statistic_item_list"`  // 教练单条统计
}

type CoachStatisticItem struct {
	JoinTime      string            `json:"join_time"`           //教练入驻时间
	CoachID       int               `json:"coach_id"`            //教练ID
	CoachName     string            `json:"coach_name"`          //教练名称
	Phone         string            `json:"phone"`               //手机号
	GymID         int               `json:"gym_id"`              //健身房id
	Bio           string            `json:"bio"`                 //教练简介
	RecReason     string            `json:"rec_reason"`          //教练推荐原因
	CourseIdList  string            `json:"course_id_list"`      //教练可上课程列表，英文逗号分割
	GoodAt        string            `json:"good_at"`             //教练擅长领域
	StatisticCalc StatisticCalcInfo `json:"statistic_calc_info"` //计算统计数据
}

// 计算统计数据
type StatisticCalcInfo struct {
	TrailPackageUv              int            `json:"trail_package_uv"`                // 体验用户数
	TrailPackageUidList         map[int64]bool `json:"trail_package_uid_list"`          // 体验课课包用户数
	TrailBookingCountUv         int            `json:"trail_booking_count_uv"`          // 体验约课人数
	TrailBookingCountPv         int            `json:"trail_booking_count_pv"`          // 体验约课次数
	TrailLessonWriteOffUv       int            `json:"trail_lesson_writeoff_uv"`        // 体验课核销人数
	TrailLessonWriteOffPv       int            `json:"trail_lesson_writeoff_pv"`        // 体验课核销次数
	PaidPackageUv               int            `json:"paid_package_uv"`                 // 正式课课包付费用户数
	PaidPackageUidList          map[int64]bool `json:"paid_package_uid_list"`           // 正式课课包付费用户数
	PaidPackageTotalLessonCount int            `json:"paid_package_total_lesson_count"` // 正式课付费课时次数
	PaidPackageSalesRevenue     int            `json:"paid_package_sales_revenue"`      // 正式课付费销售额
	PaidLessonWriteOffUv        int            `json:"paid_lesson_writeoff_uv"`         // 正式课核销人数
	PaidLessonWriteOffPv        int            `json:"paid_lesson_writeoff_pv"`         // 正式课核销次数
	PaidLessonWriteOffAmount    int            `json:"paid_lesson_writeoff_amount"`     // 正式课核销金额
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

	mapCoachId2StatisticCalcInfo := make(map[int]StatisticCalcInfo)
	for _, v := range mapCoach {
		var item StatisticCalcInfo
		item.TrailPackageUidList = make(map[int64]bool)
		item.PaidPackageUidList = make(map[int64]bool)
		mapCoachId2StatisticCalcInfo[v.CoachID] = item
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
			Printf("GetAllCoursePackageList empty, StatisticTs:%s vecAllPackageModel.len:%d\n", req.StatisticTs, len(vecAllPackageModel))
			break
		}
		turnPageTs = tmpVecAllUserModel[len(tmpVecAllUserModel)-1].Ts
		vecAllPackageModel = append(vecAllPackageModel, tmpVecAllUserModel...)
	}

	for _, v := range vecAllPackageModel {
		if v.PackageType == model.Enum_PackageType_PaidPackage {
			tmp := mapCoachId2StatisticCalcInfo[v.CoachId]
			if _, ok := tmp.PaidPackageUidList[v.Uid]; !ok {
				tmp.PaidPackageUv += 1
				tmp.PaidPackageUidList[v.Uid] = true
				mapCoachId2StatisticCalcInfo[v.CoachId] = tmp
			}

			tmp = mapCoachId2StatisticCalcInfo[v.CoachId]
			tmp.PaidPackageTotalLessonCount += v.TotalCnt
			tmp.PaidPackageSalesRevenue += v.Price
			mapCoachId2StatisticCalcInfo[v.CoachId] = tmp
		}
	}

	for _, v := range vecAllPackageModel {
		if v.PackageType == model.Enum_PackageType_TrialFree {

			//如果是付费用户，则免费课用户不计入
			tmp := mapCoachId2StatisticCalcInfo[v.CoachId]
			if _, ok := tmp.PaidPackageUidList[v.Uid]; ok {
				continue
			}

			if _, ok := tmp.TrailPackageUidList[v.Uid]; !ok {
				tmp.TrailPackageUv += 1
				tmp.TrailPackageUidList[v.Uid] = true
				mapCoachId2StatisticCalcInfo[v.CoachId] = tmp
			}
		}
	}

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

	//for _, v := range vecAllSingleLesson {
	//	if v.Status == model.En_LessonStatusCompleted {
	//		rsp.TotalClassesAttended += 1
	//		rsp.TotalRedemptionAmount += int64(mapCourse[v.CourseID].Price)
	//	}
	//	if v.ScheduleBegTs > dayBegTs {
	//		if v.Status == model.En_LessonStatusCompleted {
	//			rsp.TodayCompletedClasses += 1
	//		} else {
	//			rsp.TodayBookedClasses += 1
	//		}
	//	}
	//}
	//
	//for _, v := range vecAllSingleLesson {
	//	if v.ScheduleBegTs < dayBegTs {
	//		continue
	//	}
	//	var stLessonStatisticItem LessonStatisticItem
	//	_, _, packageType := comm.ParseCoursePackageId(v.PackageID)
	//	stCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentById(v.AppointmentID)
	//	if err != nil {
	//		Printf("GetAppointmentById err, err:%+v AppointmentID:%d\n", err, v.AppointmentID)
	//	}
	//	if stCoachAppointmentModel != nil {
	//		t := time.Unix(stCoachAppointmentModel.CreateTs, 0)
	//		stLessonStatisticItem.BookingTime = "课程发起预约的时间 " + t.Format("2006年01月02日 15:04")
	//	}
	//
	//	stLessonStatisticItem.LessonID = v.LessonID
	//	if packageType == model.Enum_PackageType_PaidPackage {
	//		stLessonStatisticItem.LessonType = "正式课"
	//	} else {
	//		stLessonStatisticItem.LessonType = "体验课"
	//	}
	//
	//	if v.Status == model.En_LessonStatus_Scheduled {
	//		stLessonStatisticItem.LessonStatus = "已预约"
	//	} else if v.Status == model.En_LessonStatusCompleted {
	//		stLessonStatisticItem.LessonStatus = "已完成"
	//	} else if v.Status == model.En_LessonStatusCanceled {
	//		stLessonStatisticItem.LessonStatus = "已取消"
	//	} else if v.Status == model.En_LessonStatusMissed {
	//		stLessonStatisticItem.LessonStatus = "已旷课"
	//	}
	//	t := time.Unix(v.ScheduleBegTs, 0)
	//	stLessonStatisticItem.ScheduleBegTs = "课程开始时间 " + t.Format("2006年01月02日 15:04")
	//	t = time.Unix(v.WriteOffTs, 0)
	//	stLessonStatisticItem.WriteOffTs = "课程核销时间 " + t.Format("2006年01月02日 15:04")
	//
	//	stLessonStatisticItem.CommentContent = v.CommentContent
	//	rsp.LessonStatisticItemList = append(rsp.LessonStatisticItemList, stLessonStatisticItem)
	//}

	for _, v := range mapCoach {
		var stCoachStatisticItem CoachStatisticItem
		t := time.Unix(v.JoinTs, 0)
		stCoachStatisticItem.JoinTime = "教练入驻时间 " + t.Format("2006年01月02日 15:04")
		stCoachStatisticItem.CoachID = v.CoachID
		stCoachStatisticItem.CoachName = v.CoachName
		stCoachStatisticItem.Phone = v.Phone
		stCoachStatisticItem.GymID = v.GymID
		stCoachStatisticItem.Bio = v.Bio
		stCoachStatisticItem.RecReason = v.RecReason
		stCoachStatisticItem.CourseIdList = v.CourseIdList
		stCoachStatisticItem.GoodAt = v.GoodAt

		if item,ok:=mapCoachId2StatisticCalcInfo[v.CoachID];ok{
			stCoachStatisticItem.StatisticCalc = item
		}
		rsp.CoachStatisticItemList = append(rsp.CoachStatisticItemList, stCoachStatisticItem)
	}
	return
}
