package main

import (
	"fmt"
	"path"
	"runtime"
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
