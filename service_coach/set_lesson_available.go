package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SetLessonAvailableReq 请求结构
// 还需要传教练id+场地id
type SetLessonAvailableReq struct {
	StartTs int64 `json:"start_ts"`
	EndTs   int64 `json:"end_ts"`

	//mock数据测试使用
	GymId    int  `json:"gym_id"`   //场地id,目前每个教练只有一个场地，可以先不传
	CoachId  int  `json:"coach_id"` //教练id
	MockData bool `json:"mock_data"`
}

type SetLessonAvailableRsp struct {
	Code          int    `json:"code"`
	ErrorMsg      string `json:"errorMsg,omitempty"`
	AppointmentID int    `json:"appointment_id"` //预约ID
}

func getSetLessonAvailableReq(r *http.Request) (SetLessonAvailableReq, error) {
	req := SetLessonAvailableReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// SetLessonAvailableHandler 教练设置可预约时间
func SetLessonAvailableHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getSetLessonAvailableReq(r)
	rsp := &SetLessonAvailableRsp{}
	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()
	Printf("SetLessonAvailableHandler start, req:%+v strOpenId:%s\n", req, strOpenId)

	if req.MockData {
		//连续设置7天
		//10~12
		SetDataTest(req.CoachId, req.GymId, 10*3600) //9点
		SetDataTest(req.CoachId, req.GymId, 11*3600) //10点

		//14~18
		SetDataTest(req.CoachId, req.GymId, 14*3600) //14点
		SetDataTest(req.CoachId, req.GymId, 15*3600) //15点
		SetDataTest(req.CoachId, req.GymId, 16*3600) //16点
		SetDataTest(req.CoachId, req.GymId, 17*3600) //16点
		return
	}

	if err != nil || len(strOpenId) == 0 {
		rsp.Code = -10003
		rsp.ErrorMsg = "参数错误"
		return
	}

	stUserInfoModel, err := comm.GetUserInfoByOpenId(strOpenId)
	if err != nil || stUserInfoModel == nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		Printf("getLoginUid fail, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}
	coachId := stUserInfoModel.CoachId

	mapCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		return
	}

	if req.CoachId == 0 {
		req.CoachId = coachId
	}
	if req.GymId == 0 {
		req.GymId = mapCoach[coachId].GymID
	}

	SetLessonAvailable(req, rsp)
}

func SetLessonAvailable(req SetLessonAvailableReq, rsp *SetLessonAvailableRsp) {
	if req.GymId == 0 ||
		req.StartTs == 0 ||
		req.EndTs == 0 ||
		req.StartTs >= req.EndTs {
		rsp.Code = -10003
		rsp.ErrorMsg = "参数错误"
		return
	}

	unDayBegTs := comm.GetTodayBegTsByTs(req.StartTs)
	unDayBegTsE := comm.GetTodayBegTsByTs(req.EndTs)
	if unDayBegTs != unDayBegTsE {
		rsp.Code = -10003
		rsp.ErrorMsg = "参数错误"
	}

	vecCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentScheduleOneDay(req.GymId, req.CoachId, unDayBegTs)
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		Printf("GetAppointmentScheduleOneDay err, err:%+v req:%+v\n", err, req)
		return
	}

	if !checkTsInterval(vecCoachAppointmentModel, req.StartTs, req.EndTs) {
		rsp.Code = -900
		rsp.ErrorMsg = "时间区间内已设置，请重新选择时间区间。"
		Printf("checkTsInterval err, vecCoachAppointmentModel:%+v req:%+v\n", vecCoachAppointmentModel, req)
		return
	}

	diff := req.EndTs - req.StartTs
	if diff%3600 != 0 {
		rsp.Code = -10003
		rsp.ErrorMsg = "时间段设置需要包含完整的小时。"
		return
	}
	unitCnt := diff / 3600

	unNowTs := time.Now().Unix()
	var i int64
	for i = 0; i < unitCnt; i++ {
		var stCoachAppointmentModel model.CoachAppointmentModel
		stCoachAppointmentModel.CoachID = req.CoachId //TODO 后续填教练端登录态uid
		stCoachAppointmentModel.GymId = req.GymId
		stCoachAppointmentModel.AppointmentDate = unDayBegTs
		stCoachAppointmentModel.StartTime = req.StartTs + i*3600
		stCoachAppointmentModel.EndTime = stCoachAppointmentModel.StartTime + 3600
		stCoachAppointmentModel.Status = model.Enum_Appointment_Status_Available
		stCoachAppointmentModel.CreateTs = unNowTs
		stCoachAppointmentModel.UpdateTs = unNowTs
		err := dao.ImpAppointment.SetAppointmentSchedule(stCoachAppointmentModel)
		dayTime := time.Unix(stCoachAppointmentModel.AppointmentDate, 0)
		startTime := time.Unix(stCoachAppointmentModel.StartTime, 0)
		endTime := time.Unix(stCoachAppointmentModel.EndTime, 0)
		timeFormat := "2006-01-02 15:04:05 MST"

		if err != nil {
			rsp.Code = -977
			rsp.ErrorMsg = err.Error()
			Printf("SetAppointmentSchedule err, err:%+v stCoachAppointmentModel:%+v dayTime:%s startTime:%s endTime:%s\n",
				err, stCoachAppointmentModel, dayTime.Format(timeFormat), startTime.Format(timeFormat), endTime.Format(timeFormat))
			return
		}

		tmpSetRes, err := dao.ImpAppointment.GetAppointmentByBegTsAndEndTs(stCoachAppointmentModel.GymId, stCoachAppointmentModel.CoachID,
			stCoachAppointmentModel.StartTime, stCoachAppointmentModel.EndTime)
		if err != nil {
			//非关键路径
			Printf("GetAppointmentByBegTsAndEndTs err, err:%+v stCoachAppointmentModel:%+v\n", err, stCoachAppointmentModel)
		} else {
			rsp.AppointmentID = tmpSetRes.AppointmentID
		}

		Printf("SetAppointmentSchedule succ, stCoachAppointmentModel:%+v dayTime:%s startTime:%s endTime:%s\n",
			stCoachAppointmentModel, dayTime.Format(timeFormat), startTime.Format(timeFormat), endTime.Format(timeFormat))
	}
}

func SetDataTest(coachId int, gymId int, addTs int64) error {
	var i int64
	todayBegTs := comm.GetTodayEndTs()
	unNowTs := time.Now().Unix()
	//连续设置7天
	for i = 0; i < 7; i++ {
		var stCoachAppointmentModel model.CoachAppointmentModel
		stCoachAppointmentModel.CoachID = coachId //TODO 后续填教练端登录态uid
		stCoachAppointmentModel.GymId = gymId
		stCoachAppointmentModel.AppointmentDate = todayBegTs + i*86400
		stCoachAppointmentModel.StartTime = stCoachAppointmentModel.AppointmentDate + addTs
		stCoachAppointmentModel.EndTime = stCoachAppointmentModel.StartTime + 3600
		stCoachAppointmentModel.Status = model.Enum_Appointment_Status_Available
		stCoachAppointmentModel.CreateTs = unNowTs
		stCoachAppointmentModel.UpdateTs = unNowTs
		err := dao.ImpAppointment.SetAppointmentSchedule(stCoachAppointmentModel)
		dayTime := time.Unix(stCoachAppointmentModel.AppointmentDate, 0)
		startTime := time.Unix(stCoachAppointmentModel.StartTime, 0)
		endTime := time.Unix(stCoachAppointmentModel.EndTime, 0)
		timeFormat := "2006-01-02 15:04:05 MST"
		if err != nil {
			Printf("SetAppointmentSchedule err, err:%+v stCoachAppointmentModel:%+v dayTime:%s startTime:%s endTime:%s\n",
				err, stCoachAppointmentModel, dayTime.Format(timeFormat), startTime.Format(timeFormat), endTime.Format(timeFormat))
			return err
		}
		Printf("SetAppointmentSchedule succ, stCoachAppointmentModel:%+v dayTime:%s startTime:%s endTime:%s\n",
			stCoachAppointmentModel, dayTime.Format(timeFormat), startTime.Format(timeFormat), endTime.Format(timeFormat))
	}
	return nil
}

func checkTsInterval(vecCoachAppointmentModel []model.CoachAppointmentModel, startTs int64, endTs int64) bool {
	if len(vecCoachAppointmentModel) == 0 {
		return true
	}
	var stReqBookInterval Interval
	stReqBookInterval.Start = startTs
	stReqBookInterval.End = endTs
	for _, v := range vecCoachAppointmentModel {
		var stInterval Interval
		stInterval.Start = v.StartTime
		stInterval.End = v.EndTime

		if stInterval.Contains(stReqBookInterval) {
			return false
		}

		if stInterval.Intersects(stReqBookInterval) {
			return false
		}
	}
	return true
}

func checkAppointmentTsInterval(v model.CoachAppointmentModel, startTs int64, endTs int64) bool {
	var stReqBookInterval Interval
	stReqBookInterval.Start = startTs
	stReqBookInterval.End = endTs

	var stInterval Interval
	stInterval.Start = v.StartTime
	stInterval.End = v.EndTime

	if stInterval.Contains(stReqBookInterval) {
		return false
	}

	if stInterval.Intersects(stReqBookInterval) {
		return false
	}

	return true
}

// Interval 表示一个区间
type Interval struct {
	Start int64
	End   int64
}

// Contains 判断区间 A 是否包含区间 B
func (a Interval) Contains(b Interval) bool {
	return a.Start <= b.Start && a.End >= b.End
}

// Intersects 判断区间 A 是否与区间 B 有交集（只有边界是交集不算）
func (a Interval) Intersects(b Interval) bool {
	return a.Start < b.End && b.Start < a.End
}
