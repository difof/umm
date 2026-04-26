package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIIntegration(t *testing.T) {
	binary := buildBinary(t)

	t.Run("normal no-ui stat flow", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		output := runCmd(t, binary, "--root", root, "--no-ui", "--pattern", "needle", "--only-stat", "list")
		if !strings.Contains(output, filepath.Join(root, "one.txt")) {
			t.Fatalf("expected stat output to contain file path, got %q", output)
		}
	})

	t.Run("hidden emitter plus preview flow", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		emit := runCmd(t, binary, "__emit-search", "--root", root, "--pattern", `one\.txt`, "--only-filename")
		line := strings.TrimSpace(strings.SplitN(emit, "\n", 2)[0])
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			t.Fatalf("unexpected emitter output: %q", emit)
		}

		preview := runCmd(t, binary, "preview", parts[0], parts[1])
		if !strings.Contains(preview, "needle") {
			t.Fatalf("expected preview output to contain file contents, got %q", preview)
		}
	})

	t.Run("interactive normal flow with fake fzf", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--pattern", "needle")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=stdin-first",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive normal run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		if !strings.Contains(string(logged), filepath.Join(root, "one.txt")) {
			t.Fatalf("expected fake editor to receive selected file, got %q", string(logged))
		}
	})

	t.Run("interactive normal flow passes themed args to fzf", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		xdg := t.TempDir()
		writeFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: lattice-dark\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		fzfLog := filepath.Join(binDir, "fzf.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--pattern", "needle")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=stdin-first",
			"FAKE_FZF_LOG="+fzfLog,
			"XDG_CONFIG_HOME="+xdg,
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive themed run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(fzfLog)
		if err != nil {
			t.Fatalf("ReadFile fzf log: %v", err)
		}
		text := string(logged)
		if !strings.Contains(text, "--color=dark,") || !strings.Contains(text, "--separator=─") || !strings.Contains(text, "--preview-window=top:60%") {
			t.Fatalf("expected themed fzf args, got %q", text)
		}
	})

	t.Run("UMM_THEME overrides configured theme when valid", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		xdg := t.TempDir()
		writeFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: lattice-dark\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		fzfLog := filepath.Join(binDir, "fzf.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--pattern", "needle")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=stdin-first",
			"FAKE_FZF_LOG="+fzfLog,
			"XDG_CONFIG_HOME="+xdg,
			"UMM_THEME=lattice-light",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive env-themed run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(fzfLog)
		if err != nil {
			t.Fatalf("ReadFile fzf log: %v", err)
		}
		text := string(logged)
		if !strings.Contains(text, "--color=light,") {
			t.Fatalf("expected UMM_THEME to force light theme, got %q", text)
		}
	})

	t.Run("interactive git flow passes themed args to fzf", func(t *testing.T) {
		root := t.TempDir()
		runGit(t, root, "init")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "Test User")
		writeFile(t, filepath.Join(root, "tracked.txt"), "hello\n")
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-m", "initial commit")

		xdg := t.TempDir()
		writeFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: lattice-dark\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		fzfLog := filepath.Join(binDir, "fzf.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--git", "--git-mode", "tracked")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=ctrl-o-stdin-first",
			"FAKE_FZF_LOG="+fzfLog,
			"XDG_CONFIG_HOME="+xdg,
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive themed git run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(fzfLog)
		if err != nil {
			t.Fatalf("ReadFile fzf log: %v", err)
		}
		text := string(logged)
		if !strings.Contains(text, "--color=dark,") || !strings.Contains(text, "--prompt=> Git: ") || !strings.Contains(text, "--preview-window=top:60%") {
			t.Fatalf("expected themed git fzf args, got %q", text)
		}
	})

	t.Run("interactive startup with empty query streams initial results", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "first\n")
		writeFile(t, filepath.Join(root, "two.txt"), "second\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root)
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=stdin-first",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive startup run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		content := string(logged)
		if !strings.Contains(content, filepath.Join(root, "one.txt")) && !strings.Contains(content, filepath.Join(root, "two.txt")) {
			t.Fatalf("expected initial startup results to open a file, got %q", content)
		}
	})

	t.Run("interactive normal flow handles spaced query reload", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "two words here\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--pattern", "two words")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=change-reload-first",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive spaced-query run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		if !strings.Contains(string(logged), filepath.Join(root, "one.txt")) {
			t.Fatalf("expected fake editor to receive spaced-query result, got %q", string(logged))
		}
	})

	t.Run("interactive normal flow handles shell metachar query", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "semi:semicolon\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--pattern", "semi:semicolon")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=change-reload-first",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive metachar-query run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		if !strings.Contains(string(logged), filepath.Join(root, "one.txt")) {
			t.Fatalf("expected fake editor to receive metachar-query result, got %q", string(logged))
		}
	})

	t.Run("interactive dirname flow with fake fzf", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "cmd", "tool.txt"), "cmd tool\n")

		binDir := t.TempDir()
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--only-dirname", "--pattern", "cmd")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR=true",
			"FAKE_FZF_MODE=stdin-first",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive dirname run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		want := filepath.Join(root, "cmd")
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected dirname output to contain %q, got %q", want, stdout.String())
		}
	})

	t.Run("git no-ui flow", func(t *testing.T) {
		root := t.TempDir()
		runGit(t, root, "init")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "Test User")
		writeFile(t, filepath.Join(root, "tracked.txt"), "hello\n")
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-m", "initial commit")
		runGit(t, root, "tag", "v1.0.0")

		output := runCmd(t, binary, "--root", root, "--git", "--no-ui", "--pattern", `tag:\s+v1\.0\.0`)
		if !strings.Contains(output, "tag:") || !strings.Contains(output, "v1.0.0") {
			t.Fatalf("expected git output to contain tag summary, got %q", output)
		}
	})

	t.Run("non-interactive flow works with broken selected theme", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		xdg := t.TempDir()
		writeFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: missing-theme\n")

		cmd := exec.Command(binary, "--root", root, "--no-ui", "--pattern", "needle", "--only-stat", "list")
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdg)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("no-ui run with broken theme failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), filepath.Join(root, "one.txt")) {
			t.Fatalf("expected no-ui output, got %q", stdout.String())
		}
	})

	t.Run("interactive flow fails clearly with broken selected theme", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "one.txt"), "needle\n")

		xdg := t.TempDir()
		writeFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: missing-theme\n")

		binDir := t.TempDir()
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--pattern", "needle")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR=true",
			"XDG_CONFIG_HOME="+xdg,
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			t.Fatal("expected interactive run to fail with broken theme")
		}
		if !strings.Contains(stderr.String(), "missing-theme") {
			t.Fatalf("expected missing-theme error, got %q", stderr.String())
		}
	})

	t.Run("theme recovery commands work with broken selected theme", func(t *testing.T) {
		xdg := t.TempDir()
		writeFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: missing-theme\n")

		cmd := exec.Command(binary, "theme", "list")
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdg)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("theme list recovery failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "lattice-dark") {
			t.Fatalf("expected theme list output, got %q", stdout.String())
		}

		cmd = exec.Command(binary, "theme", "set", "lattice-dark")
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdg)
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("theme set recovery failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		data, err := os.ReadFile(filepath.Join(xdg, "umm", "umm.yml"))
		if err != nil {
			t.Fatalf("ReadFile config: %v", err)
		}
		if !strings.Contains(string(data), "theme: lattice-dark") {
			t.Fatalf("expected recovered config theme, got %q", string(data))
		}
	})

	t.Run("no-ui open-ask stat covers all matches", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "a.go"), "package a\n")
		writeFile(t, filepath.Join(root, "b.go"), "package b\n")

		cmd := exec.Command(binary, "--root", root, "--no-ui", "--open-ask", "--pattern", "package")
		cmd.Env = append(os.Environ(), "EDITOR=true", "UMM_TEST_OPEN_ASK_CHOICE=stat")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("no-ui open-ask stat run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), filepath.Join(root, "a.go")) || !strings.Contains(stdout.String(), filepath.Join(root, "b.go")) {
			t.Fatalf("expected stat output for all matches, got %q", stdout.String())
		}
	})

	t.Run("no-ui open-ask editor opens all file matches", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "a.go"), "package a\n")
		writeFile(t, filepath.Join(root, "b.go"), "package b\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)

		cmd := exec.Command(binary, "--root", root, "--no-ui", "--open-ask", "--pattern", "package")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"UMM_TEST_OPEN_ASK_CHOICE=editor",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("no-ui open-ask editor run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		content := string(logged)
		if !strings.Contains(content, filepath.Join(root, "a.go")) || !strings.Contains(content, filepath.Join(root, "b.go")) {
			t.Fatalf("expected fake editor to receive all file matches, got %q", content)
		}
	})

	t.Run("editor command with args is supported", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "main.go"), "package main\n")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)

		cmd := exec.Command(binary, "--root", root, "--no-ui", "--pattern", "package")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor")+" --wait",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("editor-with-args run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		content := string(logged)
		if !strings.Contains(content, "--wait") || !strings.Contains(content, filepath.Join(root, "main.go")) {
			t.Fatalf("expected fake editor to receive fixed args and file path, got %q", content)
		}
	})

	t.Run("no-ui open-ask system opens all compatible matches", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "a.go"), "package a\n")
		writeFile(t, filepath.Join(root, "b.go"), "package b\n")

		binDir := t.TempDir()
		systemLog := filepath.Join(binDir, "system.log")
		installFakeSystemOpeners(t, binDir, systemLog)

		cmd := exec.Command(binary, "--root", root, "--no-ui", "--open-ask", "--pattern", "package")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR=true",
			"UMM_TEST_OPEN_ASK_CHOICE=system",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("no-ui open-ask system run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(systemLog)
		if err != nil {
			t.Fatalf("ReadFile system log: %v", err)
		}
		content := string(logged)
		if !strings.Contains(content, filepath.Join(root, "a.go")) || !strings.Contains(content, filepath.Join(root, "b.go")) {
			t.Fatalf("expected fake system opener to receive all matches, got %q", content)
		}
	})

	t.Run("interactive git ctrl-o with fake fzf", func(t *testing.T) {
		root := t.TempDir()
		runGit(t, root, "init")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "Test User")
		writeFile(t, filepath.Join(root, "tracked.txt"), "hello\n")
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-m", "initial commit")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)
		installFakeFZF(t, filepath.Join(binDir, "fzf"))

		cmd := exec.Command(binary, "--root", root, "--git", "--git-mode", "tracked")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"FAKE_FZF_MODE=ctrl-o-stdin-first",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("interactive git run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		if !strings.Contains(string(logged), filepath.Join(root, "tracked.txt")) {
			t.Fatalf("expected fake editor to receive tracked file, got %q", string(logged))
		}
	})

	t.Run("git no-ui open-ask editor uses tracked subset only", func(t *testing.T) {
		root := t.TempDir()
		runGit(t, root, "init")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "Test User")
		writeFile(t, filepath.Join(root, "tracked.txt"), "hello\n")
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-m", "initial commit")

		binDir := t.TempDir()
		editorLog := filepath.Join(binDir, "editor.log")
		installFakeEditor(t, filepath.Join(binDir, "fake-editor"), editorLog)

		cmd := exec.Command(binary, "--root", root, "--git", "--no-ui", "--open-ask", "--pattern", ".")
		cmd.Env = append(os.Environ(),
			"PATH="+binDir+":"+os.Getenv("PATH"),
			"EDITOR="+filepath.Join(binDir, "fake-editor"),
			"UMM_TEST_OPEN_ASK_CHOICE=editor",
		)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("git no-ui open-ask run failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		}

		logged, err := os.ReadFile(editorLog)
		if err != nil {
			t.Fatalf("ReadFile editor log: %v", err)
		}
		content := string(logged)
		if !strings.Contains(content, filepath.Join(root, "tracked.txt")) {
			t.Fatalf("expected tracked file to be opened, got %q", content)
		}
		if strings.Contains(content, "commit:") {
			t.Fatalf("expected non-file git objects to be excluded from open action, got %q", content)
		}
	})
}

func buildBinary(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "umm-test")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	return binary
}

func runCmd(t *testing.T, binary string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "EDITOR=true")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %s failed: %v\nstdout:\n%s\nstderr:\n%s", binary, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func installFakeEditor(t *testing.T, path string, logPath string) {
	t.Helper()
	script := "#!/bin/sh\n: > \"$FAKE_EDITOR_LOG\"\nfor arg in \"$@\"; do\n  printf '%s\\n' \"$arg\" >> \"$FAKE_EDITOR_LOG\"\ndone\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile fake editor: %v", err)
	}
	t.Setenv("FAKE_EDITOR_LOG", logPath)
}

func installFakeSystemOpeners(t *testing.T, dir string, logPath string) {
	t.Helper()
	script := "#!/bin/sh\nfor arg in \"$@\"; do\n  printf '%s\\n' \"$arg\" >> \"$FAKE_SYSTEM_LOG\"\ndone\n"
	for _, name := range []string{"open", "xdg-open"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatalf("WriteFile fake %s: %v", name, err)
		}
	}
	t.Setenv("FAKE_SYSTEM_LOG", logPath)
}

func installFakeFZF(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
mode="${FAKE_FZF_MODE:-stdin-first}"
if [ -n "$FAKE_FZF_LOG" ]; then
  : > "$FAKE_FZF_LOG"
  for arg in "$@"; do
    printf '%s\n' "$arg" >> "$FAKE_FZF_LOG"
  done
fi
query=""
start_cmd=""
change_cmd=""
prev=""
for arg in "$@"; do
  case "$arg" in
    --query=*) query="${arg#--query=}" ;;
  esac
  if [ "$prev" = "--bind" ]; then
    case "$arg" in
      start:reload:*) start_cmd="${arg#start:reload:}" ;;
      change:reload:*) change_cmd="${arg#change:reload:}" ;;
    esac
    prev=""
    continue
  fi
  prev="$arg"
done

case "$mode" in
  start-reload-first)
    if [ -n "$start_cmd" ]; then
      quoted_query=$(printf '%s' "$query" | sed "s/'/'\\''/g")
      eval_cmd=$(printf '%s' "$start_cmd" | sed "s/{q}/'$quoted_query'/g")
      /bin/sh -c "$eval_cmd" | sed -n '1p'
    fi
    ;;
  change-reload-first)
    if [ -n "$change_cmd" ]; then
      quoted_query=$(printf '%s' "$query" | sed "s/'/'\\''/g")
      eval_cmd=$(printf '%s' "$change_cmd" | sed "s/{q}/'$quoted_query'/g")
      /bin/sh -c "$eval_cmd" | sed -n '1p'
    fi
    ;;
  ctrl-o-stdin-first)
    printf 'ctrl-o\n'
    sed -n '1p'
    ;;
  stdin-first|*)
    sed -n '1p'
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile fake fzf: %v", err)
	}
}
