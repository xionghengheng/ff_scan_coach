package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/xionghengheng/ff_plib/comm"
	"github.com/xionghengheng/ff_plib/db/dao"
	"github.com/xionghengheng/ff_plib/db/model"
)

// 前端管理平台接口

type GetAllCoachListReq struct {
	// 无需参数，获取全量教练
}

// 简单的健身房信息
type SimpleShowGymInfo struct {
	GymID   int    `json:"gym_id"`   //健身房id
	GymName string `json:"gym_name"` //健身房名称
}

// 简单的课程信息
type SimpleShowCourseInfo struct {
	CourseID   int    `json:"course_id"`   //课程id
	CourseName string `json:"course_name"` //课程名称
}

// CoachInfoForFrontend 专门给前端使用的教练信息结构（包含数据库原始字段）
type CoachInfoForFrontend struct {
	VecBindGymInfo      []SimpleShowGymInfo    `json:"vec_gym_info"`          //教练绑定的所有健身房列表
	VecCourseInfo       []SimpleShowCourseInfo `json:"vec_course_info"`       //教练可上的所有课程列表
	CoachID             int                    `json:"coach_id"`              //教练id
	CoachName           string                 `json:"coach_name"`            //教练名称
	Avatar              string                 `json:"avatar"`                //教练头像url
	CircleAvatar        string                 `json:"circle_avatar"`         //教练圆形头像url
	Bio                 string                 `json:"bio"`                   //教练简介
	GoodAt              string                 `json:"good_at"`               //教练擅长领域
	Phone               string                 `json:"phone"`                 //手机号
	QualifyType         int                    `json:"qualify_type"`          //教练资质类型
	SkillCertification  string                 `json:"skill_certification"`   //教练的技能认证（逗号分隔）
	Style               string                 `json:"style"`                 //教练风格（逗号分隔）
	YearsOfWork         string                 `json:"years_of_work"`         //从业时长
	TotalCompleteLesson string                 `json:"total_complete_lesson"` //累计上课节数
	BTestCoach          bool                   `json:"b_test_coach"`          //是否测试教练
	CanShow             int                    `json:"can_show"`              //是否可展示
	QualifyDetail       QualifyDetail          `json:"qualify_detail"`        //教练资质详细描述
}

type QualifyDetail struct {
	Title string `json:"title"` //标题
	Desc  string `json:"desc"`  //具体描述
}

type CoachQualifyDesc struct {
	MapQualifyType2Desc map[int]QualifyDetail `json:"map_qualify_type_2_desc"` //教练资质类型对应的具体描述
}

type GetAllCoachListRsp struct {
	Code         int                    `json:"code"`
	ErrorMsg     string                 `json:"errorMsg,omitempty"`
	VecCoachInfo []CoachInfoForFrontend `json:"coach_list,omitempty"` //教练列表（包含资质描述）
	TotalCount   int                    `json:"total_count"`          //教练总数
	VecAllGym    []SimpleShowGymInfo    `json:"all_gym_list"`         //全量门店列表
	VecAllCourse []SimpleShowCourseInfo `json:"all_course_list"`      //全量课程列表
}

func getGetAllCoachListHandler(r *http.Request) (GetAllCoachListReq, error) {
	req := GetAllCoachListReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	defer r.Body.Close()
	return req, nil
}

// GetAllCoachListHandler 获取全量教练列表
func GetAllCoachListHandler(w http.ResponseWriter, r *http.Request) {
	req, err := getGetAllCoachListHandler(r)
	rsp := &GetAllCoachListRsp{}

	//打日志要加换行，不然不会刷到屏幕
	Printf("GetAllCoachListHandler req start, req:%+v\n", req)

	defer func() {
		msg, err := json.Marshal(rsp)
		if err != nil {
			fmt.Fprint(w, "内部错误")
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(msg)
	}()

	// 获取所有教练信息
	mapCoach, err := comm.GetAllCoach()
	if err != nil {
		rsp.Code = -922
		rsp.ErrorMsg = err.Error()
		return
	}

	// 获取所有健身房信息
	mapGym, err := comm.GetAllGym()
	if err != nil {
		rsp.Code = -921
		rsp.ErrorMsg = err.Error()
		return
	}

	// 获取所有课程信息
	mapCourse := make(map[int]string) // courseID -> courseName
	vecCourseInfoModel, err := dao.ImpCourse.GetCourseList()
	if err != nil {
		rsp.Code = -920
		rsp.ErrorMsg = err.Error()
		return
	}
	for _, v := range vecCourseInfoModel {
		mapCourse[v.CourseID] = v.Name
	}

	// 获取资质描述映射
	stCoachQualifyDesc := getCoachQualifyDesc()

	// 构建全量门店列表
	for gymId, gymInfo := range mapGym {
		rsp.VecAllGym = append(rsp.VecAllGym, SimpleShowGymInfo{
			GymID:   gymId,
			GymName: gymInfo.LocName,
		})
	}
	// 按照门店ID排序
	sort.Slice(rsp.VecAllGym, func(i, j int) bool {
		return rsp.VecAllGym[i].GymID < rsp.VecAllGym[j].GymID
	})

	// 构建全量课程列表
	for courseId, courseName := range mapCourse {
		rsp.VecAllCourse = append(rsp.VecAllCourse, SimpleShowCourseInfo{
			CourseID:   courseId,
			CourseName: courseName,
		})
	}
	// 按照课程ID排序
	sort.Slice(rsp.VecAllCourse, func(i, j int) bool {
		return rsp.VecAllCourse[i].CourseID < rsp.VecAllCourse[j].CourseID
	})

	// 过滤并转换教练信息
	for _, coachModel := range mapCoach {

		// 转换图片链接格式：cloud:// -> https://
		avatar := convertCloudUrlToHttps(coachModel.Avatar)
		circleAvatar := convertCloudUrlToHttps(coachModel.CircleAvatar)

		// 创建前端专用的教练信息结构
		coachInfoForFrontend := CoachInfoForFrontend{
			CoachID:             coachModel.CoachID,
			CoachName:           coachModel.CoachName,
			Avatar:              avatar,
			CircleAvatar:        circleAvatar,
			Bio:                 coachModel.Bio,
			GoodAt:              coachModel.GoodAt,
			Phone:               coachModel.Phone,
			QualifyType:         coachModel.QualifyType,
			SkillCertification:  coachModel.SkillCertification,
			Style:               coachModel.Style,
			YearsOfWork:         coachModel.YearsOfWork,
			TotalCompleteLesson: coachModel.TotalCompleteLesson,
			BTestCoach:          coachModel.BTestCoach,
			CanShow:             coachModel.CanShow,
		}

		// 构建教练绑定的健身房列表
		for _, gymId := range comm.GetAllGymIds(coachModel.GymIDs) {
			if gymInfo, ok := mapGym[gymId]; ok {
				coachInfoForFrontend.VecBindGymInfo = append(coachInfoForFrontend.VecBindGymInfo, SimpleShowGymInfo{
					GymID:   gymId,
					GymName: gymInfo.LocName,
				})
			}
		}

		// 构建教练可上的课程列表
		if len(coachModel.CourseIdList) > 0 {
			vecCourseId := strings.Split(coachModel.CourseIdList, ",")
			for _, id := range vecCourseId {
				// trim掉可能存在的换行符和空格
				id = strings.TrimSpace(id)
				nId, err := strconv.ParseInt(id, 10, 64)
				if err != nil {
					Printf("ParseInt err, err:%+v id:%d CoachID:%d CourseIdList:%s\n", err, id, coachModel.CoachID, coachModel.CourseIdList)
					continue
				}
				if courseName, ok := mapCourse[int(nId)]; ok {
					coachInfoForFrontend.VecCourseInfo = append(coachInfoForFrontend.VecCourseInfo, SimpleShowCourseInfo{
						CourseID:   int(nId),
						CourseName: courseName,
					})
				}
			}
		}

		// 根据教练的资质类型，添加对应的资质描述
		if qualifyDetail, ok := stCoachQualifyDesc.MapQualifyType2Desc[coachModel.QualifyType]; ok {
			coachInfoForFrontend.QualifyDetail = qualifyDetail
		}

		rsp.VecCoachInfo = append(rsp.VecCoachInfo, coachInfoForFrontend)
	}

	// 按照教练ID排序，确保每次返回顺序一致
	sort.Slice(rsp.VecCoachInfo, func(i, j int) bool {
		return rsp.VecCoachInfo[i].CoachID < rsp.VecCoachInfo[j].CoachID
	})

	rsp.TotalCount = len(rsp.VecCoachInfo)
}

func getCoachQualifyDesc() CoachQualifyDesc {
	var stCoachQualifyDesc CoachQualifyDesc
	stCoachQualifyDesc.MapQualifyType2Desc = make(map[int]QualifyDetail)
	var stQualifyDetail QualifyDetail
	stQualifyDetail.Title = "基础教练"
	stQualifyDetail.Desc = "教练资质1-2年，证书：国内健身私人教练注册认证，累计授课节数500+；"
	stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Basic] = stQualifyDetail

	stQualifyDetail.Title = "中级教练"
	stQualifyDetail.Desc = "教练资质2-4年，证书：国内健身私人教练注册认证+其他专业证书认证，累计授课节数1000+；"
	stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Intermediate] = stQualifyDetail

	stQualifyDetail.Title = "高级教练"
	stQualifyDetail.Desc = "教练资质5-8年，证书：NSCA-CPT/NASM-CPT/ACSM-CPT/ACE-CPT等国际证书认证，累计授课节数2000+；"
	stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Advanced] = stQualifyDetail

	stQualifyDetail.Title = "资深教练"
	stQualifyDetail.Desc = "教练资质8-10年+，证书：NSCA-CPT/NASM-CPT/ACSM-CPT/ACE-CPT等国际证书认证，累计授课节数5000+；"
	stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Senior] = stQualifyDetail

	//
	//stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Basic] = "教练资质1-2年，证书：国内健身私人教练注册认证，累计授课节数500+；"
	//stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Intermediate] = "教练资质2-4年，证书：国内健身私人教练注册认证+其他专业证书认证，累计授课节数1000+；"
	//stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Advanced] = "教练资质5-8年，证书：NSCA-CPT/NASM-CPT/ACSM-CPT/ACE-CPT等国际证书认证，累计授课节数2000+；"
	//stCoachQualifyDesc.MapQualifyType2Desc[model.Enum_Coach_QualifyType_Senior] = "教练资质8-10年+，证书：NSCA-CPT/NASM-CPT/ACSM-CPT/ACE-CPT等国际证书认证，累计授课节数5000+；"

	return stCoachQualifyDesc
}

// convertCloudUrlToHttps 将腾讯云cloud://格式的URL转换为https://格式
// 例如: cloud://prod-8gl9g7u4ad06b98e.7072-prod-8gl9g7u4ad06b98e-1326535808/coach/new/250X200/吴建宏的副本.png
// 转换为: https://7072-prod-8gl9g7u4ad06b98e-1326535808.tcb.qcloud.la/coach/new/250X200/吴建宏的副本.png
func convertCloudUrlToHttps(cloudUrl string) string {
	if cloudUrl == "" {
		return ""
	}

	// 如果不是cloud://开头，直接返回原URL
	if !strings.HasPrefix(cloudUrl, "cloud://") {
		return cloudUrl
	}

	// 去掉 cloud:// 前缀
	urlWithoutPrefix := strings.TrimPrefix(cloudUrl, "cloud://")

	// 分割字符串，格式为: prod-8gl9g7u4ad06b98e.7072-prod-8gl9g7u4ad06b98e-1326535808/path/to/file
	parts := strings.SplitN(urlWithoutPrefix, "/", 2)
	if len(parts) != 2 {
		// 格式不正确，返回原URL
		return cloudUrl
	}

	// parts[0] 是环境信息，格式为: prod-8gl9g7u4ad06b98e.7072-prod-8gl9g7u4ad06b98e-1326535808
	// 需要提取出 7072-prod-8gl9g7u4ad06b98e-1326535808 部分
	envInfo := parts[0]
	dotIndex := strings.Index(envInfo, ".")
	if dotIndex == -1 {
		// 格式不正确，返回原URL
		return cloudUrl
	}

	bucketId := envInfo[dotIndex+1:] // 7072-prod-8gl9g7u4ad06b98e-1326535808
	filePath := parts[1]              // coach/new/250X200/吴建宏的副本.png

	// 构建https URL
	httpsUrl := fmt.Sprintf("https://%s.tcb.qcloud.la/%s", bucketId, filePath)

	return httpsUrl
}
