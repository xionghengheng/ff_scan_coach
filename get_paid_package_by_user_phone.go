package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/jinzhu/gorm"
	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
)

type GetPaidPackageByUserPhoneReq struct {
	PhoneNumber string `json:"phone_number"` // 用户手机号
}

type GetPaidPackageByUserPhoneRsp struct {
	Code               int               `json:"code"`
	ErrorMsg           string            `json:"errorMsg,omitempty"`
	VecPaidPackageItem []PaidPackageItem `json:"vec_paid_package_item"`
}

func getGetPaidPackageByUserPhoneReq(r *http.Request) (GetPaidPackageByUserPhoneReq, error) {
	req := GetPaidPackageByUserPhoneReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetPaidPackageByUserPhoneHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getGetPaidPackageByUserPhoneReq(r)
	rsp := &GetPaidPackageByUserPhoneRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetPaidPackageByUserPhoneHandler start, req:%+v\n", req)

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
		Printf("GetPaidPackageByUserPhoneHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("GetPaidPackageByUserPhoneHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("GetPaidPackageByUserPhoneHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	// 验证手机号参数
	if req.PhoneNumber == "" {
		rsp.Code = -996
		rsp.ErrorMsg = "手机号不能为空"
		Printf("GetPaidPackageByUserPhoneHandler phone_number is empty\n")
		return
	}

	// 根据手机号获取用户信息
	stUserInfoModel, err := dao.ImpUser.GetUserByPhone(req.PhoneNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rsp.Code = -911
			rsp.ErrorMsg = "手机号对应的用户未找到"
		} else {
			rsp.Code = -922
			rsp.ErrorMsg = "通过手机号拉取用户信息失败"
		}
		Printf("GetPaidPackageByUserPhoneHandler GetUserByPhone err, err:%+v PhoneNumber:%s\n", err, req.PhoneNumber)
		return
	}

	// 根据用户ID获取付费课包列表
	vecPayCoursePackageModel, err := dao.ImpCoursePackage.GetPayCoursePackageList(stUserInfoModel.UserID)
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = "获取用户付费课包列表失败"
		Printf("GetPaidPackageByUserPhoneHandler GetPayCoursePackageList err, err:%+v uid:%d\n", err, stUserInfoModel.UserID)
		return
	}

	// 如果没有付费课包，直接返回空列表
	if len(vecPayCoursePackageModel) == 0 {
		rsp.Code = 0
		rsp.VecPaidPackageItem = []PaidPackageItem{}
		Printf("GetPaidPackageByUserPhoneHandler no paid package found, uid:%d phone:%s\n", stUserInfoModel.UserID, req.PhoneNumber)
		return
	}

	// 获取所有教练信息
	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -944
		rsp.ErrorMsg = "获取教练信息失败"
		Printf("GetPaidPackageByUserPhoneHandler GetAllCoach err, err:%+v\n", err)
		return
	}

	// 获取所有课程信息
	mapALlCourseModel, err := comm.GetAllCourse()
	if err != nil {
		rsp.Code = -955
		rsp.ErrorMsg = "获取课程信息失败"
		Printf("GetPaidPackageByUserPhoneHandler GetAllCourse err, err:%+v\n", err)
		return
	}

	// 获取所有用户信息
	mapAllUserModel, err := comm.GetAllUser()
	if err != nil {
		rsp.Code = -966
		rsp.ErrorMsg = "获取用户信息失败"
		Printf("GetPaidPackageByUserPhoneHandler GetAllUser err, err:%+v\n", err)
		return
	}

	// 获取所有场地信息
	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -977
		rsp.ErrorMsg = "获取场地信息失败"
		Printf("GetPaidPackageByUserPhoneHandler GetAllGym err, err:%+v\n", err)
		return
	}

	// 转换数据格式
	for _, v := range vecPayCoursePackageModel {

		vecPaymentOrderModel, err := dao.ImpPaymentOrder.GetOrderByPackageId(v.Uid, v.PackageID)
		if err != nil || len(vecPaymentOrderModel) == 0 {
			Printf("GetPaidPackageByUserPhoneHandler GetUserByPhone err, err:%+v PhoneNumber:%s\n", err, req.PhoneNumber)
			continue
		}

		item := ConvertPackageItemModel2PaidRspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym)
		item.WeixinPayOrderId = vecPaymentOrderModel[0].OrderID
		item.PayPice = vecPaymentOrderModel[0].Price
		rsp.VecPaidPackageItem = append(rsp.VecPaidPackageItem, item)
	}

	rsp.Code = 0
	Printf("GetPaidPackageByUserPhoneHandler success, uid:%d phone:%s package_count:%d\n", stUserInfoModel.UserID, req.PhoneNumber, len(rsp.VecPaidPackageItem))
	return
}
