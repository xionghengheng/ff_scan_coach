package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

type GetTrailLessonListReq struct {
}

type GetTrailLessonListRsp struct {
	Code                 int                 `json:"code"`
	ErrorMsg             string              `json:"errorMsg,omitempty"`
	VecTrailBookedLesson []TrailBookedLesson `json:"vec_trail_booked_lesson,omitempty"` //学员已预约的所有体验课列表
}

type TrailBookedLesson struct {
	PackageID          string `json:"package_id"`                //课包的唯一标识符（用户id_获取课包的时间戳）
	LessonID           string `json:"lesson_id"`                 //单节课的唯一标识符（用户id_场地id_课程id_教练id_发起预约的时间戳）
	CourseId           int    `json:"course_id,omitempty"`       //课程id
	CourseTitle        string `json:"course_title,omitempty"`    //课程名称
	CourseDuration     int    `json:"course_duration,omitempty"` //课时
	GymId              int    `json:"gym_id,omitempty"`          //门店id
	GymName            string `json:"gym_name,omitempty"`        //门店名称
	ScheduleBegTs      int64  `json:"schedule_beg_ts,omitempty"` //预约时间-开始时间
	ScheduleEndTs      int64  `json:"schedule_end_ts,omitempty"` //预约时间-结束时间
	AppointmentID      int    `json:"appointment_id"`            //预约ID
	TraineeUid         int64  `json:"trainee_uid"`               //学员uid
	TraineeUserName    string `json:"trainee_user_name"`         //学员名称
	TraineeUserHeadPic string `json:"trainee_user_head_pic"`     //学员头像
	TraineeUserPhone   string `json:"trainee_user_phone"`        //学员手机号
}

// GetTrailLessonListHandler 拉取已预约的所有体验课课程列表
func GetTrailLessonListHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	rsp := &GetTrailLessonListRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetTrailLessonListHandler start, openid:%s\n", strOpenId)

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

	var vecBookedLesson []model.CoursePackageSingleLessonModel
	vecTrailCoursePackageModel, err := dao.ImpCoursePackage.GetTrailCoursePackageListByCoachId(coachId, 100)
	if err != nil {
		rsp.Code = -866
		rsp.ErrorMsg = err.Error()
		Printf("GetTrailCoursePackageListByCoachId err, strOpenId:%s coachId:%d err:%+v\n", strOpenId, coachId, err)
		return
	}

	for _, v := range vecTrailCoursePackageModel {
		vecCoursePackageSingleLessonModel, err := dao.ImpCoursePackageSingleLesson.GetSingleLessonListByPackageId(v.Uid, v.PackageID)
		if err != nil {
			rsp.Code = -866
			rsp.ErrorMsg = err.Error()
			Printf("GetSingleLessonListByPackageId err, strOpenId:%s coachId:%d err:%+v\n", strOpenId, coachId, err)
			continue
		}
		if len(vecCoursePackageSingleLessonModel) == 0 {
			continue
		}
		for _, lesson := range vecCoursePackageSingleLessonModel {
			if lesson.Status == model.En_LessonStatus_Scheduled {
				vecBookedLesson = append(vecBookedLesson, lesson)
			}
		}
	}

	if len(vecBookedLesson) == 0 {
		return
	}
	// 按照 ScheduleBegTs 从小到大排序，确保最先开始的课程在最前面
	sort.Slice(vecBookedLesson, func(i, j int) bool {
		return vecBookedLesson[i].ScheduleBegTs < vecBookedLesson[j].ScheduleBegTs
	})

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
	var vecAllUid []int64
	for _, v := range vecBookedLesson {
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

	for _, v := range vecBookedLesson {
		var stTrailBookedLesson TrailBookedLesson
		stTrailBookedLesson.PackageID = v.PackageID
		stTrailBookedLesson.LessonID = v.LessonID
		stTrailBookedLesson.CourseId = v.CourseID
		stTrailBookedLesson.CourseTitle = mapCourse[v.CourseID].Name
		stTrailBookedLesson.CourseDuration = 60
		stTrailBookedLesson.GymId = v.GymId
		stTrailBookedLesson.GymName = mapGym[v.GymId].LocName
		stTrailBookedLesson.ScheduleBegTs = v.ScheduleBegTs
		stTrailBookedLesson.ScheduleEndTs = v.ScheduleEndTs
		stTrailBookedLesson.AppointmentID = v.AppointmentID
		stTrailBookedLesson.TraineeUid = v.Uid
		stTrailBookedLesson.TraineeUserName = mapUser[v.Uid].Nick
		stTrailBookedLesson.TraineeUserHeadPic = mapUser[v.Uid].HeadPic
		if mapUser[v.Uid].PhoneNumber != nil {
			stTrailBookedLesson.TraineeUserPhone = *mapUser[v.Uid].PhoneNumber
		}
		rsp.VecTrailBookedLesson = append(rsp.VecTrailBookedLesson, stTrailBookedLesson)
	}
}
