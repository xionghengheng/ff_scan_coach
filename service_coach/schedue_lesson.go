package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/service"
	"encoding/json"
	"fmt"
	"net/http"
)

type SchedueLessonReq struct {
	TraineeUid    int64  `json:"trainee_uid"`     // 学员uid
	PackageID     string `json:"package_id"`      // 课包的唯一标识符（用户id_获取课包的时间戳）
	ScheduleBegTs int64  `json:"schedule_beg_ts"` // 安排上课开始时间
	ScheduleEndTs int64  `json:"schedule_end_ts"` // 安排上课结束时间
}

type SchedueLessonRsp struct {
	Code          int    `json:"code"`
	ErrorMsg      string `json:"errorMsg,omitempty"`
	PackageID     string `json:"package_id"`     // 课包的唯一标识符（用户id_获取课包的时间戳）
	AppointmentID int    `json:"appointment_id"` // 预约ID
	LessonID      string `json:"lesson_id"`      // 单节课的唯一标识符（用户id_场地id_课程id_教练id_发起预约的时间戳）
}

func getSchedueLessonReq(r *http.Request) (SchedueLessonReq, error) {
	req := SchedueLessonReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// SchedueLessonHandler 教练主动排课
func SchedueLessonHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getSchedueLessonReq(r)
	rsp := &SchedueLessonRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("SchedueLessonHandler start, openid:%s\n", strOpenId)

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
		rsp.Code = -993
		rsp.ErrorMsg = err.Error()
		return
	}

	if len(strOpenId) == 0 || req.TraineeUid == 0 || len(req.PackageID) == 0 ||
		req.ScheduleBegTs == 0 || req.ScheduleEndTs == 0 || req.ScheduleBegTs >= req.ScheduleEndTs {
		rsp.Code = -10003
		rsp.ErrorMsg = "param err"
		return
	}

	if req.ScheduleEndTs-req.ScheduleBegTs != 3600 {
		rsp.Code = -10003
		rsp.ErrorMsg = "排课错误，时间间隔必须为1小时"
		return
	}

	stUserInfoModel, err := comm.GetUserInfoByOpenId(strOpenId)
	if err != nil || stUserInfoModel == nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		Printf("getLoginUid fail, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}
	uid := stUserInfoModel.UserID
	coachId := stUserInfoModel.CoachId

	if stUserInfoModel.IsCoach == false || coachId == 0 {
		rsp.Code = -900
		rsp.ErrorMsg = "not coach return"
		Printf("not coach err, strOpenId:%s uid:%d\n", strOpenId, uid)
		return
	}

	stCoachModel, err := dao.ImpCoach.GetCoachById(coachId)
	if err != nil || stUserInfoModel == nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		Printf("GetCoachById err, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}

	stCoursePackageModel, err := dao.ImpCoursePackage.GetCoursePackageById(req.PackageID)
	if err != nil{
		rsp.Code = -600
		rsp.ErrorMsg = err.Error()
		Printf("GetCoursePackageById err, err:%+v coachId:%d req:%+v\n", err, coachId, req)
		return
	}
	if stCoursePackageModel.RemainCnt == 0{
		rsp.Code = -111
		rsp.ErrorMsg = "课包没有剩余次数，无法排课"
		Printf("课包没有剩余次数，无法排课 err, coachId:%d req:%+v\n", coachId, req)
		return
	}

	unDayBegTs := comm.GetTodayBegTsByTs(req.ScheduleBegTs)
	vecTraineeAppointmentModel, err := dao.ImpAppointment.GetUserAppointmentRecordOneDay(req.TraineeUid, unDayBegTs)
	if err != nil {
		rsp.Code = -677
		rsp.ErrorMsg = err.Error()
		Printf("GetUserAppointmentRecordOneDay err, err:%=v coachId:%d TraineeUid:%d\n", err, coachId, req.TraineeUid)
		return
	}
	if !checkTsInterval(vecTraineeAppointmentModel, req.ScheduleBegTs, req.ScheduleEndTs) {
		rsp.Code = -633
		rsp.ErrorMsg = "该学员的时间段已被占用。"
		Printf("checkTsInterval err, vecTraineeAppointmentModel:%+v req:%+v\n", vecTraineeAppointmentModel, req)
		return
	}

	vecCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentScheduleOneDay(stCoachModel.GymID, coachId, unDayBegTs)
	if err != nil {
		rsp.Code = -677
		rsp.ErrorMsg = err.Error()
		Printf("GetAppointmentScheduleOneDay err, err:%=v coachId:%d TraineeUid:%d\n", err, coachId, req.TraineeUid)
		return
	}
	for _, v := range vecCoachAppointmentModel {
		if v.UserID > 0 {
			if !checkAppointmentTsInterval(v, req.ScheduleBegTs, req.ScheduleEndTs) {
				rsp.Code = -622
				rsp.ErrorMsg = "该时间段已其他学员占用"
				Printf("该时间段已其他学员占用, coachId:%d TraineeUid:%d req:%+v v:%+v\n", coachId, req.TraineeUid, req, v)
				return
			}
			continue
		}

		if req.ScheduleBegTs == v.StartTime && req.ScheduleEndTs == v.EndTime {
			rsp.PackageID = req.PackageID
			rsp.AppointmentID = v.AppointmentID
			Printf("获取完全匹配空闲的时间区间, coachId:%d req:%+v AppointmentID:%d\n", coachId, req, rsp.AppointmentID)
			break
		}

		if !checkAppointmentTsInterval(v, req.ScheduleBegTs, req.ScheduleEndTs) {
			rsp.Code = -611
			rsp.ErrorMsg = "该时间段和教练预设时间段有交叉，请先取消之前预设置时间段"
			Printf("时间段和教练预设时间段有交叉，请先取消之前预设置时间段, coachId:%d TraineeUid:%d req:%+v v:%+v\n", coachId, req.TraineeUid, req, v)
			return
		}
	}

	if rsp.AppointmentID > 0 {
		var stBookLessonReq service.BookLessonReq
		var stBookLessonRsp service.BookLessonRsp
		stBookLessonReq.AppointmentID = rsp.AppointmentID
		stBookLessonReq.CoursePackageId = rsp.PackageID
		service.BookLesson(req.TraineeUid, stBookLessonReq, &stBookLessonRsp)
		if stBookLessonRsp.Code != 0 {
			rsp.Code = stBookLessonRsp.Code
			rsp.ErrorMsg = stBookLessonRsp.ErrorMsg
			Printf("BookLesson err, coachId:%d TraineeUid:%d req:%+v stBookLessonReq:%+v\n", coachId, req.TraineeUid, req, stBookLessonReq)
			return
		}
		Printf("BookLesson succ, coachId:%d TraineeUid:%d req:%+v stBookLessonReq:%+v\n", coachId, req.TraineeUid, req, stBookLessonReq)
		rsp.LessonID = stBookLessonRsp.LessonID
		return
	}

	//没有预约id，需要设置预约，然后再排课
	var stSetLessonAvailableReq SetLessonAvailableReq
	var stSetLessonAvailableRsp SetLessonAvailableRsp
	stSetLessonAvailableReq.GymId = stCoachModel.GymID
	stSetLessonAvailableReq.CoachId = coachId
	stSetLessonAvailableReq.StartTs = req.ScheduleBegTs
	stSetLessonAvailableReq.EndTs = req.ScheduleEndTs
	stSetLessonAvailableReq.MockData = false
	SetLessonAvailable(stSetLessonAvailableReq, &stSetLessonAvailableRsp)
	if stSetLessonAvailableRsp.Code != 0 {
		rsp.Code = stSetLessonAvailableRsp.Code
		rsp.ErrorMsg = stSetLessonAvailableRsp.ErrorMsg
		Printf("SetLessonAvailable err, coachId:%d TraineeUid:%d req:%+v stSetLessonAvailableReq:%+v\n", coachId, req.TraineeUid, req, stSetLessonAvailableReq)
		return
	}

	var stBookLessonReq service.BookLessonReq
	var stBookLessonRsp service.BookLessonRsp
	stBookLessonReq.AppointmentID = stSetLessonAvailableRsp.AppointmentID
	stBookLessonReq.CoursePackageId = req.PackageID
	service.BookLesson(req.TraineeUid, stBookLessonReq, &stBookLessonRsp)
	if stBookLessonRsp.Code != 0 {
		rsp.Code = stBookLessonRsp.Code
		rsp.ErrorMsg = stBookLessonRsp.ErrorMsg
		Printf("BookLesson err, coachId:%d TraineeUid:%d req:%+v stBookLessonReq:%+v\n", coachId, req.TraineeUid, req, stBookLessonReq)
		return
	}
	Printf("BookLesson succ, coachId:%d TraineeUid:%d req:%+v stBookLessonReq:%+v\n", coachId, req.TraineeUid, req, stBookLessonReq)
	rsp.LessonID = stBookLessonRsp.LessonID
	rsp.PackageID = req.PackageID
	rsp.AppointmentID = stBookLessonReq.AppointmentID
}
