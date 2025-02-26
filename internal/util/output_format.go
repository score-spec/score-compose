package util

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/olekukonko/tablewriter"
)

type OutputFormatter interface {
	Display()
}

type JSONOutputFormatter[T any] struct {
	Data T
}

type TableOutputFormatter struct {
	Headers []string
	Rows    [][]string
}

func (t *TableOutputFormatter) Display() {
	table := tablewriter.NewWriter(os.Stdout)
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
	output, err := json.MarshalIndent(j.Data, "", "  ")
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to marshal data: %v", err))
		return
	}
	fmt.Println(string(output))
}
