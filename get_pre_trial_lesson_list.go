package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
)

// GetPreTrialLessonListReq 获取预体验课列表请求
type GetPreTrialLessonListReq struct {
}

// GetPreTrialLessonListRsp 获取预体验课列表响应
type GetPreTrialLessonListRsp struct {
	Code     int                  `json:"code"`
	ErrorMsg string               `json:"errorMsg,omitempty"`
	List     []PreTrialLessonItem `json:"list,omitempty"`
}

// PreTrialLessonItem 预体验课列表项
type PreTrialLessonItem struct {
	Id            int64  `json:"id"`              // 记录ID
	LinkToken     string `json:"link_token"`      // 链接token
	LinkStatus    int    `json:"link_status"`     // l链接状态：0-待使用，1-已使用，2-已过期
	UserPhone     string `json:"user_phone"`      // 用户手机号
	TrainingNeed  string `json:"training_need"`   // 训练需求
	GymId         int    `json:"gym_id"`          // 门店ID
	GymName       string `json:"gym_name"`        // 门店名称
	CoachId       int    `json:"coach_id"`        // 教练ID
	CoachName     string `json:"coach_name"`      // 教练名称
	LessonDate    string `json:"lesson_date"`     // 体验课日期（格式化）
	LessonTimeBeg string `json:"lesson_time_beg"` // 体验课开始时间（格式化）
	LessonTimeEnd string `json:"lesson_time_end"` // 体验课结束时间（格式化）
	Price         int    `json:"price"`           // 体验课价格（元）
	CreatedBy     string `json:"created_by"`      // 创建人（顾问）
	StatusText    string `json:"status_text"`     // 状态文本
	CreatedTs     string `json:"created_ts"`      // 创建时间
	UpdateTs      string `json:"update_ts"`       // 更新时间
}

// getGetPreTrialLessonListReq 解析请求参数
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

	// 验证管理员用户名和密码
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
		Printf("GetPreTrialLessonListHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("GetPreTrialLessonListHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("GetPreTrialLessonListHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	// 查询预体验课列表
	list, err := dao.ImpPreTrailManage.GetTrailManageList(0, 1)
	if err != nil {
		rsp.Code = -2002
		rsp.ErrorMsg = fmt.Sprintf("查询预体验课列表失败: %v", err)
		Printf("GetPreTrialLessonListHandler GetPreTrialLessonList err:%+v\n", err)
		return
	}

	// 获取所有门店和教练信息，用于查询最新名称
	mapGym, _ := comm.GetAllGym()
	mapCoach, _ := comm.GetAllCoach()

	// 转换为响应格式
	rsp.List = make([]PreTrialLessonItem, 0, len(list))
	for _, item := range list {
		statusText := "待使用"
		switch item.LinkStatus {
		case PreTrialLessonStatusUsed:
			statusText = "已使用"
		case PreTrialLessonStatusExpired:
			statusText = "已过期"
		case PreTrialLessonStatusCanceled:
			statusText = "已取消"
		}

		// 根据ID获取门店名称和教练名称
		gymName := ""
		if gymInfo, ok := mapGym[item.GymID]; ok {
			gymName = gymInfo.LocName
		}
		coachName := ""
		if coachInfo, ok := mapCoach[item.CoachID]; ok {
			coachName = coachInfo.CoachName
		}

		rsp.List = append(rsp.List, PreTrialLessonItem{
			Id:            item.ID,
			LinkToken:     item.LinkToken,
			UserPhone:     item.UserPhone,
			TrainingNeed:  item.TrainingNeed,
			GymId:         item.GymID,
			GymName:       gymName,
			CoachId:       item.CoachID,
			CoachName:     coachName,
			LessonDate:    time.Unix(item.LessonDate, 0).Format("2006-01-02"),
			LessonTimeBeg: time.Unix(item.LessonTimeBeg, 0).Format("15:04"),
			LessonTimeEnd: time.Unix(item.LessonTimeEnd, 0).Format("15:04"),
			Price:         item.Price,
			CreatedBy:     item.CreatedBy,
			LinkStatus:    item.LinkStatus,
			StatusText:    statusText,
			CreatedTs:     time.Unix(item.CreatedTs, 0).Format("2006-01-02 15:04:05"),
			UpdateTs:      time.Unix(item.UpdatedTs, 0).Format("2006-01-02 15:04:05"),
		})
	}

	rsp.Code = 0
	Printf("GetPreTrialLessonListHandler success, count:%d\n", len(rsp.List))
}
