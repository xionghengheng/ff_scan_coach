package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
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
	CoachID   int    `json:"coach_id"`   // 教练ID
	CoachName string `json:"coach_name"` // 姓名
	AvatarUrl string `json:"avatar_url"` // 头像URL
	Gender    string `json:"gender"`     // 性别
	GoodAt    string `json:"good_at"`    // 擅长内容

	// 付费课包相关
	DealUserCount      int    `json:"deal_user_count"`      // 成交用户数
	PaidConversionRate string `json:"paid_conversion_rate"` // 付费转化率（暂时可空）
	SecondRenewalRate  string `json:"second_renewal_rate"`  // 二次续费率
	ThirdRenewalRate   string `json:"third_renewal_rate"`   // 三次续费率
	MonthLessonCount   int    `json:"month_lesson_count"`   // 近1个月付费课包的消课量

	// 单次课程相关
	TotalCommentCount     int    `json:"total_comment_count"`       // 用户累计评价数
	MonthSalesRevenue     int    `json:"month_sales_revenue"`       // 近1个月付费课包的销售额
	ActiveStatus          string `json:"active_status"`             // 教练活跃状态
	Last30DaysLessonCount int    `json:"last_30_days_lesson_count"` // 近30天核销课程数

	// 近30天新增指标、超时率、改课率、训练总结率
	Last30DaysOvertimeCount   int    `json:"last_30_days_overtime_count"`   // 近30天超时核销数
	OvertimeRate              string `json:"overtime_rate"`                 // 超时率（超时核销课数/总课程数）
	Last30DaysRescheduleCount int    `json:"last_30_days_reschedule_count"` // 近30天系统改课数
	RescheduleRate            string `json:"reschedule_rate"`               // 改课率（改课次数/预约课程总数）
	Last30DaysSummaryCount    int    `json:"last_30_days_summary_count"`    // 近30天训练总结数
	SummaryRate               string `json:"summary_rate"`                  // 训练总结率（填写训练总结数/近30天实际核销上课数）
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

// 获取全量教练画像数据
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

	// 计算时间戳
	nowTs := time.Now().Unix()
	last30DaysBegTs := nowTs - 30*24*3600       // 近30天开始时间
	monthBegTs := comm.GetMonthBegTsByTs(nowTs) // 本月开始时间

	// 并发获取所有数据
	var wg sync.WaitGroup
	var mapCoach map[int]model.CoachModel
	var mapAllUserModel map[int64]model.UserInfoModel
	var vecAllPackageModel []model.CoursePackageModel
	var vecAllSingleLesson []model.CoursePackageSingleLessonModel
	var errCoach, errUser, errPackage, errLesson error

	// 获取所有教练信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		mapCoach, errCoach = comm.GetAllCoach()
	}()

	// 获取所有用户信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		mapAllUserModel, errUser = comm.GetAllUser()
	}()

	// 获取所有课包数据
	wg.Add(1)
	go func() {
		defer wg.Done()
		var turnPageTs int64
		for i := 0; i <= 5000; i++ {
			tmpVecAllPackageModel, err := dao.ImpCoursePackage.GetAllCoursePackageList(turnPageTs)
			if err != nil {
				errPackage = err
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
	}()

	// 获取所有单节课程数据
	wg.Add(1)
	go func() {
		defer wg.Done()
		var turnPageTs int64
		for i := 0; i <= 5000; i++ {
			tmpVecAllSingleLesson, err := dao.ImpCoursePackageSingleLesson.GetAllSingleLessonList(turnPageTs)
			if err != nil {
				errLesson = err
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
	}()

	// 等待所有goroutine完成
	wg.Wait()

	// 检查错误
	if errCoach != nil {
		rsp.Code = -922
		rsp.ErrorMsg = "获取教练信息失败"
		Printf("GetCoachProfileHandler GetAllCoach err, err:%+v\n", errCoach)
		return
	}
	if errUser != nil {
		rsp.Code = -933
		rsp.ErrorMsg = "获取用户信息失败"
		Printf("GetCoachProfileHandler GetAllUser err, err:%+v\n", errUser)
		return
	}
	if errPackage != nil {
		rsp.Code = -944
		rsp.ErrorMsg = "获取课包信息失败"
		return
	}
	if errLesson != nil {
		rsp.Code = -955
		rsp.ErrorMsg = "获取课程信息失败"
		return
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

// 构建单个教练的画像数据
func buildCoachProfile(
	coach model.CoachModel,
	mapAllUserModel map[int64]model.UserInfoModel,
	vecAllPackageModel []model.CoursePackageModel,
	vecAllSingleLesson []model.CoursePackageSingleLessonModel,
	last30DaysBegTs int64,
	monthBegTs int64,
) CoachProfileItem {
	profile := CoachProfileItem{
		CoachID:            coach.CoachID,
		CoachName:          coach.CoachName,
		AvatarUrl:          comm.ConvertCloudUrlToHttps(coach.Avatar),
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
	lessonStats := calculateLessonStats(coach.CoachID, vecAllSingleLesson, last30DaysBegTs, monthBegTs)
	profile.TotalCommentCount = lessonStats.TotalCommentCount
	profile.MonthLessonCount = lessonStats.MonthLessonCount
	profile.Last30DaysLessonCount = lessonStats.Last30DaysWriteOffLessonCount

	profile.Last30DaysOvertimeCount = lessonStats.Last30DaysOvertimeCount
	profile.OvertimeRate = lessonStats.OvertimeRate
	profile.Last30DaysRescheduleCount = lessonStats.Last30DaysRescheduleCount
	profile.RescheduleRate = lessonStats.RescheduleRate
	profile.Last30DaysSummaryCount = lessonStats.Last30DaysSummaryCount
	profile.SummaryRate = lessonStats.SummaryRate

	// 判断教练活跃状态
	profile.ActiveStatus = calculateActiveStatus(profile.Last30DaysLessonCount)

	return profile
}

// 计算课包相关统计数据
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

// LessonStatsResult 课程统计结果
type LessonStatsResult struct {
	TotalCommentCount             int    // 用户累计评价数（所有有评价内容的课程数）
	MonthLessonCount              int    // 近1个月付费课包的消课量（本月核销的付费课包课程数）
	Last30DaysWriteOffLessonCount int    // 近30天实际核销上课数（已完成状态且核销时间在近30天内）
	Last30DaysOvertimeCount       int    // 近30天超时核销数（核销时间超过预约结束时间2小时以上的课程数）
	OvertimeRate                  string // 超时率 = 超时核销课数 / 近30天总课程数 * 100%
	Last30DaysRescheduleCount     int    // 近30天系统改课数（教练主动取消用户预约的课程数）
	RescheduleRate                string // 改课率 = 改课次数 / 近30天预约课程总数 * 100%
	Last30DaysSummaryCount        int    // 近30天训练总结数（近30天核销课程中填写了训练总结的数量）
	SummaryRate                   string // 训练总结率 = 填写训练总结数 / 近30天实际核销上课数 * 100%
}

// 计算课程相关统计数据
func calculateLessonStats(
	coachID int,
	vecAllSingleLesson []model.CoursePackageSingleLessonModel,
	last30DaysBegTs int64,
	monthBegTs int64,
) LessonStatsResult {
	result := LessonStatsResult{}

	// 近30天预约课程总数（用于计算改课率）
	var last30DaysBookedCount int
	// 近30天总课程数（用于计算超时率）
	var last30DaysTotalLessonCount int

	for _, lesson := range vecAllSingleLesson {
		if lesson.CoachId != coachID {
			continue
		}

		// 统计评价数
		if len(lesson.CommentContent) > 0 {
			result.TotalCommentCount++
		}

		// 近30天的课程统计
		if lesson.CreateTs >= last30DaysBegTs {
			// 统计近30天预约课程总数（用户主动预约的课程）
			last30DaysBookedCount++

			// 统计近30天改课数（教练主动取消用户预约的课程）
			if lesson.Status == model.En_LessonStatusCanceled && lesson.CancelByCoach {
				result.Last30DaysRescheduleCount++
			}
		}

		// 只统计已完成的课程
		if lesson.Status == model.En_LessonStatusCompleted {
			// 统计近30天核销课程数
			if lesson.WriteOffTs >= last30DaysBegTs {
				result.Last30DaysWriteOffLessonCount++
				last30DaysTotalLessonCount++

				// 统计超时核销数（预约时间2小时后核销记为超时）
				// 超时定义：用户具体预约时间的2小时内核销记为正常，超过2小时核销记为超时
				if lesson.ScheduleEndTs > 0 && lesson.WriteOffTs > lesson.ScheduleEndTs+2*3600 {
					result.Last30DaysOvertimeCount++
				}

				// 统计训练总结数（填写了训练总结的课程）
				if len(lesson.TrainContent) > 0 {
					result.Last30DaysSummaryCount++
				}
			}

			// 统计近1个月付费课包的消课量
			_, _, packageType := comm.ParseCoursePackageId(lesson.PackageID)
			if packageType == model.Enum_PackageType_PaidPackage && lesson.WriteOffTs >= monthBegTs {
				result.MonthLessonCount++
			}
		}
	}

	// 计算超时率（超时核销课数/近30天总课程数）
	if last30DaysTotalLessonCount > 0 {
		overtimeRate := float64(result.Last30DaysOvertimeCount) / float64(last30DaysTotalLessonCount) * 100
		result.OvertimeRate = fmt.Sprintf("%.2f%%", overtimeRate)
	} else {
		result.OvertimeRate = "0.00%"
	}

	// 计算改课率（改课次数/预约课程总数）
	if last30DaysBookedCount > 0 {
		rescheduleRate := float64(result.Last30DaysRescheduleCount) / float64(last30DaysBookedCount) * 100
		result.RescheduleRate = fmt.Sprintf("%.2f%%", rescheduleRate)
	} else {
		result.RescheduleRate = "0.00%"
	}

	// 计算训练总结率（填写训练总结数/近30天实际核销上课数）
	if result.Last30DaysWriteOffLessonCount > 0 {
		summaryRate := float64(result.Last30DaysSummaryCount) / float64(result.Last30DaysWriteOffLessonCount) * 100
		result.SummaryRate = fmt.Sprintf("%.2f%%", summaryRate)
	} else {
		result.SummaryRate = "0.00%"
	}

	return result
}

// 计算教练活跃状态
// 教练活跃状态定义：
// 活跃：近 30 天 核销课程≥ 10节
// 非活跃: 近30天 核销课程< 10节
func calculateActiveStatus(last30DaysLessonCount int) string {
	// 活跃：近30天核销课程≥10节
	if last30DaysLessonCount >= 10 {
		return "活跃"
	}
	return "非活跃"
}
