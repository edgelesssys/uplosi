package config

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
)

//go:embed validation.rego
var validationPolicy string

type Validator struct{}

func (v *Validator) Validate(ctx context.Context, config Config) error {
	opts := []func(*rego.Rego){
		rego.Query("data.config.deny"),
		rego.Module("validation.rego", validationPolicy),
		rego.Input(config),
	}
	r := rego.New(opts...)
	res, err := r.Eval(ctx)
	if err != nil {
		return fmt.Errorf("evaluating policy: %w", err)
	}
	fmt.Println(res)

	var resErr error
	for _, result := range res {
		for _, expression := range result.Expressions {
			var expressionValues []any
			if vals, ok := expression.Value.([]any); ok {
				expressionValues = vals
			}

			for _, v := range expressionValues {
				switch val := v.(type) {
				// Policies that only return a single string (e.g. deny[msg])
				case string:
					resErr = errors.Join(resErr, fmt.Errorf(val))
				}
			}
		}
	}

	return resErr
}
