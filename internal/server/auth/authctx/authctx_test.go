package authctx

import (
	"context"
	"testing"
)

func TestUserIDFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		want   string
		wantOK bool
	}{
		{
			name:   "present",
			ctx:    ContextWithUserID(context.Background(), "u1"),
			want:   "u1",
			wantOK: true,
		},
		{
			name:   "absent",
			ctx:    context.Background(),
			want:   "",
			wantOK: false,
		},
		{
			name:   "empty value",
			ctx:    ContextWithUserID(context.Background(), ""),
			want:   "",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := UserIDFromContext(tt.ctx)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("UserIDFromContext() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
