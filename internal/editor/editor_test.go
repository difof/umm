package editor

import (
	"reflect"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name   string
		editor string
		line   int
		want   []string
	}{
		{name: "nvim line", editor: "nvim", line: 12, want: []string{"+12", "file.go"}},
		{name: "code line", editor: "code", line: 12, want: []string{"--goto", "file.go:12"}},
		{name: "subl line", editor: "subl", line: 12, want: []string{"file.go:12"}},
		{name: "default no line", editor: "custom", line: 0, want: []string{"file.go"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildArgs(test.editor, "file.go", test.line)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("BuildArgs() = %v, want %v", got, test.want)
			}
		})
	}
}
