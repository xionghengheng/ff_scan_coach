package main

import (
	"FunFitnessTrainer/db"
	"FunFitnessTrainer/service"
	"FunFitnessTrainer/service_coach"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	if err := db.Init(); err != nil {
		panic(fmt.Sprintf("mysql init failed with %+v", err))
	}

	autoScanAllCoursePackageSingleLesson()

	//http.HandleFunc("/", service.IndexHandler)
	//http.HandleFunc("/api/count", service.CounterHandler)

	//获取用户信息
	http.HandleFunc("/api/getUserInfo", service.GetUserHandler)

	//更新用户信息
	http.HandleFunc("/api/updateUserInfo", service.UpdateUserInfoHandler)

	//获取推荐的健身房列表
	http.HandleFunc("/api/getGymList", service.GetGymListHandler)
	http.HandleFunc("/api/getCoachList", service.GetCoachListHandler)

	//获取主页
	http.HandleFunc("/api/getHomePage", service.GetHomePageHandler)

	//获取课程列表
	http.HandleFunc("/api/getCourseList", service.GetCourseListHandler)

	//发送手机验证码,绑定手机号
	http.HandleFunc("/api/sendSmsCode", service.SendSmsCodeBindPhoneHandler)

	//验证手机验证码
	http.HandleFunc("/api/verifySmsCode", service.VerifySmsCodeHandler)

	//获取训练计划
	http.HandleFunc("/api/getTrainingPlan", service.GetTrainingPlanHandler)

	//下单
	http.HandleFunc("/api/placeOrder", service.PlaceOrderHandler)

	//支付成功回调
	http.HandleFunc("/api/payResultCallback", service.PayResultCallbackHandler)

	//查询订单状态
	http.HandleFunc("/api/queryOrder", service.QueryOrder)

	//查询订单列表
	http.HandleFunc("/api/getOrderList", service.GetOrderListHandler)

	//取消订单
	http.HandleFunc("/api/cancelOrder", service.CancelOrderHandler)

	//获取某个课包的单节课列表
	http.HandleFunc("/api/getLessonListOfPackage", service.GetLessonListOfPackageHandler)

	http.HandleFunc("/api/updateCoachOrGym2Package", service.UpdateCoachOrGym2PackageHandler)

	//获取某个课包的单节课详情信息
	http.HandleFunc("/api/getLessonDetailOfPackage", service.GetLessonDetailOfPackageHandler)

	//查看某个教练可预约的课程
	http.HandleFunc("/api/getCoachSchedule", service.GetCoachScheduleHandler)

	//预约课程
	http.HandleFunc("/api/bookLesson", service.BookLessonHandler)

	//预约课程
	http.HandleFunc("/api/cancelBookLesson", service.CancelBookLessonHandler)

	//上报各种窗口或者bar的关闭状态
	http.HandleFunc("/api/reportAction", service.ReportActionHandler)

	//邀请码生成和校验
	http.HandleFunc("/api/genInvitationCode", service.GenInvitationCodeHandler)
	http.HandleFunc("/api/checkInvitationCode", service.CheckInvitationCodeHandler)

	//私教页
	http.HandleFunc("/api/getCoachPage", service.GetCoachPageHandler)

	//根据健身房id获取健身房详情
	http.HandleFunc("/api/getGymDetail", service.GetGymDetailHandler)

	//获取教练详情页
	http.HandleFunc("/api/getCoachDetail", service.GetCoachDetailHandler)

	//获取教练购买课程详情页
	http.HandleFunc("/api/getCoachPayOrderDetail", service.GetCoachPayOrderDetailHandler)

	//提交评价
	http.HandleFunc("/api/commitComment", service.CommitCommentHandler)

	//----------------------------教练端接口---------------------------------------//
	//教练端设置可预约课程
	http.HandleFunc("/api/coach/setLessonAvailable", service_coach.SetLessonAvailableHandler)

	//删除教练已设置的可预约item
	http.HandleFunc("/api/coach/delAppointment", service_coach.DelAppointmentHandler)


	//核销课程
	http.HandleFunc("/api/writeOffLesson", service_coach.WriteOffLessonHandler)

	//拉取主页
	http.HandleFunc("/api/coach/getHomePage", service_coach.GetHomePageHandler)

	//拉取已预约的体验课列表
	http.HandleFunc("/api/coach/getTrailLessonList", service_coach.GetTrailLessonListHandler)

	//拉取学员主页信息
	http.HandleFunc("/api/coach/getTraineePage", service_coach.GetTraineePageHandler)

	//拉取学员的课包列表
	http.HandleFunc("/api/coach/getPackageList", service_coach.GetPackageListHandler)

	//拉取排课页
	http.HandleFunc("/api/coach/getSchedulingPage", service_coach.GetSchedulingPageHandler)

	//拉取排课页
	http.HandleFunc("/api/coach/getSchedulingDetail", service_coach.GetSchedulingDetailHandler)


	//拉取个人页
	http.HandleFunc("/api/coach/getPersonalPage", service_coach.GetPersonalPageHandler)

	//拉取学员列表
	http.HandleFunc("/api/coach/getTraineeList", service_coach.GetTraineeListHandler)

	//拉取学员评价列表
	http.HandleFunc("/api/coach/getTraineeCommentList", service_coach.GetTraineeCommentListHandler)



	//教练端发起排课
	http.HandleFunc("/api/coach/schedueLesson", service_coach.SchedueLessonHandler)

	//测试接口，修改身份
	http.HandleFunc("/api/coach/changeRole", service_coach.ChangeRoleHandler)

	service_coach.InitUserInfoCache()

	//----------------------------测试接口---------------------------------------//
	//测试接口，清空用户信息
	http.HandleFunc("/api/cleanUser", service.CleanUserHandler)

	//测试接口，清空用户信息
	http.HandleFunc("/api/test", service.ForTestHandler)



	log.Fatal(http.ListenAndServe(":80", nil))
}

func autoScanAllCoursePackageSingleLesson() {
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(300))
		for range ticker.C {
			service.ScanAllCoursePackageSingleLesson()
		}
	}()
}

