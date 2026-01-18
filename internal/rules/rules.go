package rules

import (
	"strings"

	"github.com/tony-montemuro/http/internal/lws"
)

func Extract(s string) []string {
	rules := []string{}

	parts := strings.Split(s, ",")
	for i, part := range parts {
		if i+1 == len(parts) {
			rules = append(rules, lws.TrimLeft(part))
		} else {
			rules = append(rules, lws.Trim(part))
		}
	}

	return rules
}
