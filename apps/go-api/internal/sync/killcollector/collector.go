package killcollector

// collector.go — LE COLLECTEUR : il enchaine telechargement, decodage et ecriture,
// et c est TOUT ce qu il fait. Chacune des trois responsabilites vit ailleurs et n a pas le
// droit de migrer ici :
//
//	decoder un film        games/halo_infinite/film/killsource   ne touche ni base ni reseau
//	telecharger les chunks killsource_bridge.go                  ne decode pas
//	ecrire les lignes      persist.KillSourcePersister           ne decide pas QUOI ecrire
//	enchainer les trois    CE FICHIER                            ne contient aucune logique de decodage
//
// # POURQUOI UNE PASSE SEPAREE, ET PAS LE SYNC PRIMAIRE
//
// LE FILM N EXISTE PAS ENCORE QUAND LE MATCH ARRIVE. Le sync primaire ecrit le match dans
// `match_registry` quelques secondes apres sa fin ; le film Theater se publie plus tard. Un
// collecteur branche dans le sync primaire echouerait donc sur la quasi-totalite des matchs
// frais, et « reussirait » surtout sur les vieux. C est une passe SEPAREE, sur des matchs DEJA
// presents au registre.
//
// # LE COUT EST UNE CONTRAINTE DE CONCEPTION, PAS UN DETAIL
//
//	4v4 (8-10 joueurs)      8 a 30 s par film
//	BTB (36 participants)   11 minutes  (mesure du 2026-07-31 sur 4f77afc1)
//
// Consequences, toutes tenues ici :
//   - TACHE DE FOND, jamais dans le chemin d une requete HTTP. Le type ne fournit aucun
//     handler et aucune methode ne doit etre appelee depuis `api/` ;
//   - UNE LIMITE DE TEMPS PAR MATCH + un compteur d abandons. Sans limite, un seul film
//     pathologique bloque la passe entiere ;
//   - UN SEUL DECODAGE A LA FOIS DANS LE PROCESS. Les parametres de replication de `filmdec`
//     sont des GLOBAUX DE PAQUET ; `killsource.Decode` serialise deja par un verrou et remet
//     ces globaux a zero a chaque entree. **Ne pas contourner** : paralleliser deux films
//     n accelere rien et contaminerait les deux. Le collecteur traite donc les matchs EN SERIE,
//     et c est un choix, pas une simplification.
//
// # TITLE-AGNOSTIC
//
// Le DECODEUR est title-specific (`games/halo_infinite/`) et c est correct : le format de film
// est propre a Halo Infinite. LE COLLECTEUR, lui, se branche sur la CAPABILITY
// `film.kill_source` — jamais sur `slug == "halo_infinite"` (ratchet no_slug_comparison_test.go).
// Capability absente -> `games.ErrCapabilityNotSupported`, le cycle CONTINUE, aucun panic.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// KillSourceDecoderRev — la version du decodeur, ecrite sur CHAQUE ligne produite.
//
// Elle ne sert pas a faire joli : c est elle qui permettra de savoir QUELS matchs redecoder
// apres un changement de decodage, au lieu de tout reprendre (1 325 films a 8-30 s = 3 a 11 h).
// LA FAIRE EVOLUER a chaque changement de decodage qui change les lignes produites.
const KillSourceDecoderRev = "killsource-2026-07-31"

// defaultKillSourceTimeout — la limite de temps PAR MATCH.
//
// CALIBRAGE SUR MESURE, PAS SUR INTUITION (2026-08-01, machine locale) :
//
//	8 chunks     1,6 s        69 chunks   1 145 s  (19 min 05)
//	28 chunks   10,4 s        63 chunks     575 s  ( 9 min 35)
//	33 chunks   14,4 s
//
// ⚠ LA COURBE EST VIOLEMMENT SUPERLINEAIRE, ET C EST CE QUI FIXE CETTE VALEUR : passer de 63 a
// 69 chunks (+9,5 %) DOUBLE le temps (+99 %). Le cout par chunk va de 0,44 s en regime 4v4 a
// 16,6 s sur le plus gros film du corpus — un facteur 38. Une telle non-linearite suggere une
// FAILLE LOGIQUE (un cout quadratique quelque part), pas un manque de puissance : elle est A
// PROFILER, et elle est consignee au plan comme telle. Elle n est PAS traitee ici.
//
// La limite vaut donc 45 min : ~2,4x le pire cas MESURE, parce qu avec une telle pente le
// prochain film un peu plus gros peut couter beaucoup plus. Une limite trop juste ne
// protegerait de rien — elle transformerait un film lent mais valide en PERTE DE DONNEE.
const defaultKillSourceTimeout = 45 * time.Minute

// Compteurs de sante du collecteur (ADR 0009 : entiers, snake_case, aucun ratio).
const (
	metricCollected   = "killsource_matchs_collectes"
	metricNoFilm      = "killsource_films_absents"
	metricNoKillFeed  = "killsource_sans_killfeed"
	metricDecodeError = "killsource_erreurs_decodage"
	metricTimeout     = "killsource_abandons_delai"
	metricWriteError  = "killsource_erreurs_ecriture"
	metricDeaths      = "killsource_morts_ecrites"
	metricNotPublish  = "killsource_passes_non_publiables"
	metricAssistExtra = "killsource_assist_extra_count"
)

// KillSourceRoster : la resolution `gamertag -> xuid` pour UN match.
//
// Elle est ICI et pas dans le decodeur parce que le film ne porte AUCUN xuid cote replication :
// il ne rend que des noms. Un nom non resolu n est pas une erreur — c est le cas normal d un BOT
// (qui n a pas de xuid) et le cas honnete d un nom que le roster n a pas su rattacher.
type KillSourceRoster interface {
	// RosterForMatch rend la table `gamertag -> xuid` des participants du match.
	RosterForMatch(ctx context.Context, matchID string) (map[string]string, error)
}

// KillSourceCollector : la passe de fond qui remplit `shared.match_kill_events`.
type KillSourceCollector struct {
	client        filmChunkFetcher
	roster        KillSourceRoster
	acquireShared persist.SharedWriterFn
	caps          games.CapabilityMap
	timeout       time.Duration
}

// NewKillSourceCollector construit le collecteur.
//
//   - client        : source des chunks (le pont l utilise) ;
//   - roster        : resolution nom -> xuid, contre les participants du match ;
//   - acquireShared : ouverture RW de shared_matches_v2.duckdb AVEC son lease (ADR 0013) — la
//     meme fonction que celle du CombinedPersister ;
//   - caps          : la CapabilityMap du titre (capabilities.toml). C est elle qui autorise ou
//     refuse la passe, jamais une comparaison de slug ;
//   - timeout       : limite par match. 0 = defaut.
func NewKillSourceCollector(
	client filmChunkFetcher,
	roster KillSourceRoster,
	acquireShared persist.SharedWriterFn,
	caps games.CapabilityMap,
	timeout time.Duration,
) *KillSourceCollector {
	if timeout <= 0 {
		timeout = defaultKillSourceTimeout
	}
	return &KillSourceCollector{
		client:        client,
		roster:        roster,
		acquireShared: acquireShared,
		caps:          caps,
		timeout:       timeout,
	}
}

// KillSourceOutcome : ce qui est arrive a UN match. Le collecteur ne rend jamais une erreur pour
// un film absent ou sans kill-feed — ce sont des ETATS, pas des pannes, et les confondre ferait
// arreter une passe de backfill sur le premier vieux match.
type KillSourceOutcome string

const (
	// OutcomeWritten : la passe a ete decodee et ecrite.
	OutcomeWritten KillSourceOutcome = "ecrit"
	// OutcomeNoFilm : aucun film pour ce match. Cas NORMAL — les films Theater expirent cote
	// serveur, au moins 28 % des matchs n en auront jamais.
	OutcomeNoFilm KillSourceOutcome = "film-absent"
	// OutcomeNoKillFeed : film present mais sans chunk HIGHLIGHT : aucun couple tueur/victime
	// n est reconstituable, il n y a rien a publier.
	OutcomeNoKillFeed KillSourceOutcome = "sans-killfeed"
	// OutcomeTimeout : le decodage a depasse la limite de temps du match.
	OutcomeTimeout KillSourceOutcome = "abandon-delai"
	// OutcomeNotSupported : le titre n expose pas la capability. Le cycle continue.
	OutcomeNotSupported KillSourceOutcome = "capability-absente"
)

// CollectMatch : LE point d entree, un match a la fois.
//
// Il journalise a l ENTREE et a la SORTIE avec `match_id` et la duree — c est ce qui rendra le
// backfill lisible pendant qu il tourne, et c est aussi la seule facon de voir venir l anomalie
// de cout du BTB.
//
// Contrat d erreur : une erreur rendue est une VRAIE panne (reseau, base, decodage casse). Un
// film absent, un film sans kill-feed, un depassement de delai et une capability absente sont
// des OUTCOMES, pas des erreurs — la passe appelante continue.
func (c *KillSourceCollector) CollectMatch(ctx context.Context, matchID string) (KillSourceOutcome, error) {
	if !c.caps.Has(games.CapFilmKillSource) {
		// Degradation gracieuse : ni panic, ni erreur remontee — le cycle continue.
		slog.DebugContext(ctx, "killsource: capability absente, passe ignoree",
			"match_id", matchID, "capability", string(games.CapFilmKillSource),
			"err", games.ErrCapabilityNotSupported)
		return OutcomeNotSupported, nil
	}

	start := time.Now()
	slog.InfoContext(ctx, "killsource: decodage du film — debut", "match_id", matchID)

	// LA LIMITE DE TEMPS PAR MATCH. Elle couvre telechargement ET decodage : le cout observe
	// est domine par le decodage, mais un CDN qui ne repond pas bloquerait tout autant.
	matchCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	outcome, deaths, err := c.collect(matchCtx, matchID)
	dur := time.Since(start)

	// Le depassement de delai se reconnait au ctx du MATCH, pas a celui de l appelant : un
	// arret demande par l appelant (Ctrl-C, shutdown) n est pas un abandon du collecteur.
	if err != nil && errors.Is(matchCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
		observability.AddInt(metricTimeout, 1)
		slog.WarnContext(ctx, "killsource: abandon sur limite de temps",
			"match_id", matchID, "duration", dur, "limite", c.timeout)
		return OutcomeTimeout, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "killsource: decodage du film — echec",
			"match_id", matchID, "duration", dur, "err", err)
		return outcome, err
	}

	slog.InfoContext(ctx, "killsource: decodage du film — fin",
		"match_id", matchID, "duration", dur, "resultat", string(outcome), "morts", deaths)
	return outcome, nil
}

// collect : le corps, sans la journalisation ni la mesure de duree.
func (c *KillSourceCollector) collect(ctx context.Context, matchID string) (KillSourceOutcome, int, error) {
	src, found, err := ChunkSourceForMatch(ctx, c.client, matchID)
	if err != nil {
		observability.AddInt(metricDecodeError, 1)
		return OutcomeNoFilm, 0, err
	}
	if !found {
		observability.AddInt(metricNoFilm, 1)
		return OutcomeNoFilm, 0, nil
	}

	// `nil` = la CONFIGURATION GELEE, celle qui a produit les chiffres publies. Ne jamais
	// passer d Options ici sans une raison ecrite : ce sont elles qui definissent le decodage.
	res, err := killsource.Decode(ctx, matchID, src, nil)
	if err != nil {
		// Un film sans kill-feed n est pas une panne : c est un film dont on ne peut rien
		// publier. Le distinguer evite qu un backfill s arrete sur un vieux match.
		if errors.Is(err, killsource.ErrNoKillFeed) {
			observability.AddInt(metricNoKillFeed, 1)
			slog.InfoContext(ctx, "killsource: film sans kill-feed, rien a publier",
				"match_id", matchID)
			return OutcomeNoKillFeed, 0, nil
		}
		observability.AddInt(metricDecodeError, 1)
		return OutcomeNoFilm, 0, fmt.Errorf("decodage %s: %w", matchID, err)
	}

	roster, err := c.roster.RosterForMatch(ctx, matchID)
	if err != nil {
		return OutcomeNoFilm, 0, fmt.Errorf("roster %s: %w", matchID, err)
	}

	batch := BuildKillSourceBatch(matchID, res, roster)
	if err := c.write(ctx, batch); err != nil {
		observability.AddInt(metricWriteError, 1)
		return OutcomeWritten, 0, err
	}

	publishKillSourceMetrics(res, batch)
	return OutcomeWritten, len(batch.Deaths), nil
}

// write : l ecriture, sous le lease RW de shared (ADR 0013 — un seul writer par DB).
//
// Le chemin est le persister DEDIE et pas `BatchBuilder` : une passe de decodage arrive sur un
// match DEJA insere, donc sans `Shared.Match`, et `SharedPersister` y serait un no-op. Le chemin
// builder existe (`SetKillSource`) et reste le bon quand un film serait pret des le sync
// primaire — ce qui n arrive pas aujourd hui.
func (c *KillSourceCollector) write(ctx context.Context, batch persist.KillSourceBatch) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", batch.MatchID, err)
	}
	defer release()
	return persist.NewKillSourcePersister(db).PersistPass(ctx, batch)
}

// BuildKillSourceBatch traduit le resultat du decodeur en lignes ecrivables.
//
// C EST LA SEULE FONCTION DU LOT OU LES TROIS ETATS DE L ASSISTANT PEUVENT SE PERDRE, et elle
// est exportee pour que le backfill (session suivante) emprunte EXACTEMENT le meme chemin — une
// seconde traduction serait une seconde chance de les confondre.
//
// Les trois traductions qui ne sont pas des copies de champ :
//
//	Assist.Index == -1        -> nil (NULL). Ecrire -1 fabriquerait un indice de replication.
//	KillerDamage.Known false  -> nil (NULL). « Non mesure » n est jamais zero.
//	AssistDamage              -> nil AUSSI quand l assistant n est pas NOMME. ⚠ `Known` peut
//	                             valoir VRAI avec un `Name` VIDE (assistant REFUSE) : le champ
//	                             etait present, la part est mesuree, mais son PORTEUR est refuse
//	                             — et sans assistant nomme ce bloc porte une constante par film
//	                             qui ne veut rien dire (elle vaut 20 sur certains films). Le
//	                             persister refuse cette ligne, et il a raison.
//
// AUCUN PLAFOND A 100 sur les parts : 1,7 % des kill-events vont jusqu a 228, ce sont des
// donnees. Le seul plafond applique est celui du TYPE (uint8, 255) — et si une valeur le
// depassait, c est le type qu il faudrait elargir, pas la valeur qu il faudrait ecreter.
func BuildKillSourceBatch(matchID string, res *killsource.Result, roster map[string]string) persist.KillSourceBatch {
	batch := persist.KillSourceBatch{
		MatchID:     matchID,
		DecoderRev:  KillSourceDecoderRev,
		Publishable: res.LineByLinePublishable(),
		Deaths:      make([]persist.KillEventInsert, 0, len(res.Kills)),
	}
	for i := range res.Kills {
		batch.Deaths = append(batch.Deaths, killToInsert(&res.Kills[i], roster))
	}
	return batch
}

// killToInsert : UNE mort. Decoupe de [BuildKillSourceBatch] pour rester sous le plafond de
// longueur du depot (80 lignes).
func killToInsert(k *killsource.Kill, roster map[string]string) persist.KillEventInsert {
	d := persist.KillEventInsert{
		TimeMS:             k.TimeMS,
		VictimGamertag:     k.Victim,
		VictimXUID:         roster[k.Victim],
		FeedKillerGamertag: k.Feed.Killer,
		FeedKillerXUID:     roster[k.Feed.Killer],
		FeedPresent:        k.Feed.Present,
		AssistGamertag:     k.Assist.Name,
		AssistKnown:        k.Assist.Known,
		AssistRejected:     k.Assist.Rejected,
		AssistExtra:        k.Assist.Extra,
		SourceTag:          k.Source.Tag,
		Diverges:           k.Diverges,
		ReadPath:           string(k.Read.Path),
		ReadOrigin:         string(k.Read.Origin),
	}
	if k.Assist.Name != "" {
		d.AssistXUID = roster[k.Assist.Name]
	}
	// La categorie voyage AVEC le tag : les deux sortent de la meme lecture du dead-state, et
	// le persister refuse une demi-source. Tag nul = source NON MESUREE, donc categorie vide.
	if k.Source.Tag != 0 {
		d.SourceCategory = k.Source.Category.Name()
	} else {
		// Sans source mesuree, la divergence est INDEFINISSABLE (elle compare les deux
		// verites). Ecrire FALSE la ferait passer pour « mesure : pas de divergence ».
		d.Diverges = false
	}
	if k.Assist.Index >= 0 {
		idx := k.Assist.Index
		d.AssistIndex = &idx
	}
	if k.KillerDamage.Known {
		d.KillerDamagePct = pctToU8(k.KillerDamage.Pct)
	}
	if k.AssistDamage.Known && k.Assist.Name != "" {
		d.AssistDamagePct = pctToU8(k.AssistDamage.Pct)
	}
	return d
}

// pctToU8 : la part de degats, bornee par le TYPE et par lui seul.
//
// ⚠ `uint8` plafonne a 255 pour un maximum MESURE a 228 : la marge existe mais elle est mince.
// Si une valeur superieure apparaissait, c est le TYPE qu il faudrait elargir — pas la valeur
// qu il faudrait plafonner. Le log est la pour qu on l apprenne au lieu de le subir.
func pctToU8(pct int) *uint8 {
	if pct < 0 {
		pct = 0
	}
	if pct > 255 {
		slog.Warn("killsource: part de degats au-dela de la capacite du type UTINYINT — "+
			"ELARGIR LE TYPE, ne pas plafonner la valeur", "pct", pct)
		pct = 255
	}
	v := uint8(pct)
	return &v
}

// publishKillSourceMetrics publie les compteurs de sante (ADR 0009).
//
// `killsource_assist_extra_count` est LE declencheur de migration vers une table fille : le jour
// ou il bouge, l hypothese de schema « un seul assistant » est en defaut. Il est publie ICI en
// plus d etre stocke sur les lignes, parce qu un compteur qu il faut interroger en SQL pour voir
// bouger n alerte personne.
func publishKillSourceMetrics(res *killsource.Result, batch persist.KillSourceBatch) {
	observability.AddInt(metricCollected, 1)
	observability.AddInt(metricDeaths, int64(len(batch.Deaths)))
	if !batch.Publishable {
		observability.AddInt(metricNotPublish, 1)
	}
	extra := 0
	for i := range batch.Deaths {
		extra += batch.Deaths[i].AssistExtra
	}
	if extra > 0 {
		observability.AddInt(metricAssistExtra, int64(extra))
	}
	for _, p := range res.Health.ExpvarPairs() {
		observability.AddInt(p.Name, p.Value)
	}
}
