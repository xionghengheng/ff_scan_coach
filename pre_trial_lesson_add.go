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

// 创建预体验课请求
type CreatePreTrialLessonReq struct {
	UserPhone     string `json:"user_phone"`      // 用户手机号（微信绑定）
	TrainingNeed  string `json:"training_need"`   // 训练需求
	GymId         int    `json:"gym_id"`          // 门店ID
	GymName       string `json:"gym_name"`        // 门店名称
	CoachId       int    `json:"coach_id"`        // 教练ID
	CoachName     string `json:"coach_name"`      // 教练名称
	LessonDate    int64  `json:"lesson_date"`     // 体验课日期（时间戳）
	LessonTimeBeg int64  `json:"lesson_time_beg"` // 体验课开始时间（时间戳）
	LessonTimeEnd int64  `json:"lesson_time_end"` // 体验课结束时间（时间戳）
	Price         int    `json:"price"`           // 体验课价格（元）
	CreatedBy     string `json:"created_by"`      // 创建人（顾问）
}

// 创建预体验课响应
type CreatePreTrialLessonRsp struct {
	Code     int                         `json:"code"`
	ErrorMsg string                      `json:"errorMsg,omitempty"`
	Data     CreatePreTrialLessonRspData `json:"data,omitempty"`
}

// 响应数据
type CreatePreTrialLessonRspData struct {
	Id          int64  `json:"id"`            // 记录ID
	H5LinkToken string `json:"h5_link_token"` // 生成的token，用于H5页面访问
	CreatedAt   string `json:"created_at"`    // 创建时间
}

// 解析请求参数
func getCreatePreTrialLessonReq(r *http.Request) (CreatePreTrialLessonReq, error) {
	req := CreatePreTrialLessonReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// 创建预体验课处理函数
func CreatePreTrialLessonHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getCreatePreTrialLessonReq(r)
	rsp := &CreatePreTrialLessonRsp{}

	Printf("CreatePreTrialLessonHandler start, openid:%s req:%+v\n", strOpenId, req)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	// 验证管理员身份
	authResult := ValidateAdminAuth(r)
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

	// 参数校验
	checkParamResult := checkCreatePreTrialLessonParam(&req)
	if !checkParamResult.Success {
		rsp.Code = checkParamResult.Code
		rsp.ErrorMsg = checkParamResult.ErrorMsg
		return
	}

	// 生成数据
	nowTs := time.Now().Unix()

	// 构建预体验课记录（先不填token）
	preTrialLesson := model.PreTrailManageModel{
		UserPhone:     req.UserPhone,
		TrainingNeed:  req.TrainingNeed,
		GymID:         req.GymId,
		CoachID:       req.CoachId,
		LessonDate:    req.LessonDate,
		LessonTimeBeg: req.LessonTimeBeg,
		LessonTimeEnd: req.LessonTimeEnd,
		Price:         req.Price,
		CreatedBy:     req.CreatedBy,
		LinkStatus:    model.Enum_Link_Status_Pending, // 状态：待使用
		CreatedTs:     nowTs,
		UpdatedTs:     nowTs,
	}

	// 第一步：插入数据库获取记录ID
	err = dao.ImpPreTrailManage.AddTrailManage(&preTrialLesson)
	if err != nil {
		rsp.Code = -2001
		rsp.ErrorMsg = fmt.Sprintf("创建预体验课失败: %v", err)
		Printf("CreatePreTrialLesson err, err:%+v\n", err)
		return
	}
	Printf("CreatePreTrialLesson succ, preTrialLesson:%+v\n", preTrialLesson)

	// 第二步：生成包含记录ID的token
	h5Token := comm.GenerateH5LinkToken(preTrialLesson.ID, nowTs)

	// 第三步：更新token到记录中
	mapUpdates := make(map[string]interface{})
	mapUpdates["link_token"] = h5Token
	mapUpdates["updated_ts"] = time.Now().Unix()
	err = dao.ImpPreTrailManage.UpdateTrailManage(preTrialLesson.ID, mapUpdates)
	if err != nil {
		rsp.Code = -2002
		rsp.ErrorMsg = fmt.Sprintf("更新token失败: %v", err)
		Printf("UpdateToken err, id:%d err:%+v\n", preTrialLesson.ID, err)
		return
	}
	Printf("UpdateToken succ, id:%d h5Token:%s\n", preTrialLesson.ID, h5Token)

	// 构建响应
	rsp.Code = 0
	rsp.Data = CreatePreTrialLessonRspData{
		Id:          preTrialLesson.ID,
		H5LinkToken: h5Token,
		CreatedAt:   time.Unix(nowTs, 0).Format("2006-01-02 15:04:05"),
	}

	Printf("CreatePreTrialLessonHandler success, id:%d token:%s\n", preTrialLesson.ID, h5Token)
}

// 参数校验结果
type CheckParamResult struct {
	Success  bool
	Code     int
	ErrorMsg string
}

// 校验创建预体验课请求参数
func checkCreatePreTrialLessonParam(req *CreatePreTrialLessonReq) CheckParamResult {
	if req.UserPhone == "" {
		return CheckParamResult{Success: false, Code: -1001, ErrorMsg: "用户手机号不能为空"}
	}

	if req.CoachId == 0 {
		return CheckParamResult{Success: false, Code: -1002, ErrorMsg: "教练ID无效"}
	}

	if req.GymId == 0 {
		return CheckParamResult{Success: false, Code: -1003, ErrorMsg: "门店ID无效"}
	}

	if req.LessonTimeBeg == 0 || req.LessonTimeEnd == 0 {
		return CheckParamResult{Success: false, Code: -1004, ErrorMsg: "体验课时间无效"}
	}

	// 校验时间必须为整点（时间戳对3600取余为0表示整点）
	if req.LessonTimeBeg%3600 != 0 {
		return CheckParamResult{Success: false, Code: -1006, ErrorMsg: "体验课开始时间必须为整点"}
	}
	if req.LessonTimeEnd%3600 != 0 {
		return CheckParamResult{Success: false, Code: -1007, ErrorMsg: "体验课结束时间必须为整点"}
	}

	if req.LessonTimeBeg >= req.LessonTimeEnd {
		return CheckParamResult{Success: false, Code: -1005, ErrorMsg: "体验课开始时间必须早于结束时间"}
	}

	return CheckParamResult{Success: true}
}
