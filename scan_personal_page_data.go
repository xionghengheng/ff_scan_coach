package main

import (
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
	"time"
)

// ScanCoachPersonalPageData 扫描生成教练端主页的计数信息
func ScanCoachPersonalPageData() {
	Printf("scan start, beg_time:%s", time.Now().Format("2006-01-02 15:04:05"))
	err := doScan()
	if err != nil {
		Printf("doScan err, err:%+v", err)
		return
	}
	Printf("scan end, end_time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

// ScanCoachPersonalPageData 扫描生成教练端主页的计数信息
func doScan() error {
	vecCoachModel, err := dao.ImpCoach.GetCoachAll()
	if err != nil {
		Printf("GetCoachAll err, err:%+v", err)
		return err
	}
	Printf("GetCoachAll succ, vecCoachModel:%+v", vecCoachModel)

	unMonthBegTs := GetFirstOfMonthBegTimestamp()

	for _,v := range vecCoachModel{
		if v.CoachID > 0{
			genCoachData(v.CoachID, unMonthBegTs)
		}
	}
	return nil
}

func genCoachData(coachId int, unMonthBegTs int64){

	//统计各个维度的计数
	vecPaymentOrderModel, err := dao.ImpPaymentOrder.GetOrderListByCoachId(coachId, unMonthBegTs)
	if err != nil{
		Printf("GetOrderListByCoachId err, err:%+v coachId:%d", err, coachId)
		return
	}

	if len(vecPaymentOrderModel) == 0{
		return
	}

	mapPayUser := make(map[int64]bool)
	var unSaleRevenue uint32
	for _,v := range vecPaymentOrderModel{
		unSaleRevenue += uint32(v.Price)
		if _,ok:= mapPayUser[v.PayerUID];ok{
			continue
		}else{
			mapPayUser[v.PayerUID] = true
		}
	}
	
	//添加到统计表里
	stCoachMonthlyStatisticModel := &model.CoachMonthlyStatisticModel{}
	stCoachMonthlyStatisticModel.CoachID = coachId
	stCoachMonthlyStatisticModel.MonthBegTs = unMonthBegTs
	stCoachMonthlyStatisticModel.PayUserCnt = uint32(len(mapPayUser))
	stCoachMonthlyStatisticModel.LessonCnt = 1
	stCoachMonthlyStatisticModel.LessonUserCnt = 1
	stCoachMonthlyStatisticModel.SaleRevenue = unSaleRevenue
	err = dao.ImpCoachClientMonthlyStatistic.AddItem(stCoachMonthlyStatisticModel)
	if err != nil {
		Printf("AddItem err, err:%+v coachId:%d", err, coachId)
		return
	}
	Printf("AddItem succ, coachId:%d stCoachMonthlyStatisticModel:%+v", coachId, stCoachMonthlyStatisticModel)
}