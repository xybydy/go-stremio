package types

// Manifest describes the capabilities of the addon.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/manifest.md
type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	// One of the following is required
	// Note: Can only have one in code because of how Go (de-)serialization works
	// Resources     []string       `json:"resources,omitempty"`
	ResourceItems []ResourceItem `json:"resources,omitempty"`

	Types    []string      `json:"types"` // Stremio supports "movie", "series", "channel" and "tv"
	Catalogs []CatalogItem `json:"catalogs"`

	// Optional
	IDprefixes    []string              `json:"idPrefixes,omitempty"`
	Background    string                `json:"background,omitempty"` // URL
	Logo          string                `json:"logo,omitempty"`       // URL
	ContactEmail  string                `json:"contactEmail,omitempty"`
	BehaviorHints ManifestBehaviorHints `json:"behaviorHints,omitempty"`
	AddonCatalogs []CatalogItem         `json:"addonCatalogs,omitempty"`

	Config []ConfigItem `json:"config,omitempty"`
}

// Clone returns a deep copy of m.
// We're not using one of the deep copy libraries because only few are maintained and even they have issues.
func (m Manifest) Clone() Manifest {
	var resourceItems []ResourceItem
	if m.ResourceItems != nil {
		resourceItems = make([]ResourceItem, len(m.ResourceItems))
		for i, resourceItem := range m.ResourceItems {
			resourceItems[i] = resourceItem.Clone()
		}
	}

	var types []string
	if m.Types != nil {
		types = make([]string, len(m.Types))
		copy(types, m.Types)
	}

	var catalogs []CatalogItem
	if m.Catalogs != nil {
		catalogs = make([]CatalogItem, len(m.Catalogs))
		for i, catalog := range m.Catalogs {
			catalogs[i] = catalog.Clone()
		}
	}

	var addonCatalogs []CatalogItem
	if m.AddonCatalogs != nil {
		addonCatalogs = make([]CatalogItem, len(m.AddonCatalogs))
		for i, catalog := range m.AddonCatalogs {
			addonCatalogs[i] = catalog.Clone()
		}
	}

	var configs []ConfigItem
	if m.Config != nil {
		configs = make([]ConfigItem, len(m.Config))
		for i, config := range m.Config {
			configs[i] = config.Clone()
		}
	}

	var idPrefixes []string
	if m.IDprefixes != nil {
		idPrefixes = make([]string, len(m.IDprefixes))
		copy(idPrefixes, m.IDprefixes)
	}

	return Manifest{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Version:     m.Version,

		ResourceItems: resourceItems,

		Types:    types,
		Catalogs: catalogs,

		IDprefixes:    idPrefixes,
		Background:    m.Background,
		Logo:          m.Logo,
		ContactEmail:  m.ContactEmail,
		BehaviorHints: m.BehaviorHints,
		AddonCatalogs: m.AddonCatalogs,
		Config:        m.Config,
	}
}

type ManifestBehaviorHints struct {
	// Note: Must include `omitempty`, otherwise it will be included if this struct is used in another one, even if the field of the containing struct is marked as `omitempty`
	Adult        bool `json:"adult,omitempty"`
	P2P          bool `json:"p2p,omitempty"`
	Configurable bool `json:"configurable,omitempty"`
	// If you set this to true, it will be true for the "/manifest.json" endpoint, but false for the "/:userData/manifest.json" endpoint, because otherwise Stremio won't show the "Install" button in its UI.
	ConfigurationRequired bool `json:"configurationRequired,omitempty"`
}

type ResourceItem struct {
	Name  string   `json:"name"`
	Types []string `json:"types"` // Stremio supports "movie", "series", "channel" and "tv"

	// Optional
	IDprefixes []string `json:"idPrefixes,omitempty"`
}

func (ri ResourceItem) Clone() ResourceItem {
	var types []string
	if ri.Types != nil {
		types = make([]string, len(ri.Types))
		copy(types, ri.Types)
	}

	var idPrefixes []string
	if ri.IDprefixes != nil {
		idPrefixes = make([]string, len(ri.IDprefixes))
		copy(idPrefixes, ri.IDprefixes)
	}

	return ResourceItem{
		Name:  ri.Name,
		Types: types,

		IDprefixes: idPrefixes,
	}
}

// CatalogItem represents a catalog.
type CatalogItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`

	// Optional
	Extra []ExtraItem `json:"extra,omitempty"`
}

func (ci CatalogItem) Clone() CatalogItem {
	var extras []ExtraItem
	if ci.Extra != nil {
		extras = make([]ExtraItem, len(ci.Extra))
		for i, extra := range ci.Extra {
			extras[i] = extra.Clone()
		}
	}

	return CatalogItem{
		Type: ci.Type,
		ID:   ci.ID,
		Name: ci.Name,

		Extra: extras,
	}
}

type ExtraItem struct {
	Name string `json:"name"`

	// Optional
	IsRequired   bool     `json:"isRequired,omitempty"`
	Options      []string `json:"options,omitempty"`
	OptionsLimit int      `json:"optionsLimit,omitempty"`
}

func (ei ExtraItem) Clone() ExtraItem {
	var options []string
	if ei.Options != nil {
		options = make([]string, len(ei.Options))
		copy(options, ei.Options)
	}

	return ExtraItem{
		Name: ei.Name,

		IsRequired:   ei.IsRequired,
		Options:      options,
		OptionsLimit: ei.OptionsLimit,
	}
}

type ConfigItem struct {
	ConfKey      string   `json:"key,omitempty"`
	ConfType     string   `json:"type,omitempty"`     // can be "text", "number", "password", "checkbox" or "select"
	ConfDefault  string   `json:"default,omitempty"`  // the default value, for type: "boolean" this can be set to "checked" to default to enabled
	ConfTitle    string   `json:"title,omitempty"`    // the title of the setting
	ConfOptions  []string `json:"options,omitempty"`  // the list of (string) choices for type: "select"
	ConfRequired bool     `json:"required,omitempty"` // if the value is required or not, only applies to the following types: "string", "number" (default is false)
}

func (ci ConfigItem) Clone() ConfigItem {
	var options []string
	for ci.ConfOptions != nil {
		options = make([]string, len(ci.ConfOptions))
		copy(options, ci.ConfOptions)
	}

	return ConfigItem{
		ConfKey:      ci.ConfKey,
		ConfType:     ci.ConfType,
		ConfDefault:  ci.ConfDefault,
		ConfTitle:    ci.ConfTitle,
		ConfOptions:  options,
		ConfRequired: ci.ConfRequired,
	}
}
