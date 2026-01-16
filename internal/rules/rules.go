package rules

import (
	"strings"

	"github.com/tony-montemuro/http/internal/lws"
)

type Extractor string

func (e Extractor) Extract() []string {
	rules := []string{}

	parts := strings.Split(string(e), ",")
	for i, part := range parts {
		if i+1 == len(parts) {
			rules = append(rules, lws.TrimLeft(part))
		} else {
			rules = append(rules, lws.Trim(part))
		}
	}

	return rules
}
