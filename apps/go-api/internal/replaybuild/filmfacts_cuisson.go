package replaybuild

// filmfacts_cuisson.go — LA BASCULE : REJOUER DEPUIS LES FAITS QUAND ILS SONT FRAIS, DECODER
// SINON (lot 4.1.2 du PLAN_DECODEUR_FILM, 2026-09-17).
//
// # OU LA BASCULE SE PREND, ET POURQUOI LA
//
// Dans `BuildBytes`, APRES `ResolveMapEntry` et AVANT tout chargement de film. L entree de
// catalogue sert aux DEUX branches et elle VALIDE l en-tete des faits : les positions sont des
// quanta, et les relire avec l entree d une autre carte rendrait des coordonnees FAUSSES et pas
// approximatives. La branche « relire » court-circuite donc `ouvrirManifeste`, `chargerFilm`, la
// lecture du statborg, le decodage `killsource` et `replay.BuildFromFilmAvecFaits` ;
// `json.Marshal(doc)` reste COMMUN aux deux (`serialiserDocument`).
//
// # UNE SEULE SUITE D ETAPES OBSERVEES, ET C EST UNE CONTRAINTE, PAS UN HASARD
//
// Les deux branches convergent AVANT la premiere etape observee : ce qui les separe est
// [entreesDeCuisson] — d ou viennent le statborg, le fil des morts et le resultat `killsource` —
// et rien d autre. `assemblerFilmStats`, `collecterEntreesCatalogue` et `serialiserDocument` sont
// communs. Sans cela, `observe_test.go` verrait la suite DEUX FOIS dans le source et le harnais
// d equivalence n aurait plus de sequence de reference.
//
// # CE QUE LA BRANCHE « RELIRE » NE CHANGE PAS
//
//	LE VERROU SOLO       `filmproc.AcquireSolo` est pris par les SIX points d entree, jamais par
//	                     ce paquet. Un rejeu depuis les faits ne decompresse aucun chunk et ne le
//	                     merite pas, mais le sortir du verrou deplacerait une garantie MEMOIRE
//	                     dans une boucle — la porte par laquelle quatre sinistres RAM sont
//	                     passes. RIEN NE CHANGE ICI (note de preparation de M4, §2.5).
//	LE PUITS D ARTEFACT  l ecriture reste `writeArtifactBytes` (`artifact_store.go`), avec ses
//	                     trois refus avant ecriture, sa garde anti-regression et son ecriture
//	                     atomique. `BuildBytes` ne range rien : il rend des octets, et ce sont
//	                     `BuildMatch` et le post-sync qui rangent, par ce chemin unique.
//	                     CORRECTION DU PLAN, item 4.1.2 : `no_second_artifact_sink_test` ne garde
//	                     PAS cette unicite — il compte les cablages du PUITS DE NOTIFICATION
//	                     (Discord groupe, deux appelants autorises), pas les ecritures d octets.
//	                     Ce que ce lot preserve et PROUVE est le passage par
//	                     `writeArtifactBytes` (`TestBrancheDesFaitsTraverseLeMemePuits`).
//
// # LE CHEMIN DECODE ECRIT SES FAITS — SI LE PUITS RANGERAIT SON ARTEFACT (lot J3.4, RA1-1)
//
// Les faits d une cuisson qui a decode se rangent APRES la serialisation de l artefact, et
// seulement si le puits le rangerait (`rangerLesFaits`) : un artefact appauvri refuse ne laisse
// pas derriere lui des faits cuits sous des gardes pauvres. Les faits frais qui ne couvrent pas
// les gardes de la cuisson courante font redecoder (`documentDeLaCuisson`).
//
// Ecriture ATOMIQUE (`platform/atomicfile`) : un fichier tronque par une sentinelle memoire serait
// relu comme des faits COMPLETS — la relecture ne juge que l en-tete, pas la vraisemblance des
// sections. L echec d ecriture est journalise et NON FATAL : l artefact est deja construit, et le
// perdre pour un disque plein serait une regression franche.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// EtapeRejeuDepuisLesFaits est le NOM de l etape que l observateur recoit pour dire QUELLE BRANCHE
// a servi.
//
// ELLE EST OBSERVEE DANS LES DEUX CAS, avec un booleen : sans cela, `replay-equiv` ne pourrait pas
// distinguer « cette passe a relu les faits » de « cette etape n existe pas dans ce binaire », et
// le harnais S8 comparerait deux passes sans savoir ce qu il compare.
const EtapeRejeuDepuisLesFaits = "filmFactsRejoue"

// entreesDeCuisson porte CE QUI DIFFERE entre les deux branches, et rien de plus.
//
// `faits` non nil = la branche « relire » a ete RETENUE a la bascule ; `film` non nil = la branche
// « decoder ». Les deux ne sont JAMAIS renseignes ensemble.
type entreesDeCuisson struct {
	// faits : les faits relus, FRAIS (codec, revisions, cle), et `entete` leur en-tete. nil sur le
	// chemin du film. Les GARDES DE L APPELANT se jugent plus tard, sur les options
	// (`documentDeLaCuisson`, lot J3.4).
	faits  *replay.FilmFactsFile
	entete replay.FilmFactsEntete
	// film : le film charge. nil sur le chemin des faits.
	film *decfilm.Film
	// statborg, deaths, kills : les trois entrees que les deux branches fournissent, l une en
	// lisant le film, l autre en relisant les faits.
	statborg replay.FilmStatborg
	deaths   filmDeaths
	kills    *decfilm.Result
}

// entreesDeLaCuisson resout la bascule et rend les entrees de la branche retenue.
func (b *Builder) entreesDeLaCuisson(ctx context.Context, matchID string, mapNames []string,
	filmDir string, entry decfilm.MapQuantEntry,
) (entreesDeCuisson, error) {
	if f, entete := b.lireLesFaitsFrais(ctx, matchID, entry); f != nil {
		src := entreesDesFaits(f)
		src.entete = entete
		return src, nil
	}
	tFilm := time.Now()
	film, err := chargerLeFilmDeLaCuisson(ctx, matchID, filmDir)
	if err != nil {
		return entreesDeCuisson{}, err
	}
	// UNE SEULE LECTURE DU FIL DES MORTS pour les deux consommateurs de cet etage
	// (`identifiedEvents` et `killRefs`, qui ouvraient chacun le chunk highlight).
	deaths := lireMorts(film)
	logPhase("film", matchID, tFilm)
	tStats := time.Now()
	statborg := statborgDuFilm(ctx, matchID, film)
	logPhase("stats", matchID, tStats)
	// UN SEUL décodage killsource par match : neutralDeaths ET killRefs (cf. kills.go) en
	// dérivent tous les deux, pour ne payer qu'UNE fois le verrou filmdec partagé avec la
	// cuisson du rejeu — au lieu de deux, comme avant la jointure des frags sous effet actif
	// (PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1).
	tKS := time.Now()
	kills := b.decodeKillSource(matchID, mapNames, film)
	logPhase("killsource", matchID, tKS)
	return entreesDeCuisson{film: film, statborg: statborg, deaths: deaths, kills: kills}, nil
}

// chargerLeFilmDeLaCuisson charge le film SOUS SES GARDES : manifeste finalise, morceaux decrits,
// cle connue. Un refus sort AVANT toute lecture.
//
// LE FILM EST DECOMPRESSE UNE FOIS ICI, POUR TOUTE LA CUISSON (lot 1, PLAN_CUISSON_PERF item 1.3).
// Avant, chacun des ~20 balayages de `BuildFromFilm` relisait et redecompressait le film entier
// depuis le disque. Le manifeste, deja ouvert pour le statborg, donne le type et le debut de chaque
// chunk ; les NUMEROS, eux, viennent des fichiers presents.
//
// DEUX APPELANTS : la branche « decoder » de la bascule, et `documentDeLaCuisson` quand des faits
// frais ne couvrent pas les gardes de l appelant (lot J3.4) — le meme film, les memes gardes.
func chargerLeFilmDeLaCuisson(ctx context.Context, matchID, filmDir string) (*decfilm.Film, error) {
	src := ouvrirManifeste(ctx, matchID, filmDir)
	// UN FILM NON FINALISE NE SE CUIT PAS (lot L3, 2026-09-23) : refuse AVANT le chargement quand
	// son manifeste ne porte pas le morceau des temps forts, JUSTE APRES quand des morceaux du
	// repertoire n y sont pas decrits (cf. [refuserManifesteNonFinalise]).
	if err := refuserManifesteNonFinalise(ctx, matchID, filmDir, src); err != nil {
		return nil, err
	}
	film := chargerFilm(ctx, matchID, filmDir, src)
	if err := refuserMorceauxHorsManifeste(ctx, matchID, src, film); err != nil {
		return nil, err
	}
	// LA PORTE DE LA CLÉ, AVANT TOUTE LECTURE (lot 3.1.1, cf. cle_du_film.go).
	if err := ecarterSiCleInconnue(ctx, matchID, film); err != nil {
		return nil, err
	}
	return film, nil
}

// documentCuit : ce que la production du document rend a `BuildBytes`.
type documentCuit struct {
	// doc : le document.
	doc replay.ReplayDocument
	// aPersister : les faits a ranger — nil quand le document a ete rejoue DEPUIS les faits (rien
	// de neuf a ecrire) ou quand le decoupage d i0 etait illisible.
	aPersister *replay.FilmFactsFile
	// depuisLesFaits : la branche « relire » a SERVI (et pas seulement ete retenue a la bascule).
	depuisLesFaits bool
}

// documentDeLaCuisson produit le document : DEPUIS LES FAITS quand la bascule les a juges frais
// ET qu ils couvrent les gardes de cette cuisson, DEPUIS LE FILM sinon. Il ne range RIEN : les faits
// d une cuisson decodee se rangent apres l artefact (`rangerLesFaits`, lot J3.4).
//
// LES GARDES SE JUGENT ICI, ET PAS A LA BASCULE (lot J3.4, RA1-1) : elles se derivent des OPTIONS
// ([replay.GardesDe]), et les options se construisent sur le statborg que les faits frais
// fournissent. Des faits qui ne les couvrent pas font charger le film ; le statborg et le resultat
// killsource tires de ces faits restent justes — des faits frais sont ceux du film, sous les memes
// revisions et la meme cle —, seul le balayage des canaux gardes est a refaire.
//
// LES DEUX BRANCHES CONVERGENT CHEZ L'APPELANT, sur `json.Marshal` : un second encodage aurait pu
// diverger d'un reglage, et l'equivalence a l'octet du test S8 n'aurait plus rien prouve.
func (b *Builder) documentDeLaCuisson(ctx context.Context, matchID, filmDir string,
	opts replay.Options, src entreesDeCuisson,
) (documentCuit, error) {
	if src.faits != nil {
		var entry decfilm.MapQuantEntry
		if opts.MapQuant != nil {
			entry = *opts.MapQuant
		}
		err := src.entete.Utilisable(entry, replay.GardesDe(opts))
		if err == nil {
			return documentCuit{doc: replay.BuildFromFacts(matchID, b.titleSlug, src.faits, opts),
				depuisLesFaits: true}, nil
		}
		slog.InfoContext(ctx, "cuisson: faits de film cuits sous d autres gardes — redecodage",
			"match_id", matchID, "raison", err)
		if src.film, err = chargerLeFilmDeLaCuisson(ctx, matchID, filmDir); err != nil {
			return documentCuit{}, err
		}
	}
	tDecode := time.Now()
	doc, aPersister, err := replay.BuildFromFilmAvecFaits(matchID, b.titleSlug, src.film, opts)
	logPhase("decodage", matchID, tDecode)
	if err != nil {
		return documentCuit{}, fmt.Errorf("décodage du film %s: %w", matchID, err)
	}
	completerLesFaits(aPersister, src)
	return documentCuit{doc: doc, aPersister: aPersister}, nil
}

// entreesDesFaits rend les entrees de la branche « relire ».
//
// LE FIL DES MORTS NE SE DUPLIQUE PAS : il est deja dans `FilmInputs.Deaths`. SON ERREUR, ELLE,
// VOYAGE (lot M8 des retours rejeu, 2026-09-24) : les faits portent le verdict de la lecture et sa
// cause (`FilmInputs.DeathsFeed`), et `Erreur()` la reconstruit, au meme texte. Sans elle, un fil
// vide ou illisible arrivait ici SANS erreur : les actions d objectif s identifiaient sur un pont
// sans mort et les frags sous effet actif se disaient lus, la ou le decodage du meme film refuse
// les deux (`identifiedEvents`, `killRefs`). L ancien commentaire (« des faits existants portent
// une lecture reussie ») etait faux : le decodage n echoue pas sur un fil illisible, il le publie.
func entreesDesFaits(f *replay.FilmFactsFile) entreesDeCuisson {
	return entreesDeCuisson{faits: f, statborg: f.Statborg,
		deaths: filmDeaths{list: f.Facts.Deaths, err: f.Facts.DeathsFeed.Erreur()}, kills: f.Kills}
}

// refuserManifesteNonFinalise rend [filmcache.ErrFilmNonFinalise] enveloppee quand le manifeste du
// film ne porte pas le morceau des TEMPS FORTS — la signature d un film archive avant que le
// serveur l ait finalise (`ab526724`, 2026-09-22 : 34 morceaux sur 37, fil des morts illisible,
// toute la chaine d identite tombee).
//
// CUIRE QUAND MEME PUBLIERAIT UN DOCUMENT FAUX, et plausible : c est exactement ce qui s est passe.
// Le refus est ECARTE, jamais echec : le parent (`sync/replayartifacts`) le classe sur le TEXTE de
// la sentinelle, comme `ErrUnknownFilmKey` — l erreur traverse une frontiere de processus.
//
// `src == nil` (aucun manifeste EXPLOITABLE) se juge a part, par [jugerFilmSansManifeste].
func refuserManifesteNonFinalise(ctx context.Context, matchID, filmDir string, src *filmcache.Source) error {
	if src == nil {
		return jugerFilmSansManifeste(ctx, matchID, filmDir)
	}
	if filmcache.Finalise(src.Meta(), typeDeChunkMeta) {
		return nil
	}
	slog.WarnContext(ctx, "cuisson: film ECARTE — manifeste non finalise (sans morceau des temps "+
		"forts) ; aucun artefact n est cuit", "match_id", matchID, "entrees", len(src.Meta()))
	return fmt.Errorf("%w (match %s, %d entrees au manifeste)", filmcache.ErrFilmNonFinalise,
		matchID, len(src.Meta()))
}

// jugerFilmSansManifeste decide du sort d un film dont `ouvrirManifeste` n a rien rendu — ce qui
// couvre DEUX etats que la lecture confond (elle rend nil pour les deux) et que ce lot separe
// (constat L3-R8 de la revue adverse, 2026-09-23) :
//
//	PRESENT MAIS ILLISIBLE  REFUSE. Un manifeste corrompu ne prouve pas la finalisation : le
//	                        laisser passer contournait les deux gardes de ce fichier, et le fil des
//	                        morts retombait sur « le dernier numero ». Meme sentinelle que le film
//	                        non finalise, donc meme classement ECARTE chez le parent.
//	ABSENT                  CUIT, par le chemin degrade d avant le lot (document sans courbe de
//	                        score, cf. `filmload.go`), et ce n est pas tu : le fil des morts y est
//	                        lu par le repli NOMME `repli_temps_forts_dernier_numero` (registre
//	                        `facts/fallback`, tranche « objectifs et construction »), journalise ici
//	                        en WARN. 0 cas au parc du 2026-09-23 (1 625 repertoires, 1 625
//	                        manifestes) : le cas est celui d un repertoire donne a la main a
//	                        `replay-build`.
//
// Le manifeste est RELU ici, et seulement sur ce chemin rare : `ouvrirManifeste` ne rend pas son
// erreur, et la lui faire rendre changerait le contrat de `filmload.go`. Un manifeste APPARU entre
// les deux lectures est refuse aussi — le film a ete charge sans lui ; la cuisson suivante le lira.
func jugerFilmSansManifeste(ctx context.Context, matchID, filmDir string) error {
	_, found, err := filmcache.OpenChunkDir(filmDir)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "cuisson: film ECARTE — manifeste present mais illisible, finalisation "+
			"non prouvable ; aucun artefact n est cuit", "match_id", matchID, "err", err)
		return fmt.Errorf("%w (match %s, manifeste present mais illisible : %v)",
			filmcache.ErrFilmNonFinalise, matchID, err)
	case found:
		slog.WarnContext(ctx, "cuisson: film ECARTE — manifeste apparu pendant la cuisson ; la "+
			"suivante le lira", "match_id", matchID)
		return fmt.Errorf("%w (match %s, manifeste apparu pendant la cuisson)",
			filmcache.ErrFilmNonFinalise, matchID)
	}
	slog.WarnContext(ctx, "cuisson: film SANS manifeste — finalisation non prouvable, fil des morts "+
		"par le repli repli_temps_forts_dernier_numero", "match_id", matchID, "filmDir", filmDir)
	return nil
}

// refuserMorceauxHorsManifeste rend [filmcache.ErrFilmNonFinalise] enveloppee quand le repertoire
// porte des morceaux que le manifeste ne decrit pas — l autre moitie de la signature
// d `ab526724` : les morceaux 34 a 36 etaient arrives sur le disque onze secondes apres un
// manifeste deja valide. Un tel manifeste ne dit pas le film entier ; les morceaux en trop n ont
// ni type ni debut connus.
func refuserMorceauxHorsManifeste(ctx context.Context, matchID string, src *filmcache.Source,
	film *decfilm.Film,
) error {
	if src == nil || film == nil {
		return nil
	}
	decrits := make(map[int]bool, len(src.Meta()))
	for _, m := range src.Meta() {
		decrits[m.Index] = true
	}
	var horsManifeste []int
	for _, m := range film.Meta() {
		if !decrits[m.Index] {
			horsManifeste = append(horsManifeste, m.Index)
		}
	}
	if len(horsManifeste) == 0 {
		return nil
	}
	slog.WarnContext(ctx, "cuisson: film ECARTE — morceaux presents au cache mais absents du "+
		"manifeste ; aucun artefact n est cuit", "match_id", matchID, "hors_manifeste", horsManifeste)
	return fmt.Errorf("%w (match %s, morceaux hors manifeste %v)", filmcache.ErrFilmNonFinalise,
		matchID, horsManifeste)
}

// typeDeChunkMeta : l accesseur de type que [filmcache.Finalise] recoit.
func typeDeChunkMeta(m decfilm.ChunkMeta) int { return m.ChunkType }

// filmFactsPath rend le chemin des faits de ce match, par le PathResolver et jamais a la main.
func (b *Builder) filmFactsPath(matchID string) string {
	return title.NewPathResolver(b.repoRoot).FilmFactsPath(b.titleSlug, matchID)
}

// lireLesFaitsFrais rend les faits de ce match S ILS SONT UTILISABLES, nil sinon.
//
// TOUTE RAISON DE NE PAS RELIRE EST JOURNALISEE, jamais avalee : fichier absent (le cas nominal du
// premier passage, en DEBUG), en-tete perime (INFO — c est la mesure qui dira si la recuisson
// selective de 4.4 sert), fichier illisible malgre un en-tete frais (WARN — un fichier de faits
// corrompu est un fait d exploitation).
func (b *Builder) lireLesFaitsFrais(ctx context.Context, matchID string,
	entry decfilm.MapQuantEntry,
) (*replay.FilmFactsFile, replay.FilmFactsEntete) {
	if b.sansFaitsPersistes {
		slog.DebugContext(ctx, "cuisson: faits de film IGNORES sur demande (harnais S8)",
			"match_id", matchID)
		return nil, replay.FilmFactsEntete{}
	}
	chemin := b.filmFactsPath(matchID)
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin resolu par PathResolver
	if err != nil {
		niveau := slog.LevelWarn
		if errors.Is(err, fs.ErrNotExist) {
			niveau = slog.LevelDebug
		}
		slog.Log(ctx, niveau, "cuisson: faits de film non relus", "match_id", matchID,
			"path", chemin, "err", err)
		return nil, replay.FilmFactsEntete{}
	}
	entete, err := replay.DecodeFilmFactsEntete(blob)
	if err == nil {
		err = entete.Frais(entry)
	}
	if err != nil {
		slog.InfoContext(ctx, "cuisson: faits de film perimes — redecodage", "match_id", matchID,
			"path", chemin, "raison", err)
		return nil, replay.FilmFactsEntete{}
	}
	fichier, err := replay.DecodeFilmFactsFile(blob, entry)
	if err != nil {
		slog.WarnContext(ctx, "cuisson: faits de film illisibles malgre un en-tete frais — redecodage",
			"match_id", matchID, "path", chemin, "err", err)
		return nil, replay.FilmFactsEntete{}
	}
	slog.InfoContext(ctx, "cuisson: faits de film FRAIS — rejeu si les gardes de l appelant les "+
		"couvrent", "match_id", matchID, "path", chemin, "bytes", len(blob))
	return fichier, entete
}

// horlogeDesChunks rend `index de chunk -> start_ms` pour les chunks QUE LE MANIFESTE DECRIT.
//
// MEME FILTRE QUE [chunksDuManifeste], et pour la meme raison : un chunk hors manifeste n a pas de
// debut connu, et l inscrire a zero dirait au balayage de l anneau « ce chunk commence a 0 » au
// lieu de « je ne sais pas ». La map est TOUJOURS non nil — un film sans chunk datable rend une
// horloge vide, pas une horloge absente.
func horlogeDesChunks(film *decfilm.Film) map[int]int {
	chunks := chunksDuManifeste(film)
	clock := make(map[int]int, len(chunks))
	for _, c := range chunks {
		clock[c.Index] = c.StartMS
	}
	return clock
}

// serialiserDocument est LA SECONDE MOITIE, COMMUNE AUX DEUX BRANCHES : le document devient des
// octets, l observateur voit la branche puis l etape `artifact`, et la ligne de journal dit la
// duree totale.
//
// COMMUNE VEUT DIRE QU IL N Y A QU UNE SORTIE : un second `json.Marshal` sur la branche « relire »
// aurait pu diverger sur un reglage d encodage, et l equivalence a l octet du test S8 n aurait
// plus rien prouve.
func (b *Builder) serialiserDocument(matchID string, entry decfilm.MapQuantEntry,
	doc replay.ReplayDocument, depuisLesFaits bool, debutTotal time.Time,
) (Built, error) {
	if len(doc.Tracks) == 0 {
		return Built{}, fmt.Errorf("%w (match %s)", ErrNoTracks, matchID)
	}
	tMarshal := time.Now()
	blob, err := json.Marshal(doc)
	logPhase("marshal", matchID, tMarshal)
	if err != nil {
		return Built{}, fmt.Errorf("sérialisation artefact %s: %w", matchID, err)
	}
	b.observe(EtapeRejeuDepuisLesFaits, depuisLesFaits)
	b.observe("artifact", blob)
	slog.Info("cuisson: octets construits", "match_id", matchID, "depuis_les_faits", depuisLesFaits,
		"duration", time.Since(debutTotal), "tracks", len(doc.Tracks), "bytes", len(blob))
	return Built{Blob: blob, Module: entry.Module, Tracks: len(doc.Tracks)}, nil
}

// SansFaitsPersistes force la branche DECODE : les faits sur disque sont IGNORES, le film est
// relu, et les faits sont RE-ECRITS a la sortie. Chainable.
//
// # CE N EST PAS UN INTERRUPTEUR DE FONCTIONNALITE (CLAUDE.md regle 11)
//
// La bascule est ACTIVE par defaut et le reste ; ce reglage existe pour UNE raison, et elle est
// mesurable : le test S8 (`cmd/replay-equiv -deux-passes`) doit jouer LES DEUX BRANCHES DU MEME
// COMMIT et comparer leurs artefacts a l octet. Sans lui, la seconde passe relirait les faits
// ecrits par la premiere et le harnais comparerait « faits contre faits » — une equivalence
// VACUANTE, exactement le defaut contre lequel `alerterCatalogueVide` a ete ecrit.
//
// CRITERE DE RETRAIT : le jour ou le harnais saurait forcer la branche autrement (deux racines de
// cache, par exemple), ce reglage se retire avec son dernier appelant. Il n a qu UN appelant, et
// `TestReglageDuDecodageForceNAQuUnAppelant` le tient.
func (b *Builder) SansFaitsPersistes() *Builder {
	b.sansFaitsPersistes = true
	return b
}

// completerLesFaits pose les DEUX SECTIONS QUE `replay` NE PEUT PAS CONNAITRE.
//
// # LE TROU QUE CETTE FONCTION FERME, ET C EST S8 QUI L A TROUVE (2026-09-18)
//
// `replay.faitsDuBalayage` remplit ce que la couche de PUBLICATION sait : la couverture,
// les entrees, l identite du film, le rapport de replis du balayage. Elle ne peut pas remplir le
// STATBORG ni le resultat KILLSOURCE — les deux naissent ICI (`statborgDuFilm`,
// `decodeKillSource`), dans le paquet d ASSEMBLAGE, et la couche de publication ne les voit
// jamais passer. Personne ne les posait : le fichier ecrit portait deux sections VIDES.
//
// LA CONSEQUENCE ETAIT SILENCIEUSE ET TOTALE. Un rejeu depuis de tels faits reconstruit ses
// entrees de calque sur une section statborg vide, donc `assemblerFilmStats` rend un `filmStats`
// VIDE : le document perd sa courbe de score, ses actions d objectif, son drapeau, sa couronne,
// son crane et son armement — et rien ne l aurait dit, parce que l en-tete etait FRAIS. Le seul
// gate qui pouvait l attraper est le S8 (deux passes du meme commit comparees a l octet), et il
// l a attrape a la PREMIERE ecriture.
//
// LE GARDE-RAIL QUI LE TIENT MAINTENANT, ET QUI N A PAS BESOIN DE FILM :
// [TestLesFaitsEcritsPortentLeursCinqSections] balaie PAR REFLEXION tous les champs exportes de
// `replay.FilmFactsFile` apres la chaine complete de production. Une section ajoutee au type que
// personne ne remplirait le fait rougir.
func completerLesFaits(f *replay.FilmFactsFile, src entreesDeCuisson) {
	if f == nil {
		return
	}
	f.Statborg = src.statborg
	f.Kills = src.kills
}
