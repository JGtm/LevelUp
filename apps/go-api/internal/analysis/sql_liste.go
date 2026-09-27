package analysis

// sql_liste.go — UNE LISTE LIÉE EN UN SEUL PARAMÈTRE `VARCHAR[]` (lot B du plan perf
// « compaction et périmètre joueur », 2026-09-27, ADR 0036 I2 et sa conséquence sur les longues
// listes).
//
// Le coût d'un `IN (?, ?, …)` de plusieurs milliers de valeurs est la LIAISON de ses paramètres,
// pas le filtre (copie compactée : 7 190 matchs, 304-341 ms en `IN`, 192-249 ms en un
// paramètre). La liste passe donc en UN paramètre, sous l'une de deux formes — et le choix n'est
// pas libre :
//
//	SQLDansListe             `list_contains(?, col)` : une CONSTANTE, que DuckDB pousse sous la
//	                         fenêtre d'une vue `_latest` (sa clé de partition `match_id`). Évaluée
//	                         ligne à ligne, en temps proportionnel à la liste : réservée aux listes
//	                         de match_id qui DOIVENT traverser une fenêtre (mesuré : 3 000 xuids sur
//	                         la table brute du journal, 38-44 s ; la même liste en semi-jointure,
//	                         0,11-0,12 s) ;
//	SQLDansListeParJointure  `col IN (SELECT unnest(?))` : une semi-jointure par hachage, pour tout
//	                         le reste (listes de xuids, filtres sur des TABLES). Elle ne descend PAS
//	                         sous une fenêtre (mutation M1a du lot B) : jamais sur une vue `_latest`.
//
// SOURCE UNIQUE du texte : les gabarits d'`analysis` (annuaire des noms) appellent ces fonctions,
// les lectures de `platform/duckdb` passent par `clauseListe…` (perimetre_liste.go), qui y ajoute
// l'argument. Aucun autre `?::VARCHAR[]` ne s'écrit à la main (ratchet
// archlint/liste_liee_ratchet_test.go ; deux copies antérieures y sont consignées).

// sqlListeLiee : le paramètre, une liste de chaînes. L'argument lié est un `[]string` non nil.
const sqlListeLiee = "?::VARCHAR[]"

// SQLDansListe rend le prédicat CONSTANT qui borne `col` à la liste liée (cf. en-tête : réservé
// à `match_id` sous une fenêtre `_latest`). Une liste vide ne retient rien, jamais « tous ».
func SQLDansListe(col string) string {
	return "list_contains(" + sqlListeLiee + ", " + col + ")"
}

// SQLDansListeParJointure rend le prédicat de SEMI-JOINTURE qui borne `col` à la liste liée
// (cf. en-tête : jamais sur une vue `_latest`). Une liste vide ne retient rien.
func SQLDansListeParJointure(col string) string {
	return col + " IN (" + SQLListeEnLignes() + ")"
}

// SQLListeEnLignes rend la liste liée dépliée en lignes (une colonne), pour une CTE.
func SQLListeEnLignes() string {
	return "SELECT unnest(" + sqlListeLiee + ")"
}
