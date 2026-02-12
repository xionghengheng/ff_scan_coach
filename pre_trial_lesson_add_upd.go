package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

// 更新预体验课请求（用户手机号和价格不支持修改）
type UpdatePreTrialLessonReq struct {
	Id            int64  `json:"id"`              // 记录ID（必填）
	TrainingNeed  string `json:"training_need"`   // 训练需求
	GymId         int    `json:"gym_id"`          // 门店ID
	CoachId       int    `json:"coach_id"`        // 教练ID
	CourseId      int    `json:"course_id"`       // 课程ID
	LessonDate    int64  `json:"lesson_date"`     // 体验课日期（时间戳）
	LessonTimeBeg int64  `json:"lesson_time_beg"` // 体验课开始时间（时间戳）
	LessonTimeEnd int64  `json:"lesson_time_end"` // 体验课结束时间（时间戳）
}

// 更新预体验课响应
type UpdatePreTrialLessonRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

// 解析更新请求参数
func getUpdatePreTrialLessonReq(r *http.Request) (UpdatePreTrialLessonReq, error) {
	req := UpdatePreTrialLessonReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// 更新预体验课处理函数
func UpdatePreTrialLessonHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getUpdatePreTrialLessonReq(r)
	rsp := &UpdatePreTrialLessonRsp{}

	Printf("UpdatePreTrialLessonHandler start, openid:%s req:%+v\n", strOpenId, req)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	// 身份验证：优先通过OpenID识别顾问，否则走管理员账号密码校验
	authResult := ValidateConsultantOrAdminAuth(r)
	if !authResult.Success {
		rsp.Code = authResult.Code
		rsp.ErrorMsg = authResult.ErrorMsg
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	// 校验ID
	if req.Id <= 0 {
		rsp.Code = -1001
		rsp.ErrorMsg = "记录ID无效"
		return
	}

	// 查询原记录
	preTrialLesson, err := dao.ImpPreTrailManage.GetTrailManageById(req.Id)
	if err != nil {
		rsp.Code = -2001
		rsp.ErrorMsg = "查询预体验课记录失败"
		Printf("UpdatePreTrialLessonHandler GetTrailManageById err, id:%d err:%+v\n", req.Id, err)
		return
	}

	if preTrialLesson == nil {
		rsp.Code = -2002
		rsp.ErrorMsg = "预体验课记录不存在"
		Printf("UpdatePreTrialLessonHandler record not found, id:%d\n", req.Id)
		return
	}

	// 检查状态：已过期、已取消和已完成（已使用）的不支持更新
	// 同时需要检查是否实时过期（待使用状态但已超过24小时）
	linkStatus := comm.GetRealLinkStatus(preTrialLesson.LinkStatus, preTrialLesson.CreatedTs)

	switch linkStatus {
	case model.Enum_Link_Status_Used:
		rsp.Code = -3001
		rsp.ErrorMsg = "已使用的预体验课不支持更新"
		Printf("UpdatePreTrialLessonHandler record already used, id:%d\n", req.Id)
		return
	case model.Enum_Link_Status_Expired:
		rsp.Code = -3002
		rsp.ErrorMsg = "已过期的预体验课不支持更新"
		Printf("UpdatePreTrialLessonHandler record expired, id:%d\n", req.Id)
		return
	case model.Enum_Link_Status_Cancel:
		rsp.Code = -3003
		rsp.ErrorMsg = "已取消的预体验课不支持更新"
		Printf("UpdatePreTrialLessonHandler record canceled, id:%d\n", req.Id)
		return
	}

	// 校验更新参数
	checkResult := checkUpdatePreTrialLessonParam(&req)
	if !checkResult.Success {
		rsp.Code = checkResult.Code
		rsp.ErrorMsg = checkResult.ErrorMsg
		return
	}

	// 构建更新字段（用户手机号和价格不支持更新）
	mapUpdates := make(map[string]interface{})
	if req.TrainingNeed != "" {
		mapUpdates["training_need"] = req.TrainingNeed
	}
	if req.GymId > 0 {
		mapUpdates["gym_id"] = req.GymId
	}
	if req.CoachId > 0 {
		mapUpdates["coach_id"] = req.CoachId
	}
	if req.CourseId > 0 {
		mapUpdates["course_id"] = req.CourseId
	}
	if req.LessonDate > 0 {
		mapUpdates["lesson_date"] = req.LessonDate
	}
	if req.LessonTimeBeg > 0 {
		mapUpdates["lesson_time_beg"] = req.LessonTimeBeg
	}
	if req.LessonTimeEnd > 0 {
		mapUpdates["lesson_time_end"] = req.LessonTimeEnd
	}
	mapUpdates["updated_ts"] = time.Now().Unix()

	// 执行更新
	err = dao.ImpPreTrailManage.UpdateTrailManage(req.Id, mapUpdates)
	if err != nil {
		rsp.Code = -2003
		rsp.ErrorMsg = fmt.Sprintf("更新预体验课失败: %v", err)
		Printf("UpdatePreTrialLessonHandler UpdateTrailManage err, id:%d err:%+v\n", req.Id, err)
		return
	}

	rsp.Code = 0
	Printf("UpdatePreTrialLessonHandler success, id:%d\n", req.Id)
}

// 校验更新预体验课请求参数
func checkUpdatePreTrialLessonParam(req *UpdatePreTrialLessonReq) CheckParamResult {
	// 如果传了时间参数，需要校验
	if req.LessonTimeBeg > 0 && req.LessonTimeEnd > 0 {
		// 校验时间必须为整点
		if req.LessonTimeBeg%3600 != 0 {
			return CheckParamResult{Success: false, Code: -1006, ErrorMsg: "体验课开始时间必须为整点"}
		}
		if req.LessonTimeEnd%3600 != 0 {
			return CheckParamResult{Success: false, Code: -1007, ErrorMsg: "体验课结束时间必须为整点"}
		}

		if req.LessonTimeBeg >= req.LessonTimeEnd {
			return CheckParamResult{Success: false, Code: -1005, ErrorMsg: "体验课开始时间必须早于结束时间"}
		}
	}

	return CheckParamResult{Success: true}
}
