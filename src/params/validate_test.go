package params

import (
	"testing"
)

func TestParams_Valid(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr bool
	}{
		{
			name:    "valid default params",
			params:  Default,
			wantErr: false,
		},
		{
			name:    "invalid zero value",
			params:  Params{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := tt.params.Valid()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Valid() unexpectedly rejected %+v with error: %v", tt.params, gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("Valid() succeeded unexpectedly on %+v", tt.params)
			}
		})
	}
}
