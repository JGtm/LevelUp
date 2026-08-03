package killcollector

// credit.go — LE SECOND PRODUCTEUR DE `shared.match_kill_events` : LE CREDIT SEUL, DEPUIS LA BASE.
//
// # POURQUOI IL EXISTE, ET CE QU IL COUTE
//
// Les films Theater EXPIRENT cote serveur. Mesure du 2026-08-01 : 1 926 matchs au registre,
// **1 343** porteurs de couples tueur -> victime, **949** films en cache — donc **394 matchs
// (29,3 %) n auront JAMAIS de film**. Un `match_kill_events` alimente par le seul decodeur de
// film couvrirait 949 matchs sur 1 343 : remplacer `killer_victim_pairs` par une vue posee
// dessus ne serait plus une deduplication, ce serait une REGRESSION SILENCIEUSE de 29,3 %.
//
// Ce producteur ferme ce trou, et il ne coute presque rien : `killer_victim_pairs` est
// AUJOURD HUI produit depuis la table `highlight_events` (cf. `persistCombatCompletion`), donc
// la meme source, le meme algorithme d appariement, la meme tolerance de 5 ms. C est une
// transformation SQL -> SQL : ni reseau, ni tokens, ni decodage.
//
// # CE QU IL SAIT, ET CE QU IL NE SAIT PAS — LES TROIS ETATS SONT LE POINT
//
//	assist_known = FALSE              ON NE SAIT PAS. Le kill-feed de l API ne porte aucun
//	                                  assistant : ecrire « pas d assistant » fabriquerait un fait
//	                                  jamais observe.
//	source_tag / source_category      ABSENTES (NULL), pas nulles-comme-zero. La source du degat
//	                                  se lit dans le film, et il n y en a pas.
//	diverges                          NULL. Elle compare les DEUX verites ; sans la seconde elle
//	                                  est indefinissable — FALSE se lirait « mesure : pas de
//	                                  divergence ».
//	killer/assist_damage_pct          NULL. « Non mesure » n est jamais zero.
//
// # LA PRESEANCE — INVERSEE LE 2026-08-03 : LE CREDIT EST LA BASE, LE FILM ENRICHIT
//
// La vue `_latest` retient LA DERNIERE PASSE par match. Ce producteur REFUSAIT donc d ecrire un
// match dont la passe courante venait d un film, pour ne pas effacer la source du degat fatal.
// Le refus protegeait la source — au prix d un quart des morts : la passe de film ne publie que
// 74,4 % de ce que le credit porte a 98,5 % (`.ai/CONCEPTION_INVERSION_PRESEANCE.md`).
//
// Il ECRIT DESORMAIS TOUJOURS, et il recompose : sa base credit + l enrichissement de la passe de
// film courante, RELU EN BASE (`persist.FilmPassForMatch`) et fusionne par
// `persist.MergeCreditAndFilm` sur la clef `(match_id, time_ms)` a tolerance ZERO. Rien n est
// perdu d aucun cote. `covertParUnFilm` n a pas disparu : meme requete, sens inverse — elle
// decide s il y a un enrichissement A RELIRE, pas un refus d ecrire.
//
// # TITLE-AGNOSTIC PAR CONSTRUCTION
//
// Aucune capability, aucun slug : ce producteur lit `highlight_events` de la base partagee DU
// TITRE COURANT (chemin resolu par `PathResolver`). Un titre qui ne peuple pas cette table ne
// produit rien — degradation gracieuse sans une ligne de branchement.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// CreditDecoderRev — la version du producteur credit-seul.
//
// Elle est du meme espace que [KillSourceDecoderRev] (la colonne `decoder_rev` est commune) mais
// designe un AUTRE producteur : c est elle qui permettra de rejouer les matchs credit-seul sans
// toucher aux matchs decodes depuis un film.
const CreditDecoderRev = "highlight-credit-2026-08-01"

// Portee des lignes credit-seul : `killscope.ReadPathCreditBackfill` / `killscope.OriginCreditOnly`.
//
// Elle N EST PAS redeclaree ici. Ce vocabulaire est partage par quatre ecrivains et il vit dans
// une feuille sans import (`domain/killscope`) — une copie locale qui deriverait d un caractere
// rendrait la preseance aveugle sans rien casser d autre. Garde-rail :
// `internal/archlint/no_raw_kill_scope_literal_test.go`.

// creditToleranceMS : la fenetre d appariement kill <-> death.
//
// 5 ms, et ce n est pas un reglage : c est EXACTEMENT la valeur qu emploie la completion qui
// produit `killer_victim_pairs` aujourd hui (`persistCombatCompletion`). Une autre valeur
// produirait un jeu de couples different de celui que la table servait — le remplacement ne
// serait plus comparable a ce qu il remplace.
const creditToleranceMS = int64(5)

// Compteurs de sante du producteur credit-seul (ADR 0009).
const (
	metricCreditMatches = "killsource_credit_matchs_ecrits"
	metricCreditDeaths  = "killsource_credit_morts_ecrites"
	// metricCreditEnriched remplace `killsource_credit_matchs_deja_couverts_par_un_film`, qui
	// comptait des REFUS d ecrire. La preseance inversee, ces matchs ne sont plus refuses : ils
	// sont ECRITS avec l enrichissement du film recopie dessus. Le compteur change donc de nom
	// avec ce qu il mesure — garder l ancien nom sur une quantite differente ferait mentir un
	// graphe.
	metricCreditEnriched = "killsource_credit_matchs_enrichis_par_un_film"
	metricCreditEmpty    = "killsource_credit_matchs_sans_evenement"
)

// CreditCollector : le producteur credit-seul.
type CreditCollector struct {
	read          *sql.DB
	acquireShared persist.SharedWriterFn
}

// NewCreditCollector construit le producteur.
//
//   - read          : handle de LECTURE sur shared (`OpenReadForQuery` si la base est
//     potentiellement tenue en RW par le process — jamais `OpenReadOnly` force) ;
//   - acquireShared : ouverture RW AVEC son lease (ADR 0013), la meme que le collecteur de film.
func NewCreditCollector(read *sql.DB, acquireShared persist.SharedWriterFn) *CreditCollector {
	return &CreditCollector{read: read, acquireShared: acquireShared}
}

// CreditOutcome : ce qui est arrive a UN match.
type CreditOutcome string

const (
	// CreditWritten : la base credit a ete ecrite, sans enrichissement (aucun film sur ce match).
	CreditWritten CreditOutcome = "ecrit"
	// CreditEnriched : la base credit a ete ecrite AVEC l enrichissement de la passe de film
	// courante recopie dessus. Remplace l ancien `film-prioritaire`, qui designait un REFUS.
	CreditEnriched CreditOutcome = "ecrit-enrichi"
	// CreditNoEvents : aucun couple reconstituable. Cas NORMAL sur un match dont les
	// `highlight_events` n ont jamais ete charges.
	CreditNoEvents CreditOutcome = "sans-evenement"
)

// CollectMatch produit la passe d UN match : sa base credit, enrichie du film s il y en a un.
//
// Contrat d erreur : une erreur rendue est une VRAIE panne (base injoignable, ecriture refusee,
// identites divergentes entre les deux cotes de la fusion). L absence d evenements est un
// OUTCOME — une passe de masse continue.
func (c *CreditCollector) CollectMatch(ctx context.Context, matchID string) (CreditOutcome, int, error) {
	events, err := c.evenementsDuMatch(ctx, matchID)
	if err != nil {
		return CreditNoEvents, 0, err
	}
	base := BuildCreditBatch(matchID, events)
	if len(base.Deaths) == 0 {
		observability.AddInt(metricCreditEmpty, 1)
		return CreditNoEvents, 0, nil
	}

	batch, enrichi, err := c.recomposer(ctx, base)
	if err != nil {
		return CreditNoEvents, 0, err
	}
	if err := c.write(ctx, batch); err != nil {
		return CreditNoEvents, 0, err
	}
	observability.AddInt(metricCreditMatches, 1)
	observability.AddInt(metricCreditDeaths, int64(len(batch.Deaths)))
	if enrichi {
		observability.AddInt(metricCreditEnriched, 1)
		return CreditEnriched, len(batch.Deaths), nil
	}
	return CreditWritten, len(batch.Deaths), nil
}

// recomposer : la base credit, plus l enrichissement de la passe de film courante s il y en a un.
//
// La sonde `covertParUnFilm` reste en tete pour la meme raison qu avant : elle est un `COUNT(*)`
// sur l index `(match_id, written_at)`, la lecture complete des lignes de film ne suit que si elle
// repond oui — et ~70 % des matchs n ont pas de film.
func (c *CreditCollector) recomposer(
	ctx context.Context, base persist.KillSourceBatch,
) (persist.KillSourceBatch, bool, error) {
	couvert, err := c.covertParUnFilm(ctx, base.MatchID)
	if err != nil {
		return persist.KillSourceBatch{}, false, err
	}
	if !couvert {
		return base, false, nil
	}
	film, err := persist.FilmPassForMatch(ctx, c.read, base.MatchID)
	if err != nil {
		return persist.KillSourceBatch{}, false, err
	}
	batch, st, err := persist.MergeCreditAndFilm(base, film)
	if err != nil {
		return persist.KillSourceBatch{}, false, err
	}
	persist.PublishMergeStats(ctx, base.MatchID, st)
	slog.DebugContext(ctx, "killsource credit: passe recomposee sur la base credit",
		"match_id", base.MatchID, "base_credit", len(base.Deaths), "publiees", len(batch.Deaths),
		"enrichies", st.Enriched, "orphelins", st.Orphans)
	return batch, true, nil
}

// covertParUnFilm : LE DETECTEUR D ENRICHISSEMENT, lu par la VUE et pas par la table.
//
// Meme requete qu avant le 2026-08-03, sens inverse : elle decidait un REFUS D ECRIRE, elle
// decide desormais s il y a quelque chose a RELIRE avant d ecrire.
//
// La question n est pas « ce match a-t-il DEJA eu une passe de film » mais « la passe COURANTE
// porte-t-elle un enrichissement » : l enrichissement d une passe supplantee n existe plus. La vue
// repond exactement a cette question.
// La question se teste EN POSITIF sur les voies de film (`persist.FilmReadPaths`), et le
// changement du 2026-08-02 n est pas cosmetique : la version precedente demandait
// `read_path <> 'highlight-events'`, donc toute voie AUTRE que la sienne passait pour un film.
// Le jour ou une seconde voie credit est apparue (le producteur live, `kill-feed`), ce test
// aurait pris ce producteur-la pour un decodage — silencieusement.
func (c *CreditCollector) covertParUnFilm(ctx context.Context, matchID string) (bool, error) {
	args := make([]any, 0, len(persist.FilmReadPaths)+1)
	args = append(args, matchID)
	marques := make([]string, 0, len(persist.FilmReadPaths))
	for _, p := range persist.FilmReadPaths {
		marques = append(marques, "?")
		args = append(args, p)
	}

	var n int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE match_id = ? AND read_path IN (%s)`, strings.Join(marques, ", "))
	if err := c.read.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return false, fmt.Errorf("killsource credit: preseance %s: %w", matchID, err)
	}
	return n > 0, nil
}

// evenementsDuMatch : les kills et les morts du match, avec le gamertag du joueur.
//
// LE GAMERTAG VIENT DE `v_gamertag_lookup`, la vue canonique — pas de `highlight_events` (qui ne
// le porte pas) et surtout pas de `match_participants.gamertag`, qui est vide en pratique
// (4 xuids nommes sur 16 996 ; cf. [SharedRoster.RosterForMatch]). Un joueur que la vue ne
// nomme pas sort avec son xuid en guise de nom : `victim_gamertag` est NOT NULL, il faut un nom,
// et le xuid en est un honnete.
func (c *CreditCollector) evenementsDuMatch(ctx context.Context, matchID string) ([]analysis.RawEvent, error) {
	rows, err := c.read.QueryContext(ctx, `
		SELECT he.xuid, LOWER(he.event_type), COALESCE(he.time_ms, 0), COALESCE(g.gamertag, '')
		FROM highlight_events he
		LEFT JOIN v_gamertag_lookup g ON g.xuid = he.xuid
		WHERE he.match_id = ?
		  AND LOWER(he.event_type) IN (?, ?)
		  AND he.xuid IS NOT NULL AND he.xuid <> ''
		ORDER BY he.time_ms, he.xuid, he.event_type`,
		matchID, analysis.EventTypeKill, analysis.EventTypeDeath)
	if err != nil {
		return nil, fmt.Errorf("killsource credit: evenements %s: %w", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	var out []analysis.RawEvent
	for rows.Next() {
		var ev analysis.RawEvent
		if err := rows.Scan(&ev.XUID, &ev.EventType, &ev.TimeMS, &ev.Gamertag); err != nil {
			return nil, fmt.Errorf("killsource credit: scan %s: %w", matchID, err)
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("killsource credit: rows %s: %w", matchID, err)
	}
	return out, nil
}

// BuildCreditBatch : les evenements deviennent des morts, sans rien inventer.
//
// EXPORTEE pour la meme raison que [BuildKillSourceBatch] : le backfill emprunte ce chemin, et
// une seconde traduction serait une seconde chance de confondre les trois etats de l assistant.
//
// L APPARIEMENT EST CELUI DE LA PRODUCTION, PAS UN NOUVEAU : `analysis.ComputeKillerVictimPairs`
// est la fonction qui produit `killer_victim_pairs` aujourd hui. Un appariement different
// donnerait un jeu de couples different de celui qu on remplace, et la comparaison avant/apres
// ne voudrait plus rien dire.
func BuildCreditBatch(matchID string, events []analysis.RawEvent) persist.KillSourceBatch {
	pairs := analysis.ComputeKillerVictimPairs(events, creditToleranceMS)
	batch := persist.KillSourceBatch{
		MatchID:    matchID,
		DecoderRev: CreditDecoderRev,
		// PUBLIABLE LIGNE PAR LIGNE : ces lignes valent EXACTEMENT ce que vaut
		// `killer_victim_pairs` aujourd hui — meme source, meme appariement — et cette table
		// est deja lue ligne par ligne (match-view, timeline K/D, penalite de depart LUSR).
		// Les declarer non publiables retirerait une capacite existante au nom d une prudence
		// qui ne repose sur aucune mesure.
		Publishable: true,
		Deaths:      make([]persist.KillEventInsert, 0, len(pairs)),
	}
	for _, p := range pairs {
		batch.Deaths = append(batch.Deaths, persist.KillEventInsert{
			TimeMS:             int(p.TimeMS),
			VictimGamertag:     p.VictimGT,
			VictimXUID:         p.VictimXUID,
			FeedKillerGamertag: p.KillerGT,
			FeedKillerXUID:     p.KillerXUID,
			// Le kill-feed porte bien cette mort : c est LUI la source de ces lignes.
			FeedPresent: true,
			// ON NE SAIT PAS s il y avait un assistant. Le reste des champs d assistant, de
			// source et de parts reste a zero — et le persister les ecrit NULL, jamais 0.
			AssistKnown: false,
			ReadPath:    killscope.ReadPathCreditBackfill,
			ReadOrigin:  killscope.OriginCreditOnly,
		})
	}
	// LA DEDUPLICATION N EST PAS UNE PRECAUTION, ELLE EST NECESSAIRE — et c est une mesure du
	// 2026-08-02 qui l impose : `highlight_events` porte 15 120 groupes `(match_id, time_ms,
	// xuid)` en DOUBLE EXACT sur les 394 matchs sans film. Appariés tels quels, ils rendaient
	// 50 125 morts la ou l oracle deduplique en compte 35 224 — 140 %. Une base qui sur-compte de
	// 42 % n est pas une base, et l invariant « une passe ne porte jamais moins que la base »
	// deviendrait ininterpretable. Une seule definition de la deduplication dans le depot.
	batch.Deaths = persist.DedupCreditDeaths(batch.Deaths)
	batch.CreditBaseCount = len(batch.Deaths)
	return batch
}

// write : l ecriture, sous le lease RW de shared (ADR 0013).
func (c *CreditCollector) write(ctx context.Context, batch persist.KillSourceBatch) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", batch.MatchID, err)
	}
	defer release()
	return persist.NewKillSourcePersister(db).PersistPass(ctx, batch)
}

// CreditSummary : ce qu une passe credit a produit.
//
// `Enriched` remplace `SkippedFilm` : la preseance inversee, un match couvert par un film n est
// plus SAUTE, il est ecrit AVEC son enrichissement. Les deux compteurs se ressemblent, ils ne
// disent pas la meme chose — et `Written` + `Enriched` totalisent les matchs ecrits.
type CreditSummary struct {
	Total       int
	Written     int
	Enriched    int
	Deaths      int
	NoEvents    int
	Errors      int
	ElapsedTime time.Duration
}

// CollectMatches : la passe de masse. Une erreur sur UN match ne l arrete pas — elle est comptee
// et journalisee ; seul l arret de l appelant interrompt, et il rend la synthese de ce qui a ete
// fait.
func (c *CreditCollector) CollectMatches(ctx context.Context, matchIDs []string) CreditSummary {
	start := time.Now()
	sum := CreditSummary{Total: len(matchIDs)}
	for _, id := range matchIDs {
		if ctx.Err() != nil {
			slog.InfoContext(ctx, "killsource credit: passe interrompue par l appelant",
				"traites", sum.Written+sum.Enriched+sum.NoEvents+sum.Errors, "total", sum.Total)
			break
		}
		outcome, deaths, err := c.CollectMatch(ctx, id)
		if err != nil {
			sum.Errors++
			slog.ErrorContext(ctx, "killsource credit: match en echec", "match_id", id, "err", err)
			continue
		}
		switch outcome {
		case CreditWritten:
			sum.Written++
			sum.Deaths += deaths
		case CreditEnriched:
			sum.Enriched++
			sum.Deaths += deaths
		case CreditNoEvents:
			sum.NoEvents++
		}
	}
	sum.ElapsedTime = time.Since(start)
	slog.InfoContext(ctx, "killsource credit: passe terminee",
		"total", sum.Total, "ecrits", sum.Written, "enrichis_par_un_film", sum.Enriched,
		"morts", sum.Deaths, "sans_evenement", sum.NoEvents,
		"erreurs", sum.Errors, "duration", sum.ElapsedTime)
	return sum
}
