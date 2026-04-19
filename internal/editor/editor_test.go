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

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Command
		wantErr bool
	}{
		{name: "simple command", input: "nvim", want: Command{Name: "nvim"}},
		{name: "command with args", input: "code --wait", want: Command{Name: "code", Args: []string{"--wait"}}},
		{name: "quoted arg", input: "emacsclient -c --alternate-editor='nvim -f'", want: Command{Name: "emacsclient", Args: []string{"-c", "--alternate-editor=nvim -f"}}},
		{name: "quoted path", input: "'/tmp/my editor' --wait", want: Command{Name: "/tmp/my editor", Args: []string{"--wait"}}},
		{name: "unterminated quote", input: "'nvim", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Parse() = %#v, want %#v", got, test.want)
			}
		})
	}
}
