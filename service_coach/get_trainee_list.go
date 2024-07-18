package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

type GetTraineeListReq struct {
	TraineeType int `json:"trainee_type"` //学员类型（En_TraineeType）
	OrderType   int `json:"order_type"`   //排序类型（En_OrderType）
}

type GetTraineeListRsp struct {
	Code           int               `json:"code"`
	ErrorMsg       string            `json:"errorMsg,omitempty"`
	VecTraineeList []TraineeListItem `json:"vec_trainee_list,omitempty"` //学员列表
}

const (
	En_GetTraineeList_TraineeType_All   int = iota + 1 // 全部
	En_GetTraineeList_TraineeType_Trail                // 体验会员
	En_GetTraineeList_TraineeType_Pay                  // 正式会员
)

const (
	En_GetTraineeList_OrderType_Default               int = iota + 1 // 默认(剩余课数越少，排越靠前)
	En_GetTraineeList_OrderType_RemainCntAsc                         // 剩余课量升序
	En_GetTraineeList_OrderType_RemainCntDesc                        // 剩余课量降序
	En_GetTraineeList_OrderType_WithoutLessonDaysAsc                 // 未上课天数升序
	En_GetTraineeList_OrderType_WithoutLessonDaysDesc                // 未上课天数降序
)

// TraineeListItem 学员item
type TraineeListItem struct {
	Uid               int64  `json:"uid"`                 //学员的用户id
	Name              string `json:"name"`                //名字
	HeadPic           string `json:"head_pic"`            //头像
	WithoutLessonDays int    `json:"without_lesson_days"` //未上课时间，单位天
	TraineeType       int    `json:"trainee_type"`        //学员类型，体验or正式（En_TraineeType）
	RemainCnt         int    `json:"remain_cnt"`          //未上课天数
}

func getGetTraineeListReq(r *http.Request) (GetTraineeListReq, error) {
	req := GetTraineeListReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetTraineeListHandler 拉取学员列表
func GetTraineeListHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetTraineeListReq(r)
	rsp := &GetTraineeListRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetTraineeListHandler start, openid:%s\n", strOpenId)

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

	if len(strOpenId) == 0 || req.TraineeType == 0 || req.OrderType == 0 {
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

	//【1】先收集候选uid
	var vecCoursePackageModel []model.CoursePackageModel
	if req.TraineeType == En_GetTraineeList_TraineeType_All {
		vecCoursePackageModel, err = dao.ImpCoursePackage.GetAllCoursePackageListByCoachId(coachId, 100)
		if err != nil {
			rsp.Code = -888
			rsp.ErrorMsg = err.Error()
			return
		}
	} else if req.TraineeType == En_GetTraineeList_TraineeType_Trail {
		vecCoursePackageModel, err = dao.ImpCoursePackage.GetTrailCoursePackageListByCoachId(coachId, 100)
		if err != nil {
			rsp.Code = -777
			rsp.ErrorMsg = err.Error()
			return
		}
	} else if req.TraineeType == En_GetTraineeList_TraineeType_Pay {
		vecCoursePackageModel, err = dao.ImpCoursePackage.GetPayCoursePackageListByCoachId(coachId, 100)
		if err != nil {
			rsp.Code = -666
			rsp.ErrorMsg = err.Error()
			return
		}
	}

	//发起合并，uid相同的数据直接合并成一个（合并原则：剩余次数累加、未上课时间戳取最大的、付费会员覆盖体验会员）
	vecMergePackageModel := mergeSameUid(vecCoursePackageModel)

	//排序
	if req.OrderType == En_GetTraineeList_OrderType_Default || req.OrderType == En_GetTraineeList_OrderType_RemainCntAsc {
		// 按照 RemainCnt 从小到大排序，确保最先开始的课程在最前面
		sort.Slice(vecMergePackageModel, func(i, j int) bool {
			return vecMergePackageModel[i].RemainCnt < vecMergePackageModel[j].RemainCnt
		})
	} else if req.OrderType == En_GetTraineeList_OrderType_RemainCntDesc {
		sort.Slice(vecMergePackageModel, func(i, j int) bool {
			return vecMergePackageModel[i].RemainCnt > vecMergePackageModel[j].RemainCnt
		})
	} else if req.OrderType == En_GetTraineeList_OrderType_WithoutLessonDaysAsc {
		sort.Slice(vecMergePackageModel, func(i, j int) bool {
			return vecMergePackageModel[i].LastLessonTs < vecMergePackageModel[j].LastLessonTs
		})
	} else if req.OrderType == En_GetTraineeList_OrderType_WithoutLessonDaysDesc {
		sort.Slice(vecMergePackageModel, func(i, j int) bool {
			return vecMergePackageModel[i].LastLessonTs > vecMergePackageModel[j].LastLessonTs
		})
	}

	var vecAllUid []int64
	mapUser := make(map[int64]model.UserInfoModel)
	for _, v := range vecMergePackageModel {
		vecAllUid = append(vecAllUid, v.Uid)
	}
	if len(vecAllUid) > 0 {
		mapUser, err = GetAllUser(vecAllUid)
		if err != nil {
			rsp.Code = -930
			rsp.ErrorMsg = err.Error()
			return
		}
	}

	for _, v := range vecMergePackageModel {
		var stRspTraineeListItem TraineeListItem
		stRspTraineeListItem.Uid = v.Uid
		if userInfo,ok :=mapUser[v.Uid];ok{
			stRspTraineeListItem.Name = userInfo.Nick
			stRspTraineeListItem.HeadPic = userInfo.HeadPic
		}else{
			//测试uid，可能注销了但是没有清理课包等其他数据
			continue
		}

		stRspTraineeListItem.WithoutLessonDays = comm.CalculateDaysSinceTimestamp(v.LastLessonTs)
		if v.PackageType == model.Enum_PackageType_PaidPackage {
			stRspTraineeListItem.TraineeType = En_GetTraineeList_TraineeType_Trail
		} else {
			stRspTraineeListItem.TraineeType = En_GetTraineeList_TraineeType_Pay
		}
		stRspTraineeListItem.RemainCnt = v.RemainCnt
		rsp.VecTraineeList = append(rsp.VecTraineeList, stRspTraineeListItem)
	}

}

func mergeSameUid(vecID []model.CoursePackageModel) []model.CoursePackageModel {
	mapRes := make(map[int64]model.CoursePackageModel)
	for _, v := range vecID {
		if res, ok := mapRes[v.Uid]; ok {
			res.RemainCnt += v.RemainCnt
			if v.LastLessonTs > res.LastLessonTs {
				res.LastLessonTs = v.LastLessonTs
			}
			if v.PackageType == model.Enum_PackageType_PaidPackage {
				res.PackageType = model.Enum_PackageType_PaidPackage
			}
			mapRes[v.Uid] = res
		} else {
			mapRes[v.Uid] = v
		}
	}

	var vecRes []model.CoursePackageModel
	for _, v := range mapRes {
		vecRes = append(vecRes, v)
	}
	return vecRes
}
