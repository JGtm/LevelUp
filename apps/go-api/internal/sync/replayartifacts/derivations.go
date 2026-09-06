package replayartifacts

// derivations.go — LE POINT D'ENTREE UNIQUE DES DERIVATIONS POST-RANGEMENT.
//
// # Le defaut que ce fichier ferme (constat A1 du registre v2)
//
// Les trois derivations d'un artefact — coup d'envoi mesure (t0film.go), resume d'usage
// (usage.go), statistiques d'Assaut (bombstats.go) — etaient cablees a la CUISSON LOCALE, et
// a elle seule : `Run` faisait `if d.Placement == PlacementWorker { enqueueAll(...); return }`
// AVANT de les appeler. Or le placement `worker` est le DEFAUT en production, et `local` y est
// meme REFUSE par construction (`replaybuild.DecidePlacement`). Le jour de l'activation de
// l'ouvrier distant, `match_usage_*`, `match_bomb_stats` et `real_start_time` seraient donc
// restes vides, sans qu'aucun compteur ne le dise.
//
// # La regle, et elle tient en une phrase
//
// LES DERIVATIONS SE DECLENCHENT SUR « UN ARTEFACT VIENT D'ETRE RANGE », JAMAIS SUR « JE VIENS
// DE CUIRE ». Les deux rangeurs appellent le MEME [Deriver] :
//
//	cuisson locale     `buildAll` (cuisson.go) -> `Run` (artifacts.go)
//	depot d'un ouvrier `ServiceRegistry.StoreBuildArtifact` (api/wire/registry_build_queue.go)
//
// Le rattrapage (derivations_backlog.go) est un TROISIEME appelant du meme point d'entree : il
// ne cuit rien, il rejoue les derivations sur des artefacts deja ranges.
//
// # UNE SEULE LECTURE DU DOCUMENT, ET C'EST LE SECOND GAIN
//
// Avant, chaque derivation ouvrait l'artefact pour son compte : `lireT0FilmArtefact`,
// `projeterResumeUsage` et `projeterStatsBombe` faisaient TROIS `os.ReadFile` + deux
// deserialisations d'un document de ~2 Mo par match. Ce fichier lit et deserialise UNE fois,
// puis passe le document aux projections. Meme doctrine que `replaybuild.ArtifactDigest`
// (item 5.3 de PLAN_CUISSON_PERF), pour la meme raison.
//
// # SUR DISQUE, PAS LE BLOB CANDIDAT
//
// La lecture porte sur l'artefact TEL QU'IL EST RANGE : `StoreArtifact` peut REFUSER les octets
// candidats (garde anti-regression) et conserver l'artefact precedent. Projeter le candidat
// ecrirait en base ce que le disque ne porte pas.
//
// # BEST-EFFORT DE BOUT EN BOUT
//
// Aucune erreur ne remonte : ni un cycle de sync ni un depot d'ouvrier ne doit echouer parce
// qu'une projection a rate. Aucune ne se tait non plus — chaque degradation a son journal et
// son compteur.

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/replaybuild"
)

// ArtefactRange : UN artefact qui vient d'etre range sur disque. C'est le seul fait qui
// declenche les derivations, et il ne dit rien de QUI l'a range.
type ArtefactRange struct {
	// MatchID est l'identite CANONIQUE du match (celle du registre cote post-sync, celle du
	// job cote ouvrier), jamais celle que le document revendique — meme regle que
	// `replaybuild.ArtifactStored.MatchID`.
	MatchID string
	// Path est le chemin du fichier RANGE.
	Path string
}

// DerivationsDeps : ce dont les derivations ont besoin, INDEPENDAMMENT du rangeur.
//
// C'est un sous-ensemble strict de [Deps] : le chemin ouvrier n'a ni client film, ni segment de
// lecture, ni placement — lui demander un [Deps] complet l'obligerait a inventer des champs
// qu'il ne possede pas.
type DerivationsDeps struct {
	// RepoRoot, TitleSlug : resolution des chemins et lecture des capabilities du titre.
	RepoRoot  string
	TitleSlug string
	// Gamertag : identite journalisee (aucun role fonctionnel). Vide sur le chemin ouvrier,
	// qui ne travaille pour aucun joueur en particulier.
	Gamertag string
	// AcquireWriter ouvre un segment d'ECRITURE shared COURT. NIL = les derivations LISENT et
	// ne persistent rien, ce qui est journalise par chaque projection. Jamais un panic.
	AcquireWriter func(ctx context.Context) (*sql.DB, func(), error)
}

// deps rend la vue [Deps] que les projections consomment. Les autres champs restent a leur
// zero : aucune projection ne les lit (garde-rail `derivations_test.go`).
func (dd DerivationsDeps) deps() Deps {
	return Deps{
		RepoRoot:      dd.RepoRoot,
		TitleSlug:     dd.TitleSlug,
		Gamertag:      dd.Gamertag,
		AcquireWriter: dd.AcquireWriter,
	}
}

// artefactLu : un artefact range, LU ET DESERIALISE UNE FOIS.
type artefactLu struct {
	matchID string
	path    string
	doc     *replay.ReplayDocument
	// octets : la taille du fichier lu. Le rattrapage des derives s'en sert pour dire si la
	// marque de derivation colle encore au CONTENU (cf. derivations_backlog.go).
	octets int
}

// Deriver applique TOUTES les derivations post-rangement a un lot d'artefacts.
//
// LES DERIVATIONS S'ECRIVENT EN BURSTS GROUPES, comme avant : les documents sont lus et
// projetes d'abord, puis chaque famille acquiert le writer a son tour et le relache aussitot.
// Un lot vide ne fait rien du tout — pas meme une lecture de capabilities.
func Deriver(ctx context.Context, dd DerivationsDeps, ranges []ArtefactRange) {
	if len(ranges) == 0 {
		return
	}
	d := dd.deps()
	lus := lireArtefacts(ctx, d, ranges)
	if len(lus) == 0 {
		return
	}
	b := &bilanDerivations{}
	// LE REPORT DU COUP D'ENVOI EN PREMIER, puis les deux projections. L'ordre est celui
	// d'avant (artifacts.go) et il n'est pas indifferent : le T0 ecrit `match_registry`, une
	// table match-of-record, et il vaut mieux qu'il passe avant les tables derivees si le
	// writer devient indisponible en cours de route.
	reporterT0Film(ctx, d, b, rapportsT0(lus))
	persisterResumesUsage(ctx, d, b, lus)
	persisterStatsBombe(ctx, d, b, lus)
	// LES POSITIONS EN DERNIER (decision utilisateur 1) : c'est la projection la plus VOLUMINEUSE
	// du lot (~215 lignes par match apres decimation, cf. positions.go). La passer apres les
	// trois autres garantit que si le writer devient indisponible en cours de route, ce sont les
	// donnees les moins couteuses a rejouer qui manquent.
	persisterPositions(ctx, d, b, lus)
	marquerDerivations(ctx, b, lus)
}

// bilanDerivations enregistre, PAR MATCH, ce que les familles n'ont PAS pu ecrire.
//
// # LE DEFAUT QU'IL FERME (constat C1 de la revue A-R1, 2026-09-06)
//
// La marque se posait EN FIN DE PASSE, sans condition. Un writer indisponible — aucun writer
// cable sur le chemin, ou acquisition en echec (lease shared tenu, B-swap en cours) — faisait
// journaliser les quatre familles, ecrire ZERO ligne, et poser la marque quand meme.
// `DerivationsUpToDate` rendait alors `true` A JAMAIS et `candidatsDerivations` excluait ce
// match DEFINITIVEMENT : `real_start_time`, `match_usage_*`, `match_bomb_stats` et
// `match_player_positions` restaient vides, sans autre reprise que la suppression manuelle du
// fichier de marque — l'inverse exact de ce que le rattrapage promet.
//
// # CE QU'IL DISTINGUE, ET POURQUOI CE N'EST PAS LA MEME CHOSE
//
//	WRITER INDISPONIBLE  aucune famille n'a pu ouvrir de segment d'ecriture : AUCUNE marque
//	                     n'est posee sur tout le lot. C'est le cas de panne, il est global.
//	ECHEC D'UN MATCH     une famille a refuse ou rate CE match (capabilities illisibles, INSERT
//	                     en erreur, etat du T0 illisible) : ce match seul n'est pas marque, et
//	                     le rattrapage rejouera ses quatre familles au prochain cycle.
//	RIEN A ECRIRE        un titre sans capability film, un document sans `t0FilmMs` ni
//	                     trajectoire, un match hors Assaut : ce sont des derivations JOUEES.
//	                     Elles se marquent — les rejouer a chaque cycle serait du travail
//	                     pur perte.
//
// LE REJEU EST TOUJOURS SUR — les quatre familles se rejouent au complet pour un match non
// marque : trois d'entre elles sont append-only (une passe neuve supersede), et le report du T0
// porte sa propre garde « deja a la meme valeur ».
type bilanDerivations struct {
	// sansWriter : le segment d'ecriture n'a pas pu etre ouvert. Etat GLOBAL a la passe.
	sansWriter bool
	// echecs : les matchs qu'au moins une famille n'a pas pu ecrire.
	echecs map[string]bool
}

// writerIndisponible enregistre qu'aucun segment d'ecriture n'a pu etre ouvert.
func (b *bilanDerivations) writerIndisponible() {
	b.sansWriter = true
}

// echec enregistre qu'une famille n'a pas pu ecrire CE match.
func (b *bilanDerivations) echec(matchID string) {
	if b.echecs == nil {
		b.echecs = make(map[string]bool, 4)
	}
	b.echecs[matchID] = true
}

// echecLot enregistre l'echec de TOUS les artefacts du lot — le cas d'une famille ecartee en
// bloc (capabilities illisibles) ou d'un writer indisponible.
func (b *bilanDerivations) echecLot(lus []artefactLu) {
	for _, a := range lus {
		b.echec(a.matchID)
	}
}

// aEchoue dit si ce match doit etre rejoue au prochain cycle.
func (b *bilanDerivations) aEchoue(matchID string) bool { return b.echecs[matchID] }

// marquerDerivations inscrit, dans l'index des artefacts, que CE contenu a ete derive a la
// revision courante. C'est ce qui permet au rattrapage de distinguer « artefact present » de
// « artefact derive », et c'est ce qui le fait CONVERGER (cf. derivations_backlog.go et
// replaybuild.DerivationsMark).
//
// ON NE MARQUE QUE CE QU'ON A PU ECRIRE (cf. [bilanDerivations]) : writer indisponible = aucune
// marque du lot ; sinon, une marque par match dont aucune famille n'a echoue.
//
// UNE MARQUE QUI ECHOUE NE FAIT RIEN ECHOUER : les derives sont deja ecrits, la seule
// consequence est que le rattrapage les rejouera. C'est journalise, jamais avale.
func marquerDerivations(ctx context.Context, b *bilanDerivations, lus []artefactLu) {
	if b.sansWriter {
		slog.WarnContext(ctx, "post-sync: derivations NON marquees — aucun segment d'ecriture "+
			"shared n'a pu etre ouvert, le rattrapage rejouera tout le lot",
			"artefacts", len(lus), "echecs", len(b.echecs))
		return
	}
	for _, a := range lus {
		if b.aEchoue(a.matchID) {
			slog.WarnContext(ctx, "post-sync: derivation NON marquee — au moins une famille n'a "+
				"pas pu ecrire ce match, le rattrapage la rejouera",
				"match_id", a.matchID, "path", a.path)
			continue
		}
		if err := replaybuild.WriteDerivationsMark(a.path, a.doc.SchemaVersion, a.octets); err != nil {
			slog.WarnContext(ctx, "post-sync: marque de derivation non ecrite — le rattrapage "+
				"rejouera ces derivations au prochain cycle",
				"match_id", a.matchID, "path", a.path, "err", err)
		}
	}
}

// lireArtefacts lit et deserialise chaque artefact UNE fois. Un artefact illisible est un echec
// de CE match, jamais du lot : il est journalise et ecarte.
func lireArtefacts(ctx context.Context, d Deps, ranges []ArtefactRange) []artefactLu {
	out := make([]artefactLu, 0, len(ranges))
	for _, r := range ranges {
		raw, err := os.ReadFile(r.Path)
		if err != nil {
			slog.WarnContext(ctx, "post-sync: artefact range mais illisible — aucune derivation",
				"gamertag", d.Gamertag, "match_id", r.MatchID, "path", r.Path, "err", err)
			continue
		}
		var doc replay.ReplayDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			slog.WarnContext(ctx, "post-sync: artefact range mais indeserialisable — aucune derivation",
				"gamertag", d.Gamertag, "match_id", r.MatchID, "path", r.Path, "err", err)
			continue
		}
		out = append(out, artefactLu{matchID: r.MatchID, path: r.Path, doc: &doc, octets: len(raw)})
	}
	return out
}

// rapportsT0 extrait les coups d'envoi mesures des documents lus.
//
// Un document sans `t0FilmMs` — schema anterieur au champ, ou detecteur qui a REFUSE de dater
// le coup d'envoi — ne donne rien : les deux cas veulent la meme chose, ne rien ecrire.
func rapportsT0(lus []artefactLu) []rapportT0Film {
	out := make([]rapportT0Film, 0, len(lus))
	for _, a := range lus {
		if a.doc.T0FilmMs == nil {
			continue
		}
		out = append(out, rapportT0Film{matchID: a.matchID, t0FilmMs: *a.doc.T0FilmMs})
	}
	return out
}
