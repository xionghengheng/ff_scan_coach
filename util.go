package main

import (
	"fmt"
	"path"
	"runtime"
	"strings"
	"time"
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

// GetFirstOfMonthBegTimestamp 返回当前时间所在月份1号的开始时间的 Unix 时间戳
func GetFirstOfMonthBegTimestamp() int64 {
	now := time.Now()
	year, month, _ := now.Date()
	location := now.Location()
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, location)
	return firstOfMonth.Unix()
}
