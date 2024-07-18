package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"net/http"
)

type GetTraineeCommentListReq struct {
}

type GetTraineeCommentListRsp struct {
	Code       int           `json:"code"`
	ErrorMsg   string        `json:"errorMsg,omitempty"`
	VecComment []CommentItem `json:"vec_comment,omitempty"` //评价列表
}

// CommentItem 学员评价item
type CommentItem struct {
	Uid        int64         `json:"uid"`         //学员的用户id
	Name       string        `json:"name"`        //名字
	HeadPic    string        `json:"head_pic"`    //头像
	CourseType int           `json:"course_type"` //1=体验课，2=正式付费课
	Ts         int64         `json:"ts"`          //上课时间
	Comment    CommentDetail `json:"comment"`     //评价具体内容
}

type CommentDetail struct {
	//评论相关内容
	Overall              int    `json:"overall"`                // 整体
	Professional         int    `json:"professional"`           // 专业
	Environment          int    `json:"environment"`            // 环境
	Service              int    `json:"service"`                // 服务
	ContinueAttendLesson int    `json:"continue_attend_lesson"` // 是否愿意继续上课，愿意、待考虑、不愿意
	CommentContent       string `json:"comment_content"`        // 评价内容
	AnonymousComment     bool   `json:"anonymous_comment"`      // 是否匿名评价
	CommentTs            int64  `json:"comment_ts"`             // 提交评价的时间
}

// GetTraineeCommentListHandler 拉取学员评价列表
func GetTraineeCommentListHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	rsp := &GetTraineeCommentListRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetTraineeCommentListHandler start, openid:%s\n", strOpenId)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	if len(strOpenId) == 0 {
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

	vecCoachClientTraineeCommentModel, err := dao.ImpCoachClientTraineeComment.GetTraineeCommentList(coachId, 100)
	if err != nil {
		rsp.Code = -800
		rsp.ErrorMsg = err.Error()
		Printf("GetTraineeCommentList err, coachId:%d err:%+v\n", coachId, err)
		return
	}
	Printf("GetTraineeCommentList succ, coachId:%d vecCoachClientTraineeCommentModel:%+v\n", coachId, vecCoachClientTraineeCommentModel)
	if len(vecCoachClientTraineeCommentModel) == 0 {
		return
	}

	var vecAllUid []int64
	mapUser := make(map[int64]model.UserInfoModel)
	for _, v := range vecCoachClientTraineeCommentModel {
		vecAllUid = append(vecAllUid, v.Uid)
	}
	mapUser, err = GetAllUser(vecAllUid)
	if err != nil {
		rsp.Code = -930
		rsp.ErrorMsg = err.Error()
		return
	}

	for _, v := range vecCoachClientTraineeCommentModel {
		rsp.VecComment = append(rsp.VecComment, transCommentModel2RspCommentItem(v, mapUser))
	}
}

func transCommentModel2RspCommentItem(in model.CoachClientTraineeCommentModel, mapUser map[int64]model.UserInfoModel) CommentItem {
	var stCommentItem CommentItem
	if in.AnonymousComment {
		stCommentItem.Name = "匿名用户"
	} else {
		stCommentItem.Uid = in.Uid
		stCommentItem.Name = mapUser[in.Uid].Nick
		stCommentItem.HeadPic = mapUser[in.Uid].HeadPic
		stCommentItem.Ts = in.ScheduleBegTs
		_, _, packageType := comm.ParseCoursePackageId(in.PackageID)
		stCommentItem.CourseType = packageType
	}

	stCommentItem.Comment.Overall = in.Overall
	stCommentItem.Comment.Professional = in.Professional
	stCommentItem.Comment.Environment = in.Environment
	stCommentItem.Comment.Service = in.Service
	stCommentItem.Comment.ContinueAttendLesson = in.ContinueAttendLesson
	stCommentItem.Comment.CommentContent = in.CommentContent
	stCommentItem.Comment.AnonymousComment = in.AnonymousComment
	if in.AnonymousComment {
		stCommentItem.Comment.CommentTs = 0
	} else {
		stCommentItem.Comment.CommentTs = in.CommentTs
	}
	return stCommentItem
}
