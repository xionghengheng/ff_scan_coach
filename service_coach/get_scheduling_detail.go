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

type GetSchedulingDetailReq struct {
	TraineeUid int64 `json:"trainee_uid"` //学员uid
}

type GetSchedulingDetailRsp struct {
	Code                     int                     `json:"code"`
	ErrorMsg                 string                  `json:"errorMsg,omitempty"`
	TraineeUserInfo          TraineeUser             `json:"trainee_user_info,omitempty"`  //学员信息
	PackageID                string                  `json:"package_id"`                   //课包的唯一标识符（用户id_获取课包的时间戳）
	CourseName               string                  `json:"course_name"`                  //课程名称
	GymId                    int                     `json:"gym_id"`                       //场地id
	GymName                  string                  `json:"gym_name"`                     //场地名称
	VecSimpleDayScheduleInfo []SimpleDayScheduleInfo `json:"vec_simple_day_schedule_info"` //可设置时间段
}

type SimpleDayScheduleInfo struct {
	DayBegTs        int64                `json:"day_beg_ts,omitempty"` //当天零点开始时间
	VecScheduleItem []SimpleScheduleItem `json:"schedule_list,omitempty"`
}

type SimpleScheduleItem struct {
	StartTs    int64  `json:"start_ts"`               //起始时间
	EndTs      int64  `json:"end_ts"`                 //结束时间
	StartTsStr string `json:"start_ts_str,omitempty"` //调试用，后面删除
	EndTsStr   string `json:"end_ts_str,omitempty"`   //调试用，后面删除
	BAvailable bool   `json:"b_available,omitempty"`  //是否可排课
}

func getGetSchedulingDetailReq(r *http.Request) (GetSchedulingDetailReq, error) {
	req := GetSchedulingDetailReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetSchedulingDetailHandler 点击排课后，拉取某个学员的可排课时间信息
func GetSchedulingDetailHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetSchedulingDetailReq(r)
	rsp := &GetSchedulingDetailRsp{}
	unDayBegTs := comm.GetTodayBegTs()

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	if len(strOpenId) == 0 || req.TraineeUid == 0 {
		rsp.Code = -10003
		rsp.ErrorMsg = "param err"
		return
	}

	coachId, err := comm.GetCoachIdByOpenId(strOpenId)
	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		Printf("getLoginUid fail, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetSchedulingDetailHandler start, openid:%s unDayBegTs:%d coachId:%d\n", strOpenId, unDayBegTs, coachId)

	if coachId == 0 {
		rsp.Code = -900
		rsp.ErrorMsg = "not coach return"
		Printf("not coach err, strOpenId:%s\n", strOpenId)
		return
	}

	stCoachModel, err := dao.ImpCoach.GetCoachById(coachId)
	if err != nil || stCoachModel == nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		Printf("GetCoachById err, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}

	//获取学员用户已预约的时间段
	vecTraineeAppointment, err := dao.ImpAppointment.GetUserAppointmentRecordFromBegTs(req.TraineeUid, unDayBegTs, 100)
	if err != nil {
		rsp.Code = -300
		rsp.ErrorMsg = err.Error()
		Printf("GetUserAppointmentRecordFromBegTs err, coachId:%d TraineeUid:%d err:%+v\n", coachId, req.TraineeUid, err)
		return
	}
	//按天规整
	mapDay2VecTraineeAppointment := make(map[int64][]model.CoachAppointmentModel)
	for _, v := range vecTraineeAppointment {
		mapDay2VecTraineeAppointment[v.AppointmentDate] = append(mapDay2VecTraineeAppointment[v.AppointmentDate], v)
	}
	Printf("GetUserAppointmentRecord succ, coachId:%d TraineeUid:%d mapDay2VecTraineeAppointment:%+v\n", coachId, req.TraineeUid, mapDay2VecTraineeAppointment)

	//获取教练当天后所有设置的时间段
	vecCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentScheduleHasUidFromBegTs(stCoachModel.GymID, coachId, unDayBegTs)
	if err != nil {
		rsp.Code = -995
		rsp.ErrorMsg = err.Error()
		return
	}
	//按天规整
	mapDay2VecCoachAppointment := make(map[int64][]model.CoachAppointmentModel)
	for _, v := range vecCoachAppointmentModel {
		mapDay2VecCoachAppointment[v.AppointmentDate] = append(mapDay2VecCoachAppointment[v.AppointmentDate], v)
	}
	Printf("GetCoachAppointmentRecord succ, coachId:%d TraineeUid:%d mapDay2VecCoachAppointment:%+v\n", coachId, req.TraineeUid, mapDay2VecCoachAppointment)

	//初始化14天可排课列表，每天的最小单元是半小时
	var vecSimpleDayScheduleInfo []SimpleDayScheduleInfo
	for i := 0; i < 14; i++ {
		var stSimpleDayScheduleInfo SimpleDayScheduleInfo
		stSimpleDayScheduleInfo.DayBegTs = unDayBegTs + int64(i*86400)
		stSimpleDayScheduleInfo.VecScheduleItem = make([]SimpleScheduleItem, 48, 48)
		stSimpleDayScheduleInfo.VecScheduleItem[0].StartTs = stSimpleDayScheduleInfo.DayBegTs
		stSimpleDayScheduleInfo.VecScheduleItem[0].EndTs = stSimpleDayScheduleInfo.DayBegTs + 1800
		stSimpleDayScheduleInfo.VecScheduleItem[0].BAvailable = true
		// 生成时间戳
		for j := 1; j < 48; j++ {
			stSimpleDayScheduleInfo.VecScheduleItem[j].StartTs = stSimpleDayScheduleInfo.VecScheduleItem[j-1].EndTs
			stSimpleDayScheduleInfo.VecScheduleItem[j].EndTs = stSimpleDayScheduleInfo.VecScheduleItem[j].StartTs + 1800
			stSimpleDayScheduleInfo.VecScheduleItem[j].BAvailable = true
		}
		vecSimpleDayScheduleInfo = append(vecSimpleDayScheduleInfo, stSimpleDayScheduleInfo)
	}
	for i := 0; i < 14; i++ {
		dayBegTs := vecSimpleDayScheduleInfo[i].DayBegTs

		//如果当天发现学员有预约记录，需要设置对应的时间段不可用
		if vecApp, ok := mapDay2VecTraineeAppointment[dayBegTs]; ok {
			for _, v := range vecApp {
				SetScheduleUnAvailable(&vecSimpleDayScheduleInfo[i], v.StartTime, v.EndTime)
			}
		}

		//如果当天发现教练有时间被占用，需要设置对应的时间段不可用
		if vecApp, ok := mapDay2VecCoachAppointment[dayBegTs]; ok {
			for _, v := range vecApp {
				if v.UserID > 0 {
					SetScheduleUnAvailable(&vecSimpleDayScheduleInfo[i], v.StartTime, v.EndTime)
				}
			}
		}
	}

	//过滤不可用时间段数据，合并可用时间段
	for _, v := range vecSimpleDayScheduleInfo {
		rsp.VecSimpleDayScheduleInfo = append(rsp.VecSimpleDayScheduleInfo, mergeScheduleInfo(coachId, req.TraineeUid, v))
	}

	vecTraineePackageListModel, err := dao.ImpCoursePackage.GetCoursePackageListByUid(req.TraineeUid)
	if len(vecTraineePackageListModel) > 0 {
		// 按照 RemainCnt 从小到大排序
		sort.Slice(vecTraineePackageListModel, func(i, j int) bool {
			return vecTraineePackageListModel[i].RemainCnt < vecTraineePackageListModel[j].RemainCnt
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

		stPackage := vecTraineePackageListModel[0]
		rsp.PackageID = stPackage.PackageID
		rsp.GymId = stPackage.GymId
		rsp.CourseName = mapCourse[stPackage.GymId].Name
		rsp.GymName = mapGym[stPackage.GymId].LocName

		//打包学员user信息
		stTraineeUserInfoModel, err := dao.ImpUser.GetUser(req.TraineeUid)
		if err != nil || stTraineeUserInfoModel == nil {
			rsp.Code = -800
			rsp.ErrorMsg = err.Error()
			Printf("GetUser err, strOpenId:%s err:%+v\n", strOpenId, err)
			return
		}
		rsp.TraineeUserInfo = transUserModel2TraineeUser(stPackage.LastLessonTs, *stTraineeUserInfoModel)
	}
}

func SetScheduleUnAvailable(stSimpleDayScheduleInfo *SimpleDayScheduleInfo, begTs int64, endTs int64) {
	bFindBeg := false
	for i := 0; i < len(stSimpleDayScheduleInfo.VecScheduleItem); i++ {
		if stSimpleDayScheduleInfo.VecScheduleItem[i].StartTs == begTs {
			stSimpleDayScheduleInfo.VecScheduleItem[i].BAvailable = false
			bFindBeg = true
		}
		if bFindBeg {
			stSimpleDayScheduleInfo.VecScheduleItem[i].BAvailable = false
			if stSimpleDayScheduleInfo.VecScheduleItem[i].EndTs == endTs {
				break
			}
		}
	}
	Printf("set UnAvailable succ, begTs:%d endTs:%d stSimpleDayScheduleInfo:%+v", begTs, endTs, stSimpleDayScheduleInfo)
}

// 睡觉时间段：最早可以8点，最晚可以到23：30
func mergeScheduleInfo(coachId int, traineeUid int64, stSchedule SimpleDayScheduleInfo) SimpleDayScheduleInfo {
	unNowTs := time.Now().Unix()
	var rsp SimpleDayScheduleInfo
	rsp.DayBegTs = stSchedule.DayBegTs
	if len(stSchedule.VecScheduleItem) == 0 {
		return rsp
	}

	for i := 0; i < len(stSchedule.VecScheduleItem); {
		scheduleItem := stSchedule.VecScheduleItem[i]
		//早8点前的时间全部过滤
		if scheduleItem.EndTs <= (stSchedule.DayBegTs + 3600*8) {
			i++
			continue
		}

		//晚11点半后的时间全部过滤
		if scheduleItem.StartTs >= (stSchedule.DayBegTs + 3600*23 + 1800) {
			i++
			continue
		}

		if !scheduleItem.BAvailable {
			i++
			continue
		}

		Printf("get AvailablePair, i:%d scheduleItem:%+v coachId:%d traineeUid:%d", i, scheduleItem, coachId, traineeUid)
		var beg int64
		var end int64
		beg = scheduleItem.StartTs
		for i < len(stSchedule.VecScheduleItem) {
			i++
			if i >= len(stSchedule.VecScheduleItem) {
				break
			}
			if !stSchedule.VecScheduleItem[i].BAvailable {
				end = stSchedule.VecScheduleItem[i].StartTs
				break
			}
		}
		//如果一直没找到，说明后面的时间都是可用的，直接到最后一条
		if end == 0 {
			end = stSchedule.VecScheduleItem[len(stSchedule.VecScheduleItem)-1].EndTs
		}

		//不足半小时的直接过滤
		if end - beg <= 1800{
			Printf("间隔不足半小时直接过滤, beg:%d end:%d coachId:%d traineeUid:%d", beg, end, coachId, traineeUid)
			continue
		}
		if unNowTs >= end{
			Printf("当前时间之前的时间不能再选, beg:%d end:%d unNowTs:%d coachId:%d traineeUid:%d", beg, end, unNowTs, coachId, traineeUid)
			continue
		}
		if unNowTs >= beg && unNowTs <= end{
			if end - unNowTs <= 3600{
				Printf("当前距离end小于1小时直接过滤, beg:%d end:%d unNowTs:%d coachId:%d traineeUid:%d", beg, end, unNowTs, coachId, traineeUid)
				continue
			}
		}

		rsp.VecScheduleItem = append(rsp.VecScheduleItem, SimpleScheduleItem{
			StartTs:    beg,
			EndTs:      end,
			StartTsStr: time.Unix(beg, 0).Format("2006-01-02 15:04:05"),
			EndTsStr:   time.Unix(end, 0).Format("2006-01-02 15:04:05"),
		})
		Printf("merge pairSucc, beg:%d end:%d coachId:%d traineeUid:%d", beg, end, coachId, traineeUid)
		i++
	}

	return rsp
}
