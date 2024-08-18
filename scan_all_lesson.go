package main

import (
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"fmt"
	"time"
)

// 扫描所有单次课程，处理旷课以及旷课退回的情况
func ScanAllCoursePackageSingleLesson() {
	Printf("scan start, beg_time:%s", time.Now().Format("2006-01-02 15:04:05"))
	err := doSingleLessonScan()
	if err != nil {
		Printf("doScan err, err:%+v", err)
		return
	}
	Printf("scan end, end_time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

// 如果当前时间已经超过了课程终止时间，还没有核销，那么则认为用户旷课，或者是教练忘记核销了
func doSingleLessonScan() error {
	now := time.Now()

	//如果当前时间超过晚上11点30分，则触发归还次数
	// 创建一个表示当天晚上11点50分的时间对象
	elevenFiftyPM := time.Date(now.Year(), now.Month(), now.Day(), 23, 30, 0, 0, now.Location())
	if now.After(elevenFiftyPM) {
		Printf("当前时间超过晚上11点30分, now:%d", now.Unix())
		vecMissedLesson, err := dao.ImpCoursePackageSingleLesson.GetSingleLessonListMissed(100)
		if err != nil {
			Printf("GetSingleLessonListMissed err, err:%+v", err)
			return err
		}

		for _, v := range vecMissedLesson {
			mapUpdates := make(map[string]interface{})
			mapUpdates["write_off_missed_return_cnt"] = true
			err = dao.ImpCoursePackageSingleLesson.UpdateSingleLesson(v.Uid, v.LessonID, mapUpdates)
			if err != nil {
				Printf("UpdateSingleLesson2StatusMissed err, err:%+v uid:%d PackageID:%s LessonID:%s", err, v.Uid, v.PackageID, v.LessonID)
				continue
			}
			Printf("UpdateSingleLesson2StatusMissed succ, uid:%d PackageID:%s LessonID:%s", v.Uid, v.PackageID, v.LessonID)

			err = dao.ImpCoursePackage.AddRemainCourseCnt(v.PackageID, 1)
			if err != nil {
				Printf("ReturnCourseCnt err, err:%+v uid:%d PackageID:%s LessonID:%s", err, v.Uid, v.PackageID, v.LessonID)
				continue
			}
			Printf("ReturnCourseCnt succ, uid:%d PackageID:%s LessonID:%s", v.Uid, v.PackageID, v.LessonID)
		}
		return nil
	}

	//每5分钟处理一次
	handleLessonMissed()

	handleSendMsg()

	return nil
}

// 处理旷课的情况
func handleLessonMissed() {
	vecNotFinishLesson, err := dao.ImpCoursePackageSingleLesson.GetSingleLessonListNotFinish(time.Now().Unix(), 100)
	if err != nil {
		Printf("GetSingleLessonListNotFinish err, err:%+v", err)
		return
	}

	//将用户课包里的单节课状态变成已旷课
	for _, v := range vecNotFinishLesson {
		mapUpdates := make(map[string]interface{})
		mapUpdates["status"] = model.En_LessonStatusMissed
		err = dao.ImpCoursePackageSingleLesson.UpdateSingleLesson(v.Uid, v.LessonID, mapUpdates)
		if err != nil {
			Printf("UpdateSingleLesson2StatusMissed err, err:%+v uid:%d PackageID:%s LessonID:%s", err, v.Uid, v.PackageID, v.LessonID)
			continue
		}
		Printf("UpdateSingleLesson2StatusMissed succ, uid:%d PackageID:%s LessonID:%s", v.Uid, v.PackageID, v.LessonID)
	}
	return
}

func handleSendMsg() {
	unNowTs := time.Now().Unix()
	vecNotSendMsgLesson, err := dao.ImpCoursePackageSingleLesson.GetTodaySingleLessonListNotSendMsgGoLesson(unNowTs+4000, 1000)
	if err != nil {
		Printf("GetSingleLessonListNotFinish err, err:%+v", err)
		return
	}

	for _, v := range vecNotSendMsgLesson {
		if v.ScheduleBegTs == 0 {
			continue
		}

		//开课前一小时，发送消息通知用户上课
		if unNowTs >= v.ScheduleBegTs-3600 {
			mapUpdates := make(map[string]interface{})
			mapUpdates["send_msg_go_lesson"] = true
			err = dao.ImpCoursePackageSingleLesson.UpdateSingleLesson(v.Uid, v.LessonID, mapUpdates)
			if err != nil {
				Printf("UpdateSingleLesson2StatusSendMsg err, err:%+v uid:%d PackageID:%s LessonID:%s", err, v.Uid, v.PackageID, v.LessonID)
				continue
			}

			//模板配置链接：https://mp.weixin.qq.com/wxamp/newtmpl/tmpldetail?type=2&pri_tmpl_id=kENL0EQdSD5gvtUAPh58n923AwBEio7tec6e1bC2sb0&flag=undefined&token=1034864027&lang=zh_CN
			stGymInfoModel, err := dao.ImpGym.GetGymInfoByGymId(v.GymId)
			stCourseModel, err := dao.ImpCourse.GetCourseById(v.CourseID)
			stUserModel, err := dao.ImpUser.GetUser(v.Uid)
			t := time.Unix(v.ScheduleBegTs, 0)
			stWxSendMsg2UserReq := comm.WxSendMsg2UserReq{
				ToUser:           stUserModel.WechatID,
				TemplateID:       "kENL0EQdSD5gvtUAPh58n923AwBEio7tec6e1bC2sb0",
				Page:             fmt.Sprintf("pages/course/order-info-detail/index?lesson_id=%s", v.LessonID),
				MiniprogramState: "trial",
				Lang:             "zh_CN",
				Data: map[string]comm.MsgDataField{
					"thing1": {Value: stCourseModel.Name},            //课程名称
					"date2":  {Value: t.Format("2006年01月02日 15:04")}, //上课时间
					"thing3": {Value: stGymInfoModel.LocName},        //上课地点
					"thing4": {Value: "课程还有1小时开始了，出发前往场地热身吧"},        //温馨提示
				},
			}
			err = comm.SendMsg2User(v.Uid, stWxSendMsg2UserReq)
			if err != nil {
				Printf("sendMsg2User err, err:%+v uid:%d PackageID:%s LessonID:%s", err, v.Uid, v.PackageID, v.LessonID)
			} else {
				Printf("sendMsg2User succ, uid:%d PackageID:%s LessonID:%s", v.Uid, v.PackageID, v.LessonID)
			}
			Printf("UpdateSingleLesson2StatusSendMsg succ, uid:%d PackageID:%s LessonID:%s", v.Uid, v.PackageID, v.LessonID)
		}
	}
	return
}
