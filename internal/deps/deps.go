package deps

import (
	"os/exec"
	"runtime"

	"github.com/difof/errors"
)

func Has(tool string) bool {
	_, err := exec.LookPath(tool)
	return err == nil
}

func Require(tool string, feature string) error {
	if Has(tool) {
		return nil
	}

	return errors.Newf("missing required dependency %q for %s", tool, feature)
}

func SystemOpenCommand() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "open", nil
	case "linux":
		return "xdg-open", nil
	default:
		return "", errors.Newf("system open is not supported on %s", runtime.GOOS)
	}
}
