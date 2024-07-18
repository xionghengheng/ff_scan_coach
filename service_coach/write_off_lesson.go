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

// 在上课预定时间内，完成课程
type WriteOffLessonReq struct {
	LessonID string `json:"lesson_id"` // 单节课的唯一标识符（用户id_场地id_课程id_教练id_发起预约的时间戳）
}

type WriteOffLessonRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

func getWriteOffLessonReq(r *http.Request) (WriteOffLessonReq, error) {
	stWriteOffLessonReq := WriteOffLessonReq{}
	if err := json.NewDecoder(r.Body).Decode(&stWriteOffLessonReq); err != nil {
		return stWriteOffLessonReq, err
	}
	defer r.Body.Close()
	return stWriteOffLessonReq, nil
}

// WriteOffLessonHandler
func WriteOffLessonHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getWriteOffLessonReq(r)
	rsp := &WriteOffLessonRsp{}
	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()
	Printf("WriteOffLessonHandler start, req:%+v strOpenId:%s\n", req, strOpenId)

	if err != nil {
		rsp.Code = -999
		rsp.ErrorMsg = err.Error()
		return
	}

	if len(strOpenId) == 0 || len(req.LessonID) == 0 {
		rsp.Code = -10003
		rsp.ErrorMsg = "参数错误"
		return
	}

	coachId, err := comm.GetCoachIdByOpenId(strOpenId)
	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		Printf("getLoginUid fail, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}
	if coachId == 0 {
		rsp.Code = -10003
		rsp.ErrorMsg = "不是教练，无法核销课程"
		Printf("not Coach return, coachId:%d LessonID:%s\n", coachId, req.LessonID)
		return
	}

	traineeUid, _, _, coachIdFromLessonId, _ := comm.ParseCoursePackageSingleLessonID(req.LessonID)
	if coachId != coachIdFromLessonId {
		rsp.Code = -10003
		rsp.ErrorMsg = "教练无法核销不属于自己的课程"
		Printf("writeOffLesson err, coachId:%d coachIdFromLessonId:%d\n", coachId, coachIdFromLessonId)
		return
	}

	stSingleLessonModel, err := dao.ImpCoursePackageSingleLesson.GetSingleLessonById(traineeUid, req.LessonID)
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		Printf("GetSingleLessonById err, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}
	if stSingleLessonModel.Status == model.En_LessonStatusCompleted {
		rsp.Code = -113
		rsp.ErrorMsg = "已核销过了，无需重复核销"
		Printf("WriteOff dup return, LessonID:%s coachId:%d traineeUid:%d\n", req.LessonID, coachId, traineeUid)
		return
	}

	if stSingleLessonModel.WriteOffMissedReturnCnt {
		err = dao.ImpCoursePackage.SubCourseCnt(stSingleLessonModel.PackageID)
		if err != nil {
			rsp.Code = -922
			rsp.ErrorMsg = err.Error()
			Printf("SubCourseCnt err, strOpenId:%s traineeUid:%d PackageID:%d err:%+v\n", strOpenId, traineeUid, stSingleLessonModel.PackageID, err)
			return
		}
		Printf("SubCourseCnt succ, strOpenId:%s traineeUid:%d PackageID:%+s\n", strOpenId, traineeUid, stSingleLessonModel.PackageID)
	}

	//将用户课包里的单节课状态变成已完成
	mapUpdates := make(map[string]interface{})
	mapUpdates["status"] = model.En_LessonStatusCompleted
	err = dao.ImpCoursePackageSingleLesson.UpdateSingleLesson(traineeUid, req.LessonID, mapUpdates)
	if err != nil {
		rsp.Code = -966
		rsp.ErrorMsg = err.Error()
		return
	}
	Printf("UpdateSingleLesson En_LessonStatusCompleted succ, strOpenId:%s traineeUid:%d LessonID:%s\n", strOpenId, traineeUid, req.LessonID)

	//模板配置链接：https://mp.weixin.qq.com/wxamp/newtmpl/tmpldetail?type=2&pri_tmpl_id=ys_Ch2lOt-fsB-C4NdJh1zx5BgCAdgVuEJgezK0GLiA&flag=undefined&token=1034864027&lang=zh_CN
	stCourseModel, err := dao.ImpCourse.GetCourseById(stSingleLessonModel.CourseID)
	stCoachModel, err := dao.ImpCoach.GetCoachById(stSingleLessonModel.CoachId)
	t := time.Unix(stSingleLessonModel.ScheduleBegTs, 0)
	stWxSendMsg2UserReq := comm.WxSendMsg2UserReq{
		ToUser:           strOpenId,
		TemplateID:       "kENL0EQdSD5gvtUAPh58n923AwBEio7tec6e1bC2sb0",
		Page:             fmt.Sprintf("pages/course/order-info-detail/index?lesson_id=%s", stSingleLessonModel.LessonID),
		MiniprogramState: "trial",
		Lang:             "zh_CN",
		Data: map[string]comm.MsgDataField{
			"thing1": {Value: stCourseModel.Name},            //课程名称
			"name2":  {Value: stCoachModel.CoachName},        //教练名称
			"date3":  {Value: t.Format("2006年01月02日 15:04")}, //上课时间
			"thing4": {Value: "你有新的待评价课程，快去评价吧"},             //温馨提示
		},
	}
	err = comm.SendMsg2User(traineeUid, stWxSendMsg2UserReq)
	if err != nil {
		Printf("sendMsg2User err, err:%+v traineeUid:%d PackageID:%s LessonID:%s", err, traineeUid, stSingleLessonModel.PackageID, stSingleLessonModel.LessonID)
	} else {
		Printf("sendMsg2User succ, traineeUid:%d PackageID:%s LessonID:%s", traineeUid, stSingleLessonModel.PackageID, stSingleLessonModel.LessonID)
	}

	//非关键路径，把最后一次完成课程的时间更新到课包结构里
	mapPackageUpdates := make(map[string]interface{})
	mapPackageUpdates["last_lesson_ts"] = time.Now().Unix()
	err = dao.ImpCoursePackage.UpdateCoursePackage(traineeUid, stSingleLessonModel.PackageID, mapPackageUpdates)
	if err != nil {
		rsp.Code = -900
		rsp.ErrorMsg = err.Error()
		Printf("UpdateCoursePackage err, traineeUid:%d CoursePackageId:%s err:%+v mapPackageUpdates:%+v\n", traineeUid, stSingleLessonModel.PackageID, err, mapPackageUpdates)
		return
	}
	Printf("UpdateCoursePackage succ, traineeUid:%d CoursePackageId:%s err:%+v mapPackageUpdates:%+v\n", traineeUid, stSingleLessonModel.PackageID, mapPackageUpdates)
}
