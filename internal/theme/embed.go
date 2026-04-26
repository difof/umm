package theme

import (
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/difof/errors"
	builtinthemes "github.com/difof/umm/themes"
)

type BuiltinFile struct {
	FileName string
	Raw      []byte
	Theme    Theme
}

func LoadBuiltins() ([]BuiltinFile, error) {
	entries, err := fs.ReadDir(builtinthemes.FS, ".")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	builtins := make([]BuiltinFile, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".yml" {
			continue
		}

		raw, err := fs.ReadFile(builtinthemes.FS, entry.Name())
		if err != nil {
			return nil, errors.Wrapf(err, "read built-in theme %s", entry.Name())
		}

		parsed, err := Decode(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "decode built-in theme %s", entry.Name())
		}

		base := strings.TrimSuffix(entry.Name(), path.Ext(entry.Name()))
		if parsed.Name != base {
			return nil, errors.Newf("built-in theme %s declares name %q; want %q", entry.Name(), parsed.Name, base)
		}
		if _, ok := seen[parsed.Name]; ok {
			return nil, errors.Newf("duplicate built-in theme name %q", parsed.Name)
		}
		seen[parsed.Name] = struct{}{}

		builtins = append(builtins, BuiltinFile{
			FileName: entry.Name(),
			Raw:      raw,
			Theme:    parsed,
		})
	}

	sort.Slice(builtins, func(i int, j int) bool {
		return builtins[i].FileName < builtins[j].FileName
	})

	return builtins, nil
}
