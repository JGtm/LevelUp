package main

// cmd_backfill_killsource_sante.go — LA RESTITUTION DES COMPTEURS DE LA PASSE.
//
// Fichier dedie extrait de `cmd_backfill_killsource.go` le 2026-08-29, quand l ajout de
// `--online` a pousse celui-ci au-dela des 500 lignes (regle 5 : la dette ne s accroit pas).
// Aucun changement de comportement — le bloc a ete deplace tel quel, plus le parametre
// `enLigne`.

import (
	"fmt"

	"levelup/go-api/internal/observability"
)

// santeCompteurs : les compteurs a rendre en fin de passe.
//
// ILS SONT PUBLIES EN `expvar` (ADR 0009), ce qui les rend interrogeables sur un SERVEUR — mais
// cette commande est un process qui s arrete : sans cette restitution, ses compteurs mourraient
// avec lui. Un compteur qu on ne peut lire qu apres coup n alerte personne, et c est
// precisement ce que le gate de cette phase demande de constater.
var santeCompteurs = []string{
	"killsource_matchs_collectes", "killsource_morts_ecrites",
	"killsource_films_absents", "killsource_sans_killfeed",
	"killsource_erreurs_decodage", "killsource_abandons_delai", "killsource_erreurs_ecriture",
	"killsource_passes_non_publiables", "killsource_assist_extra_count",
	"killsource_tirs_matchs_ventiles", "killsource_tirs_lignes_ecrites",
	"killsource_tirs_indices_non_resolus", "killsource_tirs_erreurs_ecriture",
	"killsource_credit_matchs_ecrits", "killsource_credit_morts_ecrites",
	"killsource_credit_matchs_enrichis_par_un_film", "killsource_credit_matchs_sans_evenement",
	"killsource_fusion_morts_enrichies", "killsource_fusion_orphelins_film",
	"killsource_orphelins_film_humain_contre_humain", "killsource_fusion_instants_ambigus",
}

// afficherSante restitue les compteurs de sante de la passe.
//
// `enLigne` ajoute les compteurs de la source reseau — cache/telecharge/archive. Ils ne sont
// pas toujours affiches parce qu un zero sur une passe hors ligne ne veut rien dire, alors
// qu un `killsource_films_archive_erreurs` non nul sur une passe en ligne veut dire qu on est
// en train de perdre des films irremplacables.
func afficherSante(enLigne bool) {
	fmt.Println("compteurs de sante de la passe :")
	noms := santeCompteurs
	if enLigne {
		noms = append(append([]string{}, compteursEnLigne...), santeCompteurs...)
	}
	for _, nom := range noms {
		fmt.Printf("  %-52s %d\n", nom, observability.LoadCounter(nom))
	}
	fmt.Println("  ⚠ killsource_assist_extra_count : surplus d assistants observes. Mesure du " +
		"2026-08-03 : 5 lignes a 1 sur 124 694 — anecdotique. Declencheur de migration vers une " +
		"table fille (ADR 0026) : UNE ligne a >= 2, ou plus de 0,1 % des lignes a >= 1.")
	fmt.Println("  ⚠ killsource_orphelins_film_humain_contre_humain : lignes de film sans mort " +
		"de credit en face ET portant DEUX xuids. 13 sur 74 569 au 2026-08-02, mecanisme non " +
		"demontre — la seule des trois populations d orphelins a surveiller.")
}
