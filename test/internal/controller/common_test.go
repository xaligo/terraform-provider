package controller_test

import (
	"testing"

	"github.com/xaligo/terraform-provider/internal/controller"
)

func TestIsCLIInvocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arguments []string
		want      bool
	}{
		{arguments: nil, want: false},
		{arguments: []string{"-debug"}, want: false},
		{arguments: []string{"convert"}, want: true},
		{arguments: []string{"serve"}, want: true},
		{arguments: []string{"version"}, want: true},
		{arguments: []string{"--help"}, want: true},
	}
	for _, test := range tests {
		if got := controller.IsCLIInvocation(test.arguments); got != test.want {
			t.Errorf("IsCLIInvocation(%v) = %v, want %v", test.arguments, got, test.want)
		}
	}
}
