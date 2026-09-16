package killcollector

// collector.go — LE COLLECTEUR : il enchaine telechargement, decodage et ecriture,
// et c est TOUT ce qu il fait. Chacune des trois responsabilites vit ailleurs et n a pas le
// droit de migrer ici :
//
//	decoder un film        games/halo_infinite/film/internal/facts/killsource   ne touche ni base ni reseau
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
//	                        du mur de cout (`grammar.consumeObjectMultiplayerProperties`)
//
// Consequences, toutes tenues ici :
//   - TACHE DE FOND, jamais dans le chemin d une requete HTTP. Le type ne fournit aucun
//     handler et aucune methode ne doit etre appelee depuis `api/` ;
//   - UNE LIMITE DE TEMPS PAR MATCH + un compteur d abandons. Sans limite, un seul film
//     pathologique bloque la passe entiere ;
//   - UN SEUL DECODAGE A LA FOIS DANS LE PROCESS. Les parametres de replication de `grammar`
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
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
)

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
// `grammar.consumeObjectMultiplayerProperties`, qui sautait le corps d un TLV en lisant un octet
// a la fois, plafonne a 1 048 576 iterations — sur une traversee mal alignee, la longueur lue est
// du bruit et declenchait ce million de lectures. Le cout par chunk allait de 0,20 s a 16,6 s
// (facteur 83) ; il va desormais de 0,13 s a 0,68 s.
//
// LA LIMITE RESTE A 45 MIN, et ce n est pas de la negligence : elle vaut ~57x le pire cas mesure
// aujourd hui. Une limite serree ne protegerait de rien de plus (aucun film connu n en approche)
// et transformerait un film lent mais VALIDE en perte de donnee. Elle est la pour le cas
// pathologique inconnu, pas pour le regime nominal.
const defaultKillSourceTimeout = 45 * time.Minute

// KillSourceRoster : la resolution `gamertag -> xuid` pour UN match.
//
// Elle est ICI et pas dans le decodeur parce que les CHUNKS DE REPLICATION ne portent aucun
// xuid : ils ne rendent que des noms. Un nom non resolu n est pas une erreur — c est le cas
// normal d un BOT (qui n a pas de xuid) et le cas honnete d un nom que le roster n a pas su
// rattacher.
//
// ⚠ « LE FILM NE PORTE AUCUN XUID » ETAIT ECRIT ICI, ET C EST FAUX DEPUIS LE LOT 1.5 : `chunk_00`
// porte les trente-deux enregistrements de slot, XUID et gamertag compris, et le lot 1.8 les
// emploie dans le decodeur (`killsource/film_table.go`). Ce qui reste vrai : la table est celle du
// DEBUT du film et elle ignore les bots, donc la base garde la resolution de PUBLICATION.
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
	// budget : duree maximale d une passe MULTI-MATCHS. 0 = sans borne (backfill de nuit).
	// Verifie ENTRE deux matchs, jamais au milieu d une ecriture.
	budget  time.Duration
	timeout time.Duration
	// mapNames / mapBounds : cablage OPTIONNEL de la RESOLUTION DE CARTE du collecteur — cf.
	// WithPositionCapture. Les deux sont necessaires ensemble ; nil = positions desactivees,
	// degradation journalisee PAR MATCH (configuration, pas panne). DEUX passes les partagent
	// depuis le lot 1.9.4 : les positions (G.2bis) ET les distances de touche (`map_identity.go`).
	mapNames  port.ReplayMapNameRepo
	mapBounds *decfilm.MapQuantCatalog
	// filmDir : la CONFIGURATION du numerateur film (precision par arme + distance, collectHits
	// — acquis du chantier precision remis le 2026-09-01, exposition API retiree). nil = passe
	// non configuree (chemin live sans cache) -> precision ignoree. Voir ConfigureFilmAccuracy.
	filmDir FilmDirResolver
}

// FilmDirResolver rend le repertoire disque des chunks d un film (chunk_NN.bin, format
// grammar.ReadFilmChunk), ou "" si le film n est pas sur disque pour ce match. Le numerateur de
// precision par arme (collectHits) est DIR-BASE : il rejoue le film avec les scanners filmdec
// (positions bipedes, damage_aftermath) qui lisent des fichiers chunk. Le cache disque local
// (data/cache/film_chunks/{matchID}) satisfait ce format ; le chemin live (chunks en memoire,
// hors cache) n a pas de repertoire -> la passe est alors sautee proprement.
type FilmDirResolver func(matchID string) string

// ConfigureFilmAccuracy branche le numerateur film (collectHits). `dir` resout le repertoire de
// chunks d un match. Sans cet appel, la passe de precision par arme ne tourne pas (degradation
// gracieuse).
//
// ELLE NE PREND PLUS DE CHEMIN DE CATALOGUE DEPUIS LE LOT 1.9.4 : le collecteur n en a qu UN,
// celui de [KillSourceCollector.WithPositionCapture]. Deux configurations du meme catalogue, ce
// seraient deux verites possibles pour la meme carte (`map_identity.go`).
func (c *KillSourceCollector) ConfigureFilmAccuracy(dir FilmDirResolver) {
	c.filmDir = dir
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

// WithBudget borne la duree d une passe multi-matchs. 0 (defaut) = sans borne.
//
// A UTILISER PARTOUT OU LA PASSE PARTAGE SON TEMPS AVEC AUTRE CHOSE — typiquement le cycle de
// sync, dont les etapes suivantes attendent derriere. Le solde est repris a la passe suivante,
// la collecte etant idempotente (`decoder_rev` fait foi). Le backfill CLI, lui, ne borne rien.
func (c *KillSourceCollector) WithBudget(d time.Duration) *KillSourceCollector {
	c.budget = d
	return c
}

// WithPositionCapture active la capture des positions monde par kill (`shared.kill_positions`,
// G.2bis), EN PLUS des morts et des tirs. Chainable, meme patron que les Withers du rejeu 2D
// (WithFrameInterval, WithLocalFilmCache) : un reglage optionnel qui ne grossit pas la liste de
// parametres positionnels de [NewKillSourceCollector] (deja a 5, le plafond du depot).
//
// LES DEUX ARGUMENTS SONT NECESSAIRES ENSEMBLE. `mapNames` resout les identites de carte
// candidates d un match (meme port que le rejeu 2D, `port.ReplayMapNameRepo` — implemente par
// `platform/duckdb.ReplayMapRepo`) ; `mapBounds` est le catalogue de bornes de dequantification
// (`decfilm.LoadMapQuantCatalog`, meme fichier que replaybuild). Passer l un sans l autre revient
// a ne rien cabler : [collectPositions] verifie les deux et degrade proprement (Debug, pas
// d erreur) si l un des deux manque.
//
// nil, nil DESACTIVE explicitement la capture (retour a l etat par defaut du constructeur).
func (c *KillSourceCollector) WithPositionCapture(
	mapNames port.ReplayMapNameRepo, mapBounds *decfilm.MapQuantCatalog,
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
