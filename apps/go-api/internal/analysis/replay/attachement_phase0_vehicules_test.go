package replay

// attachement_phase0_vehicules_test.go — ITEM 0.3 : LE JOUEUR DANS UN VÉHICULE.
//
// L'ORACLE, ET POURQUOI IL EST GÉOMÉTRIQUE. Aucun événement du film ne dit « ce joueur est
// monté dans ce véhicule » : ni le statborg, ni les événements nommés, ni le fil des morts.
// Ce qu'on a, ce sont deux nuages de positions décodés par deux chaînes indépendantes — les
// bipèdes (chemin delta, `ScanFilmBipedPositions`) et les objets du monde de l'archétype
// `ti=40` (`ScanFilmWorldObjects`). Un Spartan assis dans un véhicule est, par construction
// du moteur, à la position du véhicule : la COÏNCIDENCE PROLONGÉE de deux pistes est donc
// l'oracle, et le plan l'a écrit avant la mesure (décision 4(b)) — position du bipède à
// moins de 1,5 m de celle du véhicule pendant au moins 3 s.
//
// LE SECOND FILM SE CHOISIT SUR PREUVE, pas sur un nom de playlist : `TestAttachement-
// Phase0CensusVehicules` recense les archétypes des images-clés d'une liste de films
// candidats et publie ceux qui portent des slots `ti=40`. Le corpus ci-dessous n'est arrêté
// qu'après ce recensement.

import (
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// attVehiculeTI est l'archétype « véhicule (présumé) » de la table ECS. Présumé est le mot
// juste : la table le donne sans confirmation, et c'est l'item 0.3 qui l'éprouve.
const attVehiculeTI = uint32(40)

// attFilmsVehiculesDefaut — le corpus à véhicules de l'item 0.3. `084a804d` est le film
// Fortitude Heavies donné par le plan ; `a349fea8` (Fragmentation Heavies, Total Control) est
// le second, arrêté SUR PREUVE par `TestAttachementPhase0CensusVehicules` — une playlist
// « BTB Heavies » relevée au registre ne dit rien du flux, des slots `ti=40` dans ses
// images-clés si.
var attFilmsVehiculesDefaut = []string{"084a804d", "a349fea8"}

// attFilmsVehiculesEnv permet de substituer le corpus (liste séparée par des virgules) —
// c'est ce qui permet au recensement de proposer un second film sans réécrire le code.
const attFilmsVehiculesEnv = "ATT_FILMS_VEHICULES"

// attFilmsVehicules rend le corpus effectif de l'item 0.3.
func attFilmsVehicules() []string {
	if v := os.Getenv(attFilmsVehiculesEnv); v != "" {
		var out []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return attFilmsVehiculesDefaut
}

// attCandidatsEnv porte la liste des films à recenser (séparés par des virgules). Sans
// elle, le recensement se saute : balayer les 951 films du cache n'est pas une mesure,
// c'est une attente.
const attCandidatsEnv = "ATT_CANDIDATS"

// attCensusTI recense les archétypes vus dans les images-clés d'un film, avec le nombre de
// slots distincts par archétype. Lecture des seuls chunks demandés : un recensement n'a pas
// besoin du film entier pour dire si un archétype y vit.
func attCensusTI(dir string, maxChunks int) (map[int]map[int]bool, int) {
	n := filmdec.CountFilmChunks(dir)
	if maxChunks > 0 && n > maxChunks {
		n = maxChunks
	}
	out := map[int]map[int]bool{}
	images := 0
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			images++
			for _, r := range filmdec.WalkKeyframeWorld(p.Payload(data)) {
				if r.Slot < 0 {
					continue
				}
				if out[r.TI] == nil {
					out[r.TI] = map[int]bool{}
				}
				out[r.TI][r.Slot] = true
			}
		}
	}
	return out, images
}

// TestAttachementPhase0CensusVehicules recense `ti=40` chez des films candidats.
//
// C'EST LA PREUVE QUI CHOISIT LE SECOND FILM. Un film « BTB » nommé comme tel dans une base
// ne prouve rien du contenu de son flux ; un slot `ti=40` dans ses images-clés, si.
func TestAttachementPhase0CensusVehicules(t *testing.T) {
	root := attRequireRoot(t)
	liste := os.Getenv(attCandidatsEnv)
	if liste == "" {
		t.Skipf("recensement non demandé : %s vide (liste de films séparés par des virgules)",
			attCandidatsEnv)
	}
	type ligne struct {
		id                    string
		slots40, bipedes, tis int
		images                int
	}
	var out []ligne
	for _, id := range strings.Split(liste, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		cens, images := attCensusTI(objChunkDir(root, id), 3)
		out = append(out, ligne{
			id: id, slots40: len(cens[int(attVehiculeTI)]), bipedes: len(cens[objBipedTI]),
			tis: len(cens), images: images,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].slots40 > out[j].slots40 })
	for _, l := range out {
		t.Logf("%s : %d images-clés · %d archétypes · %d slots ti=40 · %d slots bipède",
			l.id, l.images, l.tis, l.slots40, l.bipedes)
	}
}
