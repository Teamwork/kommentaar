package yamlout

import "testing"

func TestMarshal(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{[]string{"="}, "- \"=\"\n"},
		{[]string{"!=", "==", "IN"}, "- '!='\n- ==\n- IN\n"},
		{[]string{"yes", "1:20", "<<"}, "- \"yes\"\n- \"1:20\"\n- \"<<\"\n"},
		{map[string]any{"a": map[string]string{"b": "="}}, "a:\n  b: \"=\"\n"},
		{map[string]string{"<<": "x"}, "\"<<\": x\n"},
		{map[string]any{"a": map[string]string{"d": " x\ny", "e": "="}}, "a:\n  d: |2-\n     x\n    y\n  e: \"=\"\n"},
	}

	for _, tt := range tests {
		got, err := Marshal(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tt.want {
			t.Errorf("Marshal(%v)\ngot:  %q\nwant: %q", tt.in, got, tt.want)
		}
	}
}
