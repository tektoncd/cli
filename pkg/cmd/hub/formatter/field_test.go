// Copyright © 2020 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/cli/pkg/cmd/hub/gen/http/resource/client"
)

func TestFormatName(t *testing.T) {
	name := FormatName("abc", "0.1")
	assert.Equal(t, name, "abc (0.1)")
}

func TestFormatCatalogName(t *testing.T) {
	catalog := FormatCatalogName("foo")
	assert.Equal(t, catalog, "Foo")
}

func TestFormatDesc(t *testing.T) {

	// Description greater than 40 char
	desc := FormatDesc("Buildah task builds source into a container image and then pushes it to a container registry.", 40)
	assert.Equal(t, "Buildah task builds source into a conta...", desc)

	// Description less than 40 char
	desc = FormatDesc("Buildah task builds images.", 40)
	assert.Equal(t, "Buildah task builds images.", desc)

	// No Description
	desc = FormatDesc("", 40)
	assert.Equal(t, "---", desc)
}

func TestFormatTags(t *testing.T) {

	tagName1 := "tag1"
	tagName2 := "tag2"

	res := []*client.TagResponseBody{
		{
			Name: &tagName1,
		},
		{
			Name: &tagName2,
		},
	}

	tags := FormatTags(res)
	assert.Equal(t, "tag1, tag2", tags)

	// No Tags
	tags = FormatTags(nil)
	assert.Equal(t, "---", tags)
}

func TestFormatCategories(t *testing.T) {

	categoryName1 := "category1"
	categoryName2 := "category2"

	res := []*client.CategoryResponseBody{
		{
			Name: &categoryName1,
		},
		{
			Name: &categoryName2,
		},
	}

	categories := FormatCategories(res)
	assert.Equal(t, "category1, category2", categories)

	// No Categories
	categories = FormatTags(nil)
	assert.Equal(t, "---", categories)
}

func TestFormatPlatforms(t *testing.T) {

	pName1 := "linux/amd64"
	pName2 := "linux/s390x"

	pRes := []*client.PlatformResponseBody{
		{
			Name: &pName1,
		},
		{
			Name: &pName2,
		},
	}

	platforms := FormatPlatforms(pRes)
	assert.Equal(t, "linux/amd64, linux/s390x", platforms)

	// No Tags
	platforms = FormatTags(nil)
	assert.Equal(t, "---", platforms)
}

func TestWrapText(t *testing.T) {

	// Description of resource with just summa
	desc := WrapText("The Buildpacks task builds source into a container image and pushes it to a registry, using Cloud Native Buildpacks.", 80, 16)
	assert.Equal(t, "The Buildpacks task builds source into a container image and"+"\n"+" pushes it to a registry, using Cloud Native Buildpacks.", desc)

	// Description of resource with summary and description
	desc = WrapText("Buildah task builds source into a container image and then pushes it to a container registry."+"\n"+"Buildah Task builds source into a container image using Project Atomic's Buildah build tool.It uses Buildah's support for building from Dockerfiles, using its buildah bud command.This command executes the directives in the Dockerfile to assemble a container image, then pushes that image to a container registry.", 80, 16)
	assert.Equal(t, "Buildah task builds source into a container image and then"+"\n"+" pushes it to a container registry. Buildah Task builds source into a container"+"\n"+" image using Project Atomic's Buildah build tool.It uses Buildah's support for"+"\n"+" building from Dockerfiles, using its buildah bud command.This command executes"+"\n"+" the directives in the Dockerfile to assemble a container image, then pushes"+"\n"+" that image to a container registry.", desc)
}

func TestFormatVersion(t *testing.T) {
	got := FormatVersion("0.1", false, false)
	assert.Equal(t, "0.1", got)

	got = FormatVersion("0.1", false, true)
	assert.Equal(t, "0.1 (Deprecated)", got)

	got = FormatVersion("0.1", true, false)
	assert.Equal(t, "0.1 (Latest)", got)
}

func TestIcon(t *testing.T) {
	got := Icon("bullet")
	assert.Equal(t, "∙ ", got)
}

func TestDecorate(t *testing.T) {
	got := DecorateAttr("bold", "world")
	assert.Equal(t, "world", got)
}
