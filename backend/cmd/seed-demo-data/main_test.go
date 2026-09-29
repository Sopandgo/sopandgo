package main

import "testing"

func TestSeedDemoEnabled(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{value: "", want: true},
		{value: "true", want: true},
		{value: "TRUE", want: true},
		{value: "false", want: false},
		{value: "FALSE", want: false},
		{value: " false ", want: false},
		{value: "0", want: true},
		{value: "no", want: true},
	}

	for _, tc := range cases {
		if got := seedDemoEnabled(tc.value); got != tc.want {
			t.Errorf("seedDemoEnabled(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
