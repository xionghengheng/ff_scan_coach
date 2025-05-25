package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

type GetAllPaidLessonReq struct {
}

type GetAllPaidLessonRsp struct {
	Code              int    `json:"code"`
	ErrorMsg          string `json:"errorMsg,omitempty"`
	VecPaidLessonItem []PaidLessonItem
}

type PaidLessonItem struct {
	Uid              int64  `json:"uid"`                // 用户id
	UserName         string `json:"user_name"`          // 用户名称
	PhoneNumber      string `json:"phone_number"`       // 手机号
	PackageID        string `json:"package_id"`         // 课包的唯一标识符（用户id_获取课包的时间戳）
	LessonID         string `json:"lesson_id"`          // 单节课的唯一标识符（用户id_场地id_课程id_教练id_发起预约的时间戳）
	TotalCnt         int    `json:"total_cnt"`          // 课包中总的课程次数
	RemainCnt        int    `json:"remain_cnt"`         // 课包中剩余的课程次数
	GymId            int    `json:"gym_id"`             // 场地id
	GymName          string `json:"gym_name"`           // 场地id
	CourseId         int    `json:"course_id"`          // 课程id
	CourseName       string `json:"course_name"`        // 课程名称
	CoursePrice      int    `json:"course_price"`       // 课程价格
	CoachId          int    `json:"coach_id"`           // 教练id
	CoachName        string `json:"coach_name"`         // 教练名称
	CreateTs         int64  `json:"create_ts"`          // 记录生成时间，发起预约的时间
	ScheduleBegTs    int64  `json:"schedule_beg_ts"`    // 单节课的安排上课时间
	ScheduleEndTs    int64  `json:"schedule_end_ts"`    // 单节课的安排上课时间
	Status           int    `json:"status"`             // 单次课状态(已预约、已完成、已取消、已旷课)
	LessonName       string `json:"lesson_name"`        // 单节课的名称
	Duration         int    `json:"duration"`           // 单节课的时长，单位秒
	CancelByCoach    bool   `json:"cancel_by_coach"`    // 是否是教练取消
	ScheduledByCoach bool   `json:"scheduled_by_coach"` // 是否为教练排课
	WriteOffTs       int64  `json:"write_off_ts"`       // 核销时间
}

func getGetAllPaidLessonReq(r *http.Request) (GetAllPaidLessonReq, error) {
	req := GetAllPaidLessonReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetAllPaidLessonHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetAllPaidLessonReq(r)
	rsp := &GetAllPaidLessonRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetAllPaidLessonHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	mapAllPaidPackageModel := make(map[string]model.CoursePackageModel)
	var turnPageTs int64
	for i := 0; i <= 5000; i++ {
		tmpVecAllTrailPackageModel, err := dao.ImpCoursePackage.GetAllPaidCoursePackageList(turnPageTs)
		if err != nil {
			Printf("GetAllCoursePackageList err, i:%d err:%+v\n", i, err)
			return
		}
		if len(tmpVecAllTrailPackageModel) == 0 {
			Printf("GetAllCoursePackageList empty, i:%d mapAllPaidPackageModel.len:%d\n", i, len(mapAllPaidPackageModel))
			break
		}
		turnPageTs = tmpVecAllTrailPackageModel[len(tmpVecAllTrailPackageModel)-1].Ts
		for _, v := range tmpVecAllTrailPackageModel {
			mapAllPaidPackageModel[v.PackageID] = v
		}
	}

	var vecAllSingleLesson []model.CoursePackageSingleLessonModel
	turnPageTs = 0
	for i := 0; i <= 5000; i++ {
		tmpVecAllSingleLesson, err := dao.ImpCoursePackageSingleLesson.GetAllSingleLessonList(turnPageTs)
		if err != nil {
			rsp.Code = -911
			rsp.ErrorMsg = err.Error()
			Printf("GetAllSingleLessonList err, turnPageTs:%d err:%+v\n", turnPageTs, err)
			return
		}
		if len(tmpVecAllSingleLesson) == 0 {
			Printf("GetAllSingleLessonList empty, turnPageTs:%d vecAllSingleLesson.len:%d\n", turnPageTs, len(vecAllSingleLesson))
			break
		}
		turnPageTs = tmpVecAllSingleLesson[len(tmpVecAllSingleLesson)-1].CreateTs
		vecAllSingleLesson = append(vecAllSingleLesson, tmpVecAllSingleLesson...)
	}

	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		return
	}

	mapALlCourseModel, err := comm.GetAllCourse()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		return
	}

	mapAllUserModel, err := comm.GetAllUser()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		return
	}

	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		return
	}

	for _, v := range vecAllSingleLesson {
		rsp.VecPaidLessonItem = append(rsp.VecPaidLessonItem, ConvertCourseItemModel2PaidRspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym, mapAllPaidPackageModel))
	}
	return
}

// 转换函数
func ConvertCourseItemModel2PaidRspItem(item model.CoursePackageSingleLessonModel,
	mapAllCoach map[int]model.CoachModel,
	mapALlCourseModel map[int]model.CourseModel,
	mapAllUserModel map[int64]model.UserInfoModel,
	mapGym map[int]model.GymInfoModel,
	mapAllPaidPackageModel map[string]model.CoursePackageModel) PaidLessonItem {

	strPhone := ""
	phone := mapAllUserModel[item.Uid].PhoneNumber
	if phone != nil {
		strPhone = *phone
	}

	return PaidLessonItem{
		Uid:              item.Uid,
		UserName:         mapAllUserModel[item.Uid].Nick,
		PhoneNumber:      strPhone,
		PackageID:        item.PackageID,
		LessonID:         item.LessonID,
		TotalCnt:         mapAllPaidPackageModel[item.PackageID].TotalCnt,
		RemainCnt:        mapAllPaidPackageModel[item.PackageID].RemainCnt,
		GymId:            mapGym[item.GymId].GymID,
		GymName:          mapGym[item.GymId].LocName,
		CourseId:         item.CourseID,
		CourseName:       mapALlCourseModel[item.CourseID].Name,
		CoursePrice:      mapALlCourseModel[item.CourseID].Price,
		CoachId:          item.CoachId,
		CoachName:        mapAllCoach[item.CoachId].CoachName,
		CreateTs:         item.CreateTs,
		ScheduleBegTs:    item.ScheduleBegTs,
		ScheduleEndTs:    item.ScheduleEndTs,
		Status:           item.Status,
		LessonName:       item.LessonName,
		Duration:         3600,
		CancelByCoach:    item.CancelByCoach,
		ScheduledByCoach: item.ScheduledByCoach,
		WriteOffTs:       item.WriteOffTs,
	}
}
