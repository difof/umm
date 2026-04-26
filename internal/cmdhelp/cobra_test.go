package cmdhelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAttachAppendixAppendsAfterDefaultHelp(t *testing.T) {
	cmd := &cobra.Command{Use: "demo", Short: "demo command"}
	AttachAppendix(cmd, Document{Title: "Appendix"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	text := stdout.String()
	defaultHelp := strings.Index(text, "demo command")
	appendix := strings.Index(text, "Appendix")
	if defaultHelp == -1 || appendix == -1 {
		t.Fatalf("expected default help and appendix, got %q", text)
	}
	if appendix <= defaultHelp {
		t.Fatalf("expected appendix after default help, got %q", text)
	}
}

func TestAttachAppendixDoesNotAppendToSubcommandHelp(t *testing.T) {
	root := &cobra.Command{Use: "demo", Short: "demo root"}
	child := &cobra.Command{Use: "child", Short: "demo child"}
	root.AddCommand(child)
	AttachAppendix(root, Document{Title: "Appendix"})

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"child", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	text := stdout.String()
	if strings.Contains(text, "Appendix") {
		t.Fatalf("expected child help without appendix, got %q", text)
	}
	if !strings.Contains(text, "demo child") {
		t.Fatalf("expected child help text, got %q", text)
	}
}
