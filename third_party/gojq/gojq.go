package gojq

import (
	"fmt"
	"strings"
)

type step struct {
	key     string
	iterate bool
}

type Query struct {
	steps []step
}

type Iter struct {
	values []any
	index  int
}

func Parse(expr string) (*Query, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" || expr[0] != '.' {
		return nil, fmt.Errorf("expression must start with '.'")
	}

	parts := strings.Split(expr[1:], ".")
	steps := make([]step, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		item := step{key: part}
		if strings.HasSuffix(part, "[]") {
			item.key = strings.TrimSuffix(part, "[]")
			item.iterate = true
		}
		if item.key == "" {
			return nil, fmt.Errorf("invalid jq segment %q", part)
		}
		steps = append(steps, item)
	}
	return &Query{steps: steps}, nil
}

func (q *Query) Run(input any) *Iter {
	values := []any{input}
	for _, step := range q.steps {
		nextValues := make([]any, 0)
		for _, current := range values {
			object, ok := current.(map[string]any)
			if !ok {
				return &Iter{values: []any{fmt.Errorf("cannot access key %q on non-object", step.key)}}
			}
			value, ok := object[step.key]
			if !ok {
				continue
			}
			if step.iterate {
				items, ok := value.([]any)
				if !ok {
					return &Iter{values: []any{fmt.Errorf("cannot iterate over key %q", step.key)}}
				}
				nextValues = append(nextValues, items...)
				continue
			}
			nextValues = append(nextValues, value)
		}
		values = nextValues
	}
	return &Iter{values: values}
}

func (it *Iter) Next() (any, bool) {
	if it.index >= len(it.values) {
		return nil, false
	}
	value := it.values[it.index]
	it.index++
	return value, true
}

