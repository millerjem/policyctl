package audit

import (
	"strings"
	"unicode/utf8"
)

const (
	ansiGreen = "\x1b[32m"
	ansiRed   = "\x1b[31m"
	ansiReset = "\x1b[0m"
)

type Column struct {
	Header string
	Width  int
	Value  func(Row) string
}

func DefaultColumns() []Column {
	return []Column{
		{Header: "STATUS", Width: 6, Value: func(row Row) string { return colorStatus(row.Status) }},
		{Header: "NAME", Width: 24, Value: func(row Row) string { return row.Name }},
		{Header: "ACTION", Width: 24, Value: func(row Row) string { return row.Action }},
		{Header: "RESOURCE", Width: 44, Value: func(row Row) string { return row.Resource }},
		{Header: "EXPECTED", Width: 8, Value: func(row Row) string { return row.Expected }},
		{Header: "DECISION", Width: 12, Value: func(row Row) string { return row.Decision }},
		{Header: "MISSING CONTEXT", Width: 15, Value: func(row Row) string { return row.MissingContext }},
	}
}

func RenderTable(rows []Row, columns []Column) string {
	var builder strings.Builder

	writeBorder(&builder, "╭", "┬", "╮", columns)
	writeLine(&builder, headerRow(columns), columns)
	writeBorder(&builder, "├", "┼", "┤", columns)
	for i, row := range rows {
		writeWrappedRow(&builder, row, columns)
		if i == len(rows)-1 {
			continue
		}
		writeBorder(&builder, "├", "┼", "┤", columns)
	}
	writeBorder(&builder, "╰", "┴", "╯", columns)

	return builder.String()
}

func headerRow(columns []Column) Row {
	values := make([]string, len(columns))
	for i, column := range columns {
		values[i] = column.Header
	}
	return rowFromValues(values)
}

func writeWrappedRow(builder *strings.Builder, row Row, columns []Column) {
	cells := make([][]string, len(columns))
	height := 1
	for i, column := range columns {
		lines := wrap(column.Value(row), column.Width)
		cells[i] = lines
		if len(lines) > height {
			height = len(lines)
		}
	}

	for lineIndex := 0; lineIndex < height; lineIndex++ {
		builder.WriteString("│")
		for columnIndex, column := range columns {
			value := ""
			if lineIndex < len(cells[columnIndex]) {
				value = cells[columnIndex][lineIndex]
			}
			builder.WriteString(" ")
			builder.WriteString(padRight(value, column.Width))
			builder.WriteString(" │")
		}
		builder.WriteString("\n")
	}
}

func writeLine(builder *strings.Builder, row Row, columns []Column) {
	writeWrappedRow(builder, row, columns)
}

func writeBorder(builder *strings.Builder, left, middle, right string, columns []Column) {
	builder.WriteString(left)
	for i, column := range columns {
		if i > 0 {
			builder.WriteString(middle)
		}
		builder.WriteString(strings.Repeat("─", column.Width+2))
	}
	builder.WriteString(right)
	builder.WriteString("\n")
}

func wrap(value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{""}
	}

	parts := strings.Fields(value)
	if len(parts) == 0 {
		return []string{""}
	}

	lines := make([]string, 0)
	current := ""
	for _, part := range parts {
		for displayLen(part) > width {
			prefix := take(part, width)
			part = part[len(prefix):]
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, prefix)
		}

		if current == "" {
			current = part
			continue
		}
		if displayLen(current)+1+displayLen(part) <= width {
			current += " " + part
			continue
		}
		lines = append(lines, current)
		current = part
	}
	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func padRight(value string, width int) string {
	padding := width - displayLen(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func displayLen(value string) int {
	return len([]rune(stripANSI(value)))
}

func take(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width])
}

func colorStatus(status string) string {
	switch status {
	case "PASS":
		return ansiGreen + status + ansiReset
	case "FAIL":
		return ansiRed + status + ansiReset
	default:
		return status
	}
}

func stripANSI(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); {
		if value[i] == '\x1b' {
			i++
			if i < len(value) && value[i] == '[' {
				i++
				for i < len(value) {
					b := value[i]
					i++
					if b >= '@' && b <= '~' {
						break
					}
				}
			}
			continue
		}

		r, size := utf8.DecodeRuneInString(value[i:])
		builder.WriteRune(r)
		i += size
	}
	return builder.String()
}

func rowFromValues(values []string) Row {
	row := Row{}
	if len(values) > 0 {
		row.Status = values[0]
	}
	if len(values) > 1 {
		row.Name = values[1]
	}
	if len(values) > 2 {
		row.Action = values[2]
	}
	if len(values) > 3 {
		row.Resource = values[3]
	}
	if len(values) > 4 {
		row.Expected = values[4]
	}
	if len(values) > 5 {
		row.Decision = values[5]
	}
	if len(values) > 6 {
		row.MissingContext = values[6]
	}
	return row
}
