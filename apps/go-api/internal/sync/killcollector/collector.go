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
//	4v4 (8-10 joueurs)      1 a 10 s par film
//	le plus gros du corpus  47 s (69 chunks)   — mesures du 2026-08-01, APRES le correctif
//	                        du mur de cout (`filmdec.consumeObjectMultiplayerProperties`)
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

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/haloclient"
)

// KillSourceDecoderRev — la version du decodeur, ecrite sur CHAQUE ligne produite.
//
// Elle ne sert pas a faire joli : c est elle qui permettra de savoir QUELS matchs redecoder
// apres un changement de decodage, au lieu de tout reprendre (1 325 films a 8-30 s = 3 a 11 h).
// LA FAIRE EVOLUER a chaque changement de decodage qui change les lignes produites.
const KillSourceDecoderRev = "killsource-2026-07-31"

// defaultKillSourceTimeout — la limite de temps PAR MATCH.
//
// CALIBRAGE SUR MESURE, PAS SUR INTUITION. Les deux colonnes disent l histoire :
//
//	chunks    avant le 2026-08-01     apres
//	   8               1,6 s          1,0 s
//	  33              14,4 s          7,3 s
//	  42             131 s           24,8 s
//	  63             575 s           51 s
//	  69           1 145 s           46,7 s      <- x24,5
//
// LE MUR A ETE PROFILE PUIS CORRIGE (J4 session 2) : 78 % du temps partait dans
// `filmdec.consumeObjectMultiplayerProperties`, qui sautait le corps d un TLV en lisant un octet
// a la fois, plafonne a 1 048 576 iterations — sur une traversee mal alignee, la longueur lue est
// du bruit et declenchait ce million de lectures. Le cout par chunk allait de 0,20 s a 16,6 s
// (facteur 83) ; il va desormais de 0,13 s a 0,68 s.
//
// LA LIMITE RESTE A 45 MIN, et ce n est pas de la negligence : elle vaut ~57x le pire cas mesure
// aujourd hui. Une limite serree ne protegerait de rien de plus (aucun film connu n en approche)
// et transformerait un film lent mais VALIDE en perte de donnee. Elle est la pour le cas
// pathologique inconnu, pas pour le regime nominal.
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
	// IdentitiesForMatch rend tout ce que la passe doit savoir des participants : leurs deux
	// tables de noms et la reference `shots_fired` de l API.
	IdentitiesForMatch(ctx context.Context, matchID string) (MatchIdentities, error)
}

// KillSourceCollector : la passe de fond qui remplit `shared.match_kill_events`.
type KillSourceCollector struct {
	client        filmChunkFetcher
	roster        KillSourceRoster
	acquireShared persist.SharedWriterFn
	caps          games.CapabilityMap
	timeout       time.Duration
	// mapNames / mapBounds : cablage OPTIONNEL de la capture des positions
	// (`shared.kill_positions`, G.2bis). Les deux sont necessaires ensemble — cf.
	// WithPositionCapture. nil = positions desactivees, degradation journalisee
	// PAR MATCH en Debug (jamais une erreur : c est une configuration, pas une panne).
	mapNames  port.ReplayMapNameRepo
	mapBounds *filmdec.MapQuantCatalog
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

// WithPositionCapture active la capture des positions monde par kill (`shared.kill_positions`,
// G.2bis), EN PLUS des morts et des tirs. Chainable, meme patron que les Withers du rejeu 2D
// (WithFrameInterval, WithLocalFilmCache) : un reglage optionnel qui ne grossit pas la liste de
// parametres positionnels de [NewKillSourceCollector] (deja a 5, le plafond du depot).
//
// LES DEUX ARGUMENTS SONT NECESSAIRES ENSEMBLE. `mapNames` resout les identites de carte
// candidates d un match (meme port que le rejeu 2D, `port.ReplayMapNameRepo` — implemente par
// `platform/duckdb.ReplayMapRepo`) ; `mapBounds` est le catalogue de bornes de dequantification
// (`filmdec.LoadMapQuantCatalog`, meme fichier que replaybuild). Passer l un sans l autre revient
// a ne rien cabler : [collectPositions] verifie les deux et degrade proprement (Debug, pas
// d erreur) si l un des deux manque.
//
// nil, nil DESACTIVE explicitement la capture (retour a l etat par defaut du constructeur).
func (c *KillSourceCollector) WithPositionCapture(
	mapNames port.ReplayMapNameRepo, mapBounds *filmdec.MapQuantCatalog,
) *KillSourceCollector {
	c.mapNames, c.mapBounds = mapNames, mapBounds
	return c
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
// Le nombre de morts ecrites est RENDU, pas seulement journalise : la synthese d une passe de
// masse doit pouvoir le totaliser. Une premiere version le laissait dans le log — le compteur de
// `KillSourceSummary` restait donc a zero pendant que chaque match en annoncait des dizaines,
// et c est le genre de zero qu on croit longtemps.
func (c *KillSourceCollector) CollectMatch(ctx context.Context, matchID string) (KillSourceOutcome, int, error) {
	if !c.caps.Has(games.CapFilmKillSource) {
		// Degradation gracieuse : ni panic, ni erreur remontee — le cycle continue.
		slog.DebugContext(ctx, "killsource: capability absente, passe ignoree",
			"match_id", matchID, "capability", string(games.CapFilmKillSource),
			"err", games.ErrCapabilityNotSupported)
		return OutcomeNotSupported, 0, nil
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
		return OutcomeTimeout, 0, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "killsource: decodage du film — echec",
			"match_id", matchID, "duration", dur, "err", err)
		return outcome, 0, err
	}

	slog.InfoContext(ctx, "killsource: decodage du film — fin",
		"match_id", matchID, "duration", dur, "resultat", string(outcome), "morts", deaths)
	return outcome, deaths, nil
}

// collect : le corps, sans la journalisation ni la mesure de duree.
//
// UN FILM TELECHARGE UNE FOIS, DEUX TABLES ECRITES. Les morts (`match_kill_events`) et les tirs
// (`match_weapon_shots`) sortent de la MEME passe : re-decoder un film pour la seconde table
// couterait une seconde fois la passe chere du chantier, et donnerait deux occasions de sortir
// deux etats differents de la meme base.
func (c *KillSourceCollector) collect(ctx context.Context, matchID string) (KillSourceOutcome, int, error) {
	chunks, found, err := FilmChunksForMatch(ctx, c.client, matchID)
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
	res, err := killsource.Decode(ctx, matchID, ChunkSourceOf(chunks), nil)
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

	ids, err := c.roster.IdentitiesForMatch(ctx, matchID)
	if err != nil {
		return OutcomeNoFilm, 0, fmt.Errorf("roster %s: %w", matchID, err)
	}

	batch := BuildKillSourceBatch(matchID, res, ids)
	publiees, err := c.write(ctx, batch)
	if err != nil {
		observability.AddInt(metricWriteError, 1)
		return OutcomeWritten, 0, err
	}
	publishKillSourceMetrics(res, batch)

	// LA VENTILATION DES TIRS, SUR LES MEMES CHUNKS. Son echec ne remet PAS en cause les morts :
	// elles sont deja ecrites, elles sont justes, et les deux tables repondent a deux questions
	// differentes. Il se journalise et se compte — jamais il n avale la passe.
	c.collectShots(ctx, matchID, chunks, ids)

	// LES POSITIONS, SUR LES MEMES CHUNKS ET LA MEME LISTE DE MORTS (G.2bis) — memes garanties
	// que les tirs : best-effort, jamais de remise en cause des morts deja ecrites. `batch.Deaths`
	// est la liste PRE-FUSION (avant MergeCreditAndFilm dans c.write) : seule population qui peut
	// structurellement avoir une position (cf. l en-tete de positions.go).
	c.collectPositions(ctx, matchID, chunks, ids, batch.Deaths)

	return OutcomeWritten, publiees, nil
}

// collectShots : la seconde ecriture de la passe — `shared.match_weapon_shots`.
//
// BEST-EFFORT ASSUME, ET C EST UN CHOIX : les morts sont la donnee qui motive le chantier (46,5 %
// de doublons dans `killer_victim_pairs`), les tirs sont un enrichissement. Remonter une erreur
// de ventilation ferait recompter le match comme un echec alors que ses morts sont en base — et
// un backfill le rejouerait, redecodant un film pour rien.
func (c *KillSourceCollector) collectShots(
	ctx context.Context, matchID string, chunks []haloclient.FilmChunk, parts MatchIdentities,
) {
	// DEUX FAMILLES DE DONNÉES, DEUX CAPABILITIES (§4.3 du plan de branchement). Le film
	// est déjà téléchargé et décodé pour les morts ; un titre peut malgré tout exposer le
	// kill enrichi SANS exposer la ventilation des tirs — elle a ses propres réserves
	// (Fiesta et BTB non livrables). Dégradation gracieuse : on n'écrit rien, on ne
	// remonte pas d'erreur, et la passe des morts reste acquise.
	if !c.caps.Has(games.CapFilmWeaponShots) {
		slog.DebugContext(ctx, "killsource: tirs par arme — capability absente, ventilation ignorée",
			"match_id", matchID, "capability", string(games.CapFilmWeaponShots),
			"err", games.ErrCapabilityNotSupported)
		return
	}
	batch := BuildWeaponShotsBatch(matchID, ReplicationChunks(chunks), parts.XUIDs, parts.ShotsFired)
	if err := c.writeShots(ctx, batch); err != nil {
		observability.AddInt(metricShotsWriteFail, 1)
		slog.ErrorContext(ctx, "killsource: ecriture de la ventilation des tirs echouee",
			"match_id", matchID, "err", err)
		return
	}
	publishShotsPass(ctx, batch, len(parts.XUIDs))
}

// write : la FUSION puis l ecriture, sous le lease RW de shared (ADR 0013 — un seul writer par DB).
//
// ─── LE DECODAGE NE PUBLIE PLUS SEUL ───────────────────────────────────────────────────────
//
// Depuis l inversion de preseance (2026-08-03), une passe de film ne remplace plus rien : elle se
// POSE sur la base credit du match. La base est relue ici, dans le meme handle que l ecriture
// (`persist.CreditBaseForMatch` — les couples de `killer_victim_pairs`, dedupliques sur
// l identite, exactement la definition que la migration de reprise emploie), puis
// `persist.MergeCreditAndFilm` apparie sur `(match_id, time_ms)` a tolerance ZERO.
//
// UN MATCH SANS COUPLE REND UNE BASE VIDE, et ce n est pas une erreur : toutes les lignes du film
// deviennent alors des orphelines et la passe vaut ce qu elle valait avant l inversion. Le cas se
// produit sur un match dont la completion combat n a pas encore tourne — le producteur live
// recomposera la passe quand elle arrivera.
//
// Le chemin est le persister DEDIE et pas `BatchBuilder` : une passe de decodage arrive sur un
// match DEJA insere, donc sans `Shared.Match`, et `SharedPersister` y serait un no-op. Le chemin
// builder existe (`SetKillSource`) et reste le bon quand un film serait pret des le sync
// primaire — ce qui n arrive pas aujourd hui.
func (c *KillSourceCollector) write(ctx context.Context, film persist.KillSourceBatch) (int, error) {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return 0, fmt.Errorf("lease shared %s: %w", film.MatchID, err)
	}
	defer release()

	base, err := persist.CreditBaseForMatch(ctx, db, film.MatchID)
	if err != nil {
		return 0, err
	}
	batch, st, err := persist.MergeCreditAndFilm(base, film)
	if err != nil {
		return 0, err
	}
	persist.PublishMergeStats(ctx, film.MatchID, st)
	slog.InfoContext(ctx, "killsource: passe fusionnee sur la base credit",
		"match_id", film.MatchID, "base_credit", len(base.Deaths), "lignes_film", len(film.Deaths),
		"publiees", len(batch.Deaths), "enrichies", st.Enriched, "orphelins", st.Orphans)
	if err := persist.NewKillSourcePersister(db).PersistPass(ctx, batch); err != nil {
		return 0, err
	}
	return len(batch.Deaths), nil
}

// writeShots : l ecriture des tirs, sous son PROPRE lease.
//
// DEUX LEASES COURTS PLUTOT QU UN LONG. Le lease RW de shared est la ressource la plus disputee
// du process (un seul writer, ADR 0013) : le tenir pendant la ventilation — resolution d indices
// au bit pres et scan de tous les chunks — le bloquerait pour rien. Les deux tables sont
// append-only et independantes : rien n exige qu elles s ecrivent dans la meme transaction.
func (c *KillSourceCollector) writeShots(ctx context.Context, batch persist.WeaponShotsBatch) error {
	db, release, err := c.acquireShared(ctx)
	if err != nil {
		return fmt.Errorf("lease shared %s: %w", batch.MatchID, err)
	}
	defer release()
	return persist.NewWeaponShotsPersister(db).PersistPass(ctx, batch)
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
func BuildKillSourceBatch(matchID string, res *killsource.Result, ids MatchIdentities) persist.KillSourceBatch {
	batch := persist.KillSourceBatch{
		MatchID:     matchID,
		DecoderRev:  KillSourceDecoderRev,
		Publishable: res.LineByLinePublishable(),
		Deaths:      make([]persist.KillEventInsert, 0, len(res.Kills)),
	}
	for i := range res.Kills {
		batch.Deaths = append(batch.Deaths, killToInsert(&res.Kills[i], ids))
	}
	return batch
}

// killToInsert : UNE mort. Decoupe de [BuildKillSourceBatch] pour rester sous le plafond de
// longueur du depot (80 lignes).
//
// LES TROIS NOMS PASSENT PAR [MatchIdentities.Resoudre] — victime, tueur, assistant. Aucun ne se
// resout « a la main » : le film peut donner un gamertag OU un xuid, et la regle qui les
// distingue n existe qu a un seul endroit.
func killToInsert(k *killsource.Kill, ids MatchIdentities) persist.KillEventInsert {
	victimeXUID, victimeNom := ids.Resoudre(k.Victim)
	tueurXUID, tueurNom := ids.Resoudre(k.Feed.Killer)
	d := persist.KillEventInsert{
		TimeMS:             k.TimeMS,
		VictimGamertag:     victimeNom,
		VictimXUID:         victimeXUID,
		FeedKillerGamertag: tueurNom,
		FeedKillerXUID:     tueurXUID,
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
		d.AssistXUID, d.AssistGamertag = ids.Resoudre(k.Assist.Name)
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
