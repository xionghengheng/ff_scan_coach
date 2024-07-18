package main

import (
	"github.com/xionghengheng/ff_plib/db/dao"
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
	return nil
}
