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
// ─── LA PRESEANCE : LE FILM GAGNE, TOUJOURS ────────────────────────────────────────────────
//
// La vue `_latest` retient LA DERNIERE PASSE. Une passe credit-seul ecrite apres un decodage de
// film n ajouterait pas des lignes : elle DEVIENDRAIT la generation servie, et effacerait de la
// lecture la source du degat fatal. D ou le refus d ecrire quand la passe courante d un match
// vient d un film — meme regle, meme sens, que `killcollector.CreditCollector`.

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
// La preseance se teste EN POSITIF sur cette liste, jamais en negatif sur les voies credit :
// un test « read_path <> 'credit' » ferait passer toute voie future pour un film, et la
// premiere voie ajoutee bloquerait silencieusement le producteur live. Exportee parce que
// `killcollector` decide la meme preseance sur la meme liste — une seule copie dans le depot.
var FilmReadPaths = []string{"marche", "scan"}

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
	return batch
}

// persistCreditKillEvents ecrit une passe credit-seul DANS LA TRANSACTION DU CALLER.
//
// C est ce qui la distingue de [KillSourcePersister.PersistPass], qui ouvre la sienne : ici la
// passe doit tomber ou passer AVEC le reste du match. Un match dont les participants sont ecrits
// mais dont les morts manquent serait un match a moitie synchronise, et rien ne le signalerait.
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

	couvert, err := matchCoveredByFilmPass(ctx, tx, in.MatchID)
	if err != nil {
		return err
	}
	if couvert {
		// LA PRESEANCE, et c est un succes : ecrire ici retirerait de la lecture la source du
		// degat fatal que le film a mesuree.
		return nil
	}

	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: credit kill events %s: %w", in.MatchID, err)
	}
	if _, err := insertKillEventRows(ctx, tx, in, pass, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

// matchCoveredByFilmPass : la passe COURANTE de ce match vient-elle d un decodage de film ?
//
// La question se pose a la VUE et pas a la table, et ce n est pas un detail : « ce match a-t-il
// deja eu une passe de film » et « la passe servie AUJOURD HUI vient-elle d un film » sont deux
// questions differentes. Une passe de film ancienne, deja supplantee, ne doit pas bloquer
// eternellement le producteur live.
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
