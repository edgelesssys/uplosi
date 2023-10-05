package uploader

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		config  Config
		wantErr bool
	}{
		"with subsc": {
			config:  Config{Provider: "azure", Azure: AzureConfig{SubscriptionID: "test"}},
			wantErr: false,
		},
		"without subsc": {
			config:  Config{Provider: "azure", Azure: AzureConfig{SubscriptionID: "test"}},
			wantErr: false,
		},
		"unknown provider": {
			config:  Config{Provider: "foo"},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := context.Background()

			v := Validator{}

			err := v.Validate(ctx, tc.config)
			if err != nil {
				t.Log(err)
			}
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}
