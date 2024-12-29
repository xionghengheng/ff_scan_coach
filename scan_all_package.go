package main

import (
	"fmt"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"os"
	"time"
)

func ScanAllPackage() {
	Printf("scan start, beg_time:%s", time.Now().Format("2006-01-02 15:04:05"))
	err := doPackageScan()
	if err != nil {
		Printf("doPackageScan err, err:%+v", err)
		return
	}
	Printf("scan end, end_time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

func doPackageScan() error {

	if !comm.IsProd() {
		//体验课包过期提醒
		handleSendMsgWhenTrailPackageExpire()
	}

	return nil
}

func handleSendMsgWhenTrailPackageExpire() {
	unNowTs := time.Now().Unix()

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

	for _, v := range vecAllTrailPackageModel {
		if v.RemainCnt == 0 || v.SendMsgTrailExpire || v.Ts > unNowTs || unNowTs-v.Ts < 7*86400 {
			continue
		}

		// 已经过期很久的存量课包，也不通知了



		mapUpdates := make(map[string]interface{})
		mapUpdates["send_msg_trail_expire"] = true
		err := dao.ImpCoursePackage.UpdateCoursePackage(v.Uid, v.PackageID, mapUpdates)
		if err != nil {
			Printf("[send_msg_trail_expire]UpdateCoursePackage err, uid:%d PackageID:%s err:%+v\n", v.Uid, v.PackageID, err)
			return
		}

		t := time.Unix(v.Ts+14*86400, 0)
		stCourseModel, err := dao.ImpCourse.GetCourseById(v.CourseId)
		stUserModel, err := dao.ImpUser.GetUser(v.Uid)
		stWxSendMsg2UserReq := comm.WxSendMsg2UserReq{
			ToUser:           stUserModel.WechatID,
			TemplateID:       "aeCItcVr9A9iVnoujFbA0jGyopFKAujrCCPVhtvM3FM",
			Page:             "pages/home/index/index",
			MiniprogramState: os.Getenv("MiniprogramState"),
			Lang:             "zh_CN",
			Data: map[string]comm.MsgDataField{
				"thing3":  {Value: stCourseModel.Name},             //课程名称
				"number1": {Value: fmt.Sprintf("%d", v.RemainCnt)}, //剩余课时
				"time4":   {Value: t.Format("2006年01月02日 15:04")},  //到期时间
				"thing2":  {Value: "体验课有效期还剩余7天，请预约上课吧！"},          //备注
			},
		}
		err = comm.SendMsg2User(v.Uid, stWxSendMsg2UserReq)
		if err != nil {
			Printf("[PackageExpire]sendWxMsg2User err, err:%+v uid:%d PackageID:%s", err, v.Uid, v.PackageID)
		} else {
			Printf("[PackageExpire]sendWxMsg2User succ, uid:%d PackageID:%s", v.Uid, v.PackageID)
		}
	}
}
