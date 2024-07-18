package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"net/http"
	"strconv"
	"time"
)

type GetHomePageReq struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Passback  string  `json:"passback"` //课程安排翻页回传，首次传空
}

type GetHomePageRsp struct {
	Code                int                `json:"code"`
	ErrorMsg            string             `json:"errorMsg,omitempty"`
	NowTime             int64              `json:"now_time,omitempty"`
	CoachName           string             `json:"coach_name,omitempty"`
	Banner              BannerInfo         `json:"banner,omitempty"`                 //顶部banner
	VecTrialTraineeUser []TrialTraineeUser `json:"vec_trial_trainee_user,omitempty"` //体验学员列表
	VecScheduleInfo     []DayScheduleInfo  `json:"vec_schedule_info,omitempty"`      //近7天课程列表
	Passback            string             `json:"passback"`                         //翻页带上来
	BHasMore            bool               `json:"b_has_more"`                       //是否还有后续页
}

type BannerItem struct {
	PicUrl  string `json:"pic_url,omitempty"`  //图片url
	JumpUrl string `json:"jump_url,omitempty"` //跳转链接
}

type BannerInfo struct {
	VecBannerItem []BannerItem `json:"banner_item_list,omitempty"` //顶部banner轮播列表
}

// TrialUser 体验学员用户item
type TrialTraineeUser struct {
	Uid           int64  `json:"uid"`            //学员的用户id
	Name          string `json:"name"`           //名字
	HeadPic       string `json:"head_pic"`       //头像
	ConvertStatus int    `json:"convert_status"` //学员的转化状态（参考 Enum_UserConvertStatus）
}

type DayScheduleInfo struct {
	DayBegTs        int64          `json:"day_beg_ts,omitempty"` //当天零点开始时间
	VecScheduleItem []ScheduleItem `json:"schedule_list,omitempty"`
}

type ScheduleItem struct {
	LessonID           string `json:"lesson_id"`             //单节课的唯一标识符（用户id_场地id_课程id_教练id_发起预约的时间戳）
	PackageID          string `json:"package_id"`            //关联的课包的唯一标识符
	AppointmentID      int    `json:"appointment_id"`        //预约ID
	StartTs            int64  `json:"start_ts"`              //起始时间
	EndTs              int64  `json:"end_ts"`                //结束时间
	Status             int    `json:"status"`                //课程状态
	GymName            string `json:"gym_name"`              //场地名称
	CourseName         string `json:"course_name"`           //课程名称
	CourseChargeType   int    `json:"course_charge_type"`    //课程付费类型（体验课=1，正式付费课=2）
	TraineeUid         int64  `json:"trainee_uid"`           //学员uid
	TraineeUserName    string `json:"trainee_user_name"`     //学员名称
	TraineeUserHeadPic string `json:"trainee_user_head_pic"` //学员头像
	TraineeUserPhone   string `json:"trainee_user_phone"`    //学员手机号
	TraineePlan        int    `json:"trainee_plan"`          //用户信息里的健身目标（减脂减重、增肌增重、塑型体态）
	PackageProgress    string `json:"package_progress"`      //课程进度 比如第23/25节
	ScheduledByCoach   bool   `json:"scheduled_by_coach"`    //是否教练排课
	ScheduleType       int    `json:"schedule_type"`         //格子类型（排课页不同颜色格子展示，参考En_ScheduleType）
}

const (
	En_ScheduleType_Available       int = iota + 1 // 空闲，已被教练设置为可预约状态，不展示学员名字；（绿色块）
	En_ScheduleType_BookedByTrainee                // 被占据，学员主动预约，展示学员名字；（绿色块）
	En_ScheduleType_BookedByCoach                  // 被占据，教练自己排的课，展示学员名字；（蓝色块）
)

const (
	En_LessonStatus_Scheduled int = iota + 1 // 待上课
	En_LessonStatusCompleted                 // 已完成
	En_LessonStatusCanceled                  // 已取消
	En_LessonStatusMissed                    // 已旷课
)

const (
	Enum_Course_ChargeType_Paid      = iota + 1 // 1 正式课-付费
	Enum_Course_ChargeType_FreeTrial            // 2 体验课-免费
)

// 定义用户状态的枚举
const (
	Enum_UserConvertStatus_Invalid int = iota // 无效
	Enum_UserConvertStatus_New                // 新用户
	Enum_UserConvertStatus_Not                // 未转化
)

type AppointmentItem struct {
	Uid           int64
	AppointmentID int
}

func getGetHomePageReq(r *http.Request) (GetHomePageReq, error) {
	req := GetHomePageReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetHomePageHandler 获取训练计划
func GetHomePageHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetHomePageReq(r)
	rsp := &GetHomePageRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("CoachGetHomePageHandler start, openid:%s\n", strOpenId)

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

	vecBannerModel, err := dao.ImpCoachClientHomePageBanner.GetHomePageBannerList()
	if err != nil || vecBannerModel == nil {
		rsp.Code = -900
		rsp.ErrorMsg = err.Error()
		Printf("GetHomePageBannerList err, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}
	for _, v := range vecBannerModel {
		var stBannerItem BannerItem
		stBannerItem.PicUrl = v.PicUrl
		rsp.Banner.VecBannerItem = append(rsp.Banner.VecBannerItem, stBannerItem)
	}

	//组装体验用户头像列表
	var vecTrialTraineeUser []TrialTraineeUser
	vecTrailCoursePackageModel, err := dao.ImpCoursePackage.GetTrailCoursePackageListByCoachId(coachId, 50)
	if err != nil {
		rsp.Code = -866
		rsp.ErrorMsg = err.Error()
		Printf("GetTrailCoursePackageListByCoachId err, strOpenId:%s coachId:%d err:%+v\n", strOpenId, coachId, err)
		return
	}
	if len(vecTrailCoursePackageModel) > 0 {
		var vecNew []TrialTraineeUser
		var vecNotConvert []TrialTraineeUser
		for _, v := range vecTrailCoursePackageModel {
			vecPayCoursePackageModel, err := dao.ImpCoursePackage.GetPayCoursePackageListByCoachIdAndUid(coachId, v.Uid)
			if err != nil {
				Printf("GetPayCoursePackageListByCoachIdAndUid err, strOpenId:%s coachId:%d uid:%d err:%+v\n", strOpenId, coachId, uid, err)
				continue
			}
			//已转化，直接过滤
			if len(vecPayCoursePackageModel) > 0 {
				continue
			}

			stTraineeReddotModel, err := dao.ImpCoachClientTraineeReddot.GetTraineeReddot(coachId, v.Uid)
			if err != nil && err != gorm.ErrRecordNotFound {
				Printf("GetTraineeReddot err, strOpenId:%s coachId:%d uid:%d err:%+v\n", strOpenId, coachId, uid, err)
				continue
			}
			if err == gorm.ErrRecordNotFound || stTraineeReddotModel == nil {
				var stTrialTraineeUser TrialTraineeUser
				stTrialTraineeUser.Uid = v.Uid
				stTrialTraineeUser.ConvertStatus = Enum_UserConvertStatus_New
				vecNew = append(vecNew, stTrialTraineeUser)
			} else {
				var stTrialTraineeUser TrialTraineeUser
				stTrialTraineeUser.Uid = v.Uid
				stTrialTraineeUser.ConvertStatus = Enum_UserConvertStatus_Not
				vecNotConvert = append(vecNotConvert, stTrialTraineeUser)
			}
		}
		Printf("get TrialTrainee succ, coachId:%d vecNew:%+v vecNotConvert:%+v\n", coachId, vecNew, vecNotConvert)
		vecTrialTraineeUser = append(vecNew, vecNotConvert...)
	}

	stCoachModel, err := dao.ImpCoach.GetCoachById(coachId)
	if err != nil || stUserInfoModel == nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		Printf("GetCoachById err, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}

	var unDayBegTs int64
	if len(req.Passback) > 0 {
		unDayBegTs, _ = strconv.ParseInt(req.Passback, 10, 64)
	} else {
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
	var vecAppointmentItem []AppointmentItem
	mapDayBegTs2AppointmentModel := make(map[int64][]model.CoachAppointmentModel)
	for _, v := range vecCoachAppointmentModel {
		//过滤掉可预约的课程
		if v.Status == model.Enum_Appointment_Status_Available {
			continue
		}
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

	mapCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		return
	}

	mapUser := make(map[int64]model.UserInfoModel)
	for _, v := range vecTrialTraineeUser {
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

			//需要的是单次课里更新的状态
			stScheduleRspItem.PackageID = mapAppointmentID2SingleLessonInfo[v.AppointmentID].PackageID
			stScheduleRspItem.LessonID = mapAppointmentID2SingleLessonInfo[v.AppointmentID].LessonID
			stScheduleRspItem.Status = mapAppointmentID2SingleLessonInfo[v.AppointmentID].Status
			stScheduleRspItem.StartTs = v.StartTime
			stScheduleRspItem.EndTs = v.EndTime

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
			rsp.VecScheduleInfo[i].VecScheduleItem = append(rsp.VecScheduleInfo[i].VecScheduleItem, stScheduleRspItem)
		}
	}

	for _, v := range vecTrialTraineeUser {
		if userInfo, ok := mapUser[v.Uid]; ok {
			v.Name = userInfo.Nick
			v.HeadPic = userInfo.HeadPic
			rsp.VecTrialTraineeUser = append(rsp.VecTrialTraineeUser, v)
		}
	}

	rsp.NowTime = time.Now().Unix()
	rsp.CoachName = mapCoach[coachId].CoachName

	if len(req.Passback) == 0 {
		unNextWeekTs := unDayBegTs + 604800
		rsp.Passback = fmt.Sprintf("%d", unNextWeekTs)
		rsp.BHasMore = true
	} else {
		rsp.BHasMore = false
	}
}
