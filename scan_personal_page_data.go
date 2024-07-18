package main

import (
	"github.com/xionghengheng/fun_fit_plib/comm"
	"github.com/xionghengheng/fun_fit_plib/db/dao"
	"encoding/json"
	"fmt"
	"time"
)

// ScanCoachPersonalPageData 扫描生成教练端主页的计数信息
func ScanCoachPersonalPageData() {
	Printf("scan start, beg_time:%s", time.Now().Format("2006-01-02 15:04:05"))
	err := doScan()
	if err != nil {
		Printf("doScan err, err:%+v", err)
		return
	}
	Printf("scan end, end_time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

// ScanCoachPersonalPageData 扫描生成教练端主页的计数信息
func doScan() error {

	dao.ImpPaymentOrder.AddOrder()

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

	if req.Role == "coach" {
		mapUserUpdates := map[string]interface{}{}
		if req.BindCoachId == 0 {
			req.BindCoachId = 1
		}
		mapUserUpdates["is_coach"] = true
		mapUserUpdates["coach_id"] = req.BindCoachId
		err = dao.ImpUser.UpdateUserInfo(uid, mapUserUpdates)
		if err != nil {
			rsp.Code = -10007
			rsp.ErrorMsg = err.Error()
			Printf("UpdateUserInfo err, err:%+v uid:%d mapUpdates:%+v\n", err, uid, mapUserUpdates)
			return
		}
		Printf("UpdateUserInfo succ, uid:%d coachId:%d mapUpdates:%+v\n", uid, req.BindCoachId, mapUserUpdates)
	} else {
		mapUserUpdates := map[string]interface{}{}
		mapUserUpdates["is_coach"] = false
		mapUserUpdates["coach_id"] = 0
		err = dao.ImpUser.UpdateUserInfo(uid, mapUserUpdates)
		if err != nil {
			rsp.Code = -10007
			rsp.ErrorMsg = err.Error()
			Printf("UpdateUserInfo err, err:%+v uid:%d mapUpdates:%+v\n", err, uid, mapUserUpdates)
			return
		}
		Printf("UpdateUserInfo succ, uid:%d coachId:%d mapUpdates:%+v\n", uid, req.BindCoachId, mapUserUpdates)
	}
}
