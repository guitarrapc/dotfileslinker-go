package main

import "testing"

func TestContainsFlagForce(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "force flag", args: []string{"--force"}, want: true},
		{name: "force flag is case insensitive", args: []string{"--FORCE"}, want: true},
		{name: "legacy force value is not accepted", args: []string{"--force=y"}, want: false},
		{name: "force flag is absent", args: []string{"--verbose"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsFlag(tt.args, "--force"); got != tt.want {
				t.Fatalf("containsFlag(%v, %q) = %v, want %v", tt.args, "--force", got, tt.want)
			}
		})
	}
}
