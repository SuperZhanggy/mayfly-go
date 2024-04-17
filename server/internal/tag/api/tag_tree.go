package api

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"sort"
	"strings"
)

type TagTree struct {
	TagTreeApp application.TagTree `inject:""`
}

func (p *TagTree) GetTagTree(rc *req.Ctx) {
	tagType := entity.TagType(rc.QueryInt("type"))
	// 超管返回所有标签树
	if rc.GetLoginAccount().Id == consts.AdminId {
		var tagTrees vo.TagTreeVOS
		p.TagTreeApp.ListByQuery(&entity.TagTreeQuery{Type: tagType}, &tagTrees)
		rc.ResData = tagTrees.ToTrees(0)
		return
	}

	// 获取用户可以操作访问的标签路径
	tagPaths := p.TagTreeApp.ListTagByAccountId(rc.GetLoginAccount().Id)

	rootTag := make(map[string][]string, 0)
	for _, accountTagPath := range tagPaths {
		root := strings.Split(accountTagPath, "/")[0] + entity.CodePathSeparator
		tags := rootTag[root]
		tags = append(tags, accountTagPath)
		rootTag[root] = tags
	}

	// 获取所有以root标签开头的子标签
	var tags []*entity.TagTree
	p.TagTreeApp.ListByQuery(&entity.TagTreeQuery{CodePathLikes: collx.MapKeys(rootTag), Type: tagType}, &tags)

	tagTrees := make(vo.TagTreeVOS, 0)
	for _, tag := range tags {
		tagPath := tag.CodePath
		root := strings.Split(tagPath, "/")[0] + entity.CodePathSeparator
		// 获取用户可操作的标签路径列表
		accountTagPaths := rootTag[root]
		for _, accountTagPath := range accountTagPaths {
			if strings.HasPrefix(tagPath, accountTagPath) || strings.HasPrefix(accountTagPath, tagPath) {
				tagTrees = append(tagTrees, tag)
				break
			}
		}
	}

	rc.ResData = tagTrees.ToTrees(0)
}

func (p *TagTree) ListByQuery(rc *req.Ctx) {
	cond := new(entity.TagTreeQuery)
	tagPaths := rc.Query("tagPaths")
	cond.CodePaths = strings.Split(tagPaths, ",")
	var tagTrees vo.TagTreeVOS
	p.TagTreeApp.ListByQuery(cond, &tagTrees)
	rc.ResData = tagTrees
}

func (p *TagTree) SaveTagTree(rc *req.Ctx) {
	tagForm := &form.TagTree{}
	tagTree := req.BindJsonAndCopyTo(rc, tagForm, new(entity.TagTree))

	rc.ReqParam = fmt.Sprintf("tagTreeId: %d, tagName: %s, code: %s", tagTree.Id, tagTree.Name, tagTree.Code)

	biz.ErrIsNil(p.TagTreeApp.SaveTag(rc.MetaCtx, tagForm.Pid, tagTree))
}

func (p *TagTree) DelTagTree(rc *req.Ctx) {
	biz.ErrIsNil(p.TagTreeApp.Delete(rc.MetaCtx, uint64(rc.PathParamInt("id"))))
}

// 获取用户可操作的标签路径
func (p *TagTree) TagResources(rc *req.Ctx) {
	resourceType := int8(rc.PathParamInt("rtype"))
	accountId := rc.GetLoginAccount().Id
	tagResources := p.TagTreeApp.GetAccountTags(accountId, &entity.TagTreeQuery{Type: entity.TagType(resourceType)})

	tagPath2Resource := collx.ArrayToMap[*entity.TagTree, string](tagResources, func(tagResource *entity.TagTree) string {
		return tagResource.GetTagPath()
	})

	tagPaths := collx.MapKeys(tagPath2Resource)
	sort.Strings(tagPaths)
	rc.ResData = tagPaths
}

// 统计当前用户指定标签下关联的资源数量
func (p *TagTree) CountTagResource(rc *req.Ctx) {
	tagPath := rc.Query("tagPath")
	accountId := rc.GetLoginAccount().Id

	machineCodes := entity.GetCodeByPath(entity.TagTypeMachine, p.TagTreeApp.GetAccountTagCodePaths(accountId, entity.TagTypeMachineAuthCert, tagPath)...)
	dbCodes := entity.GetCodeByPath(entity.TagTypeDb, p.TagTreeApp.GetAccountTagCodePaths(accountId, entity.TagTypeDbName, tagPath)...)

	rc.ResData = collx.M{
		"machine": len(machineCodes),
		"db":      len(dbCodes),
		"redis":   len(p.TagTreeApp.GetAccountTagCodes(accountId, consts.ResourceTypeRedis, tagPath)),
		"mongo":   len(p.TagTreeApp.GetAccountTagCodes(accountId, consts.ResourceTypeMongo, tagPath)),
	}
}
