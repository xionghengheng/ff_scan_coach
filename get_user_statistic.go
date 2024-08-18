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
	StatisticTs int64 `json:"statistic_ts"`
}

type GetUserStatiticRsp struct {
	Code                     int    `json:"code"`
	ErrorMsg                 string `json:"errorMsg,omitempty"`
	TotalUsers               int    `json:"total_users"`                // 总注册人数
	TotalSubscriptions       int    `json:"total_subscriptions"`        // 总订阅数
	TotalSubscriptionRevenue int    `json:"total_subscription_revenue"` // 订阅支付总金额
	UnsubscribedUsers        int    `json:"unsubscribed_users"`         // 注册但未订阅用户数
	NewUsersToday            int    `json:"new_users_today"`            // 今日新增注册数
	NewSubscriptionsToday    int    `json:"new_subscriptions_today"`    // 今日新增订阅数

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
	if req.StatisticTs == 0 {
		dayBegTs = comm.GetTodayBegTsByTs(time.Now().Unix())
	} else {
		dayBegTs = comm.GetTodayBegTsByTs(req.StatisticTs)
	}

	vecAllUserModel, err := dao.ImpUser.GetAllUser()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		Printf("GetAllUser err, strOpenId:%s StatisticTs:%d err:%+v\n", strOpenId, req.StatisticTs, err)
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
	return
}
