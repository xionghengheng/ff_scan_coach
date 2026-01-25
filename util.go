package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
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

// 固定密钥（16字节，用于AES-128加密）
const aesSecretKey = "FF_TRIAL_KEY_16!"

// 生成H5链接token（包含记录ID，使用AES加密+Base64编码）
func generateH5LinkToken(recordId int64, createTs int64) string {
	// 使用记录ID+教练ID+时间戳生成原始数据
	data := fmt.Sprintf("%d_%d", recordId, createTs)

	// AES加密
	encrypted, err := aesEncrypt([]byte(data), []byte(aesSecretKey))
	if err != nil {
		Printf("generateH5LinkToken aesEncrypt err:%v\n", err)
		return ""
	}

	// URL安全的Base64编码
	return base64.URLEncoding.EncodeToString(encrypted)
}

// AES加密（CBC模式）
func aesEncrypt(plainText, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// PKCS7填充
	blockSize := block.BlockSize()
	padding := blockSize - len(plainText)%blockSize
	padText := make([]byte, len(plainText)+padding)
	copy(padText, plainText)
	for i := len(plainText); i < len(padText); i++ {
		padText[i] = byte(padding)
	}

	// 使用密钥前16字节作为IV
	iv := key[:blockSize]
	encrypted := make([]byte, len(padText))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, padText)

	return encrypted, nil
}

// AES解密（CBC模式）
func aesDecrypt(cipherText, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	if len(cipherText) < blockSize || len(cipherText)%blockSize != 0 {
		return nil, fmt.Errorf("密文长度无效")
	}

	// 使用密钥前16字节作为IV
	iv := key[:blockSize]
	decrypted := make([]byte, len(cipherText))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, cipherText)

	// 去除PKCS7填充
	padding := int(decrypted[len(decrypted)-1])
	if padding > blockSize || padding == 0 {
		return nil, fmt.Errorf("填充无效")
	}
	// 校验填充
	for i := len(decrypted) - padding; i < len(decrypted); i++ {
		if decrypted[i] != byte(padding) {
			return nil, fmt.Errorf("填充校验失败")
		}
	}

	return decrypted[:len(decrypted)-padding], nil
}

// 解密H5链接token，返回recordId, coachId, createTs
func decryptH5LinkToken(token string) (recordId int64, createTs int64, err error) {
	// Base64解码
	cipherText, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return 0, 0, fmt.Errorf("Base64解码失败: %v", err)
	}

	// AES解密
	plainText, err := aesDecrypt(cipherText, []byte(aesSecretKey))
	if err != nil {
		return 0, 0, fmt.Errorf("AES解密失败: %v", err)
	}

	// 解析原始数据: recordId_coachId_createTs
	var rId, ts int64
	_, err = fmt.Sscanf(string(plainText), "%d_%d_%d", &rId, &ts)
	if err != nil {
		return 0, 0, fmt.Errorf("解析数据失败: %v", err)
	}

	return rId, ts, nil
}
