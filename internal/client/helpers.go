package client

import (
	"github.com/lib/pq"
	"strings"
)

func pgQuoteListOfLiterals(list []string) string {
	quoted := make([]string, len(list))
	for i, item := range list {
		quoted[i] = pq.QuoteLiteral(item)
	}
	return strings.Join(quoted, ", ")
}
