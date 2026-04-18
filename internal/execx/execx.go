package execx

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/difof/errors"
)

func Output(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = mergeEnv(env)

	output, err := command.Output()
	if err != nil {
		return "", errors.Wrapf(err, "run %s", name)
	}

	return string(output), nil
}

func OutputBytes(ctx context.Context, dir string, env []string, stdin io.Reader, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = mergeEnv(env)
	command.Stdin = stdin

	output, err := command.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "run %s", name)
	}

	return output, nil
}

func CombinedOutput(ctx context.Context, dir string, env []string, stdin io.Reader, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = mergeEnv(env)
	command.Stdin = stdin

	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), errors.Wrapf(err, "run %s", name)
	}

	return string(output), nil
}

func Run(ctx context.Context, dir string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = mergeEnv(env)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr

	if err := command.Run(); err != nil {
		return errors.Wrapf(err, "run %s", name)
	}

	return nil
}

func InteractiveOutput(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	return InteractiveOutputWithInput(ctx, dir, env, os.Stdin, name, args...)
}

func InteractiveOutputWithInput(ctx context.Context, dir string, env []string, stdin io.Reader, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = mergeEnv(env)
	command.Stdin = stdin
	command.Stderr = os.Stderr

	var stdout bytes.Buffer
	command.Stdout = &stdout

	if err := command.Run(); err != nil {
		return stdout.String(), errors.Wrapf(err, "run %s", name)
	}

	return stdout.String(), nil
}

func ExitCode(err error) (int, bool) {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, false
	}

	return exitErr.ExitCode(), true
}

func mergeEnv(extra []string) []string {
	if len(extra) == 0 {
		return os.Environ()
	}

	env := os.Environ()
	env = append(env, extra...)
	return env
}
