package project

import (
	"fmt"
	"os"
	"strings"
)

// Severity classifies how a BaseUnit Violation should be treated during synthesis.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}
	return "warning"
}

// Violation describes a single policy failure detected by a Validator.
type Violation struct {
	File     string
	Rule     string
	Message  string
	Severity Severity
}

func (v Violation) Error() string {
	return fmt.Sprintf("%s [%s]: %s", v.File, v.Rule, v.Message)
}

// Validator checks that synthesized project output meets policy requirements.
// Validators run after all files are written, asserting postconditions on the
// output rather than influencing what gets written.
type Validator interface {
	Validate(meta Meta, rc map[string]any, dir string) ([]Violation, error)
}

func (p *Project) runValidators(meta Meta, rc map[string]any) error {
	var errs []string
	for _, v := range p.validators {
		violations, err := v.Validate(meta, rc, p.root)
		if err != nil {
			return err
		}
		for _, viol := range violations {
			switch viol.Severity {
			case SeverityError:
				errs = append(errs, viol.Error())
			case SeverityWarning:
				fmt.Fprintf(os.Stderr, "baseunit warning: %s\n", viol.Error())
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("baseunit violations:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}
