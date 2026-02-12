package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

// GetPreTrialLessonListReq 获取预体验课列表请求
type GetPreTrialLessonListReq struct {
	Passback string `json:"passback"`  // 翻页标记，首次请求传空字符串，后续传上次返回的passback
	PageSize int    `json:"page_size"` // 每页数量
}

// GetPreTrialLessonListRsp 获取预体验课列表响应
type GetPreTrialLessonListRsp struct {
	Code     int                  `json:"code"`
	ErrorMsg string               `json:"errorMsg,omitempty"`
	List     []PreTrialLessonItem `json:"list,omitempty"`
	Passback string               `json:"passback"` // 下一页的翻页标记，为空字符串表示没有更多数据
}

// PreTrialLessonItem 预体验课列表项
type PreTrialLessonItem struct {
	Id             int64  `json:"id"`               // 记录ID
	LinkToken      string `json:"link_token"`       // 链接token
	LinkStatus     int    `json:"link_status"`      // 链接状态：0-待使用，1-已使用，2-已过期
	LinkStatusText string `json:"link_status_text"` // 状态文本
	UserPhone      string `json:"user_phone"`       // 用户手机号
	TrainingNeed   string `json:"training_need"`    // 训练需求
	GymId          int    `json:"gym_id"`           // 门店ID
	GymName        string `json:"gym_name"`         // 门店名称
	CoachId        int    `json:"coach_id"`         // 教练ID
	CoachName      string `json:"coach_name"`       // 教练名称
	CourseId       int    `json:"course_id"`        // 课程ID
	CourseName     string `json:"course_name"`      // 课程名称
	LessonDate     string `json:"lesson_date"`      // 体验课日期（格式化）
	LessonTimeBeg  string `json:"lesson_time_beg"`  // 体验课开始时间（格式化）
	LessonTimeEnd  string `json:"lesson_time_end"`  // 体验课结束时间（格式化）
	Price          int    `json:"price"`            // 体验课价格（元）
	CreatedBy      string `json:"created_by"`       // 创建人（顾问）
	CreatedTs      string `json:"created_ts"`       // 创建时间
	UpdateTs       string `json:"update_ts"`        // 更新时间
}

// 解析请求参数
func getGetPreTrialLessonListReq(r *http.Request) (GetPreTrialLessonListReq, error) {
	req := GetPreTrialLessonListReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// 获取预体验课列表处理函数
func GetPreTrialLessonListHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetPreTrialLessonListReq(r)
	rsp := &GetPreTrialLessonListRsp{}

	Printf("GetPreTrialLessonListHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	// 处理翻页参数，设置默认值
	var offset int64
	if len(req.Passback) > 0 {
		offset, _ = strconv.ParseInt(req.Passback, 10, 64)
	}
	if offset < 0 {
		offset = 0
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20 // 默认每页20条
	}
	if pageSize > 100 {
		pageSize = 100 // 最大每页100条
	}

	// 查询预体验课列表（passback作为offset使用）
	list, err := dao.ImpPreTrailManage.GetTrailManageList(int(offset), pageSize)
	if err != nil {
		rsp.Code = -2002
		rsp.ErrorMsg = fmt.Sprintf("查询预体验课列表失败: %v", err)
		Printf("GetPreTrialLessonListHandler GetPreTrialLessonList err:%+v\n", err)
		return
	}
	Printf("GetTrailManageList succ, offset:%d pageSize:%d list:%+v\n", offset, pageSize, list)

	// 获取所有门店、教练和课程信息，用于查询最新名称
	mapGym, _ := comm.GetAllGym()
	mapCoach, _ := comm.GetAllCoach()
	mapCourse, _ := comm.GetAllCourse()

	// 转换为响应格式
	rsp.List = make([]PreTrialLessonItem, 0, len(list))
	for _, item := range list {
		// 实时计算过期状态
		linkStatus := comm.GetRealLinkStatus(item.LinkStatus, item.CreatedTs)

		statusText := "待使用"
		switch linkStatus {
		case model.Enum_Link_Status_Used:
			statusText = "已使用"
		case model.Enum_Link_Status_Expired:
			statusText = "已过期"
		case model.Enum_Link_Status_Cancel:
			statusText = "已取消"
		}

		// 根据ID获取门店名称、教练名称和课程名称
		gymName := ""
		if gymInfo, ok := mapGym[item.GymID]; ok {
			gymName = gymInfo.LocName
		}
		coachName := ""
		if coachInfo, ok := mapCoach[item.CoachID]; ok {
			coachName = coachInfo.CoachName
		}
		courseName := ""
		if courseInfo, ok := mapCourse[item.CourseID]; ok {
			courseName = courseInfo.Name
		}

		rsp.List = append(rsp.List, PreTrialLessonItem{
			Id:             item.ID,
			LinkToken:      item.LinkToken,
			UserPhone:      item.UserPhone,
			TrainingNeed:   item.TrainingNeed,
			GymId:          item.GymID,
			GymName:        gymName,
			CoachId:        item.CoachID,
			CoachName:      coachName,
			CourseId:       item.CourseID,
			CourseName:     courseName,
			LessonDate:     time.Unix(item.LessonDate, 0).Format("2006-01-02"),
			LessonTimeBeg:  time.Unix(item.LessonTimeBeg, 0).Format("15:04"),
			LessonTimeEnd:  time.Unix(item.LessonTimeEnd, 0).Format("15:04"),
			Price:          item.Price,
			CreatedBy:      item.CreatedBy,
			LinkStatus:     linkStatus,
			LinkStatusText: statusText,
			CreatedTs:      time.Unix(item.CreatedTs, 0).Format("2006-01-02 15:04:05"),
			UpdateTs:       time.Unix(item.UpdatedTs, 0).Format("2006-01-02 15:04:05"),
		})
	}

	// 设置下一页的passback（当前offset + 返回数量）
	// 如果返回数量小于pageSize，说明没有更多数据，passback设为空字符串
	if len(list) >= pageSize {
		rsp.Passback = strconv.FormatInt(offset+int64(len(list)), 10)
	} else {
		rsp.Passback = "" // 没有更多数据
	}

	rsp.Code = 0
	Printf("GetPreTrialLessonListHandler success, count:%d passback:%s\n", len(rsp.List), rsp.Passback)
}
