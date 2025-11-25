package main

import (
	"github.com/xionghengheng/ff_plib/db/pass_card_dao"
	"github.com/xionghengheng/ff_plib/db/pass_card_model"
	"time"
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
