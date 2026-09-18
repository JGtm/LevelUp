package replaybuild

// artifact_digest.go — LA LECTURE D'UN ARTEFACT DEJA RANGE : les gardes, et rien d'autre.
//
// Ce fichier ne CONSTRUIT rien. Il porte les fonctions qui INTERROGENT un artefact sur disque
// (est-il a la version de schema courante ? porte-t-il des compteurs de joueur ?) et l'ecriture
// qui le range. Elles vivaient dans `replaybuild.go` et l'avaient pousse au-dela des 500 lignes
// du depot : ce fichier est ne d'une SCISSION PURE, aucun corps n'a change, aucun commentaire
// n'a ete touche — seul l'emplacement bougeait.
//
// DEPUIS : `ArtifactHasPlayerCounters` a ete SUPPRIMEE (lot 6, constat 6.3 — plus aucun appelant
// de production depuis que les gardes lisent le digest une seule fois, item 5.3). Sa doctrine
// vit sur [Digest.HasPlayerCounters], qui est le predicat qu'elle decrivait.

import (
	"encoding/json"
	"os"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/observability"
)

// ArtifactUpToDate dit si l'artefact au chemin donné existe ET porte la version de schéma
// courante. C'est LA clé de reprise des backfills (cf. replay.SchemaVersion) : un artefact
// d'une version antérieure se lit « à re-cuire », jamais « à jour ». Un fichier illisible
// est traité comme périmé (il sera réécrit), pas comme une erreur.
//
// C'EST UNE VUE DU DIGEST, PLUS UNE LECTURE (PLAN_CUISSON_PERF item 5.3). Elle reste exportée
// parce qu'elle a des appelants qui n'ont besoin que d'elle (`cmd_backfill_replay.go`,
// `registry_build_queue.go`) ; ceux qui posent DEUX questions sur le même fichier lisent
// [ArtifactDigest] UNE fois et interrogent le résultat.
func ArtifactUpToDate(path string) bool {
	d, ok := ArtifactDigest(path)
	return ok && d.UpToDate()
}

// Digest : les seules marques d'un artefact que les gardes ont à lire. Une SEULE forme de
// lecture pour tous — deux structures anonymes concurrentes finiraient par diverger sur le nom
// d'un champ, et le garde deviendrait muet sans que rien ne le signale.
//
// EXPORTÉ AU LOT 5 DE PLAN_CUISSON_PERF (item 5.3) : les appelants qui posent DEUX questions
// sur le même artefact (« à jour ? » puis « avec des compteurs ? ») le lisaient DEUX FOIS,
// c'est-à-dire deux `os.ReadFile` et deux désérialisations d'un document de ~2 Mo par match et
// par cycle. Ils lisent désormais une fois et interrogent le résultat.
type Digest struct {
	MatchID       string
	SchemaVersion int
	Players       int
	Tracks        int
	Bytes         int
	// Layers : les revisions de couche que l artefact DECLARE, calque par calque (schema 62,
	// `film/replay/layers.go`). Nil = artefact anterieur au schema 62, qui ne declarait rien —
	// et c est une REPONSE : on ne peut alors pas prouver que son decodage est intact.
	//
	// C EST LA CINQUIEME CLE que `digestFromBytes` deserialise, et elle ne coute presque rien :
	// le parseur n ouvre toujours pas le reste du document, ce qui garde la lecture bon marche sur
	// ~1,2 Mio (item 5.3 de PLAN_CUISSON_PERF).
	Layers map[string]string
}

// UpToDate : cet artefact est-il A JOUR, c est-a-dire au schema courant ET decode par les
// revisions courantes ?
//
// ELLE EST CONSERVEE, ET ELLE EST DESORMAIS UNE VUE DU VERDICT (lot 4.4.1) : ses appelants
// n ont besoin que de « rien a faire ou non », et leur faire porter la presence des faits
// n apprendrait rien de plus a leur decision. Ceux qui doivent CHOISIR entre republier et
// redecoder lisent [ArtifactVerdict].
//
// CE QUI CHANGE POUR EUX : un artefact au bon schema mais decode par une grammaire perimee se
// lit desormais « pas a jour ». C est le sens de la recuisson selective — et c est aussi ce que
// `coverage.decoder` rendait constatable sans que personne ne le lise.
func (d Digest) UpToDate() bool { return d.Verdict(true) == VerdictAJour }

// HasPlayerCounters : l'artefact PORTE-T-IL des compteurs de joueur (`scoreTimeline.players`
// non vide).
//
// CE QU'IL AFFIRME, ET CE QU'IL N'AFFIRME PAS. Il constate une PROPRIÉTÉ DU DOCUMENT, il ne
// devine pas comment il a été cuit. L'implication ne vaut que dans un sens :
//
//	compteurs présents  =>  les lignes de match ont été fournies   (sûr)
//	compteurs absents   =>  les lignes de match manquaient          (FAUX en général)
//
// TROIS FAÇONS LÉGITIMES d'être vide MALGRÉ des faits complets, toutes constatées dans le code :
// (a) le film n'a aucun enregistrement d'entité à lire (cas journalisé, `matchfacts.go:70-73`) ;
// (b) l'appariement slot -> joueur échoue — `SlotIdentityFrom` écarte les triplets ambigus, et
// les `COALESCE(..., 0)` de `replay_facts_repo.go` peuvent rendre plusieurs (0,0,0)
// indistinguables ; (c) aucun compteur ne bouge dans la fenêtre lue (`PlayerScore` vide).
//
// UN APPELANT NE DOIT DONC JAMAIS EN DÉDUIRE « à re-cuire » À LUI SEUL. Le vide est une
// PRÉSOMPTION d'appauvrissement, pas une preuve : c'est pour cela que `replayartifacts.enqueueAll`
// exige EN PLUS de tenir des lignes de match (`len(facts.Players) > 0`), et que le rangement
// d'artefact (`StoreArtifact`) refuse de son côté toute régression. Le pire résidu possible est
// alors UN cycle d'ouvrier gâché — jamais un artefact rétrogradé, jamais une boucle qui converge
// vers rien.
//
// POURQUOI CE SIGNAL PLUTÔT QU'UN AUTRE. Mesuré sur deux témoins le 2026-08-24 : 8 joueurs avec
// faits, 0 sans, sur 7344d24f comme sur 530820e5. Les autres candidats sont pires :
// `coverage.score.teamIdentity` vaut légitimement `unresolved` sur 7 des 34 artefacts du cache
// POURTANT cuits avec faits, et `objectives` est vide de plein droit sur un Slayer.
//
// PAS DE CHAMP DÉDIÉ DANS LE DOCUMENT, ET C'EST DÉLIBÉRÉ : un marqueur `factsApplied` — qui
// LUI porterait l'implication dans les deux sens — forcerait un incrément de
// `replay.SchemaVersion`, donc la re-cuisson de tout le cache, aujourd'hui bloquée par la bombe
// RAM de `NamedEventsFrom` (registre du 2026-08-24). C'est la dette assumée de ce choix.
//
// CET EN-TÊTE VIENT D'`ArtifactHasPlayerCounters` (lot 6, constat 6.3) : cette forme-là lisait le
// disque pour poser la seule question des compteurs, et n'avait plus aucun appelant de production
// depuis que les gardes lisent le digest UNE fois (item 5.3). Elle a été supprimée ; sa doctrine,
// elle, décrit le PRÉDICAT et vaut donc pour cette vue.
func (d Digest) HasPlayerCounters() bool { return d.Players > 0 }

// ArtifactDigest lit UNE FOIS l'artefact au chemin donné et rend ses marques. ok=false si
// absent ou illisible : dans le doute, l'appelant traite l'artefact comme inexploitable, jamais
// comme bon — il ne peut PAS distinguer les deux cas ici, et c'est voulu (les deux appellent la
// même conduite ; celui qui a besoin de la nuance fait son propre `os.Stat`, cf.
// `cmd_backfill_replay_repair.go`).
//
// LE COMPTEUR EST LA MESURE DE CETTE LECTURE, et il a un rôle : c'est par lui que les gardes de
// fraîcheur (post-sync, backfill) prouvent qu'ils n'ouvrent qu'UNE fois par artefact. Un cycle
// qui doublerait ses lectures se verrait dans /debug/vars avant de se voir dans un profil.
func ArtifactDigest(path string) (Digest, bool) {
	observability.IncCounter(CompteurLecturesArtefact)
	raw, err := os.ReadFile(path)
	if err != nil {
		return Digest{}, false
	}
	return digestFromBytes(raw)
}

// CompteurLecturesArtefact : le nombre de lectures disque d'artefact faites par les gardes.
const CompteurLecturesArtefact = "replay_artifact_digest_reads_total"

// digestFromBytes lit les marques d'un artefact déjà en mémoire (cas du dépôt d'un ouvrier :
// le blob est là, le relire depuis le disque serait absurde).
func digestFromBytes(raw []byte) (Digest, bool) {
	var head struct {
		MatchID       string            `json:"matchId"`
		SchemaVersion int               `json:"schemaVersion"`
		Tracks        []json.RawMessage `json:"tracks"`
		ScoreTimeline struct {
			Players []json.RawMessage `json:"players"`
		} `json:"scoreTimeline"`
		Layers map[string]string `json:"layers"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return Digest{}, false
	}
	return Digest{
		MatchID:       head.MatchID,
		SchemaVersion: head.SchemaVersion,
		Players:       len(head.ScoreTimeline.Players),
		Tracks:        len(head.Tracks),
		Bytes:         len(raw),
		Layers:        head.Layers,
	}, true
}

// writeArtifact sérialise le document et l'écrit ATOMIQUEMENT (cf.
// writeArtifactBytes, artifact_store.go) ; renvoie la taille en octets. Même
// écriture que le dépôt d'un ouvrier : le service de lecture sert le fichier tel
// quel, il ne doit jamais tomber sur un artefact à moitié écrit.
// La taille rendue est celle de ce qui est FINALEMENT sur le disque, pas celle du document
// qu'on voulait écrire : quand le garde anti-régression conserve l'artefact en place, annoncer
// la taille du candidat ferait croire à une écriture qui n'a pas eu lieu.
func writeArtifact(outPath, titleSlug, matchID string, doc replay.ReplayDocument) (int, error) {
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	surDisque, err := writeArtifactBytes(outPath, titleSlug, matchID, blob)
	if err != nil {
		return 0, err
	}
	return surDisque.Bytes, nil
}

// Verdict : ce qu il faut FAIRE d un artefact deja range. Trois sorties, et pas quatre.
//
// # POURQUOI TROIS
//
// Jusqu au lot 4.4.1 la question etait binaire — `UpToDate()`, une egalite de `SchemaVersion` —
// donc toute montee de schema marquait le parc entier « a recuire », et recuire voulait dire
// REDECODER le film : ~15 a 100 s par match. Or la montee de schema la plus frequente ne change
// QUE la publication : les faits decodes sont les memes, et depuis le lot 4.1 ils sont sur le
// disque. Republier depuis les faits est alors affaire de secondes.
//
// # LE PIEGE QU IL N Y A PAS
//
// `Republier` retombe sur `Redecoder` quand le fichier de faits manque, et il n existe AUCUN
// quatrieme etat « je republierais si j avais les faits ». Un tel etat serait un piege a
// diagnostic : l appelant devrait le traduire en conduite, et deux appelants le traduiraient
// differemment. La conduite est ici, une fois.
type Verdict string

// Les trois verdicts.
const (
	// VerdictAJour : schema courant ET decodage intact. Rien a faire.
	VerdictAJour Verdict = "a-jour"
	// VerdictRepublier : le decodage est intact, seule la PUBLICATION a bouge — rejouer depuis
	// les faits, sans ouvrir le film.
	VerdictRepublier Verdict = "republier"
	// VerdictRedecoder : une revision de couche a bouge, OU les faits manquent sur le disque, OU
	// l artefact ne declare pas ses couches (anterieur au schema 62). Decodage complet, sous le
	// verrou solo.
	VerdictRedecoder Verdict = "redecoder"
)

// Verdict rend la conduite a tenir sur cet artefact, `faitsPresents` disant si le fichier de
// faits du match est sur le disque.
//
// L ORDRE DES TROIS QUESTIONS EST LE PROPOS DE CETTE FONCTION :
//
//  1. le DECODAGE est-il intact ? Sinon rien d autre ne compte : les faits persistes eux-memes
//     ont ete produits par la couche qui a bouge, donc les rejouer reproduirait l ancien
//     decodage. `Redecoder`, toujours.
//  2. le schema est-il le courant ? Alors il n y a rien a faire.
//  3. sinon seule la publication a bouge — `Republier` SI les faits sont la, `Redecoder` sinon.
func (d Digest) Verdict(faitsPresents bool) Verdict {
	switch {
	case !d.decodageIntact():
		return VerdictRedecoder
	case d.SchemaVersion == replay.SchemaVersion:
		return VerdictAJour
	case !faitsPresents:
		return VerdictRedecoder
	default:
		return VerdictRepublier
	}
}

// decodageIntact : toutes les revisions de couche DECLAREES par l artefact sont-elles celles du
// binaire courant ?
//
// UN ARTEFACT QUI NE DECLARE RIEN N EST PAS INTACT, et ce n est pas une severite gratuite : sans
// `layers` (schema < 62) on ne peut pas prouver sous quelle grammaire il a ete cuit, et republier
// depuis des faits dont on ne sait pas s ils sont a jour produirait un document faux en silence.
// Le defaut sur est donc `Redecoder`, exactement comme avant ce lot.
//
// LA FAMILLE DE PUBLICATION EST EXCLUE DE CE TEST, et c est le point du lot : c est elle qui
// bouge a chaque montee de schema, et c est precisement ce cas que `Republier` sert. La comparer
// ici rendrait le troisieme verdict inatteignable.
func (d Digest) decodageIntact() bool {
	if len(d.Layers) == 0 {
		return false
	}
	courantes := replay.RevisionsCourantesDesCouches()
	for _, lue := range d.Layers {
		famille := replay.FamilleDeRevision(lue)
		if famille == "" || famille == famillePublication {
			continue
		}
		attendue, connue := courantes[famille]
		if !connue || attendue != lue {
			return false
		}
	}
	return true
}

// famillePublication : la seule famille que `decodageIntact` ignore (cf. son en-tete).
const famillePublication = "publication"

// ArtifactVerdict lit UNE FOIS l artefact du match et rend sa conduite, en tenant compte de la
// presence du fichier de faits. ok=false quand l artefact est absent ou illisible : l appelant
// cuit alors depuis le film, et il n y a pas de verdict a rendre sur un fichier qui n existe pas.
//
// C EST LA FORME QUE LES SITES DE DECISION EMPLOIENT : eux seuls tiennent le resolveur de chemins,
// et la presence des faits ne se devine pas depuis un digest. Le resolveur PLUTOT QUE LA RACINE
// parce que les deux appelants de production en tiennent deja un — en reconstruire un ici ferait
// une seconde source du meme chemin.
func ArtifactVerdict(res *title.PathResolver, titleSlug, matchID string) (Verdict, bool) {
	d, ok := ArtifactDigest(res.ReplayArtifactPath(titleSlug, matchID))
	if !ok {
		return VerdictRedecoder, false
	}
	return d.Verdict(faitsSurDisque(res.FilmFactsPath(titleSlug, matchID))), true
}

// faitsSurDisque : le fichier de faits de ce match est-il la ? Un `os.Stat`, pas une lecture —
// le verdict n a pas besoin de son contenu, et la fraicheur de l en-tete est jugee au moment du
// rejeu par `lireLesFaitsFrais` (filmfacts_cuisson.go), qui redecode si elle ne tient pas.
func faitsSurDisque(chemin string) bool {
	st, err := os.Stat(chemin)
	return err == nil && !st.IsDir()
}
