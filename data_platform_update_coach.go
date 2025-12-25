package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
)

// UpdateCoachReq 更新教练基础属性请求
type UpdateCoachReq struct {
	CoachID             int    `json:"coach_id"`                        //教练id（必填）
	CoachName           string `json:"coach_name,omitempty"`            //教练名称
	Phone               string `json:"phone,omitempty"`                 //手机号
	Bio                 string `json:"bio,omitempty"`                   //教练简介
	GoodAt              string `json:"good_at,omitempty"`               //教练擅长领域
	Style               string `json:"style,omitempty"`                 //教练风格（英文逗号分隔）
	SkillCertification  string `json:"skill_certification,omitempty"`   //教练的技能认证（英文逗号分隔）
	YearsOfWork         string `json:"years_of_work,omitempty"`         //从业时长
	TotalCompleteLesson string `json:"total_complete_lesson,omitempty"` //累计上课节数
	GymIDs              string `json:"gym_ids,omitempty"`               //教练绑定的健身房id列表（英文逗号分隔）
	CourseIdList        string `json:"course_id_list,omitempty"`        //教练可上的课程id列表（英文逗号分隔）
	//QualifyType *int `json:"qualify_type,omitempty"` //教练资质类型（使用指针以区分0值和未设置）
	//LoginUserName       string `json:"login_user_name"`                 //管理平台用户名（必填）
	//LoginPasswd         string `json:"login_passwd"`                    //管理平台密码（必填）
	//Avatar              string `json:"avatar,omitempty"`                //教练头像url
	//CircleAvatar        string `json:"circle_avatar,omitempty"`         //教练圆形头像url
}

// UpdateCoachRsp 更新教练基础属性响应
type UpdateCoachRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

// getUpdateCoachReq 解析请求参数
func getUpdateCoachReq(r *http.Request) (UpdateCoachReq, error) {
	req := UpdateCoachReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// UpdateCoachHandler 更新教练基础属性接口
func UpdateCoachHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getUpdateCoachReq(r)
	rsp := &UpdateCoachRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("UpdateCoachHandler req start, req:%+v\n", req)

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
		Printf("UpdateCoachHandler parse req err, err:%+v\n", err)
		return
	}

	// 验证用户名和密码
	adminUserName := os.Getenv("ADMIN_USER_NAME")
	adminPasswd := os.Getenv("ADMIN_PASSWD")
	if len(adminUserName) == 0 || len(adminPasswd) == 0 {
		rsp.Code = -900
		rsp.ErrorMsg = "后台配置错误"
		Printf("conf err, adminUserName:%s adminPasswd:%s\n", adminUserName, adminPasswd)
		return
	}

	// 从header中提取用户名和密码进行校验
	username := r.Header.Get("X-Username")
	if username == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Username header"
		Printf("UpdateCoachHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("UpdateCoachHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("UpdateCoachHandler auth failed, username:%s\n", username)
		return
	}

	// 参数校验
	if req.CoachID <= 0 {
		rsp.Code = -997
		rsp.ErrorMsg = "教练ID不能为空"
		Printf("UpdateCoachHandler CoachID invalid, CoachID:%d\n", req.CoachID)
		return
	}

	//if len(req.LoginUserName) == 0 || len(req.LoginPasswd) == 0 {
	//	rsp.Code = -995
	//	rsp.ErrorMsg = "缺少管理平台用户名或密码"
	//	Printf("UpdateCoachHandler missing auth info\n")
	//	return
	//}

	//// 验证管理平台用户名和密码
	//adminUserName := os.Getenv("ADMIN_USER_NAME")
	//adminPasswd := os.Getenv("ADMIN_PASSWD")
	//if len(adminUserName) == 0 {
	//	adminUserName = "admin" // 默认值，实际应该从环境变量获取
	//}
	//if len(adminPasswd) == 0 {
	//	adminPasswd = "admin123" // 默认值，实际应该从环境变量获取
	//}
	//
	//if req.LoginUserName != adminUserName || req.LoginPasswd != adminPasswd {
	//	rsp.Code = -994
	//	rsp.ErrorMsg = "管理平台用户名或密码错误"
	//	Printf("UpdateCoachHandler auth failed, LoginUserName:%s\n", req.LoginUserName)
	//	return
	//}

	// 验证教练是否存在
	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -921
		rsp.ErrorMsg = err.Error()
		Printf("UpdateCoachHandler GetAllCoach err, err:%+v\n", err)
		return
	}

	coachInfo, exists := mapAllCoach[req.CoachID]
	if !exists {
		rsp.Code = -992
		rsp.ErrorMsg = "未找到对应教练"
		Printf("UpdateCoachHandler coach not found, CoachID:%d\n", req.CoachID)
		return
	}

	// 构建更新字段map，只更新值不同的字段
	mapUpdates := make(map[string]interface{})

	if len(req.CoachName) > 0 && req.CoachName != coachInfo.CoachName {
		mapUpdates["coach_name"] = req.CoachName
	}
	if len(req.Bio) > 0 && req.Bio != coachInfo.Bio {
		mapUpdates["bio"] = req.Bio
	}
	if len(req.GoodAt) > 0 && req.GoodAt != coachInfo.GoodAt {
		mapUpdates["good_at"] = req.GoodAt
	}
	if len(req.Phone) > 0 && req.Phone != coachInfo.Phone {
		mapUpdates["phone"] = req.Phone
	}
	if len(req.SkillCertification) > 0 && req.SkillCertification != coachInfo.SkillCertification {
		mapUpdates["skill_certification"] = req.SkillCertification
	}
	if len(req.Style) > 0 && req.Style != coachInfo.Style {
		mapUpdates["style"] = req.Style
	}
	if len(req.YearsOfWork) > 0 && req.YearsOfWork != coachInfo.YearsOfWork {
		mapUpdates["years_of_work"] = req.YearsOfWork
	}
	if len(req.TotalCompleteLesson) > 0 && req.TotalCompleteLesson != coachInfo.TotalCompleteLesson {
		mapUpdates["total_complete_lesson"] = req.TotalCompleteLesson
	}
	if len(req.GymIDs) > 0 && req.GymIDs != coachInfo.GymIDs {
		mapUpdates["gym_ids"] = req.GymIDs
	}
	if len(req.CourseIdList) > 0 && req.CourseIdList != coachInfo.CourseIdList {
		mapUpdates["course_id_list"] = req.CourseIdList
	}

	// 检查是否有字段需要更新
	if len(mapUpdates) == 0 {
		rsp.Code = -996
		rsp.ErrorMsg = "没有需要更新的字段"
		Printf("UpdateCoachHandler no fields to update, CoachID:%d\n", req.CoachID)
		return
	}

	// 执行更新操作
	err = dao.ImpCoach.UpdateCoachInfo(req.CoachID, mapUpdates)
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = "更新教练信息失败"
		Printf("UpdateCoachHandler UpdateCoachInfo err, err:%+v CoachID:%d mapUpdates:%+v\n", err, req.CoachID, mapUpdates)
		return
	}

	Printf("UpdateCoachHandler succ, CoachID:%d mapUpdates:%+v\n", req.CoachID, mapUpdates)
	rsp.Code = 0
}
