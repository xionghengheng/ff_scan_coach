package main

import (
	"encoding/json"
	"fmt"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"net/http"
	"time"
)

type GetUserStatisticReq struct {
	StatisticTs string `json:"statistic_ts"`
}

type UserStatisticInfo struct {
	UserID                  int64  `json:"user_id"`                   //用户uid
	WechatID                string `json:"wechat_id"`                 //微信openid
	PhoneNumber             string `json:"phone_number"`              //手机号
	Nick                    string `json:"nick,omitempty"`            //昵称
	HeadPic                 string `json:"head_pic"`                  //头像
	Gender                  string `json:"gender"`                    //"0=男", "1=女", "2=other"
	Age                     int    `json:"age"`                       //年龄
	Weight                  int    `json:"weight"`                    //体重
	Height                  int    `json:"height"`                    //身高
	FitnessExperience       int    `json:"fitness_experience"`        //健身经验
	FitnessGoal             int    `json:"fitness_goal"`              //健身目标
	DesiredWeight           int    `json:"desired_weight"`            //期望体重
	TimeFrame               int    `json:"time_frame"`                //期望多快达到
	PreferredBodyPart       string `json:"preferred_body_part"`       //最期望增强部位
	WeeklyExerciseFrequency int    `json:"weekly_exercise_frequency"` //每周运动次数
	PreferredPriceRange     int    `json:"preferred_price_range"`     //偏好价格档位
	PreferredLocationID     int    `json:"preferred_location_id"`     //偏好健身房场地ID
	VipType                 string `json:"vip_type"`                  //vip订阅类型 0=非会员 1=体验会员 2=付费年费会员
	VipExpiredTs            string `json:"vip_expired_ts"`            //vip过期时间
	RegistTs                string `json:"regist_ts"`                 //用户注册时间
	BeVipTs                 string `json:"be_vip_ts"`                 //付费成为订阅会员的时间
	TrialPackageReaminCnt   int    `json:"trial_package_reamin_cnt"`  //体验课，剩余课时数
	TrialPackageLevel       string `json:"trial_package_level"`       //体验课，档位
	TrialCoachId            int    `json:"trial_coach_id"`            //体验课，教练
	BuyPackage              bool   `json:"buy_package"`               //是否买了正式课
}

type GetUserStatiticRsp struct {
	Code                     int                 `json:"code"`
	ErrorMsg                 string              `json:"errorMsg,omitempty"`
	TotalUsers               int                 `json:"total_users"`                // 总注册人数
	TotalSubscriptions       int                 `json:"total_subscriptions"`        // 总订阅数
	TotalSubscriptionRevenue int                 `json:"total_subscription_revenue"` // 订阅支付总金额
	UnsubscribedUsers        int                 `json:"unsubscribed_users"`         // 注册但未订阅用户数
	NewUsersToday            int                 `json:"new_users_today"`            // 今日新增注册数
	NewSubscriptionsToday    int                 `json:"new_subscriptions_today"`    // 今日新增订阅数
	UserStatisticInfoList    []UserStatisticInfo `json:"user_statistic_info_list"`   // 用户统计信息列表
}

func getGetUserStatisticReq(r *http.Request) (GetUserStatisticReq, error) {
	req := GetUserStatisticReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetUserStatiticHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetUserStatisticReq(r)
	rsp := &GetUserStatiticRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetUserStatiticHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	var dayBegTs int64
	if len(req.StatisticTs) == 0 {
		dayBegTs = comm.GetTodayBegTsByTs(time.Now().Unix())
	} else {
		t, _ := time.Parse("20060102", req.StatisticTs)
		dayBegTs = comm.GetTodayBegTsByTs(t.Unix())
	}

	vecAllUserModel, err := dao.ImpUser.GetAllUser()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		Printf("GetAllUser err, StatisticTs:%d err:%+v\n", req.StatisticTs, err)
		return
	}
	for _, v := range vecAllUserModel {
		phone := v.PhoneNumber
		if phone != nil && len(*phone) > 0 {
			rsp.TotalUsers += 1
		}

		if v.BeVipTs > 0 {
			rsp.TotalSubscriptions += 1
		}

		if v.VipType == model.Enum_VipType_PaidYear {
			rsp.TotalSubscriptionRevenue += 299
		}

		if v.RegistTs >= dayBegTs {
			rsp.NewUsersToday += 1
		}

		if v.RegistTs >= dayBegTs && v.BeVipTs > 0 {
			rsp.NewSubscriptionsToday += 1
		}
	}

	rsp.UnsubscribedUsers = rsp.TotalUsers - rsp.TotalSubscriptions

	//处理单条记录信息
	for _, v := range vecAllUserModel {
		stTrailCoursePackage, err := dao.ImpCoursePackage.GetTrailCoursePackage(v.UserID)
		if err != nil {
			continue
		}
		vecPayCoursePackageModel, err := dao.ImpCoursePackage.GetPayCoursePackageList(v.UserID)
		if err != nil {
			continue
		}
		rspItem := convertUser2SUser(v)
		rspItem.TrialPackageReaminCnt = stTrailCoursePackage.RemainCnt
		if stTrailCoursePackage.CourseId == 4 {
			rspItem.TrialPackageLevel = "基础体验课"
		} else if stTrailCoursePackage.CourseId == 5 {
			rspItem.TrialPackageLevel = "中级体验课"
		} else if stTrailCoursePackage.CourseId == 6 {
			rspItem.TrialPackageLevel = "高级体验课"
		}
		rspItem.TrialCoachId = stTrailCoursePackage.CoachId
		if len(vecPayCoursePackageModel) > 0 {
			rspItem.BuyPackage = true
		} else {
			rspItem.BuyPackage = false
		}
		rsp.UserStatisticInfoList = append(rsp.UserStatisticInfoList, rspItem)
	}
	return
}

func convertUser2SUser(user model.UserInfoModel) UserStatisticInfo {
	var rsp UserStatisticInfo
	rsp.UserID = user.UserID
	rsp.WechatID = user.WechatID
	if user.PhoneNumber != nil {
		rsp.PhoneNumber = *user.PhoneNumber
	}
	rsp.Nick = user.Nick
	rsp.HeadPic = user.HeadPic
	if user.Gender == 0 {
		rsp.Gender = "男"
	} else {
		rsp.Gender = "女"
	}
	rsp.Age = user.Age
	rsp.Weight = user.Weight
	rsp.Height = user.Height
	rsp.FitnessExperience = user.FitnessExperience
	rsp.FitnessGoal = user.FitnessGoal
	rsp.DesiredWeight = user.DesiredWeight
	rsp.TimeFrame = user.TimeFrame
	rsp.PreferredBodyPart = user.PreferredBodyPart
	rsp.WeeklyExerciseFrequency = user.WeeklyExerciseFrequency
	rsp.PreferredPriceRange = user.PreferredPriceRange
	rsp.PreferredLocationID = user.PreferredLocationID

	if user.VipType == 0 {
		rsp.VipType = "非会员"
	} else if user.VipType == 1 {
		rsp.VipType = "体验会员"
	} else {
		rsp.VipType = "付费年费会员"
	}

	t := time.Unix(user.VipExpiredTs, 0)
	rsp.VipExpiredTs = t.Format("2006年01月02日 15:04")

	t = time.Unix(user.RegistTs, 0)
	rsp.RegistTs = t.Format("2006年01月02日 15:04")

	t = time.Unix(user.BeVipTs, 0)
	rsp.BeVipTs = t.Format("2006年01月02日 15:04")

	return rsp
}
