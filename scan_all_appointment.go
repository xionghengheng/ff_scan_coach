package main

import (
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"os"
	"time"
)

// 扫描所有预约，如果教练过去2天没设置预约，直接通知教练
func ScanAllAppointments() {
	Printf("scan start, beg_time:%s", time.Now().Format("2006-01-02 15:04:05"))
	err := doAppointmentsScan()
	if err != nil {
		Printf("doScan err, err:%+v", err)
		return
	}
	Printf("scan end, end_time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

func doAppointmentsScan() error {
	unDayBegTs := GetYesterdayBegTs()

	tmpMapCoach, err := comm.GetAllCoach()
	if err != nil {
		Printf("GetAllCoach err:%+v", err)
		return err
	}

	mapCoach := make(map[int]model.CoachModel)
	for k, v := range tmpMapCoach {
		mapCoach[k] = v
	}

	for _, v := range mapCoach {
		vecCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentScheduleFromBegTs(v.GymID, v.CoachID, unDayBegTs)
		if err != nil {
			Printf("GetAppointmentScheduleFromBegTs err:%+v", err)
			return err
		}
		Printf("GetAppointmentScheduleFromBegTs succ, CoachId:%d unDayBegTs:%d vecCoachAppointmentModel:%+v\n", v.CoachID, unDayBegTs, vecCoachAppointmentModel)
		if len(vecCoachAppointmentModel) == 0 {
			sendRemindMsgSetLessonAvailiable2Coach(v.CoachID)
		}
	}

	return nil
}

func GetYesterdayBegTs() int64 {
	t := time.Now()
	// 计算昨天的时间
	yesterday := t.AddDate(0, 0, -1)
	// 获取昨天零点的时间
	yesterdayMidnight := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	return yesterdayMidnight.Unix()
}

func sendRemindMsgSetLessonAvailiable2Coach(coachId int) {
	stCoachUserModel, err := dao.ImpUser.GetUserByCoachId(coachId)
	stCoachModel, err := dao.ImpCoach.GetCoachById(coachId)
	stGymModel, err := dao.ImpGym.GetGymInfoByGymId(stCoachModel.GymID)
	stWxSendMsg2UserReq := comm.WxSendMsg2UserReq{
		ToUser:           stCoachUserModel.WechatID,
		TemplateID:       "r5pEmo4PPkXIZVhBhY9mv6yTvKFENg62x0phoAMYKM4",
		Page:             "pages/home/index/index",
		MiniprogramState: os.Getenv("MiniprogramState"),
		Lang:             "zh_CN",
		Data: map[string]comm.MsgDataField{
			"thing3": {Value: stGymModel.LocName},  //上课地点名称
			"thing4": {Value: "您未设置近期的可约时间，请及时设置"}, //课程名称
		},
	}
	err = comm.SendMsg2User(stCoachUserModel.UserID, stWxSendMsg2UserReq)
	if err != nil {
		Printf("sendMsg2Coach err, err:%+v coachId:%d uid:%d WechatID:%s", err, coachId, stCoachUserModel.UserID, stCoachUserModel.WechatID)
	} else {
		Printf("sendMsg2Coach succ, coachId:%d", coachId)
	}
}
