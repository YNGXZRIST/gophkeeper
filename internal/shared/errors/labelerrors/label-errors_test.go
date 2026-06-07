package labelerrors

import (
	"errors"
	"testing"
)

func TestLabelError_Error(t *testing.T) {
	type fields struct {
		Label string
		Err   error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "success",
			fields: fields{
				Label: "label",
				Err:   errors.New("error"),
			},
			want: "[label]: error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := LabelError{
				Label: tt.fields.Label,
				Err:   tt.fields.Err,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLabelError_Unwrap(t *testing.T) {
	type fields struct {
		Label string
		Err   error
	}
	err := errors.New("error")
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "success",
			fields: fields{
				Label: "label",
				Err:   err,
			},
			wantErr: err,
		},
		{
			name: "empty error",
			fields: fields{
				Label: "label",
				Err:   nil,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := LabelError{
				Label: tt.fields.Label,
				Err:   tt.fields.Err,
			}
			err := e.Unwrap()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewLabelError(t *testing.T) {
	type args struct {
		label string
		err   error
	}
	err := errors.New("error")
	tests := []struct {
		name    string
		args    args
		wantErr LabelError
	}{
		{
			name: "empty error",
			args: args{
				label: "label",
				err:   nil,
			},
			wantErr: LabelError{
				Label: "label",
				Err:   nil,
			},
		},
		{
			name: "not empty error",
			args: args{
				label: "label",
				err:   err,
			},
			wantErr: LabelError{
				Label: "label",
				Err:   err,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewLabelError(tt.args.label, tt.args.err)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewLabelError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
