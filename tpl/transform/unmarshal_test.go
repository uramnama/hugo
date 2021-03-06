// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transform

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/gohugoio/hugo/common/hugio"

	"github.com/gohugoio/hugo/media"

	"github.com/gohugoio/hugo/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

const (
	testJSON = `
	
{
    "ROOT_KEY": {
        "title": "example glossary",
		"GlossDiv": {
            "title": "S",
			"GlossList": {
                "GlossEntry": {
                    "ID": "SGML",
					"SortAs": "SGML",
					"GlossTerm": "Standard Generalized Markup Language",
					"Acronym": "SGML",
					"Abbrev": "ISO 8879:1986",
					"GlossDef": {
                        "para": "A meta-markup language, used to create markup languages such as DocBook.",
						"GlossSeeAlso": ["GML", "XML"]
                    },
					"GlossSee": "markup"
                }
            }
        }
    }
}

	`
)

var _ resource.ReadSeekCloserResource = (*testContentResource)(nil)

type testContentResource struct {
	content string
	mime    media.Type

	key string
}

func (t testContentResource) ReadSeekCloser() (hugio.ReadSeekCloser, error) {
	return hugio.NewReadSeekerNoOpCloserFromString(t.content), nil
}

func (t testContentResource) MediaType() media.Type {
	return t.mime
}

func (t testContentResource) Key() string {
	return t.key
}

func TestUnmarshal(t *testing.T) {

	v := viper.New()
	ns := New(newDeps(v))
	assert := require.New(t)

	assertSlogan := func(m map[string]interface{}) {
		assert.Equal("Hugo Rocks!", m["slogan"])
	}

	for i, test := range []struct {
		data    interface{}
		options interface{}
		expect  interface{}
	}{
		{`{ "slogan": "Hugo Rocks!" }`, nil, func(m map[string]interface{}) {
			assertSlogan(m)
		}},
		{`slogan: "Hugo Rocks!"`, nil, func(m map[string]interface{}) {
			assertSlogan(m)
		}},
		{`slogan = "Hugo Rocks!"`, nil, func(m map[string]interface{}) {
			assertSlogan(m)
		}},
		{testContentResource{key: "r1", content: `slogan: "Hugo Rocks!"`, mime: media.YAMLType}, nil, func(m map[string]interface{}) {
			assertSlogan(m)
		}},
		{testContentResource{key: "r1", content: `{ "slogan": "Hugo Rocks!" }`, mime: media.JSONType}, nil, func(m map[string]interface{}) {
			assertSlogan(m)
		}},
		{testContentResource{key: "r1", content: `slogan = "Hugo Rocks!"`, mime: media.TOMLType}, nil, func(m map[string]interface{}) {
			assertSlogan(m)
		}},
		{testContentResource{key: "r1", content: `1997,Ford,E350,"ac, abs, moon",3000.00
1999,Chevy,"Venture ""Extended Edition""","",4900.00`, mime: media.CSVType}, nil, func(r [][]string) {
			assert.Equal(2, len(r))
			first := r[0]
			assert.Equal(5, len(first))
			assert.Equal("Ford", first[1])
		}},
		{testContentResource{key: "r1", content: `a;b;c`, mime: media.CSVType}, map[string]interface{}{"delimiter": ";"}, func(r [][]string) {
			assert.Equal(r, [][]string{[]string{"a", "b", "c"}})

		}},
		{"a,b,c", nil, func(r [][]string) {
			assert.Equal(r, [][]string{[]string{"a", "b", "c"}})

		}},
		{"a;b;c", map[string]interface{}{"delimiter": ";"}, func(r [][]string) {
			assert.Equal(r, [][]string{[]string{"a", "b", "c"}})

		}},
		{testContentResource{key: "r1", content: `
% This is a comment
a;b;c`, mime: media.CSVType}, map[string]interface{}{"DElimiter": ";", "Comment": "%"}, func(r [][]string) {
			assert.Equal(r, [][]string{[]string{"a", "b", "c"}})

		}},
		// errors
		{"thisisnotavaliddataformat", nil, false},
		{testContentResource{key: "r1", content: `invalid&toml"`, mime: media.TOMLType}, nil, false},
		{testContentResource{key: "r1", content: `unsupported: MIME"`, mime: media.CalendarType}, nil, false},
		{"thisisnotavaliddataformat", nil, false},
		{`{ notjson }`, nil, false},
		{tstNoStringer{}, nil, false},
	} {
		errMsg := fmt.Sprintf("[%d]", i)

		ns.cache.Clear()

		var args []interface{}

		if test.options != nil {
			args = []interface{}{test.options, test.data}
		} else {
			args = []interface{}{test.data}
		}

		result, err := ns.Unmarshal(args...)

		if b, ok := test.expect.(bool); ok && !b {
			assert.Error(err, errMsg)
		} else if fn, ok := test.expect.(func(m map[string]interface{})); ok {
			assert.NoError(err, errMsg)
			m, ok := result.(map[string]interface{})
			assert.True(ok, errMsg)
			fn(m)
		} else if fn, ok := test.expect.(func(r [][]string)); ok {
			assert.NoError(err, errMsg)
			r, ok := result.([][]string)
			assert.True(ok, errMsg)
			fn(r)
		} else {
			assert.NoError(err, errMsg)
			assert.Equal(test.expect, result, errMsg)
		}

	}
}

func BenchmarkUnmarshalString(b *testing.B) {
	v := viper.New()
	ns := New(newDeps(v))

	const numJsons = 100

	var jsons [numJsons]string
	for i := 0; i < numJsons; i++ {
		jsons[i] = strings.Replace(testJSON, "ROOT_KEY", fmt.Sprintf("root%d", i), 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ns.Unmarshal(jsons[rand.Intn(numJsons)])
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("no result")
		}
	}
}

func BenchmarkUnmarshalResource(b *testing.B) {
	v := viper.New()
	ns := New(newDeps(v))

	const numJsons = 100

	var jsons [numJsons]testContentResource
	for i := 0; i < numJsons; i++ {
		key := fmt.Sprintf("root%d", i)
		jsons[i] = testContentResource{key: key, content: strings.Replace(testJSON, "ROOT_KEY", key, 1), mime: media.JSONType}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ns.Unmarshal(jsons[rand.Intn(numJsons)])
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("no result")
		}
	}
}
