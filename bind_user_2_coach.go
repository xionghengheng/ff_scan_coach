package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

type BindUser2CoachReq struct {
	CoachName     string `json:"coach_name"`      //教练名称
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

	if len(req.CoachName) == 0 {
		rsp.Code = -996
		rsp.ErrorMsg = "缺少教练名称"
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

	// 根据手机号查找教练
	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -921
		rsp.ErrorMsg = err.Error()
		Printf("bindUser2CoachHandler GetAllCoach err, err:%+v\n", err)
		return
	}

	var stCoachModel model.CoachModel
	for _, coach := range mapAllCoach {
		if coach.CoachName == req.CoachName {
			stCoachModel = coach
			break
		}
	}

	if stCoachModel.CoachID == 0 {
		rsp.Code = -992
		rsp.ErrorMsg = "未找到对应教练"
		Printf("bindUser2CoachHandler coach not found, CoachPhone:%s\n", req.CoachPhone)
		return
	}

	if len(stCoachModel.Phone) == 0 {
		mapUpdates := make(map[string]interface{})
		mapUpdates["phone"] = req.CoachPhone
		err := dao.ImpCoach.UpdateCoachInfo(stCoachModel.CoachID, mapUpdates)
		if err != nil {
			rsp.Code = -922
			rsp.ErrorMsg = "更新教练手机号失败"
			Printf("UpdateCoachInfo err, err:%+v CoachName:%s CoachPhone:%s\n", err, req.CoachName, req.CoachPhone)
			return
		}
	}

	// 根据手机号获取用户信息
	stCoachUserInfoModel, err := dao.ImpUser.GetUserByPhone(req.CoachPhone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rsp.Code = -911
			rsp.ErrorMsg = "手机号对应的用户，未找到"
		} else {
			rsp.Code = -922
			rsp.ErrorMsg = "通过手机号拉取用户信息失败"
		}
		Printf("bindUser2CoachHandler GetUserByPhone err, err:%+v CoachPhone:%s\n", err, req.CoachPhone)
		return
	}

	// 更新用户的CoachId
	mapUpdates := make(map[string]interface{})
	mapUpdates["is_coach"] = true
	mapUpdates["coach_id"] = stCoachModel.CoachID
	err = dao.ImpUser.UpdateUserInfo(stCoachUserInfoModel.UserID, mapUpdates)
	if err != nil {
		rsp.Code = -991
		rsp.ErrorMsg = err.Error()
		Printf("bindUser2CoachHandler UpdateUser err, err:%+v uid:%d coachId:%d\n", err, stCoachUserInfoModel.UserID, stCoachModel.CoachID)
		return
	}

	Printf("bindUser2CoachHandler succ, uid:%d coachId:%d CoachName:%s CoachPhone:%s\n", stCoachUserInfoModel.UserID, stCoachModel.CoachID, req.CoachName, req.CoachPhone)
	rsp.Code = 0
	go TestTriggerSetCoachLessonAvaliable(stCoachModel.CoachID, req.CoachName)
}

type TestTriggerSetCoachLessonAvaliableReq struct {
	CoachId int   `json:"coach_id"` //教练id
	BegTs   int64 `json:"beg_ts"`   //开始时间
}

type TestTriggerSetCoachLessonAvaliableRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

func TestTriggerSetCoachLessonAvaliable(coachId int, coachName string) {
	// 构造请求数据
	req := TestTriggerSetCoachLessonAvaliableReq{
		CoachId: coachId,
		BegTs:   time.Now().Unix(),
	}

	// 将请求数据转换为JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		Printf("TestTriggerSetCoachLessonAvaliable marshal req failed, coachId:%d, err:%+v\n", coachId, err)
		return
	}

	// 创建带超时的HTTP客户端
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 发送HTTP POST请求
	url := "https://golang-v3fg-107847-6-1326535808.sh.run.tcloudbase.com/api/testTriggerSetCoachLessonAvaliable"
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		Printf("TestTriggerSetCoachLessonAvaliable http post failed, coachId:%d, err:%+v\n", coachId, err)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	var rsp TestTriggerSetCoachLessonAvaliableRsp
	if err := json.NewDecoder(resp.Body).Decode(&rsp); err != nil {
		Printf("TestTriggerSetCoachLessonAvaliable decode response failed, coachId:%d, err:%+v\n", coachId, err)
		return
	}

	if rsp.Code != 0 {
		Printf("TestTriggerSetCoachLessonAvaliable response error, coachId:%d, code:%d, errorMsg:%s\n", coachId, rsp.Code, rsp.ErrorMsg)
		return
	}

	Printf("TestTriggerSetCoachLessonAvaliable success, coachId:%d coachName:%s\n", coachId, coachName)
}
