package db

import (
	"reflect"
	"testing"
)

func TestNewCfg(t *testing.T) {
	type args struct {
		dns string
	}
	tests := []struct {
		name string
		args args
		want *Config
	}{
		{
			name: "TestNewCfg",
			args: args{
				"test.dns",
			},
			want: &Config{
				DNS:        "test.dns",
				MaxRetries: MaxRetries,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCfg(tt.args.dns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCfg() = %v, want %v", got, tt.want)
			}
		})
	}
}
