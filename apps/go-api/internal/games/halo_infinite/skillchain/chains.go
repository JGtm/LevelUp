package skillchain

// chains.go — LA LISTE DES CHAÎNES LUSR DU TITRE, exportée (G.5b des finitions
// v7.5, 2026-09-13 ; constat R7 de la revue E.3).
//
// POURQUOI ELLE EXISTE. `internal/sync/invariants_gate_integration_test.go`
// RECOPIAIT les quatre chaînes pour alimenter l'invariant I14 (« aucune ligne
// match_skill_rank d'une player DB Infinite ne porte la chaîne d'un autre titre »),
// avec un commentaire promettant qu'« une chaîne ajoutée au titre sans l'être ici
// rend le gate rouge ». C'était faux : `TestSkillChainLiterals_NoDrift` ne vérifie
// que cinq pair_names et les quatre chaînes existantes. Une 5e chaîne ajoutée à
// [ClassifyLUSRChain] n'aurait rien fait rougir, et l'invariant aurait classé des
// lignes LÉGITIMES comme « chaîne étrangère ».
//
// L'EXHAUSTIVITÉ EST TENUE PAR UN TEST, pas par ce commentaire :
// [TestChainsEstExhaustive] balaie un corpus de pair_names couvrant CHAQUE branche
// de [ClassifyLUSRChain] et exige (a) que toute chaîne rendue figure dans [Chains],
// (b) que toute chaîne de [Chains] soit rendue au moins une fois. Une chaîne
// ajoutée d'un seul côté rend ce test rouge.

// Chains rend les chaînes LUSR que [ClassifyLUSRChain] peut retourner, hors la
// chaîne vide (match exclu du LUSR : Ranked → CSR, Firefight → PvE).
//
// C'est la SOURCE UNIQUE pour tout consommateur qui a besoin de l'ensemble — le
// gate d'intégration en premier. Copie défensive : l'appelant ne doit pas pouvoir
// modifier la liste partagée.
func Chains() []string {
	return []string{chainArenaSlayer, chainArenaObjectif, chainBTB, chainChaos}
}
