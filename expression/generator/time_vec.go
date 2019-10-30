// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// +build ignore

package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"log"
	"path/filepath"
	"text/template"

	. "github.com/pingcap/tidb/expression/generator/helper"
)

var addTime = template.Must(template.New("").Parse(`// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by go generate in expression/generator; DO NOT EDIT.

package expression

import (
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/parser/terror"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
)

{{ define "SetNull" }}{{if .Output.Fixed}}result.SetNull(i, true){{else}}result.AppendNull(){{end}} // fixed: {{.Output.Fixed }}{{ end }}
{{ define "ConvertStringToDuration" }}
		{{ if ne .SigName "builtinAddStringAndStringSig" }}
		if !isDuration(arg1) {
			{{ template "SetNull" . }}
			continue
		}{{ end }}
		sc := b.ctx.GetSessionVars().StmtCtx
		arg1Duration, err := types.ParseDuration(sc, arg1, {{if eq .Output.TypeName "String"}}getFsp4TimeAddSub{{else}}types.GetFsp{{end}}(arg1))
		if err != nil {
			if terror.ErrorEqual(err, types.ErrTruncatedWrongVal) {
				sc.AppendWarning(err)
				{{ template "SetNull" . }}
				continue
			}
			return err
		}
{{ end }}
{{ define "strDurationAddDuration" }}
		var output string
		if isDuration(arg0) {
			output, err = strDurationAddDuration(sc, arg0, arg1Duration)
			if err != nil {
				if terror.ErrorEqual(err, types.ErrTruncatedWrongVal) {
					sc.AppendWarning(err)
					{{ template "SetNull" . }}
					continue
				}
				return err
			}
		} else {
			output, err = strDatetimeAddDuration(sc, arg0, arg1Duration)
			if err != nil {
				return err
			}
		}	
{{ end }}

{{ range . }}
{{ if .AllNull}}
func (b *{{.SigName}}) vecEval{{ .Output.TypeName }}(input *chunk.Chunk, result *chunk.Column) error {
	n := input.NumRows()
	{{ if .Output.Fixed }}
	result.Resize{{ .Output.TypeNameInColumn }}(n, true)
	{{ else }}
	result.Reserve{{ .Output.TypeNameInColumn }}(n)
	for i := 0; i < n; i++ { result.AppendNull() }
	{{ end }}
	return nil
}
{{ else }}
func (b *{{.SigName}}) vecEval{{ .Output.TypeName }}(input *chunk.Chunk, result *chunk.Column) error {
	n := input.NumRows()
{{ $reuse := (and (eq .TypeA.TypeName .Output.TypeName) .TypeA.Fixed) }}
{{ if $reuse }}
	if err := b.args[0].VecEval{{ .TypeA.TypeName }}(b.ctx, input, result); err != nil {
		return err
	}
	buf0 := result
{{ else }}
	buf0, err := b.bufAllocator.get(types.ET{{.TypeA.ETName}}, n)
	if err != nil {
		return err
	}
	defer b.bufAllocator.put(buf0)
	if err := b.args[0].VecEval{{ .TypeA.TypeName }}(b.ctx, input, buf0); err != nil {
		return err
	}
{{ end }}

{{ if eq .SigName "builtinAddStringAndStringSig" }}
	arg1Type := b.args[1].GetType()
	if mysql.HasBinaryFlag(arg1Type.Flag) {
		result.Reserve{{ .Output.TypeNameInColumn }}(n)
		for i := 0; i < n; i++ {
			result.AppendNull()
		}
		return nil
	}
{{ end }}

	buf1, err := b.bufAllocator.get(types.ET{{.TypeB.ETName}}, n)
	if err != nil {
		return err
	}
	defer b.bufAllocator.put(buf1)
	if err := b.args[1].VecEval{{ .TypeB.TypeName }}(b.ctx, input, buf1); err != nil {
		return err
	}

{{ if $reuse }}
	result.MergeNulls(buf1)
{{ else if .Output.Fixed}}
	result.Resize{{ .Output.TypeNameInColumn }}(n, false)
	result.MergeNulls(buf0, buf1)
{{ else }}
	result.Reserve{{ .Output.TypeNameInColumn}}(n)
{{ end }}

{{ if .TypeA.Fixed }}
	arg0s := buf0.{{.TypeA.TypeNameInColumn}}s()
{{ end }}
{{ if .TypeB.Fixed }}
	arg1s := buf1.{{.TypeB.TypeNameInColumn}}s()
{{ end }}
{{ if .Output.Fixed }}
	resultSlice := result.{{.Output.TypeNameInColumn}}s()
{{ end }}
	for i := 0; i < n; i++ {
		{{ if .Output.Fixed }}
		if result.IsNull(i) {
			continue
		}
		{{ else }}
		if buf0.IsNull(i) || buf1.IsNull(i) {
			result.AppendNull()
			continue
		}
		{{ end }}

		// get arg0 & arg1
		{{ if .TypeA.Fixed }}
		arg0 := arg0s[i]
		{{ else }}
		arg0 := buf0.Get{{ .TypeA.TypeNameInColumn }}(i)
		{{ end }}
		{{ if .TypeB.Fixed }}
		arg1 := arg1s[i]
		{{ else }}
		arg1 := buf1.Get{{ .TypeB.TypeNameInColumn }}(i)
		{{ end }}

		// calculate
	{{ if eq .SigName "builtinAddDatetimeAndDurationSig" }}
		output, err := arg0.Add(b.ctx.GetSessionVars().StmtCtx, types.Duration{Duration: arg1, Fsp: -1})
		if err != nil {
			return err
		}
	{{ else if eq .SigName "builtinAddDatetimeAndStringSig" }}
		{{ template "ConvertStringToDuration" . }}
		output, err := arg0.Add(sc, arg1Duration)
		if err != nil {
			return err
		}
	{{ else if eq .SigName "builtinAddDurationAndDurationSig" }}
		output, err := types.AddDuration(arg0, arg1)
		if err != nil {
			return err
		}
	{{ else if eq .SigName "builtinAddDurationAndStringSig" }}
		{{ template "ConvertStringToDuration" . }}
		output, err := types.AddDuration(arg0, arg1Duration.Duration)
		if err != nil {
			return err
		}
	{{ else if eq .SigName "builtinAddStringAndDurationSig" }}
		sc := b.ctx.GetSessionVars().StmtCtx
		fsp1 := int8(b.args[1].GetType().Decimal)
		arg1Duration := types.Duration{Duration: arg1, Fsp: fsp1}
		{{ template "strDurationAddDuration" . }}
	{{ else if eq .SigName "builtinAddStringAndStringSig" }}
		{{ template "ConvertStringToDuration" . }}
		{{ template "strDurationAddDuration" . }}
	{{ else if eq .SigName "builtinAddDateAndDurationSig" }}
		fsp0 := int8(b.args[0].GetType().Decimal)
		fsp1 := int8(b.args[1].GetType().Decimal)
		arg1Duration := types.Duration{Duration: arg1, Fsp: fsp1}
		sum, err := types.Duration{Duration: arg0, Fsp: fsp0}.Add(arg1Duration)
		if err != nil {
			return err
		}
		output := sum.String()
	{{ else if eq .SigName "builtinAddDateAndStringSig" }}
		{{ template "ConvertStringToDuration" . }}
		fsp0 := int8(b.args[0].GetType().Decimal)
		sum, err := types.Duration{Duration: arg0, Fsp: fsp0}.Add(arg1Duration)
		if err != nil {
			return err
		}
		output := sum.String()
	{{ end }}

		// commit result
	{{ if .Output.Fixed }}
		resultSlice[i] = output	
	{{ else }}
		result.Append{{ .Output.TypeNameInColumn }}(output)
	{{ end }}
	}
	return nil
}
{{ end }}{{/* if .AllNull */}}

func (b *{{.SigName}}) vectorized() bool {
	return true
}
{{ end }}{{/* range */}}
`))

var testFile = template.Must(template.New("").Parse(`// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by go generate in expression/generator; DO NOT EDIT.

package expression

import (
	"testing"

	. "github.com/pingcap/check"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/tidb/types"
)

type gener struct {
	defaultGener
}

func (g gener) gen() interface{} {
	result := g.defaultGener.gen()
	if _, ok := result.(string); ok {
		dg := &defaultGener{eType: types.ETDuration, nullRation: 0}
		d := dg.gen().(types.Duration)
		if int8(d.Duration)%2 == 0 {
			d.Fsp = 0
		} else {
			d.Fsp = 1
		}
		result = d.String()
	}
	return result
}

{{/* Add more test cases here if we have more functions in this file */}}
var vecBuiltin{{.Category}}GeneratedCases = map[string][]vecExprBenchCase{
{{ range .Functions }}
	ast.{{.FuncName}}: {
	{{ range .Sigs }} // {{ .SigName }}
		{
			retEvalType: types.ET{{ .Output.ETName }}, 
			childrenTypes: []types.EvalType{types.ET{{ .TypeA.ETName }}, types.ET{{ .TypeB.ETName }}},
			{{ if ne .FieldTypeA "" }}
			childrenFieldTypes: []*types.FieldType{types.NewFieldType(mysql.Type{{.FieldTypeA}}), types.NewFieldType(mysql.Type{{.FieldTypeB}})},
			{{ end }}
			geners: []dataGenerator{
				gener{defaultGener{eType: types.ET{{.TypeA.ETName}}, nullRation: 0.2}},
				gener{defaultGener{eType: types.ET{{.TypeB.ETName}}, nullRation: 0.2}},
			},
		},
	{{ end }}
{{ end }}
	},
}

func (s *testEvaluatorSuite) TestVectorizedBuiltin{{.Category}}EvalOneVecGenerated(c *C) {
	testVectorizedEvalOneVec(c, vecBuiltin{{.Category}}GeneratedCases)
}

func (s *testEvaluatorSuite) TestVectorizedBuiltin{{.Category}}FuncGenerated(c *C) {
	testVectorizedBuiltinFunc(c, vecBuiltin{{.Category}}GeneratedCases)
}

func BenchmarkVectorizedBuiltin{{.Category}}EvalOneVecGenerated(b *testing.B) {
	benchmarkVectorizedEvalOneVec(b, vecBuiltin{{.Category}}GeneratedCases)
}

func BenchmarkVectorizedBuiltin{{.Category}}FuncGenerated(b *testing.B) {
	benchmarkVectorizedBuiltinFunc(b, vecBuiltin{{.Category}}GeneratedCases)
}
`))

var addTimeSigsTmpl = []sig{
	{SigName: "builtinAddDatetimeAndDurationSig", TypeA: TypeDatetime, TypeB: TypeDuration, Output: TypeDatetime},
	{SigName: "builtinAddDatetimeAndStringSig", TypeA: TypeDatetime, TypeB: TypeString, Output: TypeDatetime},
	{SigName: "builtinAddDurationAndDurationSig", TypeA: TypeDuration, TypeB: TypeDuration, Output: TypeDuration},
	{SigName: "builtinAddDurationAndStringSig", TypeA: TypeDuration, TypeB: TypeString, Output: TypeDuration},
	{SigName: "builtinAddStringAndDurationSig", TypeA: TypeString, TypeB: TypeDuration, Output: TypeString},
	{SigName: "builtinAddStringAndStringSig", TypeA: TypeString, TypeB: TypeString, Output: TypeString},
	{SigName: "builtinAddDateAndDurationSig", TypeA: TypeDuration, TypeB: TypeDuration, Output: TypeString, FieldTypeA: "Date", FieldTypeB: "Duration"},
	{SigName: "builtinAddDateAndStringSig", TypeA: TypeDuration, TypeB: TypeString, Output: TypeString, FieldTypeA: "Date", FieldTypeB: "String"},

	{SigName: "builtinAddTimeDateTimeNullSig", TypeA: TypeDatetime, TypeB: TypeDatetime, Output: TypeDatetime, AllNull: true},
	{SigName: "builtinAddTimeStringNullSig", TypeA: TypeDatetime, TypeB: TypeDatetime, Output: TypeString, AllNull: true, FieldTypeA: "Date", FieldTypeB: "Datetime"},
	{SigName: "builtinAddTimeDurationNullSig", TypeA: TypeDuration, TypeB: TypeDatetime, Output: TypeDuration, AllNull: true},
}

type sig struct {
	SigName                string
	TypeA, TypeB, Output   TypeContext
	FieldTypeA, FieldTypeB string // Optional
	AllNull                bool
}

type function struct {
	FuncName string
	Sigs     []sig
}

var tmplVal = struct {
	Category  string
	Functions []function
}{
	Category: "Time",
	Functions: []function{
		{FuncName: "AddTime", Sigs: addTimeSigsTmpl},
	},
}

func generateDotGo(fileName string) error {
	w := new(bytes.Buffer)
	err := addTime.Execute(w, addTimeSigsTmpl)
	if err != nil {
		return err
	}
	data, err := format.Source(w.Bytes())
	if err != nil {
		log.Println("[Warn]", fileName+": gofmt failed", err)
		data = w.Bytes() // write original data for debugging
	}
	return ioutil.WriteFile(fileName, data, 0644)
}

func generateTestDotGo(fileName string) error {
	w := new(bytes.Buffer)
	err := testFile.Execute(w, tmplVal)
	if err != nil {
		return err
	}
	data, err := format.Source(w.Bytes())
	if err != nil {
		log.Println("[Warn]", fileName+": gofmt failed", err)
		data = w.Bytes() // write original data for debugging
	}
	return ioutil.WriteFile(fileName, data, 0644)
}

// generateOneFile generate one xxx.go file and the associated xxx_test.go file.
func generateOneFile(fileNamePrefix string) (err error) {

	err = generateDotGo(fileNamePrefix + ".go")
	if err != nil {
		return
	}
	err = generateTestDotGo(fileNamePrefix + "_test.go")
	return
}

func main() {
	var err error
	outputDir := "."
	err = generateOneFile(filepath.Join(outputDir, "builtin_time_vec_generated"))
	if err != nil {
		log.Fatalln("generateOneFile", err)
	}
}