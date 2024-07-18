package service_coach

import (
	"FunFitnessTrainer/db/dao"
	"FunFitnessTrainer/db/model"
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/golang/groupcache"
	"path"
	"runtime"
	"strconv"
	"strings"
)

// 获取调用者的文件名和函数名
func getCallerInfo(skip int) (string, string) {
	pc, file, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", "unknown"
	}
	return path.Base(file), fn.Name()
}

// 包装 fmt.Printf，增加文件名和函数名打印
func Printf(format string, args ...interface{}) {
	// 这里传递 2 以获取更上层的调用者信息
	fileName, fullFuncName := getCallerInfo(2)

	var funcName string
	vecFullFuncName := strings.Split(fullFuncName, ".")
	if len(vecFullFuncName) > 0 {
		funcName = vecFullFuncName[len(vecFullFuncName)-1]
	} else {
		funcName = fullFuncName
	}
	format = fmt.Sprintf("[%s:%s] %s\n", fileName, funcName, format)
	fmt.Printf(format, args...)
}

var userInfoCache *groupcache.Group

func InitUserInfoCache() {
	// 创建一个缓存组，大小为 64MB
	userInfoCache = groupcache.NewGroup("userInfoGroupCache", 64<<20, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			// 当缓存未命中时，提供一个回调函数来获取数据
			uid, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				return err
			}

			stUserInfoModel, err := dao.ImpUser.GetUser(uid)
			if err != nil || stUserInfoModel == nil {
				return err
			}

			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			err = enc.Encode(*stUserInfoModel)
			if err != nil {
				fmt.Println("Error serializing struct:", err)
				return err
			}

			err = dest.SetBytes(buf.Bytes())
			if err != nil {
				return err
			}
			return nil
		},
	))
}

func GetAllUser(vecUid []int64) (map[int64]model.UserInfoModel, error) {
	mapUser := make(map[int64]model.UserInfoModel)
	if len(vecUid) == 0 {
		return mapUser, nil
	}
	for _, uid := range vecUid {
		var data []byte
		key := fmt.Sprintf("%d", uid)
		// 从缓存中获取数据
		err := userInfoCache.Get(nil, key, groupcache.AllocatingByteSliceSink(&data))
		if err != nil {
			Printf("getting value err, err:%+v uid:%d", err, uid)
			continue
		}

		var stUserInfoModel model.UserInfoModel
		dec := gob.NewDecoder(bytes.NewReader(data))
		err = dec.Decode(&stUserInfoModel)
		if err != nil {
			Printf("deserializing err:%+v uid:%d", err, uid)
			continue
		}
		Printf("Deserialized succ, uid:%d stUserInfoModel:%+v", uid, stUserInfoModel)
		mapUser[stUserInfoModel.UserID] = stUserInfoModel
	}
	return mapUser, nil
}

func GetAllLessonInfo(vecAllAppointmentId []AppointmentItem) (map[int]model.CoursePackageSingleLessonModel, error) {
	mapAppointmentID2SingleLessonInfo := make(map[int]model.CoursePackageSingleLessonModel)
	for _, v := range vecAllAppointmentId {
		//注意由于单次课数据量太大，这里只选择部分字段（lesson_id, package_id, appointment_id, status）
		stCourseModel, err := dao.ImpCoursePackageSingleLesson.GetSingleLessonByAppointmentId(v.Uid, v.AppointmentID)
		if err != nil || stCourseModel == nil {
			Printf("GetSingleLessonByAppointmentId uid:%d AppointmentID:%d err:%+v", v.Uid, v.AppointmentID, err)
			return mapAppointmentID2SingleLessonInfo, err
		}
		mapAppointmentID2SingleLessonInfo[v.AppointmentID] = *stCourseModel
	}
	return mapAppointmentID2SingleLessonInfo, nil
}
