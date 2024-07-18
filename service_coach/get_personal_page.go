package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"encoding/json"
	"fmt"
	"net/http"
)

type GetPersonalPageReq struct {
	StatisticTs int64 `json:"statistic_ts"` //统计计数对应的月份（比如4月1日0点时间戳）
}

type GetPersonalPageRsp struct {
	Code         int          `json:"code"`
	ErrorMsg     string       `json:"errorMsg,omitempty"`
	CoachId      int          `json:"coach_id,omitempty"`       //教练id
	CoachName    string       `json:"coach_name,omitempty"`     //教练名称
	CoachHeadPic string       `json:"coach_head_pic,omitempty"` //教练头像
	MonthCntInfo StatisticCnt `json:"month_cnt_info,omitempty"` //月度统计数据
}

type StatisticCnt struct {
	StatisticTs   int64  `json:"statistic_ts"`              //统计计数对应的月份
	PayUserCnt    uint32 `json:"pay_user_cnt,omitempty"`    //付费用户数
	LessonCnt     uint32 `json:"lesson_cnt,omitempty"`      //上课数
	LessonUserCnt uint32 `json:"lesson_user_cnt,omitempty"` //上课用户数
	SaleRevenue   uint32 `json:"sale_revenue,omitempty"`    //销售额(单位元)
}

func getGetPersonalPageReq(r *http.Request) (GetPersonalPageReq, error) {
	req := GetPersonalPageReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetPersonalPageHandler 拉取教练端个人页
func GetPersonalPageHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetPersonalPageReq(r)
	rsp := &GetPersonalPageRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetPersonalPageHandler start, openid:%s\n", strOpenId)

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

	if len(strOpenId) == 0 || req.StatisticTs == 0 {
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
	coachId := stUserInfoModel.CoachId

	if stUserInfoModel.IsCoach == false || coachId == 0 {
		rsp.Code = -900
		rsp.ErrorMsg = "not coach return"
		Printf("not coach err, strOpenId:%s uid:%d\n", strOpenId, uid)
		return
	}

	stCoachModel, err := dao.ImpCoach.GetCoachById(coachId)
	if err != nil || stUserInfoModel == nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		Printf("GetCoachById err, strOpenId:%s err:%+v\n", strOpenId, err)
		return
	}

	rsp.CoachId = stCoachModel.CoachID
	rsp.CoachName = stCoachModel.CoachName
	rsp.CoachHeadPic = stCoachModel.Avatar
	rsp.MonthCntInfo.StatisticTs = req.StatisticTs
	rsp.MonthCntInfo.PayUserCnt = 12
	rsp.MonthCntInfo.LessonCnt = 3
	rsp.MonthCntInfo.LessonUserCnt = 9
	rsp.MonthCntInfo.SaleRevenue = 876
}
