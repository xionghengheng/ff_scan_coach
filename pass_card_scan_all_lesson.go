package main

import (
	"fmt"
	"os"
	"time"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/pass_card_dao"
	"github.com/xionghengheng/ff_plib/db/pass_card_model"
)

// 扫描所有单次课程，处理旷课以及旷课退回的情况
func ScanAllPassCardLesson() {
	Printf("scan start, beg_time:%s", time.Now().Format("2006-01-02 15:04:05"))
	err := doPassCardLessonScan()
	if err != nil {
		Printf("doPassCardLessonScan err, err:%+v", err)
		return
	}
	Printf("scan end, end_time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

func doPassCardLessonScan() error {

	//每5分钟处理一次
	handlePassCardLessonMissed()

	//锻炼时间前2小时，发送消息通知用户
	handleSendPassCardMsgBeforeLessonStart()

	return nil
}

// 处理已经超过课程结束时间的已预约课程
func handlePassCardLessonMissed() {
	nowTs := time.Now().Unix()

	vecNotFinishLesson, err := pass_card_dao.ImpPassCardLesson.GetLessonListNotFinish(time.Now().Unix(), 100)
	if err != nil {
		Printf("GetSingleLessonListNotFinish err, err:%+v", err)
		return
	}
	Printf("GetSingleLessonListNotFinish succ, vecNotFinishLesson.len:%d", len(vecNotFinishLesson))

	for _, v := range vecNotFinishLesson {
		//当前时间已经超过课程结束时间
		if nowTs <= v.ScheduleEndTs {
			continue
		}

		mapUpdates := make(map[string]interface{})
		mapUpdates["status"] = pass_card_model.En_LessonStatus_Completed
		mapUpdates["update_ts"] = nowTs
		err = pass_card_dao.ImpPassCardLesson.UpdateLesson(v.Uid, v.LessonID, mapUpdates)
		if err != nil {
			Printf("UpdateSingleLesson2StatusMissed err, err:%+v uid:%d LessonID:%s", err, v.Uid, v.LessonID)
			continue
		}
		Printf("UpdateSingleLesson2StatusMissed succ, uid:%d LessonID:%s", v.Uid, v.LessonID)
	}
	return
}

// 处理锻炼时间前2小时发送提醒消息
func handleSendPassCardMsgBeforeLessonStart() {
	nowTs := time.Now().Unix()
	// 查询未完成的课程，时间范围设置为未来3小时内（确保能覆盖2小时前的课程）
	vecNotFinishLesson, err := pass_card_dao.ImpPassCardLesson.GetLessonListNotFinish(nowTs+10800, 1000)
	if err != nil {
		Printf("handleSendPassCardMsgBeforeLessonStart GetLessonListNotFinish err, err:%+v", err)
		return
	}

	for _, v := range vecNotFinishLesson {
		if v.ScheduleBegTs == 0 {
			continue
		}

		// 锻炼时间前2小时（7200秒），发送消息通知用户
		// 使用时间窗口（2小时前到1小时50分前）来避免重复发送
		// 这样即使扫描多次，也只会在这个窗口内发送一次
		if nowTs >= v.ScheduleBegTs-7200 && nowTs < v.ScheduleBegTs-6600 {
			sendPassCardLessonRemindMsg(v.Uid, v)
		}
	}
	return
}

// 发送通卡课程锻炼时间前2小时提醒消息
func sendPassCardLessonRemindMsg(uid int64, stLessonModel pass_card_model.LessonModel) {
	stUserModel, err := dao.ImpUser.GetUser(uid)
	if err != nil {
		Printf("sendPassCardLessonRemindMsg GetUser err, err:%+v uid:%d LessonID:%s\n", err, uid, stLessonModel.LessonID)
		return
	}

	stGymModel, err := pass_card_dao.ImpGym.GetGymInfoByGymId(stLessonModel.GymId)
	if err != nil {
		Printf("sendPassCardLessonRemindMsg GetGymInfoByGymId err, err:%+v gymId:%d uid:%d LessonID:%s\n", err, stLessonModel.GymId, uid, stLessonModel.LessonID)
		return
	}

	t := time.Unix(stLessonModel.ScheduleBegTs, 0)
	tNow := time.Now()
	// 计算剩余分钟数
	remainingMinutes := (stLessonModel.ScheduleBegTs - tNow.Unix()) / 60
	remainingMinutesStr := ""
	if remainingMinutes > 0 {
		remainingMinutesStr = fmt.Sprintf("%d分钟", remainingMinutes)
	}

	stWxSendMsg2UserReq := comm.WxSendMsg2UserReq{
		ToUser:           stUserModel.WechatID,
		TemplateID:       "jg1jkbScPaO-Ng6o5f3zpMpjn_GbRqKcNqRVHgaTYvY",
		Page:             "pages/home/index/index",
		MiniprogramState: os.Getenv("MiniprogramState"),
		Lang:             "zh_CN",
		Data: map[string]comm.MsgDataField{
			"thing2":  {Value: stGymModel.LocName},            //门店
			"time3":   {Value: t.Format("2006年01月02日 15:04")}, //预约上课时间
			"number4": {Value: remainingMinutesStr},           //剩余分钟
			"thing5":  {Value: "您的锻炼预约即将开始，请准时前往~"},           //温馨提示
		},
	}
	err = comm.SendMsg2User(uid, stWxSendMsg2UserReq)
	if err != nil {
		Printf("sendPassCardLessonRemindMsg sendMsg2User err, err:%+v uid:%d LessonID:%s\n", err, uid, stLessonModel.LessonID)
	} else {
		Printf("sendPassCardLessonRemindMsg sendMsg2User succ, uid:%d LessonID:%s\n", uid, stLessonModel.LessonID)
	}
}
