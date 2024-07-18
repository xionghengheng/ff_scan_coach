package service_coach

import (
	"FunFitnessTrainer/comm"
	"FunFitnessTrainer/db/dao"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

type GetPackageListReq struct {
	TraineeUid int64 `json:"trainee_uid"` //学员uid
}

type GetPackageListRsp struct {
	Code           int           `json:"code"`
	ErrorMsg       string        `json:"errorMsg,omitempty"`
	VecPackageItem []PackageItem `json:"vec_package_item,omitempty"`
}

type PackageItem struct {
	PackageID   string `json:"package_id"`             //课包的唯一标识符（用户id_获取课包的时间戳）
	PackageType int    `json:"package_type"`           //课包类型(1=体验免费课包 2=付费)
	CourseId    int    `json:"course_id,omitempty"`    //课程id
	CourseTitle string `json:"course_title,omitempty"` //课程名称
	CourseImage string `json:"course_image"`           //课程图片
	RemainCnt   int    `json:"remain_cnt"`             //剩余课时数
	ExpireTs    int64  `json:"expire_ts"`              //课包过期时间，暂时没用
}

func getGetPackageListReq(r *http.Request) (GetPackageListReq, error) {
	req := GetPackageListReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetPackageListHandler 拉取某个学员的课包列表
func GetPackageListHandler(w http.ResponseWriter, r *http.Request) {
	strOpenId := r.Header.Get("X-WX-OPENID")
	req, err := getGetPackageListReq(r)
	rsp := &GetPackageListRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetPackageListHandler start, openid:%s\n", strOpenId)

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

	if len(strOpenId) == 0 || req.TraineeUid == 0 {
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

	stTraineeUserInfoModel, err := comm.GetUserInfoByUid(req.TraineeUid)
	if err != nil || stTraineeUserInfoModel == nil {
		rsp.Code = -900
		rsp.ErrorMsg = err.Error()
		Printf("GetUserInfoByUid err, err:%+v coachid:%d uid:%d TraineeUid:%d\n", err, coachId, uid, req.TraineeUid)
		return
	}

	vecCoursePackageModel, err := dao.ImpCoursePackage.GetAllPackageListByCoachIdAndUid(coachId, stTraineeUserInfoModel.UserID)
	if err != nil || stTraineeUserInfoModel == nil {
		rsp.Code = -911
		rsp.ErrorMsg = err.Error()
		Printf("GetAllPackageListByCoachIdAndUid err, err:%+v coachid:%d uid:%d TraineeUid:%d\n", err, coachId, uid, req.TraineeUid)
		return
	}

	// 按照 剩余课时数 从小到大排序
	sort.Slice(vecCoursePackageModel, func(i, j int) bool {
		return vecCoursePackageModel[i].RemainCnt < vecCoursePackageModel[j].RemainCnt
	})

	mapCourse, err := comm.GetAllCouse()
	if err != nil {
		rsp.Code = -950
		rsp.ErrorMsg = err.Error()
		return
	}

	for _, v := range vecCoursePackageModel {
		var stPackageItem PackageItem
		stPackageItem.PackageID = v.PackageID
		stPackageItem.PackageType = v.PackageType
		stPackageItem.CourseId = v.CourseId
		stPackageItem.CourseTitle = mapCourse[v.CourseId].Name
		stPackageItem.CourseImage = mapCourse[v.CourseId].Image
		stPackageItem.RemainCnt = v.RemainCnt
		stPackageItem.ExpireTs = v.Ts + 31536000
		rsp.VecPackageItem = append(rsp.VecPackageItem, stPackageItem)
	}
}
