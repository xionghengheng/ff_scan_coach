package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type GetTraineePageReq struct {
	TraineeUid int64 `json:"trainee_uid"` //学员uid
}

type GetTraineePageRsp struct {
	Code               int                 `json:"code"`
	ErrorMsg           string              `json:"errorMsg,omitempty"`
	TraineeUser        model.UserInfoModel `json:"trainee_user,omitempty"`
	RemainCnt          int                 `json:"remain_cnt,omitempty"`           //所有课包总的剩余课时数
	GoLessonRecentDay  int                 `json:"go_lesson_recent_day,omitempty"` //活跃度，近30天上课次数
	LastLessonTs       int64               `json:"last_lesson_ts,omitempty"`       //上次上课时间
	FirstLessonTs      int64               `json:"first_lesson_ts,omitempty"`      //首次上课时间
	BookedLessonCnt    int                 `json:"booked_lesson_cnt,omitempty"`    //预约中的课数
	CompletedLessonCnt int                 `json:"completed_lesson_cnt,omitempty"` //已完成的课数
	VecLessonInfo      []SingleLesson      `json:"vec_lesson_info,omitempty"`
}

type SingleLesson struct {
	CoursePackageId string `json:"course_package_id"` // 课包ID
	LessonID        string `json:"lesson_id"`         // 单节课的唯一标识符（用户id_场地id_课程id_教练id_发起预约的时间戳）
	AddressName     string `json:"address_name"`
	CourseName      string `json:"course_name"`
	Status          int    `json:"status,omitempty"`
	ScheduleBegTs   int64  `json:"schedule_beg_ts"` // 单节课的安排上课开始时间
	ScheduleEndTs   int64  `json:"schedule_end_ts"` // 单节课的安排上课结束时间
	AppointmentID   int    `json:"appointment_id"`  // 预约ID
}

func getGetTraineePageReq(r *http.Request) (GetTraineePageReq, error) {
	req := GetTraineePageReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetTraineePageHandler 拉取学员主页
func GetTraineePageHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetTraineePageReq(r)
	rsp := &GetTraineePageRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetTraineePageHandler start, openid:%s\n", strOpenId)

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

	if len(strOpenId) == 0 || req.TraineeUid == 0 {
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

	stTraineeUserInfoModel, err := comm.GetUserInfoByUid(req.TraineeUid)
	if err != nil || stTraineeUserInfoModel == nil {
		rsp.Code = -900
		rsp.ErrorMsg = err.Error()
		Printf("GetUserInfoByUid err, err:%+v coachid:%d uid:%d TraineeUid:%d\n", err, coachId, uid, req.TraineeUid)
		return
	}
	rsp.TraineeUser = *stTraineeUserInfoModel

	vecCoursePackageModel, err := dao.ImpCoursePackage.GetAllPackageListByCoachIdAndUid(coachId, stTraineeUserInfoModel.UserID)
	if err != nil || stTraineeUserInfoModel == nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		Printf("GetAllPackageListByCoachIdAndUid err, err:%+v coachid:%d uid:%d TraineeUid:%d\n", err, coachId, uid, req.TraineeUid)
		return
	}
	for _, v := range vecCoursePackageModel {
		rsp.RemainCnt += v.RemainCnt
	}

	unLast30DayTs := time.Now().Unix() - 2592000
	var vecAllLesson []model.CoursePackageSingleLessonModel
	for _, v := range vecCoursePackageModel {
		vecLessonModel, err := dao.ImpCoursePackageSingleLesson.GetSingleLessonListByPackageId(v.Uid, v.PackageID)
		if err != nil {
			rsp.Code = -866
			rsp.ErrorMsg = err.Error()
			Printf("GetSingleLessonListByPackageId err, strOpenId:%s coachId:%d err:%+v\n", strOpenId, coachId, err)
			continue
		}
		if len(vecLessonModel) == 0 {
			continue
		}
		for _, lesson := range vecLessonModel {
			vecAllLesson = append(vecAllLesson, lesson)
			if lesson.Status == model.En_LessonStatus_Scheduled {
				rsp.BookedLessonCnt += 1
			} else if lesson.Status == model.En_LessonStatusCompleted {
				rsp.CompletedLessonCnt += 1
				if lesson.ScheduleBegTs >= unLast30DayTs{
					rsp.GoLessonRecentDay += 1
				}
			}
		}
	}
	// 按照 ScheduleBegTs 从大到小排序，越新的课程排越前面
	sort.Slice(vecAllLesson, func(i, j int) bool {
		return vecAllLesson[i].ScheduleBegTs > vecAllLesson[j].ScheduleBegTs
	})
	for i := 0; i < len(vecAllLesson); i++ {
		if vecAllLesson[i].Status == model.En_LessonStatusCompleted {
			rsp.FirstLessonTs = vecAllLesson[i].ScheduleBegTs
			break
		}
	}
	for i := len(vecAllLesson) - 1; i >= 0; i-- {
		if vecAllLesson[i].Status == model.En_LessonStatusCompleted {
			rsp.LastLessonTs = vecAllLesson[i].ScheduleBegTs
			break
		}
	}

	if len(vecAllLesson) > 0 {
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

		for _, v := range vecAllLesson {
			var rspItem SingleLesson
			rspItem.CoursePackageId = v.PackageID
			rspItem.LessonID = v.LessonID
			rspItem.AddressName = mapGym[v.GymId].LocName
			rspItem.CourseName = mapCourse[v.CourseID].Name
			rspItem.Status = v.Status
			rspItem.ScheduleBegTs = v.ScheduleBegTs
			rspItem.ScheduleEndTs = v.ScheduleEndTs
			rspItem.AppointmentID = v.AppointmentID
			rsp.VecLessonInfo = append(rsp.VecLessonInfo, rspItem)
		}
	}

	//清红点
	var stTraineeReddotModel model.CoachClientTraineeReddotModel
	stTraineeReddotModel.CoachId = coachId
	stTraineeReddotModel.TraineeUid = req.TraineeUid
	stTraineeReddotModel.VisitTs = time.Now().Unix()
	err = dao.ImpCoachClientTraineeReddot.AddTraineeReddotVisit(&stTraineeReddotModel)
	if err != nil {
		//红点设置失败还好，报错直接忽略
		Printf("AddTraineeReddotVisit err, err:%+v coachid:%d uid:%d TraineeUid:%d\n", err, coachId, uid, req.TraineeUid)
	}
}
