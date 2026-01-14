package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type RefundReq struct {
	PayUid          int64  `json:"pay_uid"`           // 退款人uid
	OutTradeNo      string `json:"out_trade_no"`      // 退款人订单号
	RefundCourseCnt int    `json:"refund_course_cnt"` // 已退课程节数
	RefundAmount    int    `json:"refund_amount"`     // 退款金额(单位元)
}

type RefundRsp struct {
	Code     int    `json:"errcode"`
	ErrorMsg string `json:"errmsg,omitempty"`
}

func getRefundReq(r *http.Request) (RefundReq, error) {
	req := RefundReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func RefundPackagePhoneHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getRefundReq(r)
	rsp := &RefundRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("RefundPackagePhoneHandler start, openid:%s req:%+v\n", strOpenId, req)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

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
		Printf("RefundPackagePhoneHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("RefundPackagePhoneHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("RefundPackagePhoneHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	// 验证必填参数
	if req.PayUid <= 0 {
		rsp.Code = -996
		rsp.ErrorMsg = "退款人uid不能为空或无效"
		Printf("RefundPackagePhoneHandler invalid pay_uid:%d\n", req.PayUid)
		return
	}

	if req.OutTradeNo == "" {
		rsp.Code = -996
		rsp.ErrorMsg = "订单号不能为空"
		Printf("RefundPackagePhoneHandler out_trade_no is empty\n")
		return
	}

	if req.RefundAmount <= 0 {
		rsp.Code = -996
		rsp.ErrorMsg = "退款金额必须大于0"
		Printf("RefundPackagePhoneHandler invalid refund_amount:%d\n", req.RefundAmount)
		return
	}

	// TODO: 实现退款逻辑
	// 1. 查询订单信息
	// 2. 验证订单状态
	// 3. 调用微信退款接口
	// 4. 更新数据库状态

	rsp.Code = 0
	Printf("RefundPackagePhoneHandler success, pay_uid:%d out_trade_no:%s refund_amount:%d\n", 
		req.PayUid, req.OutTradeNo, req.RefundAmount)
	return
}
