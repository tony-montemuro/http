package parser

import (
	"strings"

	"github.com/tony-montemuro/http/internal/lws"
)

type rulesExtractor string

func (s rulesExtractor) extract() []string {
	rules := []string{}

	parts := strings.Split(string(s), ",")
	for i, part := range parts {
		if i+1 == len(parts) {
			rules = append(rules, lws.TrimLeft(part))
		} else {
			rules = append(rules, lws.Trim(part))
		}
	}

	return rules
}
