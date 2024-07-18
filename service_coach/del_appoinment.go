package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"net/http"
)

// DelAppointmentReq 请求结构
type DelAppointmentReq struct {
	AppointmentID int `json:"appointment_id"` //预约ID
}

type DelAppointmentRsp struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"errorMsg,omitempty"`
}

func getDelAppointmentReq(r *http.Request) (DelAppointmentReq, error) {
	req := DelAppointmentReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// DelAppointmentHandler 删除已经设置可预约的item
func DelAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getDelAppointmentReq(r)
	rsp := &DelAppointmentRsp{}
	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()
	Printf("DelAppointmentHandler start, req:%+v strOpenId:%s\n", req, strOpenId)

	if len(strOpenId) == 0 || req.AppointmentID == 0 {
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
	coachId := stUserInfoModel.CoachId

	stCoachAppointmentModel, err := dao.ImpAppointment.GetAppointmentById(req.AppointmentID)
	if err != nil && err != gorm.ErrRecordNotFound {
		rsp.Code = -933
		rsp.ErrorMsg = err.Error()
		Printf("GetAppointmentById err, err:%+v coachId:%d AppointmentID:%d\n", err, coachId, req.AppointmentID)
		return
	}
	if err == gorm.ErrRecordNotFound || stCoachAppointmentModel == nil{
		err := dao.ImpAppointment.DelAppointmentByCoach(req.AppointmentID, coachId)
		if err != nil {
			rsp.Code = -733
			rsp.ErrorMsg = err.Error()
			Printf("DelAppointmentByCoach err, err:%+v coachId:%d AppointmentID:%d\n", err, coachId, req.AppointmentID)
			return
		}
	}

	//没有被预约可以直接删除
	if stCoachAppointmentModel.UserID == 0{
		err := dao.ImpAppointment.DelAppointmentByCoach(req.AppointmentID, coachId)
		if err != nil {
			rsp.Code = -733
			rsp.ErrorMsg = err.Error()
			Printf("DelAppointmentByCoach err, err:%+v coachId:%d AppointmentID:%d\n", err, coachId, req.AppointmentID)
			return
		}
	}else{
		rsp.Code = -711
		rsp.ErrorMsg = "课程已被预约，无法直接删除"
		Printf("can not del, err:%+v coachId:%d AppointmentID:%d\n", err, coachId, req.AppointmentID)
		return
	}
}
