package editor

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ummconfig "github.com/difof/umm/internal/config"
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

func TestResolve(t *testing.T) {
	t.Run("preserves explicit editor path for basename profile matches", func(t *testing.T) {
		editorPath := filepath.Join(t.TempDir(), "nvim")
		old := os.Getenv("EDITOR")
		t.Cleanup(func() {
			if old == "" {
				_ = os.Unsetenv("EDITOR")
				return
			}
			_ = os.Setenv("EDITOR", old)
		})
		if err := os.Setenv("EDITOR", editorPath+" --clean"); err != nil {
			t.Fatalf("Setenv EDITOR: %v", err)
		}

		cmd, err := Resolve(ummconfig.Defaults())
		if err != nil {
			t.Fatalf("Resolve returned error: %v", err)
		}
		if cmd.Name != editorPath {
			t.Fatalf("Resolve() name = %q, want %q", cmd.Name, editorPath)
		}
		if !reflect.DeepEqual(cmd.Args, []string{"--clean"}) {
			t.Fatalf("Resolve() args = %#v, want %#v", cmd.Args, []string{"--clean"})
		}
		if cmd.Profile == nil {
			t.Fatal("expected built-in profile to still be applied")
		}
	})

	t.Run("exact editor aliases still resolve to configured command", func(t *testing.T) {
		old := os.Getenv("EDITOR")
		t.Cleanup(func() {
			if old == "" {
				_ = os.Unsetenv("EDITOR")
				return
			}
			_ = os.Setenv("EDITOR", old)
		})
		if err := os.Setenv("EDITOR", "my-editor --wait"); err != nil {
			t.Fatalf("Setenv EDITOR: %v", err)
		}

		cfg := ummconfig.Defaults()
		cfg.Editors["my-editor"] = ummconfig.Editor{Cmd: "actual-editor", Args: []string{"--foreground"}}

		cmd, err := Resolve(cfg)
		if err != nil {
			t.Fatalf("Resolve returned error: %v", err)
		}
		if cmd.Name != "actual-editor" {
			t.Fatalf("Resolve() name = %q, want %q", cmd.Name, "actual-editor")
		}
		wantArgs := []string{"--wait", "--foreground"}
		if !reflect.DeepEqual(cmd.Args, wantArgs) {
			t.Fatalf("Resolve() args = %#v, want %#v", cmd.Args, wantArgs)
		}
	})
}

func TestOpenUsesProvidedStdio(t *testing.T) {
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "stdio.log")
	scriptPath := filepath.Join(binDir, "fake-editor")
	script := "#!/bin/sh\nread input\nprintf 'in=%s\\n' \"$input\" > \"" + logPath + "\"\nprintf 'out-ok\\n'\nprintf 'err-ok\\n' >&2\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", scriptPath, err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Open(t.Context(), Command{Name: scriptPath}, []Target{{Path: "/tmp/file.txt"}}, strings.NewReader("stdin-ok\n"), &stdout, &stderr); err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if stdout.String() != "out-ok\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.String() != "err-ok\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", logPath, err)
	}
	if string(logged) != "in=stdin-ok\n" {
		t.Fatalf("stdin log = %q", string(logged))
	}
}
