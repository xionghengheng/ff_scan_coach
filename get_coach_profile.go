package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

type GetCoachProfileReq struct {
	// 无需参数，获取全量教练画像
}

// CoachProfileItem 教练画像数据
type CoachProfileItem struct {
	CoachID               int    `json:"coach_id"`                  // 教练ID
	CoachName             string `json:"coach_name"`                // 姓名
	Gender                string `json:"gender"`                    // 性别
	GoodAt                string `json:"good_at"`                   // 擅长内容
	DealUserCount         int    `json:"deal_user_count"`           // 成交用户数
	PaidConversionRate    string `json:"paid_conversion_rate"`      // 付费转化率（暂时可空）
	SecondRenewalRate     string `json:"second_renewal_rate"`       // 二次续费率
	ThirdRenewalRate      string `json:"third_renewal_rate"`        // 三次续费率
	TotalCommentCount     int    `json:"total_comment_count"`       // 用户累计评价数
	MonthLessonCount      int    `json:"month_lesson_count"`        // 近1个月付费课包的消课量
	MonthSalesRevenue     int    `json:"month_sales_revenue"`       // 近1个月付费课包的销售额
	ActiveStatus          string `json:"active_status"`             // 教练活跃状态
	Last30DaysLessonCount int    `json:"last_30_days_lesson_count"` // 近30天核销课程数
	Last60DaysLessonCount int    `json:"last_60_days_lesson_count"` // 近60天核销课程数
}

type GetCoachProfileRsp struct {
	Code            int                `json:"code"`
	ErrorMsg        string             `json:"errorMsg,omitempty"`
	VecCoachProfile []CoachProfileItem `json:"coach_profile_list"` // 教练画像列表
	TotalCount      int                `json:"total_count"`        // 教练总数
}

func getGetCoachProfileReq(r *http.Request) (GetCoachProfileReq, error) {
	req := GetCoachProfileReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// buildCoachProfile 构建单个教练的画像数据
func buildCoachProfile(
	coach model.CoachModel,
	mapAllUserModel map[int64]model.UserInfoModel,
	vecAllPackageModel []model.CoursePackageModel,
	vecAllSingleLesson []model.CoursePackageSingleLessonModel,
	last30DaysBegTs int64,
	last60DaysBegTs int64,
	monthBegTs int64,
) CoachProfileItem {
	profile := CoachProfileItem{
		CoachID:            coach.CoachID,
		CoachName:          coach.CoachName,
		GoodAt:             coach.GoodAt,
		PaidConversionRate: "", // 暂时可空
	}

	// 性别（从绑定的用户信息中获取）
	for _, user := range mapAllUserModel {
		if user.CoachId == coach.CoachID {
			if user.Gender == 0 {
				profile.Gender = "男"
			} else if user.Gender == 1 {
				profile.Gender = "女"
			} else {
				profile.Gender = "未知"
			}
			break
		}
	}

	// 统计课包相关数据
	profile.DealUserCount, profile.MonthSalesRevenue, profile.SecondRenewalRate, profile.ThirdRenewalRate =
		calculatePackageStats(coach.CoachID, vecAllPackageModel, monthBegTs)

	// 统计课程相关数据
	profile.TotalCommentCount, profile.MonthLessonCount, profile.Last30DaysLessonCount, profile.Last60DaysLessonCount =
		calculateLessonStats(coach.CoachID, vecAllSingleLesson, last30DaysBegTs, last60DaysBegTs, monthBegTs)

	// 判断教练活跃状态
	profile.ActiveStatus = calculateActiveStatus(profile.Last30DaysLessonCount, profile.Last60DaysLessonCount)

	return profile
}

// calculatePackageStats 计算课包相关统计数据
func calculatePackageStats(
	coachID int,
	vecAllPackageModel []model.CoursePackageModel,
	monthBegTs int64,
) (dealUserCount int, monthSalesRevenue int, secondRenewalRate string, thirdRenewalRate string) {
	// 统计成交用户数（购买付费课包的用户数）
	mapDealUser := make(map[int64]bool)
	// 统计用户购买次数（用于计算续费率）
	mapUserBuyCount := make(map[int64]int)

	for _, pkg := range vecAllPackageModel {
		if pkg.CoachId != coachID {
			continue
		}

		// 只统计付费课包
		if pkg.PackageType == model.Enum_PackageType_PaidPackage {
			mapDealUser[pkg.Uid] = true
			mapUserBuyCount[pkg.Uid]++

			// 统计近1个月的销售额
			if pkg.Ts >= monthBegTs {
				monthSalesRevenue += pkg.Price
			}
		}
	}

	dealUserCount = len(mapDealUser)

	// 计算二次续费率和三次续费率
	var secondRenewalCount int
	var thirdRenewalCount int
	for _, count := range mapUserBuyCount {
		if count >= 2 {
			secondRenewalCount++
		}
		if count >= 3 {
			thirdRenewalCount++
		}
	}

	if dealUserCount > 0 {
		secondRate := float64(secondRenewalCount) / float64(dealUserCount) * 100
		thirdRate := float64(thirdRenewalCount) / float64(dealUserCount) * 100
		secondRenewalRate = fmt.Sprintf("%.2f%%", secondRate)
		thirdRenewalRate = fmt.Sprintf("%.2f%%", thirdRate)
	} else {
		secondRenewalRate = "0.00%"
		thirdRenewalRate = "0.00%"
	}

	return
}

// calculateLessonStats 计算课程相关统计数据
func calculateLessonStats(
	coachID int,
	vecAllSingleLesson []model.CoursePackageSingleLessonModel,
	last30DaysBegTs int64,
	last60DaysBegTs int64,
	monthBegTs int64,
) (totalCommentCount int, monthLessonCount int, last30DaysLessonCount int, last60DaysLessonCount int) {
	for _, lesson := range vecAllSingleLesson {
		if lesson.CoachId != coachID {
			continue
		}

		// 统计评价数
		if len(lesson.CommentContent) > 0 {
			totalCommentCount++
		}

		// 只统计已完成的课程
		if lesson.Status == model.En_LessonStatusCompleted {
			// 统计近30天核销课程数
			if lesson.WriteOffTs >= last30DaysBegTs {
				last30DaysLessonCount++
			}

			// 统计近60天核销课程数
			if lesson.WriteOffTs >= last60DaysBegTs {
				last60DaysLessonCount++
			}

			// 统计近1个月付费课包的消课量
			_, _, packageType := comm.ParseCoursePackageId(lesson.PackageID)
			if packageType == model.Enum_PackageType_PaidPackage && lesson.WriteOffTs >= monthBegTs {
				monthLessonCount++
			}
		}
	}

	return
}

// calculateActiveStatus 计算教练活跃状态
func calculateActiveStatus(last30DaysLessonCount int, last60DaysLessonCount int) string {
	// 活跃：近30天核销课程≥10节
	if last30DaysLessonCount >= 10 {
		return "活跃"
	}
	// 非活跃：30-60天核销课程<10节
	if last60DaysLessonCount < 10 {
		return "非活跃"
	}
	return "一般"
}

// GetCoachProfileHandler 获取全量教练画像数据
func GetCoachProfileHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getGetCoachProfileReq(r)
	rsp := &GetCoachProfileRsp{}

	Printf("GetCoachProfileHandler start, req:%+v\n", req)

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
		Printf("GetCoachProfileHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("GetCoachProfileHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("GetCoachProfileHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		return
	}

	// 获取所有教练信息
	mapCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = "获取教练信息失败"
		Printf("GetCoachProfileHandler GetAllCoach err, err:%+v\n", err)
		return
	}

	// 获取所有用户信息
	mapAllUserModel, err := comm.GetAllUser()
	if err != nil {
		rsp.Code = -933
		rsp.ErrorMsg = "获取用户信息失败"
		Printf("GetCoachProfileHandler GetAllUser err, err:%+v\n", err)
		return
	}

	// 计算时间戳
	nowTs := time.Now().Unix()
	last30DaysBegTs := nowTs - 30*24*3600       // 近30天开始时间
	last60DaysBegTs := nowTs - 60*24*3600       // 近60天开始时间
	monthBegTs := comm.GetMonthBegTsByTs(nowTs) // 本月开始时间

	// 获取所有课包数据
	var vecAllPackageModel []model.CoursePackageModel
	var turnPageTs int64
	for i := 0; i <= 5000; i++ {
		tmpVecAllPackageModel, err := dao.ImpCoursePackage.GetAllCoursePackageList(turnPageTs)
		if err != nil {
			rsp.Code = -944
			rsp.ErrorMsg = "获取课包信息失败"
			Printf("GetCoachProfileHandler GetAllCoursePackageList err, i:%d err:%+v\n", i, err)
			return
		}
		if len(tmpVecAllPackageModel) == 0 {
			Printf("GetAllCoursePackageList empty, i:%d vecAllPackageModel.len:%d\n", i, len(vecAllPackageModel))
			break
		}
		turnPageTs = tmpVecAllPackageModel[len(tmpVecAllPackageModel)-1].Ts
		vecAllPackageModel = append(vecAllPackageModel, tmpVecAllPackageModel...)
	}

	// 获取所有单节课程数据
	var vecAllSingleLesson []model.CoursePackageSingleLessonModel
	turnPageTs = 0
	for i := 0; i <= 5000; i++ {
		tmpVecAllSingleLesson, err := dao.ImpCoursePackageSingleLesson.GetAllSingleLessonList(turnPageTs)
		if err != nil {
			rsp.Code = -955
			rsp.ErrorMsg = "获取课程信息失败"
			Printf("GetCoachProfileHandler GetAllSingleLessonList err, i:%d err:%+v\n", i, err)
			return
		}
		if len(tmpVecAllSingleLesson) == 0 {
			Printf("GetAllSingleLessonList empty, i:%d vecAllSingleLesson.len:%d\n", i, len(vecAllSingleLesson))
			break
		}
		turnPageTs = tmpVecAllSingleLesson[len(tmpVecAllSingleLesson)-1].CreateTs
		vecAllSingleLesson = append(vecAllSingleLesson, tmpVecAllSingleLesson...)
	}

	// 构建教练画像数据
	for _, coach := range mapCoach {
		// 跳过测试教练
		if coach.BTestCoach {
			continue
		}

		profile := buildCoachProfile(
			coach,
			mapAllUserModel,
			vecAllPackageModel,
			vecAllSingleLesson,
			last30DaysBegTs,
			last60DaysBegTs,
			monthBegTs,
		)

		rsp.VecCoachProfile = append(rsp.VecCoachProfile, profile)
	}

	// 按照教练ID排序
	sort.Slice(rsp.VecCoachProfile, func(i, j int) bool {
		return rsp.VecCoachProfile[i].CoachID < rsp.VecCoachProfile[j].CoachID
	})

	rsp.TotalCount = len(rsp.VecCoachProfile)
	rsp.Code = 0
	Printf("GetCoachProfileHandler success, total_count:%d\n", rsp.TotalCount)
}
