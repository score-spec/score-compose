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
	"log/slog"
	"os"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

type OutputFormatter interface {
	Display()
}

type JSONOutputFormatter[T any] struct {
	Data T
	Out  io.Writer
}

type YAMLOutputFormatter[T any] struct {
	Data T
	Out  io.Writer
}

type TableOutputFormatter struct {
	Headers []string
	Rows    [][]string
	Out     io.Writer
}

func (t *TableOutputFormatter) Display() {
	// Default to stdout if no output is provided
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
	// Default to stdout if no output is provided
	if j.Out == nil {
		j.Out = os.Stdout
	}
	encoder := json.NewEncoder(j.Out)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(j.Data)
	if err != nil {
		slog.Error(err.Error())
	}
}

func (f *YAMLOutputFormatter[T]) Display() {
	// Default to stdout if no output is provided
	if f.Out == nil {
		f.Out = os.Stdout
	}

	encoder := yaml.NewEncoder(f.Out)
	if err := encoder.Encode(f.Data); err != nil {
		slog.Error(err.Error())
	}
}
