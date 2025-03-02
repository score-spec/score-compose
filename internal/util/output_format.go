// Copyright 2024 Humanitec
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

package util

import (
	"encoding/json"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

type OutputFormatter interface {
	Display()
}

type JSONOutputFormatter[T any] struct {
	Data T
	Out  io.Writer
}

type TableOutputFormatter struct {
	Headers []string
	Rows    [][]string
	Out     io.Writer
}

func (t *TableOutputFormatter) Display() {
	if t.Out == nil {
		t.Out = os.Stdout
	}
	table := tablewriter.NewWriter(t.Out)
	table.SetHeader(t.Headers)
	table.AppendBulk(t.Rows)
	table.SetAutoWrapText(false)
	table.SetRowLine(true)
	table.SetCenterSeparator("+")
	table.SetColumnSeparator("|")
	table.SetRowSeparator("-")
	table.Render()
}

func (j *JSONOutputFormatter[T]) Display() {
	if j.Out == nil {
		j.Out = os.Stdout
	}
	json.NewEncoder(j.Out).Encode(j.Data)
}
