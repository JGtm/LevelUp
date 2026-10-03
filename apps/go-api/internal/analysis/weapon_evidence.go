// Package analysis — weapon_evidence.go : QUELLE TABLE prouve qu'un match porte bien sa
// donnée d'arme.
//
// # Le problème
//
// Trois sondes de santé posent la même question : « le masque dit que le détail des armes
// de ce match est fait — la donnée est-elle vraiment là ? ». Elles la posaient toutes à
// `weapon_kills`. Depuis la bascule du 2026-09-01, cette table ne sert plus que les titres
// dont l'arme est native de l'API (Halo 5) ; sur un titre à décodeur de film, la preuve
// vit dans `match_kill_events_latest`, et la table disparaît de son fichier.
//
// Une sonde qui interrogerait une table absente ne rendrait pas « rien à signaler » : elle
// échouerait (Catalog Error), et une sonde de santé en échec est pire qu'une sonde absente.
//
// # La réponse est DONNÉE, pas déduite d'un slug
//
// On regarde ce que la base CONTIENT. Un fichier qui porte encore `weapon_kills` se sonde
// là ; un fichier où elle a été supprimée se sonde sur la source de dégât. Aucune
// comparaison de slug, aucune capability à câbler jusque dans un scheduler.
package analysis

import (
	"context"
	"database/sql"
)

// WeaponEvidenceTable rend le nom de la table (ou vue) qui porte la preuve du détail des
// armes dans CETTE base.
//
// `weapon_kills` si elle existe encore, `match_kill_events_latest` sinon. Chaîne vide si
// aucune des deux n'est là (base neuve, migrations non appliquées) : l'appelant saute alors
// la sonde plutôt que d'émettre une requête vouée à l'erreur.
func WeaponEvidenceTable(ctx context.Context, db *sql.DB) string {
	if db == nil {
		return ""
	}
	for _, name := range []string{"weapon_kills", "match_kill_events_latest"} {
		var n int
		err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", name).Scan(&n)
		if err == nil && n > 0 {
			return name
		}
		// information_schema.tables ne liste pas les VUES sur toutes les versions : on
		// retente par le catalogue des vues avant de conclure à l'absence.
		err = db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM duckdb_views() WHERE view_name = ?", name).Scan(&n)
		if err == nil && n > 0 {
			return name
		}
	}
	return ""
}
