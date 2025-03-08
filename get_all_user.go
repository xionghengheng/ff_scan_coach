package main

import (
	"encoding/json"
	"fmt"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"net/http"
)

type GetAllUserReq struct {
	Type int `json:"type"` // type=0 默认全量用户； type=1 订阅用户
}

type GetAllUserRsp struct {
	Code        int    `json:"code"`
	ErrorMsg    string `json:"errorMsg,omitempty"`
	VecUserItem []UserItem
}

type UserItem struct {
	UserID                  int64   `json:"user_id"`                          //用户uid
	WechatID                string  `json:"wechat_id"`                        //微信openid
	PhoneNumber             *string `json:"phone_number" gorm:"default:null"` //手机号
	Nick                    string  `json:"nick,omitempty"`                   //昵称
	HeadPic                 string  `json:"head_pic"`                         //头像
	Gender                  int     `json:"gender"`                           //"0=男", "1=女", "2=other"
	Age                     int     `json:"age"`                              //年龄
	Weight                  int     `json:"weight"`                           //体重
	Height                  int     `json:"height"`                           //身高
	FitnessExperience       int     `json:"fitness_experience"`               //健身经验（初级=1，中级=2，高级=3）
	FitnessGoal             int     `json:"fitness_goal"`                     //健身目标
	DesiredWeight           int     `json:"desired_weight"`                   //期望体重
	TimeFrame               int     `json:"time_frame"`                       //期望多快达到
	PreferredBodyPart       string  `json:"preferred_body_part"`              //最期望增强部位
	WeeklyExerciseFrequency int     `json:"weekly_exercise_frequency"`        //每周运动次数（频次1~2次/周=1，频次3~4次/周=2，频次5~7次/周=3）
	PreferredPriceRange     int     `json:"preferred_price_range"`            //偏好价格档位(对应的体验课程id)
	PreferredLocationID     int     `json:"preferred_location_id"`            //偏好健身房场地ID
	VipType                 int     `json:"vip_type"`                         //vip订阅类型 0=非会员 1=体验会员 2=付费年费会员
	VipExpiredTs            int64   `json:"vip_expired_ts"`                   //vip过期时间
	IsCoach                 bool    `json:"is_coach"`                         //是否教练
	CoachId                 int     `json:"coach_id"`                         //如果是教练，关联的教练id
	HeadPicSafeStatus       int     `json:"head_pic_safe_status"`             //头像审核结果(参考 Enum_HeadPic_Check)
	HeadPicWaitSafe         string  `json:"head_pic_wait_safe"`               //等待审核的头像
	HeadPicSafeTraceId      string  `json:"head_pic_safe_trace_id"`           //等待审核的traceid，用户和异步回调匹配
	RegistTs                int64   `json:"regist_ts"`                        //用户注册时间
	BeVipTs                 int64   `json:"be_vip_ts"`                        //成为订阅会员的时间
	LastLoginTs             int64   `json:"last_login_ts"`                    //上次登录时间（目前只记录教练的）
}

func getGetAllUserReq(r *http.Request) (GetAllUserReq, error) {
	req := GetAllUserReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

func GetAllUserWithBindPhoneHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetAllUserReq(r)
	rsp := &GetAllUserRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetAllUserWithBindPhoneHandler start, openid:%s req:%+v\n", strOpenId, req)

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

	vecUserInfoModel, err := dao.ImpUser.GetAllUser()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		Printf("GetPageReport err, err:%+v\n", err)
		return
	}

	if req.Type == 1 {
		for _, v := range vecUserInfoModel {
			if v.BeVipTs > 0 && v.PhoneNumber != nil && len(*v.PhoneNumber) > 0 {
				rsp.VecUserItem = append(rsp.VecUserItem, ConvertUserItemModelToRspItem(v))
			}
		}
	} else {
		for _, v := range vecUserInfoModel {
			if v.PhoneNumber != nil && len(*v.PhoneNumber) > 0 {
				rsp.VecUserItem = append(rsp.VecUserItem, ConvertUserItemModelToRspItem(v))
			}
		}
	}
	return
}

// 转换函数
func ConvertUserItemModelToRspItem(item model.UserInfoModel) UserItem {
	var phoneNumber *string
	if item.PhoneNumber != nil {
		// 深拷贝手机号指针内容
		val := *item.PhoneNumber
		phoneNumber = &val
	}

	return UserItem{
		UserID:                  item.UserID,
		WechatID:                item.WechatID,
		PhoneNumber:             phoneNumber,
		Nick:                    item.Nick,
		HeadPic:                 item.HeadPic,
		Gender:                  item.Gender,
		Age:                     item.Age,
		Weight:                  item.Weight,
		Height:                  item.Height,
		FitnessExperience:       item.FitnessExperience,
		FitnessGoal:             item.FitnessGoal,
		DesiredWeight:           item.DesiredWeight,
		TimeFrame:               item.TimeFrame,
		PreferredBodyPart:       item.PreferredBodyPart,
		WeeklyExerciseFrequency: item.WeeklyExerciseFrequency,
		PreferredPriceRange:     item.PreferredPriceRange,
		PreferredLocationID:     item.PreferredLocationID,
		VipType:                 item.VipType,
		VipExpiredTs:            item.VipExpiredTs,
		IsCoach:                 item.IsCoach,
		CoachId:                 item.CoachId,
		HeadPicSafeStatus:       item.HeadPicSafeStatus,
		HeadPicWaitSafe:         item.HeadPicWaitSafe,
		HeadPicSafeTraceId:      item.HeadPicSafeTraceId,
		RegistTs:                item.RegistTs,
		BeVipTs:                 item.BeVipTs,
		LastLoginTs:             item.LastLoginTs,
	}
}
