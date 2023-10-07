/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncrementSemver(t *testing.T) {
	testCases := []struct {
		ver     string
		want    string
		wantErr bool
	}{
		{ver: "0.0.0", want: "0.0.1"},
		{ver: "0.0.1", want: "0.0.2"},
		{ver: "0.0.9", want: "0.0.10"},
		{ver: "0.0.10", want: "0.0.11"},
		{ver: "1.15", want: "1.15.1"},
		{ver: "1.15.1", want: "1.15.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.ver, func(t *testing.T) {
			assert := assert.New(t)
			got, err := incrementSemver(tc.ver)
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tc.want, got)
		})
	}
}
