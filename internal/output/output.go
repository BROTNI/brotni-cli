package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

type Printer struct {
	format Format
	out    io.Writer
}

func NewPrinter(format string) *Printer {
	f := Format(format)
	if f != FormatJSON && f != FormatTable {
		f = FormatTable
	}
	return &Printer{format: f, out: os.Stdout}
}

func (p *Printer) Format() Format {
	return p.format
}

func (p *Printer) PrintJSON(v interface{}) error {
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (p *Printer) PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)

	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}

	w.Flush()
}

func (p *Printer) Successf(format string, args ...interface{}) {
	fmt.Fprintf(p.out, format+"\n", args...)
}

func PrintError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

func PrintDebug(enabled bool, format string, args ...interface{}) {
	if enabled {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}
