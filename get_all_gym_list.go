package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/model"
)

// GetAllGymListReq 获取全量健身房列表请求
type GetAllGymListReq struct {
	// 无需参数，获取全量健身房
}

// GetAllGymListRsp 获取全量健身房列表响应
type GetAllGymListRsp struct {
	Code      int                  `json:"code"`
	ErrorMsg  string               `json:"errorMsg"`
	VecAllGym []model.GymInfoModel `json:"vec_all_gym"` // 全量健身房列表
}

// getGetAllGymListReq 解析请求参数
func getGetAllGymListReq(r *http.Request) (GetAllGymListReq, error) {
	req := GetAllGymListReq{}
	// 该接口无需参数
	return req, nil
}

// GetAllGymListHandler 获取全量健身房列表
func GetAllGymListHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getGetAllGymListReq(r)
	rsp := &GetAllGymListRsp{}

	// 打日志要加换行，不然不会刷到屏幕
	Printf("GetAllGymListHandler req start, req:%+v\n", req)

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
		Printf("GetAllGymListHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("GetAllGymListHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("GetAllGymListHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		Printf("GetAllGymListHandler parse req err, err:%+v\n", err)
		return
	}

	// 获取所有健身房信息
	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -921
		rsp.ErrorMsg = err.Error()
		Printf("GetAllGym err, err:%+v\n", err)
		return
	}

	// 构建全量健身房列表
	for _, gymInfo := range mapGym {
		rsp.VecAllGym = append(rsp.VecAllGym, gymInfo)
	}

	// 按照健身房ID排序
	sort.Slice(rsp.VecAllGym, func(i, j int) bool {
		return rsp.VecAllGym[i].GymID < rsp.VecAllGym[j].GymID
	})

	rsp.Code = 0
	rsp.ErrorMsg = "success"
	Printf("GetAllGymListHandler success, gym count:%d\n", len(rsp.VecAllGym))
	return
}
