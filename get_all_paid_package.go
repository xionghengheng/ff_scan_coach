package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

type GetAllPaidPackageReq struct {
}

type GetAllPaidPackageRsp struct {
	Code               int    `json:"code"`
	ErrorMsg           string `json:"errorMsg,omitempty"`
	VecPaidPackageItem []PaidPackageItem
}

type PaidPackageItem struct {
	Uid              int64  `json:"uid"`                 // 用户id
	UserName         string `json:"user_name"`           // 用户名称
	PhoneNumber      string `json:"phone_number"`        // 手机号
	PackageID        string `json:"package_id"`          // 课包的唯一标识符（用户id_获取课包的时间戳）
	GymId            int    `json:"gym_id"`              // 场地id
	GymName          string `json:"gym_name"`            // 场地id
	CourseId         int    `json:"course_id"`           // 课程id
	CourseName       string `json:"course_name"`         // 场地名称
	CoachId          int    `json:"coach_id"`            // 教练id
	CoachName        string `json:"coach_name"`          // 教练名称
	Ts               int64  `json:"ts"`                  // 获得课包的时间戳
	TotalCnt         int    `json:"total_cnt"`           // 课包中总的课程次数
	RemainCnt        int    `json:"remain_cnt"`          // 课包中剩余的课程次数
	CoursePrice      int    `json:"course_price"`        // 课程价格（由于价格会变动，使用用户付款时的价格换算单次课价格）
	LastLessonTs     int64  `json:"last_lesson_ts"`      // 上次约课时间
	ChangeCoachTs    int64  `json:"change_coach_ts"`     // 更换教练的时间戳
	RefundTs         int64  `json:"refund_ts"`           // 发生退款的时间
	RefundLessonCnt  int    `json:"refund_lesson_cnt"`   // 退款课程数
	RefundAmount     int    `json:"refund_amount"`       // 退款金额，单位元
	WeixinPayOrderId string `json:"weixin_pay_order_id"` // 微信支付账单id
	PayPrice         int64  `json:"pay_price"`           // 折前价格，单位元
	RealPayPrice     int64  `json:"real_pay_price"`      // 实际支付的价格，单位元
	RenewCnt         int    `json:"renew_cnt"`           // 续费次数
	IsRenew          bool   `json:"is_renew"`            // 是否为续费订单
}

func getGetAllPaidPackageReq(r *http.Request) (GetAllPaidPackageReq, error) {
	req := GetAllPaidPackageReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetAllPaidPackageHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetAllPaidPackageReq(r)
	rsp := &GetAllPaidPackageRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetAllPaidPackageHandler start, openid:%s req:%+v\n", strOpenId, req)

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
		Printf("GetAllPaidPackageHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("GetAllPaidPackageHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("GetAllPaidPackageHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	var vecAllPaidPackageModel []model.CoursePackageModel
	var turnPageTs int64
	for i := 0; i <= 5000; i++ {
		tmpVecAllTrailPackageModel, err := dao.ImpCoursePackage.GetAllPaidCoursePackageList(turnPageTs)
		if err != nil {
			Printf("GetAllCoursePackageList err, i:%d err:%+v\n", i, err)
			return
		}
		if len(tmpVecAllTrailPackageModel) == 0 {
			Printf("GetAllCoursePackageList empty, i:%d vecAllPackageModel.len:%d\n", i, len(vecAllPaidPackageModel))
			break
		}
		turnPageTs = tmpVecAllTrailPackageModel[len(tmpVecAllTrailPackageModel)-1].Ts
		vecAllPaidPackageModel = append(vecAllPaidPackageModel, tmpVecAllTrailPackageModel...)
	}

	mapAllCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		return
	}

	mapALlCourseModel, err := comm.GetAllCourse()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		return
	}

	mapAllUserModel, err := comm.GetAllUser()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		return
	}

	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		return
	}

	// 获取所有订单，构建PackageID到订单列表的映射（一个课包可能对应多个订单，续课时会新增订单）
	mapPackageId2Orders := make(map[string][]model.PaymentOrderModel)
	var orderTurnPageTs int64
	for i := 0; i <= 5000; i++ {
		vecPaymentOrder, err := dao.ImpPaymentOrder.GetAllOrderList(orderTurnPageTs)
		if err != nil {
			rsp.Code = -912
			rsp.ErrorMsg = err.Error()
			Printf("GetAllOrderList err, i:%d err:%+v\n", i, err)
			return
		}
		if len(vecPaymentOrder) == 0 {
			Printf("GetAllOrderList empty, i:%d mapPackageId2Orders.len:%d\n", i, len(mapPackageId2Orders))
			break
		}
		orderTurnPageTs = vecPaymentOrder[len(vecPaymentOrder)-1].OrderTime
		for _, order := range vecPaymentOrder {
			mapPackageId2Orders[order.PackageID] = append(mapPackageId2Orders[order.PackageID], order)
		}
	}

	for _, v := range vecAllPaidPackageModel {
		if mapAllCoach[v.CoachId].BTestCoach {
			continue
		}
		rsp.VecPaidPackageItem = append(rsp.VecPaidPackageItem, ConvertPackageItemModel2PaidRspItem(v, mapAllCoach, mapALlCourseModel, mapAllUserModel, mapGym, mapPackageId2Orders)...)
	}

	// 按获得时间从大到小排序
	sort.Slice(rsp.VecPaidPackageItem, func(i, j int) bool {
		return rsp.VecPaidPackageItem[i].Ts > rsp.VecPaidPackageItem[j].Ts
	})
}

// ConvertPackageItemModel2PaidRspItem 转换函数，每个订单（含续费）返回一个item
func ConvertPackageItemModel2PaidRspItem(item model.CoursePackageModel,
	mapAllCoach map[int]model.CoachModel,
	mapALlCourseModel map[int]model.CourseModel,
	mapAllUserModel map[int64]model.UserInfoModel,
	mapGym map[int]model.GymInfoModel,
	mapPackageId2Orders map[string][]model.PaymentOrderModel) []PaidPackageItem {

	strPhone := ""
	phone := mapAllUserModel[item.Uid].PhoneNumber
	if phone != nil {
		strPhone = *phone
	}

	var result []PaidPackageItem

	orders, ok := mapPackageId2Orders[item.PackageID]
	if !ok || len(orders) == 0 {
		// 没有订单时返回空
		return result
	}

	// 按订单时间升序排序，确保首次购买在前、续费在后
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].OrderTime < orders[j].OrderTime
	})

	// 计算续费次数（订单数量 - 1）
	renewCnt := 0
	if len(orders) > 1 {
		renewCnt = len(orders) - 1
	}

	// 为每个订单生成一个item
	for i, order := range orders {
		// 是否为续费订单（第一个订单不是续费，时间最早的是首次购买）
		isRenew := i > 0

		// 本次订单的价格
		payPrice := int64(order.Price + order.DiscountAmount)
		realPayPrice := int64(order.Price)

		// 基于本次订单价格换算单次课价格
		coursePrice := 0
		if order.CourseCnt > 0 && payPrice > 0 {
			coursePrice = int(payPrice) / order.CourseCnt
		}

		// 续费订单的Ts用订单时间，首次购买用课包时间
		ts := item.Ts
		if isRenew {
			ts = order.OrderTime
		}

		result = append(result, PaidPackageItem{
			Uid:              item.Uid,
			UserName:         mapAllUserModel[item.Uid].Nick,
			PhoneNumber:      strPhone,
			PackageID:        item.PackageID,
			GymId:            mapGym[item.GymId].GymID,
			GymName:          mapGym[item.GymId].LocName,
			CourseId:         item.CourseId,
			CourseName:       mapALlCourseModel[item.CourseId].Name,
			CoachId:          item.CoachId,
			CoachName:        mapAllCoach[item.CoachId].CoachName,
			Ts:               ts,
			TotalCnt:         item.TotalCnt,
			RemainCnt:        item.RemainCnt,
			CoursePrice:      coursePrice,
			LastLessonTs:     item.LastLessonTs,
			ChangeCoachTs:    item.ChangeCoachTs,
			RefundTs:         item.RefundTs,
			RefundLessonCnt:  item.RefundLessonCnt,
			RefundAmount:     order.RefundAmount / 100,
			WeixinPayOrderId: order.OrderID,
			PayPrice:         payPrice,
			RealPayPrice:     realPayPrice,
			RenewCnt:         renewCnt,
			IsRenew:          isRenew,
			// OrderCourseCnt:   order.CourseCnt,
		})
	}

	return result
}
