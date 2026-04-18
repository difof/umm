package resultfmt

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/difof/errors"
)

type Kind string

const (
	KindFile Kind = "file"
	KindDir  Kind = "dir"
	KindGit  Kind = "git"
)

type Result struct {
	Kind        Kind   `json:"kind"`
	PreviewMode string `json:"preview_mode"`
	Display     string `json:"display"`

	Path string `json:"path,omitempty"`
	Line int    `json:"line,omitempty"`

	Repo     string `json:"repo,omitempty"`
	GitType  string `json:"git_type,omitempty"`
	GitRef   string `json:"git_ref,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Current  bool   `json:"current,omitempty"`
	SubValue string `json:"sub_value,omitempty"`
}

func EncodeLine(result Result) (string, error) {
	encoded, err := EncodeMeta(result)
	if err != nil {
		return "", errors.Wrap(err)
	}

	display := sanitizeDisplay(result.Display)
	return result.PreviewMode + "\t" + encoded + "\t" + display, nil
}

func EncodeMeta(result Result) (string, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return "", errors.Wrap(err)
	}

	return base64.RawStdEncoding.EncodeToString(payload), nil
}

func DecodeMeta(meta string) (Result, error) {
	var result Result

	payload, err := base64.RawStdEncoding.DecodeString(meta)
	if err != nil {
		return result, errors.Wrapf(err, "decode result metadata")
	}

	if err := json.Unmarshal(payload, &result); err != nil {
		return result, errors.Wrapf(err, "decode result metadata")
	}

	return result, nil
}

func DecodeLine(line string) (Result, error) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) < 2 {
		return Result{}, errors.Newf("invalid result line %q", line)
	}

	result, err := DecodeMeta(parts[1])
	if err != nil {
		return Result{}, errors.Wrap(err)
	}

	return result, nil
}

func DecodeLines(text string) ([]Result, error) {
	results := []Result{}
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		result, err := DecodeLine(line)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, result)
	}

	return results, nil
}

func sanitizeDisplay(value string) string {
	value = strings.ReplaceAll(value, "\t", "    ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
