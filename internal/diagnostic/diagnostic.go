package diagnostic

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Diagnostic struct {
	Code      string   `json:"code"`
	Severity  Severity `json:"severity"`
	File      string   `json:"file,omitempty"`
	Line      int      `json:"line,omitempty"`
	Column    int      `json:"column,omitempty"`
	Message   string   `json:"message"`
	Fix       string   `json:"fix,omitempty"`
	Reference string   `json:"reference,omitempty"`
	Kind      string   `json:"kind,omitempty"`
}

type Result struct {
	OK          bool         `json:"ok"`
	Strict      bool         `json:"strict"`
	Root        string       `json:"root"`
	Config      string       `json:"config,omitempty"`
	Entry       string       `json:"entry,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

func ApplySeverity(diags []Diagnostic, strict bool) {
	for i := range diags {
		if diags[i].Code == "CL005" {
			diags[i].Severity = SeverityError
			continue
		}
		if strict {
			diags[i].Severity = SeverityError
		} else {
			diags[i].Severity = SeverityWarning
		}
	}
}

func Sort(diags []Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		a := diags[i]
		b := diags[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Reference < b.Reference
	})
}

func WriteHuman(w io.Writer, result Result) error {
	if len(result.Diagnostics) == 0 {
		_, err := fmt.Fprintln(w, "context-lint: ok")
		return err
	}

	for i, diag := range result.Diagnostics {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		location := diag.File
		if diag.Line > 0 {
			location += fmt.Sprintf(":%d", diag.Line)
			if diag.Column > 0 {
				location += fmt.Sprintf(":%d", diag.Column)
			}
		}
		if location != "" {
			if _, err := fmt.Fprintf(w, "%s %s %s\n", diag.Severity, diag.Code, location); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "%s %s\n", diag.Severity, diag.Code); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintf(w, "  %s\n", diag.Message); err != nil {
			return err
		}
		if diag.Fix != "" {
			fix := diag.Fix
			if !strings.HasPrefix(strings.ToLower(fix), "fix:") {
				fix = "Fix: " + fix
			}
			if _, err := fmt.Fprintf(w, "  %s\n", fix); err != nil {
				return err
			}
		}
	}

	return nil
}

func WriteJSON(w io.Writer, result Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
