package canonical

// catalog.go — types canoniques pour le catalogue Playlists/Pairs/Maps (Phase C plan catalogue).
//
// Ces types représentent la sortie d'un TitleCatalogAdapter (cf. games/catalog_adapter.go).
// Ils sont stockés dans metadata.duckdb par le CatalogFetcherService (Phase F).

// Experience classifie une playlist au niveau "expérience de jeu".
//
// Granularité ~6-8 buckets. Cohérent avec mode_category mais distinct :
// experience est playlist-level (alimentée par TOML rules), mode_category est
// pair-level (alimentée par Go enum).
type Experience string

const (
	ExperienceRanked        Experience = "ranked"
	ExperienceSocial        Experience = "social"
	ExperienceBTB           Experience = "btb"
	ExperienceFirefight     Experience = "firefight"
	ExperienceActionSack    Experience = "action_sack"
	ExperienceLimitedTime   Experience = "limited_time"
	ExperienceCustomBrowser Experience = "custom_browser"
	ExperienceUnknown       Experience = "unknown"
)

// IsKnownExperience valide qu'une Experience est dans l'enum canonique.
func IsKnownExperience(e Experience) bool {
	switch e {
	case ExperienceRanked, ExperienceSocial, ExperienceBTB,
		ExperienceFirefight, ExperienceActionSack, ExperienceLimitedTime,
		ExperienceCustomBrowser, ExperienceUnknown:
		return true
	}
	return false
}

// AllExperiences retourne la liste exhaustive des Experience supportées.
func AllExperiences() []Experience {
	return []Experience{
		ExperienceRanked, ExperienceSocial, ExperienceBTB,
		ExperienceFirefight, ExperienceActionSack, ExperienceLimitedTime,
		ExperienceCustomBrowser, ExperienceUnknown,
	}
}

// ModeCanonical représente le mode de jeu atomique du game_variant.
//
// Distinct de mode_category (catégorie UX au niveau pair) et de mode_label
// (libellé du sous-mode normalisé). Voir §4ter.4 du plan catalogue.
type ModeCanonical string

const (
	ModeSlayer        ModeCanonical = "slayer"
	ModeCTF           ModeCanonical = "ctf"
	ModeOddball       ModeCanonical = "oddball"
	ModeKOTH          ModeCanonical = "koth"
	ModeStrongholds   ModeCanonical = "strongholds"
	ModeExtraction    ModeCanonical = "extraction"
	ModeFiesta        ModeCanonical = "fiesta"
	ModeFirefightKOTR ModeCanonical = "firefight_kotr"
	ModeAttrition     ModeCanonical = "attrition"
	ModeStockpile     ModeCanonical = "stockpile"
	ModeTotalControl  ModeCanonical = "total_control"
	ModeUnknown       ModeCanonical = "unknown"
)

// CanonicalPlaylist représente une playlist hydratée depuis l'API du titre.
type CanonicalPlaylist struct {
	AssetID       string
	VersionID     string
	NameCanonical string            // EN par défaut
	Names         map[string]string // multi-lang : "fr" → "Match Rapide", "en" → "Quick Play", ...
	Experience    Experience
	IsRanked      bool
	PairLinks     []CanonicalPlaylistPairLink // pairs référencés avec leur poids
}

// CanonicalPlaylistPairLink représente un lien playlist→pair avec son poids de tirage.
type CanonicalPlaylistPairLink struct {
	PairAssetID string
	Weight      float64
}

// CanonicalPair représente une paire map+mode hydratée depuis l'API du titre.
type CanonicalPair struct {
	AssetID            string
	VersionID          string
	NameCanonical      string            // EN par défaut
	Names              map[string]string // multi-lang
	MapAssetID         string            // FK logique → CanonicalMap
	GameVariantAssetID string            // FK logique → CanonicalGameVariant
	ModeCategory       string            // sortie InferModeCategoryFromPairName ("Assassin", "BTB", etc.)
	ModeLabels         map[string]string // sortie NormalizeModeLabel par langue
}

// CanonicalMap représente une map hydratée depuis l'API du titre.
type CanonicalMap struct {
	AssetID       string
	VersionID     string
	NameCanonical string
	Names         map[string]string // multi-lang
	ImageURL      string            // URL résolue via assetResolver (KindMapImage)
}

// CanonicalGameVariant représente un game variant hydraté depuis l'API du titre.
type CanonicalGameVariant struct {
	AssetID             string
	VersionID           string
	NameCanonical       string
	Names               map[string]string // multi-lang
	ModeCanonical       ModeCanonical
	GameVariantCategory int // code numérique GameVariantCategory du titre
}
