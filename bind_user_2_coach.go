package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
)

type BindUser2CoachReq struct {
	CoachPhone    string `json:"coach_phone"`     //教练手机号
	LoginUserName string `json:"login_user_name"` //管理平台用户名
	LoginPasswd   string `json:"login_passwd"`    //管理平台密码
}

type BindUser2CoachRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg"`
}

func getBindUser2CoachReq(r *http.Request) (BindUser2CoachReq, error) {
	req := BindUser2CoachReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func bindUser2CoachHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getBindUser2CoachReq(r)
	rsp := &BindUser2CoachRsp{}

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
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	if len(req.CoachPhone) == 0 {
		rsp.Code = -996
		rsp.ErrorMsg = "缺少教练手机号"
		return
	}

	if len(req.LoginUserName) == 0 || len(req.LoginPasswd) == 0 {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少管理平台用户名或密码"
		return
	}

	// 验证管理平台用户名和密码（这里可以根据实际需求修改验证逻辑）
	// 可以通过环境变量或配置文件来设置管理平台的用户名和密码
	adminUserName := os.Getenv("ADMIN_USER_NAME")
	adminPasswd := os.Getenv("ADMIN_PASSWD")
	if len(adminUserName) == 0 {
		adminUserName = "admin" // 默认值，实际应该从环境变量获取
	}
	if len(adminPasswd) == 0 {
		adminPasswd = "admin123" // 默认值，实际应该从环境变量获取
	}

	if req.LoginUserName != adminUserName || req.LoginPasswd != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "管理平台用户名或密码错误"
		Printf("bindUser2CoachHandler auth failed, LoginUserName:%s\n", req.LoginUserName)
		return
	}

	// 根据手机号获取用户信息
	stCoachUserInfoModel, err := dao.ImpUser.GetUserByPhone(req.CoachPhone)
	if err != nil {
		rsp.Code = -993
		rsp.ErrorMsg = "未找到对应用户"
		Printf("bindUser2CoachHandler GetUserByPhone err, err:%+v CoachPhone:%s\n", err, req.CoachPhone)
		return
	}

	// 根据手机号查找教练
	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -921
		rsp.ErrorMsg = err.Error()
		Printf("bindUser2CoachHandler GetAllCoach err, err:%+v\n", err)
		return
	}

	var coachId int
	for _, coach := range mapAllCoach {
		if coach.Phone == req.CoachPhone {
			coachId = coach.CoachID
			break
		}
	}

	if coachId == 0 {
		rsp.Code = -992
		rsp.ErrorMsg = "未找到对应教练"
		Printf("bindUser2CoachHandler coach not found, CoachPhone:%s\n", req.CoachPhone)
		return
	}

	// 更新用户的CoachId
	mapUpdates := make(map[string]interface{})
	mapUpdates["is_coach"] = true
	mapUpdates["coach_id"] = coachId
	err = dao.ImpUser.UpdateUserInfo(stCoachUserInfoModel.UserID, mapUpdates)
	if err != nil {
		rsp.Code = -991
		rsp.ErrorMsg = err.Error()
		Printf("bindUser2CoachHandler UpdateUser err, err:%+v uid:%d coachId:%d\n", err, stCoachUserInfoModel.UserID, coachId)
		return
	}

	Printf("bindUser2CoachHandler succ, uid:%d coachId:%d CoachPhone:%s\n", stCoachUserInfoModel.UserID, coachId, req.CoachPhone)
	rsp.Code = 0
}
