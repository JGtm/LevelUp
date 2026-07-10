package dblease

// Kind catégorise le type de DB ciblée par un lease. Sert à étiqueter les
// métriques expvar avec une cardinalité bornée — borner au type de DB et non
// au chemin individuel évite l'explosion de cardinalité en multi-user
// (cf. ADR-0009 et plan §Métriques).
type Kind string

const (
	// KindPlayer désigne stats.duckdb d'un joueur.
	KindPlayer Kind = "player"
	// KindSharedMatches désigne shared_matches_v2.duckdb.
	KindSharedMatches Kind = "shared_matches"
	// KindSharedSocial désigne shared_social.duckdb.
	KindSharedSocial Kind = "shared_social"
	// KindMetadata désigne metadata.duckdb.
	KindMetadata Kind = "metadata"
	// KindMonitoring désigne data/global/monitoring.duckdb (persistance du
	// dashboard admin — un seul writer = process serveur).
	KindMonitoring Kind = "monitoring"
)

// allKinds liste tous les Kind connus. Sert à initialiser les compteurs expvar
// avec toutes les clés présentes dès le démarrage (sinon /debug/vars n'expose
// que les clés vues au moins une fois).
var allKinds = []Kind{KindPlayer, KindSharedMatches, KindSharedSocial, KindMetadata, KindMonitoring}
