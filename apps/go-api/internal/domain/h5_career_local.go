package domain

// H5CareerLocal porte les données career Halo 5 lues LOCALEMENT (DuckDB synchronisé)
// pour projeter un CareerSnapshot hors-ligne — notamment en démo, où aucun token
// n'est disponible et l'API cryptum live échouerait.
//
// Pourquoi ce DTO neutre (domain) plutôt qu'un type halo_5 : la source DuckDB vit
// dans internal/platform/duckdb, qui NE PEUT PAS importer internal/games/halo_5
// (cycle via internal/games). domain est un paquet feuille importé des deux côtés,
// comme canonical.MatchSummary l'est pour l'historique h5. La projection vers
// canonical.CareerSnapshot (libellés de palier FR, bornes SR) reste côté halo_5.
type H5CareerLocal struct {
	// CSRTier : palier EN du MEILLEUR CSR à vie (Bronze, Silver, Gold, Platinum,
	// Diamond, Onyx). "" si le joueur n'a aucun CSR résolu.
	CSRTier string
	// CSRSubTier : sous-palier 1..6 (0 pour Onyx ou indéfini).
	CSRSubTier int
	// CSRValue : valeur CSR brute (significative uniquement à Onyx).
	CSRValue int
	// HasCSR : true si un palier CSR à vie a été résolu.
	HasCSR bool

	// SpartanRank : niveau SR de compte 1..152 (0 si inconnu).
	SpartanRank int
	// TotalXP : XP de compte cumulé (0 si inconnu).
	TotalXP int
}
