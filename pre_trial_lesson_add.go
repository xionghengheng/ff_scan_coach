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
	CourseId      int    `json:"course_id"`       // 课程ID
	CourseName    string `json:"course_name"`     // 课程名称
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

	// 身份验证：优先通过OpenID识别顾问，否则走管理员账号密码校验
	authResult := ValidateConsultantOrAdminAuth(r)
	if !authResult.Success {
		rsp.Code = authResult.Code
		rsp.ErrorMsg = authResult.ErrorMsg
		return
	}
	if authResult.IsConsultant {
		req.CreatedBy = authResult.ConsultantNick
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	// 参数校验（包含课程价格校验）
	checkParamResult := checkCreatePreTrialLessonParam(&req)
	if !checkParamResult.Success {
		rsp.Code = checkParamResult.Code
		rsp.ErrorMsg = checkParamResult.ErrorMsg
		return
	}

	// 前置检查：教练是否绑定场地 & 用户是否已有有效预体验课
	preCheckResult := preCheckCreatePreTrialLesson(&req)
	if !preCheckResult.Success {
		rsp.Code = preCheckResult.Code
		rsp.ErrorMsg = preCheckResult.ErrorMsg
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
		CourseID:      req.CourseId,
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

	if req.CourseId == 0 {
		return CheckParamResult{Success: false, Code: -1008, ErrorMsg: "课程ID无效"}
	}

	// 校验课程ID对应的价格是否与传入价格一致
	mapCourse, err := comm.GetAllCourse()
	if err != nil {
		return CheckParamResult{Success: false, Code: -1009, ErrorMsg: "获取课程信息失败"}
	}
	courseInfo, ok := mapCourse[req.CourseId]
	if !ok {
		return CheckParamResult{Success: false, Code: -1010, ErrorMsg: "课程ID不存在"}
	}
	if courseInfo.Price != req.Price {
		return CheckParamResult{Success: false, Code: -1011, ErrorMsg: fmt.Sprintf("价格与课程不匹配，课程体验价：%d，传入价格：%d", courseInfo.Price, req.Price)}
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

// preCheckCoachBindGym 检查教练是否绑定了对应的场地
func preCheckCoachBindGym(coachId int, gymId int) CheckParamResult {
	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		Printf("preCheck GetAllCoach err, err:%+v\n", err)
		return CheckParamResult{Success: false, Code: -1020, ErrorMsg: "获取教练信息失败"}
	}
	coachModel, ok := mapAllCoach[coachId]
	if !ok {
		Printf("preCheck Coach not found, coachId:%d\n", coachId)
		return CheckParamResult{Success: false, Code: -1021, ErrorMsg: "教练不存在"}
	}
	for _, gid := range comm.GetAllGymIds(coachModel.GymIDs) {
		if gid == gymId {
			return CheckParamResult{Success: true}
		}
	}
	Printf("preCheck Coach not bound to gym, coachId:%d gymId:%d coachGymIDs:%s\n", coachId, gymId, coachModel.GymIDs)
	return CheckParamResult{Success: false, Code: -1022, ErrorMsg: "该教练未绑定所选场地，请检查教练和场地的对应关系"}
}

// preCheckUserNoDuplicatePending 检查用户是否已有有效的预体验课（待使用状态）
func preCheckUserNoDuplicatePending(userPhone string) CheckParamResult {
	existList, err := dao.ImpPreTrailManage.GetTrailManageListByPhone(userPhone)
	if err != nil {
		Printf("preCheck GetTrailManageListByPhone err, phone:%s err:%+v\n", userPhone, err)
		return CheckParamResult{Success: false, Code: -1023, ErrorMsg: "查询用户预体验课记录失败"}
	}
	for _, item := range existList {
		realStatus := comm.GetRealLinkStatus(item.LinkStatus, item.CreatedTs)
		if realStatus == model.Enum_Link_Status_Pending {
			Printf("preCheck User already has valid pre-trial lesson, phone:%s existId:%d\n", userPhone, item.ID)
			return CheckParamResult{Success: false, Code: -1024, ErrorMsg: "该用户已有一条有效的预体验课记录，不能重复添加"}
		}
	}
	return CheckParamResult{Success: true}
}

// preCheckCreatePreTrialLesson 创建预体验课前置检查：教练绑定场地 + 用户无重复
func preCheckCreatePreTrialLesson(req *CreatePreTrialLessonReq) CheckParamResult {
	if result := preCheckCoachBindGym(req.CoachId, req.GymId); !result.Success {
		return result
	}
	if result := preCheckUserNoDuplicatePending(req.UserPhone); !result.Success {
		return result
	}
	return CheckParamResult{Success: true}
}
