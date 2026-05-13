package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Bundle struct {
	defaultLocale string
	catalogs      map[string]map[string]string
}

func Load(dir string, defaultLocale string) (*Bundle, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "locales"
	}
	if strings.TrimSpace(defaultLocale) == "" {
		defaultLocale = "en"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read locales directory: %w", err)
	}

	catalogs := make(map[string]map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		locale := strings.TrimSuffix(entry.Name(), ".json")
		catalog, err := loadCatalog(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load locale %s: %w", locale, err)
		}
		catalogs[locale] = catalog
	}

	if len(catalogs) == 0 {
		return nil, fmt.Errorf("no locale files found in %s", dir)
	}
	if _, ok := catalogs[defaultLocale]; !ok {
		return nil, fmt.Errorf("default locale %s not found", defaultLocale)
	}

	return &Bundle{defaultLocale: defaultLocale, catalogs: catalogs}, nil
}

func (b *Bundle) DefaultLocale() string {
	return b.defaultLocale
}

func (b *Bundle) Resolve(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if _, ok := b.catalogs[locale]; ok {
		return locale
	}
	return b.defaultLocale
}

func (b *Bundle) T(locale string, key string, params map[string]string) string {
	catalog := b.catalogs[b.defaultLocale]
	if candidate, ok := b.catalogs[b.Resolve(locale)]; ok {
		catalog = candidate
	}

	value, ok := catalog[key]
	if !ok {
		if fallbackValue, fallbackOK := b.catalogs[b.defaultLocale][key]; fallbackOK {
			value = fallbackValue
		} else {
			return key
		}
	}

	for name, replacement := range params {
		value = strings.ReplaceAll(value, "{{"+name+"}}", replacement)
	}
	return value
}

func loadCatalog(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var catalog map[string]string
	if err := json.Unmarshal(content, &catalog); err != nil {
		return nil, err
	}
	return catalog, nil
}
