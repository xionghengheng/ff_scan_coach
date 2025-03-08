package main

import (
	"encoding/json"
	"fmt"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"net/http"
)

type GetAllTrailPackageReq struct {
	RemainCnt int `json:"type"` // 体验课剩余课时数，RemainCnt=-1 默认全部，RemainCnt=0 剩余0节，RemainCnt=1 剩余1节，RemainCnt=2 剩余2节
}

type GetAllTrailPackageRsp struct {
	Code                int    `json:"code"`
	ErrorMsg            string `json:"errorMsg,omitempty"`
	VecTrailPackageItem []TrailPackageItem
}

type TrailPackageItem struct {
	Uid          int64  `json:"uid"`            // 用户id
	UserName     string `json:"user_name"`      // 用户名称
	PhoneNumber  string `json:"phone_number"`   // 手机号
	PackageID    string `json:"package_id"`     // 课包的唯一标识符（用户id_获取课包的时间戳）
	GymId        int    `json:"gym_id"`         // 场地id
	GymName      string `json:"gym_name"`       // 场地id
	CourseId     int    `json:"course_id"`      // 课程id
	CourseName   string `json:"course_name"`    // 场地名称
	CoachId      int    `json:"coach_id"`       // 教练id
	CoachName    string `json:"coach_name"`     // 教练名称
	Ts           int64  `json:"ts"`             // 获得课包的时间戳
	TotalCnt     int    `json:"total_cnt"`      // 课包中总的课程次数
	RemainCnt    int    `json:"remain_cnt"`     // 课包中剩余的课程次数
	Price        int    `json:"price"`          // 价格
	LastLessonTs int64  `json:"last_lesson_ts"` // 上次约课时间
}

func getGetAllTrailPackageReq(r *http.Request) (GetAllTrailPackageReq, error) {
	req := GetAllTrailPackageReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetAllTrailPackageHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetAllTrailPackageReq(r)
	rsp := &GetAllTrailPackageRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetAllTrailPackageHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	var vecAllTrailPackageModel []model.CoursePackageModel
	var turnPageTs int64
	for i := 0; i <= 5000; i++ {
		tmpVecAllTrailPackageModel, err := dao.ImpCoursePackage.GetAllTrailCoursePackageList(turnPageTs)
		if err != nil {
			Printf("GetAllCoursePackageList err, i:%d err:%+v\n", i, err)
			return
		}
		if len(tmpVecAllTrailPackageModel) == 0 {
			Printf("GetAllCoursePackageList empty, i:%d vecAllPackageModel.len:%d\n", i, len(vecAllTrailPackageModel))
			break
		}
		turnPageTs = tmpVecAllTrailPackageModel[len(tmpVecAllTrailPackageModel)-1].Ts
		vecAllTrailPackageModel = append(vecAllTrailPackageModel, tmpVecAllTrailPackageModel...)
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

	if req.RemainCnt == -1 {
		for _, v := range vecAllTrailPackageModel {
			rsp.VecTrailPackageItem = append(rsp.VecTrailPackageItem, ConvertCourseItemModelToRTrailspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym))
		}
	} else if req.RemainCnt == 0 {
		for _, v := range vecAllTrailPackageModel {
			if v.RemainCnt == 0 {
				rsp.VecTrailPackageItem = append(rsp.VecTrailPackageItem, ConvertCourseItemModelToRTrailspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym))
			}
		}
	} else if req.RemainCnt == 1 {
		for _, v := range vecAllTrailPackageModel {
			if v.RemainCnt == 1 {
				rsp.VecTrailPackageItem = append(rsp.VecTrailPackageItem, ConvertCourseItemModelToRTrailspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym))
			}
		}
	} else if req.RemainCnt == 2 {
		for _, v := range vecAllTrailPackageModel {
			if v.RemainCnt == 2 {
				rsp.VecTrailPackageItem = append(rsp.VecTrailPackageItem, ConvertCourseItemModelToRTrailspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym))
			}
		}
	}
	return
}

// 转换函数
func ConvertCourseItemModelToRTrailspItem(item model.CoursePackageModel,
	mapAllCoach map[int]model.CoachModel,
	mapALlCourseModel map[int]model.CourseModel,
	mapAllUserModel map[int64]model.UserInfoModel,
	mapGym map[int]model.GymInfoModel) TrailPackageItem {

	strPhone := ""
	phone := mapAllUserModel[item.Uid].PhoneNumber
	if phone != nil {
		strPhone = *phone
	}

	return TrailPackageItem{
		Uid:          item.Uid,
		UserName:     mapAllUserModel[item.Uid].Nick,
		PhoneNumber:  strPhone,
		PackageID:    item.PackageID,
		GymId:        mapGym[item.GymId].GymID,
		GymName:      mapGym[item.GymId].LocName,
		CourseId:     item.CourseId,
		CourseName:   mapALlCourseModel[item.CourseId].Name,
		CoachId:      item.CoachId,
		CoachName:    mapAllCoach[item.CoachId].CoachName,
		Ts:           item.Ts,
		TotalCnt:     item.TotalCnt,
		RemainCnt:    item.RemainCnt,
		Price:        item.Price,
		LastLessonTs: item.LastLessonTs,
	}
}
