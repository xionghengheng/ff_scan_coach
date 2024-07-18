package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"encoding/json"
	"fmt"
	"net/http"
)

type ChangeRoleReq struct {
	Role        string `json:"role"`          // coach or user
	BindCoachId int    `json:"bind_coach_id"` // 如果转变为coach身份，绑定的coachid，传0，则默认绑定第一个教练
}

type ChangeRoleRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

func getChangeRoleReq(r *http.Request) (ChangeRoleReq, error) {
	req := ChangeRoleReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// ChangeRoleHandler 拉取学员主页
func ChangeRoleHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getChangeRoleReq(r)
	rsp := &ChangeRoleRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetTraineePageHandler start, openid:%s\n", strOpenId)

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
