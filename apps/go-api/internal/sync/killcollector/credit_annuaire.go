package killcollector

// credit_annuaire.go — L ANNUAIRE DES NOMS DE LA PASSE CREDIT : `v_gamertag_lookup` LUE UNE
// FOIS PAR PASSE, ET NON UNE FOIS PAR MATCH.
//
// # LE DEFAUT QUE CE FICHIER SUPPRIME (lot 5.12, 2026-09-21)
//
// `levelup backfill-killsource` du 2026-09-21 : la passe des films a decode 1 598 films en
// 4 h 14, puis la passe CREDIT-SEUL (« 9 144 matchs a examiner », annoncee « quelques secondes »
// par l en-tete de la commande) a tourne PLUS DE 22 HEURES sans terminer.
//
// La cause est MESUREE (`credit_cost_integration_test.go`, banc de 180 000 lignes de
// `match_kill_events` + 50 000 couples) et elle n est pas celle qu on croyait :
//
//	evenements du match AVEC v_gamertag_lookup      77,2 ms par match
//	evenements du match SANS la jointure (controle)   0,9 ms par match     <- 84x
//	preseance — COUNT(*) sur ..._latest              1,4 ms par match
//	enrichissement — persist.FilmPassForMatch        1,7 ms par match
//
// `v_gamertag_lookup` est la vue canonique d identite : elle agrege `match_participants` GROUP
// BY, DEUX balayages fenetres de `match_kill_events_latest` et deux de `killer_victim_pairs`,
// joints en FULL OUTER JOIN. Le filtre du lecteur porte sur `highlight_events.match_id` : il ne
// peut RIEN pousser dans la vue, qui est donc materialisee ENTIEREMENT — une fois par match,
// 9 144 fois, sur une table append-only qui grossit de ~160 000 lignes a chaque backfill. Le
// cout total est quadratique : (nombre de matchs) x (toute la base).
//
// # LE CORRECTIF, ET POURQUOI IL NE CHANGE AUCUN NOM
//
// La SOURCE DES NOMS RESTE `v_gamertag_lookup`, la vue canonique — la doctrine d identite
// (ADR 0035 : une seule clef, le xuid ; une seule source de nommage) est intacte. Ce qui change
// est le NOMBRE D EVALUATIONS : une, au premier match de la passe, dans une table en memoire de
// quelques milliers d entrees (16 996 xuids en production).
//
// L INSTANTANE EST-IL COHERENT ALORS QUE LA PASSE ECRIT ? Oui, et c est verifiable : les seuls
// noms que la passe credit ajoute a `match_kill_events` sont ceux qu elle vient de lire dans cet
// annuaire, ou le xuid lui-meme quand l annuaire ne nomme pas le joueur (`victim_gamertag` est
// NOT NULL, il faut un nom). Une evaluation par match ne verrait donc jamais un nom que
// l instantane n a pas : elle re-lirait le xuid-comme-nom que la passe vient d ecrire, et
// produirait la MEME chaine que le repli. La passe des films, elle, ne tourne pas en meme temps
// (les deux passes de la commande sont sequentielles).
//
// ⚠ UN COLLECTEUR = UNE PASSE. L annuaire est charge une fois pour la VIE de l instance ; la
// seule construction de production (`cmd/levelup/cmd_backfill_killsource.go`) en cree un par
// passe. Un futur appelant qui garderait un collecteur vivant des heures servirait des noms
// vieux de ces heures — pour lui, construire un collecteur par passe, pas un par process.

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// annuaireDesNoms : le nom canonique par xuid, instantane d une passe.
//
// Une carte nue derriere un type nomme, pour une raison : le repli « pas de nom » DOIT rester a
// UN endroit. Un appelant qui ferait `a.parXUID[x]` obtiendrait la meme chaine vide, mais la
// prochaine evolution du repli (un prefixe, un masquage) devrait etre recopiee chez lui.
type annuaireDesNoms struct {
	parXUID map[string]string
}

// nom rend le nom canonique du xuid, ou la chaine VIDE s il n est pas nomme.
//
// La chaine vide est EXACTEMENT ce que rendait le COALESCE de la jointure sur un LEFT JOIN sans
// correspondance : le repli sur le xuid vit plus loin (`analysis.ComputeKillerVictimPairs`), et
// il n est pas duplique ici.
func (a *annuaireDesNoms) nom(xuid string) string {
	if a == nil {
		return ""
	}
	return a.parXUID[xuid]
}

// requeteAnnuaireDesNoms : l annuaire entier, par la vue canonique.
//
// `ORDER BY xuid, gamertag` n est pas cosmetique : si la vue rendait DEUX lignes pour un meme
// xuid (deux alias pour un joueur), l ordre rend le nom retenu DETERMINISTE au lieu de dependre
// de l ordre de sortie de DuckDB. La jointure par match, elle, dupliquait l evenement dans ce
// cas et laissait la deduplication des morts nettoyer derriere.
const requeteAnnuaireDesNoms = `
	SELECT xuid, COALESCE(gamertag, '')
	FROM v_gamertag_lookup
	WHERE xuid IS NOT NULL AND xuid <> ''
	ORDER BY xuid, gamertag`

// annuaire rend l annuaire de la passe, en le chargeant au premier appel.
//
// Le chargement est JOURNALISE (nombre d entrees, duree) : c est la seule trace qui permettra de
// voir venir le jour ou cette lecture unique devient elle-meme chere.
func (c *CreditCollector) annuaire(ctx context.Context) (*annuaireDesNoms, error) {
	c.annuaireMu.Lock()
	defer c.annuaireMu.Unlock()
	if c.noms != nil {
		return c.noms, nil
	}

	debut := time.Now()
	rows, err := c.read.QueryContext(ctx, requeteAnnuaireDesNoms)
	if err != nil {
		return nil, fmt.Errorf("killsource credit: annuaire des noms: %w", err)
	}
	defer func() { _ = rows.Close() }()

	a := &annuaireDesNoms{parXUID: make(map[string]string, 32768)}
	for rows.Next() {
		var xuid, nom string
		if err := rows.Scan(&xuid, &nom); err != nil {
			return nil, fmt.Errorf("killsource credit: annuaire des noms (scan): %w", err)
		}
		// La PREMIERE ligne d un xuid gagne — l ordre de la requete la rend deterministe.
		if _, deja := a.parXUID[xuid]; !deja {
			a.parXUID[xuid] = nom
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("killsource credit: annuaire des noms (rows): %w", err)
	}

	slog.InfoContext(ctx, "killsource credit: annuaire des noms charge pour la passe",
		"xuids", len(a.parXUID), "duration", time.Since(debut))
	c.noms = a
	return a, nil
}
