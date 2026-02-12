package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/xionghengheng/ff_plib/db/dao"
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

// AdminAuthResult 管理员身份验证结果
type AdminAuthResult struct {
	Success  bool   // 验证是否成功
	Code     int    // 错误码（验证失败时使用）
	ErrorMsg string // 错误信息（验证失败时使用）
}

// ValidateAdminAuth 验证管理员身份
// 参数：r - HTTP请求，用于从header中提取用户名和密码
// 返回：AdminAuthResult - 验证结果
func ValidateAdminAuth(r *http.Request) AdminAuthResult {
	// 验证管理员用户名和密码配置
	adminUserName := os.Getenv("ADMIN_USER_NAME")
	adminPasswd := os.Getenv("ADMIN_PASSWD")
	if len(adminUserName) == 0 || len(adminPasswd) == 0 {
		Printf("ValidateAdminAuth conf err, adminUserName:%s adminPasswd:%s\n", adminUserName, adminPasswd)
		return AdminAuthResult{
			Success:  false,
			Code:     -900,
			ErrorMsg: "后台配置错误",
		}
	}

	// 从header中提取用户名进行校验
	username := r.Header.Get("X-Username")
	if username == "" {
		Printf("ValidateAdminAuth missing X-Username header\n")
		return AdminAuthResult{
			Success:  false,
			Code:     -995,
			ErrorMsg: "缺少X-Username header",
		}
	}

	// 从header中提取密码进行校验
	password := r.Header.Get("X-Password")
	if password == "" {
		Printf("ValidateAdminAuth missing X-Password header\n")
		return AdminAuthResult{
			Success:  false,
			Code:     -995,
			ErrorMsg: "缺少X-Password header",
		}
	}

	// 校验用户名和密码
	if username != adminUserName || password != adminPasswd {
		Printf("ValidateAdminAuth auth failed, username:%s\n", username)
		return AdminAuthResult{
			Success:  false,
			Code:     -994,
			ErrorMsg: "用户名或密码错误",
		}
	}

	return AdminAuthResult{
		Success: true,
	}
}

// ConsultantOrAdminAuthResult 顾问或管理员身份验证结果
type ConsultantOrAdminAuthResult struct {
	Success      bool   // 是否验证成功
	Code         int    // 错误码（验证失败时使用）
	ErrorMsg     string // 错误信息（验证失败时使用）
	IsConsultant bool   // 是否为顾问身份
	ConsultantNick string // 顾问昵称（顾问身份时有值）
}

// ValidateConsultantOrAdminAuth 验证顾问或管理员身份
// 优先通过OpenID识别顾问（需IsOfficialAssistant为true），否则走管理员账号密码校验
func ValidateConsultantOrAdminAuth(r *http.Request) ConsultantOrAdminAuthResult {
	strOpenId := r.Header.Get("X-WX-OPENID")
	if strOpenId != "" {
		consultantUser, err := dao.ImpUser.GetUserByOpenId(strOpenId)
		if err == nil && consultantUser != nil {
			if !consultantUser.IsOfficialAssistant {
				Printf("ValidateConsultantOrAdminAuth not consultant, openid:%s\n", strOpenId)
				return ConsultantOrAdminAuthResult{
					Success:  false,
					Code:     -996,
					ErrorMsg: "当前用户不是顾问，无权操作",
				}
			}
			Printf("ValidateConsultantOrAdminAuth consultant auth, openid:%s nick:%s\n", strOpenId, consultantUser.Nick)
			return ConsultantOrAdminAuthResult{
				Success:        true,
				IsConsultant:   true,
				ConsultantNick: consultantUser.Nick,
			}
		}
	}

	// 非顾问，走管理员账号密码校验
	adminResult := ValidateAdminAuth(r)
	if !adminResult.Success {
		return ConsultantOrAdminAuthResult{
			Success:  false,
			Code:     adminResult.Code,
			ErrorMsg: adminResult.ErrorMsg,
		}
	}
	return ConsultantOrAdminAuthResult{
		Success: true,
	}
}
