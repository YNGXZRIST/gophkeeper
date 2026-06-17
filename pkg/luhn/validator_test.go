package luhn

import "testing"

func Test_Validate(t *testing.T) {

	tests := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "valid 16 digits",
			str:  "5062821234567892",
			want: true,
		},
		{
			name: "valid 15 digits (Amex)",
			str:  "378282246310005",
			want: true,
		},
		{
			name: "invalid",
			str:  "5062 8217 3456 7892",
			want: false,
		},
		{
			name: "invalid check digit",
			str:  "5062821234567893",
			want: false,
		},
		{
			name: "empty",
			str:  "",
			want: false,
		},
		{
			name: "non-digits",
			str:  "1234a5678",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Validate(tt.str); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
