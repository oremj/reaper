package filter

import (
	"fmt"
	"strconv"
	"strings"
)

type Filterable interface {
	Filter(Filter) bool
}

func PrintFilters(filters map[string]Filter) string {
	var filterText []string
	for _, filter := range filters {
		filterText = append(filterText, fmt.Sprintf("%s(%s)", filter.Function, filter.Value))
	}
	return strings.Join(filterText, ", ")
}

type Filter struct {
	Function string
	Value    string
}

func (f *Filter) Int64Value() (int64, error) {
	// parseint -> base 10, 64 bit int
	i, err := strconv.ParseInt(f.Value, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}
