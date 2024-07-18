package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type GetSchedulingPageReq struct {
	Passback  string  `json:"passback"` //课程安排翻页回传，首次传空
}

type GetSchedulingPageRsp struct {
	Code            int               `json:"code"`
	ErrorMsg        string            `json:"errorMsg,omitempty"`
	VecTraineeUser  []TraineeUser     `json:"vec_trainee_user,omitempty"`  //我的学员列表
	VecScheduleInfo []DayScheduleInfo `json:"vec_schedule_info,omitempty"` //近7天课程列表
	Passback        string            `json:"passback,omitempty"`          //翻页带上来
	BHasMore        bool              `json:"b_has_more,omitempty"`        //是否还有后续页
}

// TraineeUser 学员用户item
type TraineeUser struct {
	Uid               int64  `json:"uid"`                 //学员的用户id
	Name              string `json:"name"`                //名字
	HeadPic           string `json:"head_pic"`            //头像
	Gender            int    `json:"gender"`              // "1=男", "2=女", "0=other"
	FitnessGoal       int    `json:"fitness_goal"`        //健身目标（训练计划）
	WithoutLessonDays int    `json:"without_lesson_days"` //未上课时间，单位天
}

func getGetSchedulingPageReq(r *http.Request) (GetSchedulingPageReq, error) {
	req := GetSchedulingPageReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetSchedulingPageHandler 拉取排课页
func GetSchedulingPageHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetSchedulingPageReq(r)
	rsp := &GetSchedulingPageRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetSchedulingPageHandler start, openid:%s\n", strOpenId)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	if len(strOpenId) == 0 {
		rsp.Code = -10003
		rsp.ErrorMsg = "param err"
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

	vecCoursePackageModel, err := dao.ImpCoursePackage.GetListByCoachIdAndLastFinishLessonTs(coachId, 30)
	if err != nil {
		rsp.Code = -888
		rsp.ErrorMsg = err.Error()
		return
	}
	//有可能存在一个学员有该教练的多个课包（需要按uid去重，保留前面的）
	uniqueVec(&vecCoursePackageModel)

	var unDayBegTs int64
	if len(req.Passback) > 0{
		unDayBegTs, _ = strconv.ParseInt(req.Passback, 10, 64)
	}else{
		unDayBegTs = comm.GetTodayBegTs()
	}

	unNowTs := time.Now().Unix()
	vecCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentScheduleFromBegTs(stCoachModel.GymID, coachId, unDayBegTs)
	if err != nil {
		rsp.Code = -995
		rsp.ErrorMsg = err.Error()
		return
	}

	//按天规整
	var vecAllUid []int64
	var vecAppointmentItem []AppointmentItem //已经有预约的所有item
	mapDayBegTs2AppointmentModel := make(map[int64][]model.CoachAppointmentModel)
	for _, v := range vecCoachAppointmentModel {
		mapDayBegTs2AppointmentModel[v.AppointmentDate] = append(mapDayBegTs2AppointmentModel[v.AppointmentDate], v)
		if v.UserID > 0 && v.AppointmentID > 0 {
			vecAllUid = append(vecAllUid, v.UserID)
			var stAppointmentItem AppointmentItem
			stAppointmentItem.Uid = v.UserID
			stAppointmentItem.AppointmentID = v.AppointmentID
			vecAppointmentItem = append(vecAppointmentItem, stAppointmentItem)
		}
	}
	Printf("GetAppointmentScheduleFromBegTs succ, CoachId:%d unDayBegTs:%d unNowTs:%d vecCoachAppointmentModel:%+v mapDayBegTs2AppointmentModel:%+v vecAllUid:%+v vecAppointmentItem:%+v\n",
		coachId, unDayBegTs, unNowTs, vecCoachAppointmentModel, mapDayBegTs2AppointmentModel, vecAllUid, vecAppointmentItem)

	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -960
		rsp.ErrorMsg = err.Error()
		return
	}

	mapCourse, err := comm.GetAllCouse()
	if err != nil {
		rsp.Code = -950
		rsp.ErrorMsg = err.Error()
		return
	}

	mapUser := make(map[int64]model.UserInfoModel)
	for _, v := range vecCoursePackageModel {
		vecAllUid = append(vecAllUid, v.Uid)
	}
	if len(vecAllUid) > 0 {
		mapUser, err = GetAllUser(vecAllUid)
		if err != nil {
			rsp.Code = -930
			rsp.ErrorMsg = err.Error()
			return
		}
	}

	//预约id对应的单节课信息
	mapAppointmentID2SingleLessonInfo := make(map[int]model.CoursePackageSingleLessonModel)
	if len(vecAppointmentItem) > 0 {
		mapAppointmentID2SingleLessonInfo, err = GetAllLessonInfo(vecAppointmentItem)
		if err != nil {
			rsp.Code = -800
			rsp.ErrorMsg = err.Error()
			return
		}
	}

	mapAppointmentID2PackageInfo := make(map[int]model.CoursePackageModel)
	if len(mapAppointmentID2SingleLessonInfo) > 0 {
		for k, v := range mapAppointmentID2SingleLessonInfo {
			stCoursePackageModel, err := dao.ImpCoursePackage.GetCoursePackageById(v.PackageID)
			if err != nil {
				Printf("GetCoursePackageById err, CoachId:%d PackageID:%s Uid:%d\n", coachId, v.PackageID, v.Uid)
				continue
			}
			mapAppointmentID2PackageInfo[k] = *stCoursePackageModel
		}
	}

	//设置默认值
	for i := 0; i < 7; i++ {
		var stDayScheduleInfo DayScheduleInfo
		stDayScheduleInfo.DayBegTs = unDayBegTs + int64(i*86400)
		rsp.VecScheduleInfo = append(rsp.VecScheduleInfo, stDayScheduleInfo)
	}
	for i := 0; i < 7; i++ {
		dayBegTs := rsp.VecScheduleInfo[i].DayBegTs
		if _, ok := mapDayBegTs2AppointmentModel[dayBegTs]; !ok {
			continue
		}
		vecCoachAppointmentModelOfDay := mapDayBegTs2AppointmentModel[dayBegTs]
		for _, v := range vecCoachAppointmentModelOfDay {
			var stScheduleRspItem ScheduleItem
			stScheduleRspItem.AppointmentID = v.AppointmentID
			stScheduleRspItem.StartTs = v.StartTime
			stScheduleRspItem.EndTs = v.EndTime

			if v.UserID == 0 {
				stScheduleRspItem.ScheduleType = En_ScheduleType_Available
			} else {
				if v.ScheduledByCoach {
					stScheduleRspItem.ScheduleType = En_ScheduleType_BookedByCoach
				} else {
					stScheduleRspItem.ScheduleType = En_ScheduleType_BookedByTrainee
				}

				//需要的是单次课里更新的状态
				stScheduleRspItem.ScheduledByCoach = v.ScheduledByCoach
				stScheduleRspItem.Status = mapAppointmentID2SingleLessonInfo[v.AppointmentID].Status
				stScheduleRspItem.GymName = mapGym[v.GymId].LocName
				stScheduleRspItem.CourseName = mapCourse[v.UserCourseID].Name
				stScheduleRspItem.CourseChargeType = mapCourse[v.UserCourseID].ChargeType
				stScheduleRspItem.TraineePlan = mapUser[v.UserID].FitnessGoal
				stScheduleRspItem.TraineeUid = v.UserID
				stScheduleRspItem.TraineeUserName = mapUser[v.UserID].Nick
				stScheduleRspItem.TraineeUserHeadPic = mapUser[v.UserID].HeadPic
				if mapUser[v.UserID].PhoneNumber != nil {
					stScheduleRspItem.TraineeUserPhone = *mapUser[v.UserID].PhoneNumber
				}
				stCoursePackageModel := mapAppointmentID2PackageInfo[v.AppointmentID]
				stScheduleRspItem.PackageProgress = fmt.Sprintf("剩余%d/%d节", stCoursePackageModel.RemainCnt, stCoursePackageModel.TotalCnt)
			}
			rsp.VecScheduleInfo[i].VecScheduleItem = append(rsp.VecScheduleInfo[i].VecScheduleItem, stScheduleRspItem)
		}
	}

	for _, v := range vecCoursePackageModel {
		if userInfo, ok := mapUser[v.Uid]; ok {
			rsp.VecTraineeUser = append(rsp.VecTraineeUser, transUserModel2TraineeUser(v.LastLessonTs, userInfo))
		}
	}

	if len(req.Passback) == 0{
		unNextWeekTs := unDayBegTs + 604800
		rsp.Passback = fmt.Sprintf("%d", unNextWeekTs)
		rsp.BHasMore = true
	}else{
		rsp.BHasMore = false
	}
}

func transUserModel2TraineeUser(LastLessonTs int64, userInfo model.UserInfoModel) TraineeUser {
	var stTraineeUser TraineeUser
	stTraineeUser.Uid = userInfo.UserID
	stTraineeUser.Name = userInfo.Nick
	stTraineeUser.HeadPic = userInfo.HeadPic
	stTraineeUser.Gender = userInfo.Gender
	stTraineeUser.FitnessGoal = userInfo.FitnessGoal
	stTraineeUser.WithoutLessonDays = comm.CalculateDaysSinceTimestamp(LastLessonTs)
	return stTraineeUser
}

func uniqueVec(vecID *[]model.CoursePackageModel) error {
	tmpMap := make(map[int64]bool)

	tmpVecID := make([]model.CoursePackageModel, 0)
	for _, v := range *vecID {
		if _, ok := tmpMap[v.Uid]; ok {
			continue
		} else {
			tmpMap[v.Uid] = true
			tmpVecID = append(tmpVecID, v)
		}

	}
	*vecID = tmpVecID
	return nil
}
