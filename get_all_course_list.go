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

// GetAllCourseListReq 获取全量课程列表请求
type GetAllCourseListReq struct {
	// 无需参数，获取全量课程
}

// GetAllCourseListRsp 获取全量课程列表响应
type GetAllCourseListRsp struct {
	Code         int                 `json:"code"`
	ErrorMsg     string              `json:"errorMsg"`
	VecAllCourse []model.CourseModel `json:"vec_all_course"` // 全量课程列表
}

// 预体验课，过滤出类型为5的
//const (
//	Enum_Course_Type_Trial        = iota // 0=基础
//	Enum_Course_Type_Intermediate        // 1=中级
//	Enum_Course_Type_Advanced            // 2=高级
//	Enum_Course_Type_Senior              // 3=资深
//	Enum_Course_Type_Specialty           // 4=特色
//	Enum_Course_Type_PaidPreTrial        // 5=付费的预体验课
//)

// getGetAllCourseListReq 解析请求参数
func getGetAllCourseListReq(r *http.Request) (GetAllCourseListReq, error) {
	req := GetAllCourseListReq{}
	// 该接口无需参数
	return req, nil
}

// GetAllCourseListHandler 获取全量课程列表
func GetAllCourseListHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getGetAllCourseListReq(r)
	rsp := &GetAllCourseListRsp{}

	// 打日志要加换行，不然不会刷到屏幕
	Printf("GetAllCourseListHandler req start, req:%+v\n", req)

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
		Printf("GetAllCourseListHandler missing X-Username header\n")
		return
	}

	password := r.Header.Get("X-Password")
	if password == "" {
		rsp.Code = -995
		rsp.ErrorMsg = "缺少X-Password header"
		Printf("GetAllCourseListHandler missing X-Password header\n")
		return
	}

	if username != adminUserName || password != adminPasswd {
		rsp.Code = -994
		rsp.ErrorMsg = "用户名或密码错误"
		Printf("GetAllCourseListHandler auth failed, username:%s\n", username)
		return
	}

	if err != nil {
		rsp.Code = -998
		rsp.ErrorMsg = err.Error()
		Printf("GetAllCourseListHandler parse req err, err:%+v\n", err)
		return
	}

	// 获取所有课程信息
	mapCourse, err := comm.GetAllCourse()
	if err != nil {
		rsp.Code = -921
		rsp.ErrorMsg = err.Error()
		Printf("GetAllCourse err, err:%+v\n", err)
		return
	}

	// 构建全量课程列表，同时收集课程ID用于排序
	type courseWithId struct {
		id     int
		course model.CourseModel
	}
	var vecCourseWithId []courseWithId
	for id, courseInfo := range mapCourse {
		vecCourseWithId = append(vecCourseWithId, courseWithId{id: id, course: courseInfo})
	}

	// 按照课程ID排序
	sort.Slice(vecCourseWithId, func(i, j int) bool {
		return vecCourseWithId[i].id < vecCourseWithId[j].id
	})

	// 构建返回列表
	for _, item := range vecCourseWithId {
		rsp.VecAllCourse = append(rsp.VecAllCourse, item.course)
	}

	rsp.Code = 0
	rsp.ErrorMsg = "success"
	Printf("GetAllCourseListHandler success, course count:%d\n", len(rsp.VecAllCourse))
}
