package theme

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/difof/errors"
)

type Origin string

const (
	OriginBuiltin Origin = "builtin"
	OriginUser    Origin = "user"
)

type Entry struct {
	Name      string
	Variant   Variant
	Origin    Origin
	Path      string
	Raw       []byte
	Theme     Theme
	Invalid   bool
	LoadErr   error
	Effective bool
	Shadowed  bool
}

type Catalog struct {
	entries  []Entry
	resolved map[string]Entry
}

func UserDir(configDir string) string {
	return filepath.Join(configDir, "themes")
}

func Discover(configDir string) (Catalog, error) {
	builtins, err := LoadBuiltins()
	if err != nil {
		return Catalog{}, errors.Wrap(err)
	}

	entries := make([]Entry, 0, len(builtins))
	resolved := map[string]Entry{}
	for _, builtin := range builtins {
		entry := Entry{
			Name:    builtin.Theme.Name,
			Variant: builtin.Theme.Variant,
			Origin:  OriginBuiltin,
			Path:    builtin.FileName,
			Raw:     append([]byte(nil), builtin.Raw...),
			Theme:   builtin.Theme,
		}
		entries = append(entries, entry)
		resolved[entry.Name] = entry
	}

	if configDir == "" {
		return finalizeCatalog(entries, resolved), nil
	}

	userEntries, err := loadUserEntries(UserDir(configDir))
	if err != nil {
		return Catalog{}, errors.Wrap(err)
	}
	entries = append(entries, userEntries...)
	for _, entry := range userEntries {
		resolved[entry.Name] = entry
	}

	return finalizeCatalog(entries, resolved), nil
}

func (catalog Catalog) Entries() []Entry {
	result := make([]Entry, len(catalog.entries))
	copy(result, catalog.entries)
	return result
}

func (catalog Catalog) Resolve(name string) (Entry, error) {
	entry, ok := catalog.resolved[name]
	if !ok {
		return Entry{}, errors.Newf("theme %q was not found", name)
	}
	if entry.Invalid {
		return Entry{}, errors.Wrapf(entry.LoadErr, "theme %q is invalid", name)
	}
	return entry, nil
}

func loadUserEntries(dir string) ([]Entry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err)
	}

	loaded := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		path := filepath.Join(dir, entry.Name())
		loadedEntry := Entry{
			Name:   base,
			Origin: OriginUser,
			Path:   path,
		}

		raw, err := osReadFile(path)
		if err != nil {
			loadedEntry.Invalid = true
			loadedEntry.LoadErr = errors.Wrapf(err, "read user theme %s", path)
			loaded = append(loaded, loadedEntry)
			continue
		}

		loadedEntry.Raw = append([]byte(nil), raw...)

		parsed, decodeErr := Decode(raw)
		if decodeErr != nil {
			loadedEntry.Invalid = true
			loadedEntry.LoadErr = errors.Wrapf(decodeErr, "decode user theme %s", path)
			loaded = append(loaded, loadedEntry)
			continue
		}
		if parsed.Name != base {
			loadedEntry.Invalid = true
			loadedEntry.LoadErr = errors.Newf("user theme %s declares name %q; want %q", path, parsed.Name, base)
			loaded = append(loaded, loadedEntry)
			continue
		}

		loadedEntry.Name = parsed.Name
		loadedEntry.Variant = parsed.Variant
		loadedEntry.Theme = parsed
		loaded = append(loaded, loadedEntry)
	}

	sort.Slice(loaded, func(i int, j int) bool {
		return loaded[i].Path < loaded[j].Path
	})

	return loaded, nil
}

var osReadFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func finalizeCatalog(entries []Entry, resolved map[string]Entry) Catalog {
	final := make([]Entry, len(entries))
	copy(final, entries)
	for i := range final {
		resolvedEntry, ok := resolved[final[i].Name]
		if !ok {
			continue
		}
		if final[i].Origin == resolvedEntry.Origin && final[i].Path == resolvedEntry.Path {
			final[i].Effective = true
			continue
		}
		if final[i].Origin == OriginBuiltin && resolvedEntry.Origin == OriginUser {
			final[i].Shadowed = true
		}
	}
	sort.Slice(final, func(i int, j int) bool {
		if final[i].Name != final[j].Name {
			return final[i].Name < final[j].Name
		}
		if final[i].Origin != final[j].Origin {
			return final[i].Origin < final[j].Origin
		}
		return final[i].Path < final[j].Path
	})
	return Catalog{entries: final, resolved: resolved}
}

var _ fs.DirEntry
