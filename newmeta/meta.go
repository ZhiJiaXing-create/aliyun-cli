package newmeta

import (
	"encoding/json"
	"strings"

	aliyunopenapimeta "github.com/aliyun/aliyun-cli/v3/aliyun-openapi-meta"
)

type ProductSet struct {
	Products []Product `json:"products"`
}

type Product struct {
	Code         string              `json:"code"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	EndpointType string              `json:"endpointType"`
	Endpoints    map[string]Endpoint `json:"endpoints"`
}

type Endpoint struct {
	RegionId string `json:"regionId"`
	Name     string `json:"regionName"`
	AreaId   string `json:"areaId"`
	AreaName string `json:"areaName"`
	Public   string `json:"public"`
	VPC      string `json:"vpc"`
}

type Version struct {
	Version string         `json:"version"`
	Style   string         `json:"style"`
	APIs    map[string]API `json:"apis"`
}

type API struct {
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Deprecated bool   `json:"deprecated"`
}

type APIDetail struct {
	Name        string             `json:"name"`
	Auth        []string           `json:"security"`
	Deprecated  bool               `json:"deprecated"`
	Protocol    string             `json:"protocol"`
	Method      string             `json:"method"`
	PathPattern string             `json:"pathPattern"`
	Parameters  []RequestParameter `json:"parameters"`
	Example     *APIExample        `json:"example,omitempty"`
	Title       map[string]string  `json:"title,omitempty"`
	Descriptions map[string]string `json:"descriptions,omitempty"`
}

type APIExample struct {
	UnifiedCli string `json:"unifiedCli,omitempty"`
	LegacyCli  string `json:"legacyCli,omitempty"`
}

func (api *APIDetail) IsAnonymousAPI() bool {
	for _, v := range api.Auth {
		if v == "Anonymous" {
			return true
		}
	}
	return false
}

type RequestParameter struct {
	Name        string            `json:"name"`
	Description map[string]string `json:"description,omitempty"`
	Position    string            `json:"position"`
	Type        string            `json:"type"`
	Required    bool              `json:"required"`
}

func GetProductName(language, code string) (name string, err error) {
	langDir := getMetadataPrefix(language)
	content, err := aliyunopenapimeta.Metadatas.ReadFile("products/" + langDir + "/products.json")
	if err != nil {
		return
	}

	products := new(ProductSet)
	err = json.Unmarshal(content, &products)
	if err != nil {
		return
	}

	for _, p := range products.Products {
		if strings.EqualFold(p.Code, code) {
			name = strings.TrimSpace(p.Name)
			break
		}
	}

	return
}

func GetAPI(language, code, name string) (api *API, err error) {
	content, err := aliyunopenapimeta.Metadatas.ReadFile("metadatas/" + strings.ToLower(code) + "/version.json")
	if err != nil {
		return
	}

	version := new(Version)
	err = json.Unmarshal(content, &version)
	if err != nil {
		return
	}

	if found, ok := version.APIs[name]; ok {
		api = &found
	}

	return
}

// GetAPIDetail reads the merged API JSON from metadatas/.
// All fields (including descriptions) are in a single file.
func GetAPIDetail(language, code, name string) (api *APIDetail, err error) {
	lowerCode := strings.ToLower(code)

	content, err := aliyunopenapimeta.Metadatas.ReadFile("metadatas/" + lowerCode + "/" + name + ".json")
	if err != nil {
		return
	}

	detail := new(APIDetail)
	err = json.Unmarshal(content, &detail)
	if err != nil {
		return
	}

	api = detail
	return
}

func getMetadataPrefix(language string) string {
	if language == "en" {
		return "en-US"
	}
	return "zh-CN"
}

// GetDescription returns the description string for the given language,
// with fallback to the other language.
func GetDescription(desc map[string]string, language string) string {
	if desc == nil {
		return ""
	}
	if language == "en" {
		if v, ok := desc["en"]; ok && v != "" {
			return v
		}
		return desc["zh"]
	}
	if v, ok := desc["zh"]; ok && v != "" {
		return v
	}
	return desc["en"]
}
