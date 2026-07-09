// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/newmeta"
	"github.com/stretchr/testify/assert"

	"bytes"
	"testing"
)

func TestLibrary_PrintProducts(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")

	_, isexist := library.GetApi("aos", "v1.0", "describe")
	assert.False(t, isexist)

	products := library.GetProducts()
	assert.NotNil(t, products)

	library.builtinRepo.Products = []meta.Product{
		{
			Code: "ecs",
		},
	}
	library.PrintProducts()
}

func TestLibrary_PrintProductUsage(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	library.builtinRepo = getRepository()
	err := library.PrintProductUsage("aos", true)
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)

	library.builtinRepo = getRepository()
	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)
}

func TestLibrary_PrintApiUsage(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	library.builtinRepo = getRepository()
	err := library.PrintApiUsage("aos", "DescribeRegions")
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)

	library.builtinRepo = getRepository()
	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)
}

func TestLibrary_PrintApiUsageExamples(t *testing.T) {
	originalHook := meta.HookGetApi
	defer func() {
		meta.HookGetApi = originalHook
	}()

	meta.HookGetApi = func(fn func(productCode string, version string, apiName string) (meta.Api, bool)) func(productCode string, version string, apiName string) (meta.Api, bool) {
		return func(productCode string, version string, apiName string) (meta.Api, bool) {
			return meta.Api{
				Name:       "DescribeInstances",
				Protocol:   "HTTP|HTTPS",
				Method:     "GET|POST",
				Parameters: []meta.Parameter{},
				Example: &meta.ApiExample{
					LegacyCli:  "aliyun ecs DescribeInstances --RegionId cn-hangzhou --PageSize 10",
					UnifiedCli: "aliyun ecs describe-instances --region cn-hangzhou --page-size 10",
				},
			}, true
		}
	}

	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	library.builtinRepo = getRepository()

	err := library.PrintApiUsage("ecs", "DescribeInstances")
	assert.Nil(t, err)
	output := w.String()
	assert.Contains(t, output, "Examples:")
	assert.Contains(t, output, "Legacy CLI:")
	assert.Contains(t, output, "aliyun ecs DescribeInstances --RegionId cn-hangzhou --PageSize 10")
	assert.Contains(t, output, "Recommended unified CLI:")
	assert.Contains(t, output, "aliyun ecs describe-instances --region cn-hangzhou --page-size 10")
}

func TestLibrary_PrintApiUsageExampleWithoutUnifiedCli(t *testing.T) {
	originalHook := meta.HookGetApi
	defer func() {
		meta.HookGetApi = originalHook
	}()

	meta.HookGetApi = func(fn func(productCode string, version string, apiName string) (meta.Api, bool)) func(productCode string, version string, apiName string) (meta.Api, bool) {
		return func(productCode string, version string, apiName string) (meta.Api, bool) {
			return meta.Api{
				Name:       "DescribeInstances",
				Protocol:   "HTTP|HTTPS",
				Method:     "GET|POST",
				Parameters: []meta.Parameter{},
				Example: &meta.ApiExample{
					LegacyCli: "aliyun ecs DescribeInstances --RegionId cn-hangzhou --PageSize 10",
				},
			}, true
		}
	}

	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	library.builtinRepo = getRepository()

	err := library.PrintApiUsage("ecs", "DescribeInstances")
	assert.Nil(t, err)
	output := w.String()
	assert.Contains(t, output, "Examples:")
	assert.Contains(t, output, "Legacy CLI:")
	assert.Contains(t, output, "aliyun ecs DescribeInstances --RegionId cn-hangzhou --PageSize 10")
	assert.NotContains(t, output, "Recommended unified CLI:")
	assert.NotContains(t, output, "aliyun ecs describe-instances")
}

func Test_printParameters(t *testing.T) {
	w := new(bytes.Buffer)
	params := []meta.Parameter{
		{
			Hidden: true,
		},
		{
			Position: "Domain",
		},
		{
			Type:     "RepeatList",
			Required: true,
		},
		{
			Required: false,
		},
		{
			SubParameters: []meta.Parameter{
				{
					Name: "test",
				},
			},
		},
	}
	printParameters(w, params, "", &newmeta.APIDetail{})
}

func getRepository() *meta.Repository {
	repository := meta.LoadRepository()
	return repository
}
