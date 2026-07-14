package newmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDescription(t *testing.T) {
	// nil map
	assert.Equal(t, "", GetDescription(nil, "zh"))

	// zh only
	desc := map[string]string{"zh": "中文描述"}
	assert.Equal(t, "中文描述", GetDescription(desc, "zh"))
	assert.Equal(t, "中文描述", GetDescription(desc, "en")) // fallback to zh

	// en only
	desc = map[string]string{"en": "English description"}
	assert.Equal(t, "English description", GetDescription(desc, "en"))
	assert.Equal(t, "English description", GetDescription(desc, "zh")) // fallback to en

	// both
	desc = map[string]string{"zh": "中文", "en": "English"}
	assert.Equal(t, "中文", GetDescription(desc, "zh"))
	assert.Equal(t, "English", GetDescription(desc, "en"))

	// empty zh, fallback to en
	desc = map[string]string{"zh": "", "en": "English"}
	assert.Equal(t, "English", GetDescription(desc, "zh"))
}

func TestGetProductName(t *testing.T) {
	name, err := GetProductName("en", "ecs")
	assert.Nil(t, err)
	assert.Equal(t, "Elastic Compute Service", name)
	name, err = GetProductName("zh", "ecs")
	assert.Nil(t, err)
	assert.Equal(t, "云服务器 ECS", name)
}

func TestGetAPI(t *testing.T) {
	api, err := GetAPI("en", "ecs", "DescribeRegions")
	assert.Nil(t, err)
	assert.Equal(t, "查询地域列表", api.Title)
	assert.Equal(t, "根据计费方式、资源类型等参数查询地域信息列表。", api.Summary)
	assert.Equal(t, false, api.Deprecated)

	api2, err := GetAPI("en", "ecs", "Invalid")
	assert.Nil(t, err)
	assert.Nil(t, api2)
}

func TestGetAPIDetail(t *testing.T) {
	api, err := GetAPIDetail("en", "ecs", "DescribeRegions")
	assert.Nil(t, err)
	assert.Equal(t, "DescribeRegions", api.Name)
	assert.Equal(t, "GET|POST", api.Method)
	assert.Equal(t, false, api.Deprecated)
}

func TestIsAnonymousAPI(t *testing.T) {
	akapi, err := GetAPIDetail("en", "ecs", "DescribeRegions")
	assert.Nil(t, err)
	assert.False(t, akapi.IsAnonymousAPI())
	api, err := GetAPIDetail("en", "ice", "GetPublicMediaInfo")
	assert.Nil(t, err)
	assert.True(t, api.IsAnonymousAPI())
}
