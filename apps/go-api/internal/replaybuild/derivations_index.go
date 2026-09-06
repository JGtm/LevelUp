package replaybuild

// derivations_index.go — L'INDEX DES DERIVATIONS : « ce contenu a-t-il deja ete derive ? »
//
// # Le defaut que ce fichier ferme (constat A2 du registre v2)
//
// Le rattrapage de l'etape 1.58 tenait pour « fait » tout match dont l'artefact EXISTE
// (`backlog.go`, `os.Stat` + taille > 0). Ce predicat repond a « ce match a-t-il UN rejeu »,
// ce qui etait exactement le defaut mesure a l'epoque — mais il ne dit RIEN des DERIVES.
// Consequence : un artefact range dont le resume d'usage, les statistiques d'Assaut ou le coup
// d'envoi n'ont jamais ete ecrits n'etait JAMAIS resélectionné. Les 106 artefacts locaux, tous
// anterieurs a la version de schema courante, n'avaient aucun chemin de reprise autre qu'un
// backfill d'operateur.
//
// # Ce que la marque dit, et ce qu'elle ne dit pas
//
// Elle dit : « les derivations de REVISION R ont ete jouees sur un artefact de N OCTETS ».
// Elle ne dit RIEN de ce qu'elles ont ecrit — une passe qui n'avait rien a ecrire (mode hors
// Assaut, document sans t0FilmMs) est une passe JOUEE, et la rejouer a chaque cycle serait du
// travail pur perte.
//
// # POURQUOI LA TAILLE, ET PAS UN HACHAGE DU CONTENU
//
// Le predicat doit coder « les derives collent-ils au CONTENU d'aujourd'hui ». Un hachage
// exigerait de relire l'artefact ENTIER (~2 Mo) pour chaque candidat de chaque cycle — le cout
// meme que le predicat `os.Stat` d'origine refusait, et sa justification ecrite tient toujours.
// La taille vient du `os.Stat` qu'on fait de toute facon. Un artefact re-cuit dont la sortie
// ferait EXACTEMENT la meme taille sans etre le meme document est possible en theorie ; ses
// derives seraient alors quasi identiques, et le prochain bump de [DerivationsRev] les rejouera
// de toute maniere. C'est le compromis, il est ecrit, et il n'a pas d'angle mort silencieux.
//
// # UN FICHIER A COTE, PAS UNE COLONNE
//
// La marque vit dans `<artefact>.derived.json`, a cote de l'artefact. Une table aurait lie la
// reprise des derives a une base partagee — or le rattrapage tourne AVANT tout writer, et le
// backfill CLI tourne sans base du tout. Le fichier suit l'artefact : le supprimer, c'est
// redemander ses derivations, ce qui est exactement le geste qu'un operateur veut.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"
)

// DerivationsRev — LA VERSION DES DERIVATIONS, ecrite sur chaque marque.
//
// LA FAIRE EVOLUER a chaque changement qui modifie CE QUE les derivations ecrivent : nouvelle
// famille projetee, correction d'une projection, changement de forme d'une table derivee. Le
// rattrapage rejouera alors les derivations de tout le corpus, cinq artefacts par cycle.
//
// MEME DOCTRINE QUE `killcollector.KillSourceDecoderRev`, et pour la meme raison : sans
// revision, une correction de projection ne repare que les matchs a venir, et le parc reste
// servi avec l'ancienne — sans compteur et sans reprise possible.
const DerivationsRev = "derivations-2026-09-06"

// SuffixeMarqueDerivations : l'extension du fichier de marque, posee a la place de `.json`.
//
// ELLE EST EXPORTEE PARCE QUE LA MARQUE PARTAGE LE DOSSIER DES ARTEFACTS, et que deux
// consommateurs y balaient `*.json` (constat C6 de la revue A-R1) : le cron de purge
// (`scheduler/replay_purge_cron.go`) et le bilan du backfill (`cmd/backfill_t0_film`). Sans un
// predicat COMMUN, chacun aurait recopie le litteral — troisieme exemplaire, et la premiere
// evolution du suffixe en aurait laisse un en arriere.
const SuffixeMarqueDerivations = ".derived.json"

// EstMarqueDerivations dit si ce NOM DE FICHIER est une marque de derivation et non un artefact.
//
// Le nom, pas le chemin : les deux appelants balaient un `os.ReadDir` et n'ont que l'entree.
func EstMarqueDerivations(name string) bool {
	return strings.HasSuffix(name, SuffixeMarqueDerivations)
}

// ArtefactDeLaMarque rend le chemin (ou le nom) de l'ARTEFACT que cette marque decrit —
// l'inverse exact de [DerivationsMarkPath].
//
// ELLE EXISTE POUR QUE LA RELATION MARQUE <-> ARTEFACT NE SOIT ECRITE QU'ICI. Le cron de purge
// doit pouvoir dire si une marque est ORPHELINE (son artefact a disparu : purge anterieure a la
// suppression des marques, ou geste d'operateur), et reconstruire le nom de son cote en aurait
// fait un second detenteur de la regle — le premier ajustement du suffixe en aurait laisse un
// en arriere. Constat N4 de la revue A-R2.
//
// Un chemin qui n'est PAS une marque est rendu tel quel : la fonction ne devine rien.
func ArtefactDeLaMarque(markPath string) string {
	if !EstMarqueDerivations(markPath) {
		return markPath
	}
	return strings.TrimSuffix(markPath, SuffixeMarqueDerivations) + ".json"
}

// DerivationsMark est l'etat « derive » d'UN artefact, tel qu'il est sur disque.
type DerivationsMark struct {
	// Rev : la revision des derivations jouees (cf. [DerivationsRev]).
	Rev string `json:"rev"`
	// ArtifactBytes : la taille de l'artefact AU MOMENT de la derivation. C'est elle qui dit
	// si la marque colle encore au contenu.
	ArtifactBytes int `json:"artifactBytes"`
	// ArtifactSchema : la version de schema du document derive. Purement informative — le
	// predicat de fraicheur ne s'en sert pas (un artefact perime se RE-CUIT, ce qui produira
	// une nouvelle taille et invalidera la marque). Elle est ecrite parce qu'un operateur qui
	// ouvre le fichier doit pouvoir dire de QUOI les derives viennent.
	ArtifactSchema int `json:"artifactSchema"`
	// At : l'instant de la derivation, en UTC.
	At time.Time `json:"at"`
}

// DerivationsMarkPath rend le chemin de la marque d'un artefact.
//
// LE CHEMIN EST DERIVE DE CELUI DE L'ARTEFACT, jamais reconstruit : la place canonique de
// l'artefact vient de `PathResolver.ReplayArtifactPath`, et un second calcul du dossier
// deriverait le jour ou elle bougerait.
func DerivationsMarkPath(artifactPath string) string {
	return strings.TrimSuffix(artifactPath, ".json") + SuffixeMarqueDerivations
}

// ReadDerivationsMark lit la marque d'un artefact. ok=false quand elle est absente ou
// illisible : dans les deux cas l'appelant doit conclure « pas derive », jamais « erreur ».
func ReadDerivationsMark(artifactPath string) (DerivationsMark, bool) {
	raw, err := os.ReadFile(DerivationsMarkPath(artifactPath)) //nolint:gosec // chemin derive de la place canonique
	if err != nil {
		return DerivationsMark{}, false
	}
	var m DerivationsMark
	if err := json.Unmarshal(raw, &m); err != nil {
		return DerivationsMark{}, false
	}
	return m, true
}

// WriteDerivationsMark ecrit la marque d'un artefact. La taille est celle du document
// REELLEMENT lu par les derivations, pas celle qu'on croit qu'il fait.
//
// ECRITURE SIMPLE, PAS ATOMIQUE, ET C'EST ASSUME : une marque a moitie ecrite est illisible,
// donc lue « pas derive », donc rejouee. Le pire cas est un travail refait ; il n'y a aucun
// etat a corrompre. Une ecriture atomique protegerait d'un risque qui n'existe pas ici.
func WriteDerivationsMark(artifactPath string, artifactSchema, artifactBytes int) error {
	blob, err := json.Marshal(DerivationsMark{
		Rev:            DerivationsRev,
		ArtifactBytes:  artifactBytes,
		ArtifactSchema: artifactSchema,
		At:             time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marque de derivation: %w", err)
	}
	if err := os.WriteFile(DerivationsMarkPath(artifactPath), blob, 0o600); err != nil {
		return fmt.Errorf("marque de derivation (%s): %w", DerivationsMarkPath(artifactPath), err)
	}
	return nil
}

// DerivationsUpToDate dit si l'artefact au chemin donne a DEJA ete derive a la revision
// courante, sur le contenu qu'il porte AUJOURD'HUI.
//
// Faux dans les quatre cas, qui appellent tous la meme conduite (rejouer les derivations) :
//
//	artefact absent          il n'y a rien a deriver — c'est la cuisson qui doit passer
//	marque absente/illisible jamais derive, ou marque perdue
//	revision differente      les derivations ont change depuis
//	taille differente        l'artefact a ete re-cuit depuis la derniere derivation
//
// UN SEUL `os.Stat` ET UNE LECTURE DE QUELQUES DIZAINES D'OCTETS : c'est ce qui permet de le
// jouer sur les soixante-quatre candidats d'un horizon a chaque cycle.
func DerivationsUpToDate(artifactPath string) bool {
	st, err := os.Stat(artifactPath)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	m, ok := ReadDerivationsMark(artifactPath)
	if !ok {
		return false
	}
	return m.Rev == DerivationsRev && int64(m.ArtifactBytes) == st.Size()
}

// RemoveDerivationsMark efface la marque d'un artefact — le geste « redemander les
// derivations de ce match ». Un fichier deja absent n'est pas une erreur.
func RemoveDerivationsMark(artifactPath string) error {
	err := os.Remove(DerivationsMarkPath(artifactPath))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("suppression de la marque de derivation: %w", err)
	}
	return nil
}
