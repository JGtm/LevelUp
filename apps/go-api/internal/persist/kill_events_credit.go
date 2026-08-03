package persist

// kill_events_credit.go — LE PRODUCTEUR LIVE de `shared.match_kill_events` : les couples
// tueur -> victime que le pipeline de sync rend DEJA deviennent des morts credit-seul, ecrites
// au fil de l eau, dans la meme transaction que le reste du match.
//
// ─── POURQUOI IL EXISTE ────────────────────────────────────────────────────────────────────
//
// `match_kill_events` avait deux producteurs, et TOUS LES DEUX etaient hors ligne : le decodeur
// de film et le producteur credit-seul de `killcollector`, l un comme l autre appeles par la
// seule sous-commande `levelup backfill-killsource`. Mesure du 2026-08-02 : aucun appel a l un
// ou l autre depuis `internal/api/`, `internal/sync/v2/`, `cmd/server/` ni `internal/service/`.
// Consequence : tout match synchronise APRES la derniere passe de backfill n avait aucune ligne
// dans la table — un lecteur bascule dessus aurait affiche zero mort sur les matchs recents, et
// rien ne l aurait signale.
//
// Ce fichier ferme ce trou a la source : la ou le pipeline ecrivait `killer_victim_pairs`, il
// ecrit desormais AUSSI la table canonique. Meme rows, meme transaction, meme instant.
//
// ─── ET IL SUPPRIME LE BUG DES DOUBLONS PAR CONSTRUCTION ───────────────────────────────────
//
// Les 46,5 % de doublons de `killer_victim_pairs` (250 139 lignes pour 133 886 cles, mesure du
// 2026-08-02 sur la base Infinite) viennent de DEUX ecrivains qui ne s accordent pas : le flux
// primaire INSERT sans supprimer, la completion fait DELETE-then-INSERT. Les deux ecrivent ici
// une PASSE, et la vue `match_kill_events_latest` n en retient qu une par match. Deux passes ne
// s additionnent plus : la seconde remplace la premiere. Le doublon n est pas corrige, il est
// devenu impossible — et le DELETE (donc la surface ART, ADR 0019) disparait avec lui.
//
// ─── TITLE-AGNOSTIC PAR CONSTRUCTION ───────────────────────────────────────────────────────
//
// Aucune capability, aucun slug : ce chemin traduit ce que le batch porte. Halo Infinite y
// arrive par sa completion combat (couples reconstitues depuis `highlight_events`), Halo 5 par
// son carnage natif (`ingest/kills.go`) — mesure du 2026-08-02 : 268 337 couples H5, dont ZERO
// doublon, et `highlight_events` H5 ne porte AUCUN evenement kill/death, donc le producteur
// credit-seul de `killcollector` (qui lit cette table) ne couvrirait jamais ce titre. Un titre
// qui ne rend aucun couple ne produit rien : degradation gracieuse sans une ligne de branchement.
//
// ─── LA PRESEANCE : LE CREDIT EST LA BASE, LE FILM ENRICHIT ────────────────────────────────
//
// INVERSEE LE 2026-08-03 (`.ai/CONCEPTION_INVERSION_PRESEANCE.md`). Elle etait FILM > CREDIT : ce
// producteur REFUSAIT d ecrire quand la passe courante venait d un film, pour ne pas effacer la
// source du degat fatal. Le refus protegeait bien la source — mais il laissait servie une passe
// qui ne porte que 74,4 % des morts, la ou le credit en porte 98,5 %.
//
// Desormais il ECRIT TOUJOURS, et il recompose : sa base credit + l enrichissement de la passe de
// film courante, relu en base (`FilmPassForMatch`) et fusionne par `MergeCreditAndFilm`. La source
// du degat est preservee ligne par ligne, et plus une seule mort n est perdue.
//
// `matchCoveredByFilmPass` n a PAS ete supprimee : meme requete, sens inverse. Elle ne decide plus
// un refus, elle decide s il y a un enrichissement A RELIRE — ce qui evite la lecture
// supplementaire sur les ~70 % de matchs qui n ont pas de film.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/observability"
)

// metricLiveFeedCouplesSansVictime — compteur de sante du producteur live (ADR 0009).
//
// Il vaut 0 sur les deux titres connus. Le jour ou il bouge, il designe un titre dont les
// couples arrivent degrades — et il le designe AVANT qu on cherche pourquoi un match a moins de
// morts qu il ne devrait.
const metricLiveFeedCouplesSansVictime = "killsource_live_couples_sans_victime_nommee"

// LiveFeedDecoderRev — la version du producteur live.
//
// Du meme espace que les `decoder_rev` des autres producteurs (la colonne est commune) mais
// designant celui-ci : c est elle qui permet de rejouer les matchs ecrits au fil de l eau sans
// toucher aux matchs decodes depuis un film.
const LiveFeedDecoderRev = "sync-kill-feed-2026-08-02"

// Portee des lignes du producteur live — LUE CHEZ SON PROPRIETAIRE, jamais recopiee.
//
// Ce vocabulaire est partage par quatre ecrivains (ici, la reprise de `migration`, le producteur
// credit-seul de `killcollector`, la base de demonstration de `ops`). Il vit dans
// `domain/killscope`, une feuille sans import, parce que `migration` ne peut pas importer
// `persist` sans cycle a la compilation des tests. Garde-rail :
// `internal/archlint/no_raw_kill_scope_literal_test.go`.

// FilmReadPaths — LES VOIES DE LECTURE PRODUITES PAR UN DECODAGE DE FILM.
//
// La detection se teste EN POSITIF sur cette liste, jamais en negatif sur les voies credit :
// un test « read_path <> 'credit' » ferait passer toute voie future pour un film, et la
// premiere voie ajoutee ferait taire silencieusement le producteur live.
//
// LES VALEURS NE SONT PLUS RECOPIEES ICI (2026-08-03, dette H3 refermee) : elles vivent dans
// `domain/killscope`, avec les trois valeurs de portee credit, et leur egalite avec le decodeur
// est verrouillee par un test dans `sync/killcollector` — le seul paquet qui importe les deux.
var FilmReadPaths = killscope.FilmReadPaths()

// CreditKillEventsFromPairs : les couples tueur -> victime deviennent des morts, sans rien
// inventer.
//
// LA SEULE TRADUCTION du depot entre la forme par-couple du pipeline et la forme par-mort de
// `match_kill_events` — les deux ecrivains (flux primaire et completion combat) passent par
// elle. Une seconde traduction serait une seconde chance de confondre les trois etats de
// l assistant.
//
// CE QU ELLE N ECRIT PAS, ET POURQUOI :
//
//	assist_known = FALSE          ON NE SAIT PAS. Le kill-feed ne porte aucun assistant ;
//	                              ecrire « pas d assistant » fabriquerait un fait jamais observe.
//	source_tag / source_category  absentes. La source du degat se lit dans le film.
//	diverges                      absente. Elle compare les DEUX verites.
//	parts de degats               absentes. « Non mesure » n est jamais zero.
//
// PUBLIABLE LIGNE PAR LIGNE : ces lignes valent EXACTEMENT ce que valait `killer_victim_pairs`
// — meme source, memes couples — et cette table etait deja lue ligne par ligne (match-view,
// timeline K/D, penalite de depart LUSR). Les declarer non publiables retirerait une capacite
// existante au nom d une prudence qui ne repose sur aucune mesure.
func CreditKillEventsFromPairs(ctx context.Context, matchID string, pairs []KillerVictimInsert) KillSourceBatch {
	batch := KillSourceBatch{
		MatchID:     matchID,
		DecoderRev:  LiveFeedDecoderRev,
		Publishable: true,
		Deaths:      make([]KillEventInsert, 0, len(pairs)),
	}
	sansNom := 0
	for _, p := range pairs {
		// UNE MORT SANS NOM DE VICTIME EST ECARTEE, PAS REFUSEE — ET ELLE SE COMPTE (cf. le log
		// et le compteur apres la boucle). `victim_gamertag` est la seule colonne qu une mort ne
		// peut pas ne pas avoir, et la validation du persister s applique a la PASSE ENTIERE :
		// une ligne sans nom ferait echouer tout le match, donc perdre aussi les morts nommees
		// et le reste du batch. Le cas n existe pas sur les donnees reelles — 0 ligne sans nom
		// de victime sur les 518 476 couples des deux titres, mesure du 2026-08-02 — mais un
		// couple degrade venu d un titre futur ne doit pas pouvoir faire tomber une
		// synchronisation, ni disparaitre sans laisser de trace.
		if p.VictimGamertag == "" {
			sansNom++
			continue
		}
		batch.Deaths = append(batch.Deaths, KillEventInsert{
			TimeMS:             int(p.TimeMS),
			VictimGamertag:     p.VictimGamertag,
			VictimXUID:         p.VictimXUID,
			FeedKillerGamertag: p.KillerGamertag,
			FeedKillerXUID:     p.KillerXUID,
			// Le kill-feed porte bien cette mort : c est LUI la source de ces lignes.
			FeedPresent: true,
			AssistKnown: false,
			ReadPath:    killscope.ReadPathLiveFeed,
			ReadOrigin:  killscope.OriginCreditOnly,
		})
	}
	// L ECART SE COMPTE ET SE DIT, AVANT que le batch degrade parte a l ecriture. Ce cas
	// n existe pas sur les donnees connues (0 ligne sans nom de victime sur 518 476 couples des
	// deux titres, mesure du 2026-08-02) : c est PRECISEMENT ce qui le rend dangereux muet. Le
	// jour ou un titre en produit, un `continue` sans trace ferait disparaitre des morts du
	// journal sans qu aucun compteur ne bouge — et le nombre de morts d un match n a pas de
	// valeur attendue a laquelle le comparer.
	if sansNom > 0 {
		observability.AddInt(metricLiveFeedCouplesSansVictime, int64(sansNom))
		slog.WarnContext(ctx, "persist: couples ecartes, nom de victime absent",
			"match_id", matchID, "couples_ecartes", sansNom, "couples_recus", len(pairs))
	}
	// LA BASE CREDIT EST DEDUPLIQUEE SUR L IDENTITE, ici comme partout ailleurs — une seule
	// definition dans le depot (cf. [DedupCreditDeaths]).
	batch.Deaths = DedupCreditDeaths(batch.Deaths)
	// LE PLANCHER DE LA PASSE, pose par le producteur de la base et par lui seul. Ce que la
	// fusion ajoutera ne peut que s ajouter : le persister refusera toute passe qui en publierait
	// moins (cf. `KillSourceBatch.CreditBaseCount`).
	batch.CreditBaseCount = len(batch.Deaths)
	return batch
}

// DedupCreditDeaths : LA DEFINITION DE LA BASE CREDIT, en un seul endroit.
//
// Une mort de credit est identifiee par `(time_ms, victime, tueur)` — les XUIDS, pas les
// gamertags. Ce decoupage n est pas un choix esthetique, il est le seul des trois qui rende les
// deux cardinaux mesures le 2026-08-03 :
//
//	dedup sur (time_ms) seul                  268 330 sur Halo 5 — PERD 7 morts REELLES (deux
//	                                          victimes distinctes a la meme milliseconde)
//	dedup sur toutes les colonnes             145 974 sur Halo Infinite — FABRIQUE 12 088 doublons
//	                                          (une meme mort sous deux orthographes de gamertag)
//	dedup sur (time_ms, victime, tueur)       133 886 sur Infinite / 268 337 sur Halo 5 — JUSTE
//
// La premiere occurrence gagne : l ordre d entree est celui du producteur (couples apparies dans
// l ordre du temps), donc le resultat est reproductible.
//
// ⚠ POURQUOI ELLE EST NECESSAIRE ET PAS SEULEMENT PRUDENTE : `highlight_events` porte
// 15 120 groupes `(match_id, time_ms, xuid)` en DOUBLE EXACT sur les matchs sans film. Le
// producteur credit-seul, qui apparie cette table, en tirait 50 125 morts la ou il y en a 35 224 —
// soit 140 % de l oracle. Un « credit-base » qui sur-compte de 42 % ne serait pas une base.
func DedupCreditDeaths(deaths []KillEventInsert) []KillEventInsert {
	type identite struct {
		t              int
		victime, tueur string
	}
	vus := make(map[identite]bool, len(deaths))
	out := make([]KillEventInsert, 0, len(deaths))
	for i := range deaths {
		id := identite{deaths[i].TimeMS, deaths[i].VictimXUID, deaths[i].FeedKillerXUID}
		if vus[id] {
			continue
		}
		vus[id] = true
		out = append(out, deaths[i])
	}
	return out
}

// CreditBaseForMatch : la base credit d UN match, relue depuis `killer_victim_pairs`.
//
// C EST LA MEME DEFINITION QUE CELLE DE LA MIGRATION DE REPRISE, et il n en existe pas d autre :
// les couples dedupliques sur l identite, avec la portee `kill-feed` / `credit-seul`. Elle sert le
// collecteur de FILM, qui doit poser son enrichissement sur une base et n en produit aucune.
//
// LE GAMERTAG EST CHOISI DETERMINISTEMENT (`MAX`) quand une meme identite en porte plusieurs : un
// joueur renomme laisse deux orthographes derriere lui sur des lignes qui designent la MEME mort.
// Prendre « celui qui vient » rendrait la passe non reproductible d une execution a l autre.
//
// Un match sans couple rend un batch VIDE, et ce n est pas une erreur : la fusion se comporte
// alors comme avant l inversion (le film publie ses lignes, qui sont toutes des orphelines).
func CreditBaseForMatch(ctx context.Context, q rowQuerier, matchID string) (KillSourceBatch, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT COALESCE(time_ms, 0) AS t,
		       COALESCE(victim_xuid, '') AS vx, MAX(victim_gamertag) AS vg,
		       COALESCE(killer_xuid, '') AS kx, MAX(COALESCE(killer_gamertag, '')) AS kg
		FROM killer_victim_pairs
		WHERE match_id = ? AND victim_gamertag IS NOT NULL AND victim_gamertag <> ''
		GROUP BY 1, 2, 4
		ORDER BY 1, 2, 4`, matchID)
	if err != nil {
		return KillSourceBatch{}, fmt.Errorf("persist: base credit %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	batch := KillSourceBatch{MatchID: matchID, DecoderRev: LiveFeedDecoderRev, Publishable: true}
	for rows.Next() {
		var d KillEventInsert
		if err := rows.Scan(&d.TimeMS, &d.VictimXUID, &d.VictimGamertag,
			&d.FeedKillerXUID, &d.FeedKillerGamertag); err != nil {
			return KillSourceBatch{}, fmt.Errorf("persist: base credit %s (scan): %w", matchID, err)
		}
		d.FeedPresent = true
		d.AssistKnown = false
		d.ReadPath, d.ReadOrigin = killscope.ReadPathLiveFeed, killscope.OriginCreditOnly
		batch.Deaths = append(batch.Deaths, d)
	}
	if err := rows.Err(); err != nil {
		return KillSourceBatch{}, fmt.Errorf("persist: base credit %s (rows): %w", matchID, err)
	}
	batch.CreditBaseCount = len(batch.Deaths)
	return batch, nil
}

// persistCreditKillEvents ecrit une passe RECOMPOSEE dans la transaction du caller.
//
// C est ce qui la distingue de [KillSourcePersister.PersistPass], qui ouvre la sienne : ici la
// passe doit tomber ou passer AVEC le reste du match. Un match dont les participants sont ecrits
// mais dont les morts manquent serait un match a moitie synchronise, et rien ne le signalerait.
//
// RECOMPOSEE, ET PLUS « credit-seul » : quand le match porte deja une passe de film, son
// enrichissement est RELU EN BASE et fusionne sur la base credit fraiche. Aucune mort n est perdue
// (c est la base qui fait la liste), aucune arme n est perdue (l enrichissement est recopie
// ligne a ligne). Le refus d ecrire d avant le 2026-08-03 protegeait la source au prix d un quart
// des morts.
//
// LE COUT DE LA RECOMPOSITION est une lecture supplementaire par match — et SEULEMENT pour les
// matchs porteurs d un film : `matchCoveredByFilmPass` est une sonde `COUNT(*)` sur l index
// `(match_id, written_at)` deja present, la lecture complete ne suit que si elle repond oui.
//
// Une victime sans nom fait echouer la validation du persister (`victim_gamertag` est NOT NULL,
// et c est le seul champ qu une mort ne peut pas ne pas avoir). Ce cas ne se produit pas
// aujourd hui — mesure du 2026-08-02 : 0 ligne sans nom de victime sur les 518 476 couples des
// deux titres — mais il n est pas avale pour autant : il remonte comme l erreur qu il est.
func persistCreditKillEvents(ctx context.Context, tx *sql.Tx, in KillSourceBatch) error {
	if len(in.Deaths) == 0 {
		return nil
	}
	if err := validateKillSourceBatch(in); err != nil {
		return err
	}

	batch, err := recomposerAvecFilm(ctx, tx, in)
	if err != nil {
		return err
	}
	if err := validateKillSourceBatch(batch); err != nil {
		return err
	}

	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: credit kill events %s: %w", in.MatchID, err)
	}
	if _, err := insertKillEventRows(ctx, tx, batch, pass, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

// recomposerAvecFilm : la base credit, plus l enrichissement de la passe de film courante s il y
// en a une. Rend la base telle quelle sinon — c est le cas des ~70 % de matchs sans film.
func recomposerAvecFilm(ctx context.Context, tx *sql.Tx, base KillSourceBatch) (KillSourceBatch, error) {
	couvert, err := matchCoveredByFilmPass(ctx, tx, base.MatchID)
	if err != nil {
		return KillSourceBatch{}, err
	}
	if !couvert {
		return base, nil
	}
	film, err := FilmPassForMatch(ctx, tx, base.MatchID)
	if err != nil {
		return KillSourceBatch{}, err
	}
	batch, st, err := MergeCreditAndFilm(base, film)
	if err != nil {
		return KillSourceBatch{}, err
	}
	PublishMergeStats(ctx, base.MatchID, st)
	return batch, nil
}

// matchCoveredByFilmPass : la passe COURANTE de ce match porte-t-elle un enrichissement de film ?
//
// MEME REQUETE QU AVANT LE 2026-08-03, SENS INVERSE. Elle decidait un REFUS D ECRIRE (preseance
// film > credit) ; elle decide desormais s il y a quelque chose a RELIRE avant d ecrire. Le
// vocabulaire de portee (`FilmReadPaths`) et le test EN POSITIF sur les voies de film sont
// conserves tels quels — l acquis de J4R-4.
//
// La question se pose a la VUE et pas a la table, et ce n est pas un detail : « ce match a-t-il
// deja eu une passe de film » et « la passe servie AUJOURD HUI porte-t-elle un enrichissement »
// sont deux questions differentes. Un enrichissement d une passe deja supplantee n existe plus.
func matchCoveredByFilmPass(ctx context.Context, tx *sql.Tx, matchID string) (bool, error) {
	placeholders := ""
	args := make([]any, 0, len(FilmReadPaths)+1)
	args = append(args, matchID)
	for i, p := range FilmReadPaths {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, p)
	}

	var n int
	q := fmt.Sprintf(
		`SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ? AND read_path IN (%s)`,
		placeholders)
	if err := tx.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return false, fmt.Errorf("persist: preseance film %s: %w", matchID, err)
	}
	return n > 0, nil
}
