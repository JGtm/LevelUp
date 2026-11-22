package testutil

// replay_servis_seuls.go — LES CHAMPS ET LES TYPES DU DOCUMENT DE REJEU SERVI QUI N'ONT AUCUNE SOURCE
// DANS LE DOCUMENT STOCKE.
//
// Le document stocke (`replay.ReplayDocument`, format de cuisson) et le document servi
// (`domain/replaydoc.ReplayDocument`, forme de fil) sont jumeaux, hors ces champs : des calques RESOLUS
// A LA REQUETE depuis le referentiel du titre, que la cuisson n'ecrit jamais et que le decodeur de film
// ne porte pas. Deux gardes de paquets differents confrontent les deux documents et lisent CETTE liste,
// la seule : `service/replayview/parity_test.go` (champ par champ, et copie de la projection) et
// `internal/filmcorpus/document_shape_jumeaux_test.go` (forme serialisee). Une liste par garde
// diverge ; une exception ne s'ajoute donc qu'ici.
//
// Ce fichier n'est lie qu'aux binaires de test : aucun code de production n'importe `testutil`.
//
// UNE ENTREE SE JUSTIFIE PAR UNE DATE ET UN MOTIF (`2026-11-17 : ...`), et ne survit pas a son champ :
// `ReplayServisSeulsIncoherents` rend, pour ce que le garde a VU du document servi, chaque entree sans
// date, sans motif, ou qui ne designe plus rien.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ReplayChampsServisSeuls : les champs du document SERVI sans source stockee, par `Type.Champ`, avec la
// date de la decision et son motif. Un calque resolu a la requete et declare par le decodeur est un champ
// jumeau (`medalLabels`, depuis la v1.5.0), plus un champ servi seul.
var ReplayChampsServisSeuls = map[string]string{
	"ReplayDocument.ShieldCycles": "2026-11-22 : instants des bris et des recharges de bouclier de chaque joueur mesure, calcules a la requete " +
		"par service/replay_shield_cycles.go depuis les pistes du document stocke avec analysis/shieldcycle (une seule definition).",
}

// ReplayTypesServisSeuls : les TYPES du document servi qui n'existent que pour un champ de
// `ReplayChampsServisSeuls`, par nom de type, avec la date de la decision. Les gardes les visitent sans leur
// chercher de jumeau stocke.
var ReplayTypesServisSeuls = map[string]string{
	"ShieldCycle": "2026-11-22 : valeur de `ReplayDocument.ShieldCycles`.",
}

// LES DECISIONS DE NON-PUBLICATION. Le sens inverse : ce que le document STOCKE ecrit et que l'API ne publie pas.
// Les memes trois gardes (`service/replayview/parity_test.go`, `columns_champs_test.go` et
// `internal/filmcorpus/document_shape_jumeaux_test.go`) lisent ces deux listes, retirees de la forme STOCKEE.

// ReplayChampsNonServis liste les champs du document STOCKE que l'API ne publie pas, par
// `Type.Champ`, avec la DATE de la decision et son motif. Toute entree se justifie ici, pas
// dans un commit qui retire silencieusement un champ du corps.
//
// Une entree vaut « le decodeur l'ecrit, aucun lecteur de LevelUp ne s'en sert » : un champ servi
// sans lecteur est une promesse que rien ne tient (le miroir se fait le jour ou un lecteur le lit).
var ReplayChampsNonServis = map[string]string{
	"T0FilmCoverage.Source": "2026-11-22 (decodeur v1.5.0) : la source du coup d'envoi (`roundClock`) n'a aucun lecteur ; " +
		"le coup d'envoi se lit par `t0FilmMs`, que les lecteurs de LevelUp reprennent tel quel.",
	"T0FilmCoverage.StartRead": "2026-11-22 (decodeur v1.5.0) : dit si le depart de l'horloge est lu ou deduit ; " +
		"aucun lecteur ne distingue les deux, le coup d'envoi publie est le meme.",
	"ScoreCoverage.StatsGuarded": "2026-11-22 (decodeur v1.5.0) : emissions de compteurs ecartees par la garde ; " +
		"aucun lecteur, la marque de tir a la tete tient deja compte de la garde (`eliminations[].headshot`).",
	"ScoreCoverage.FrozenRounds": "2026-11-22 (decodeur v1.5.0) : couverture des valeurs figees par manche ; " +
		"aucun lecteur, la fiche du joueur lit les courbes du tableau et les totaux de la feuille.",
	"PlayerScore.FrozenRounds": "2026-11-22 (decodeur v1.5.0) : valeurs figees par manche terminee ; aucun lecteur " +
		"(la fiche du joueur lit les courbes du tableau et les totaux de la feuille).",
	"VehicleCoverage.TurretsByRead": "2026-11-22 (decodeur v1.5.0) : pieces montees dont le porteur est lu a la " +
		"creation ; le porteur lui-meme (`vehicles[].carrier`) est servi, son compteur de lecture n'a aucun lecteur.",
	"VehicleCoverage.TurretParentUnknown": "2026-11-22 (decodeur v1.5.0) : pieces montees au parent inconnu ; " +
		"aucun lecteur, une piece sans porteur n'a simplement pas de porteur a l'ecran.",
}

// Les compteurs de couverture que la montee du decodeur v1.5.0 ajoute et que rien ne lit : la fiche du joueur ne lit
// que ce qui rend une valeur approximative (`reloads.stoppedFrames` et `withoutPlayer`, `stances.control.noTrack`,
// `noPlayer` et `withHole`) et la presence de `coverage.eliminations`. Le reste reste au decodeur, qui est public.
const (
	motifScoreboardNonLu = "2026-11-22 (decodeur v1.5.0) : compteur de la comparaison du score personnel au tableau du film ; " +
		"aucun lecteur, le verdict `complete` suffit et les lecteurs de la fiche s'y tiennent."
	motifRechargementsNonLu = "2026-11-22 (decodeur v1.5.0) : compteur de la lecture des rechargements ; aucun lecteur, la " +
		"fiche ne lit que les trames arretees et les rechargements sans joueur pour dire que le compte est un minimum."
	motifEliminationsNonLu = "2026-11-22 (decodeur v1.5.0) : compteur de la couverture des eliminations ; aucun lecteur, " +
		"seule la presence de la couverture compte et les marques se lisent sur `eliminations[]`."
	motifCommandeNonLu = "2026-11-22 (decodeur v1.5.0) : compteur de la lecture des commandes de deplacement ; aucun " +
		"lecteur, la fiche ne lit que les commandes sans vie, sans joueur ou a trou pour dire que le temps est une estimation."
)

func init() {
	for _, cle := range []string{
		"ScoreActionCoverage.ScoreboardCompared", "ScoreActionCoverage.ScoreboardMissing",
		"ScoreActionCoverage.ScoreboardBehind", "ScoreActionCoverage.TreeShift",
	} {
		ReplayChampsNonServis[cle] = motifScoreboardNonLu
	}
	for _, champ := range []string{"Frames", "ClosedFrames", "Read", "Copies", "BeforeOrigin", "AfterEnd", "Published"} {
		ReplayChampsNonServis["ReloadCoverage."+champ] = motifRechargementsNonLu
	}
	for _, champ := range []string{
		"Eliminations", "WithVictim", "HeadYes", "HeadNo", "HeadUnknown", "HeadNoCounter", "HeadAmbiguous",
		"HeadGuarded", "HeadOrphanRises", "PerfectYes", "PerfectUnknown", "PerfectAmbiguous", "PerfectOrphans",
	} {
		ReplayChampsNonServis["EliminationCoverage."+champ] = motifEliminationsNonLu
	}
	for _, champ := range []string{"Holds", "InnerHoles", "Entries", "Published", "Intervals"} {
		ReplayChampsNonServis["StanceControlCoverage."+champ] = motifCommandeNonLu
	}
}

// ReplayTypesNonServis liste les TYPES du document stocke que l'API ne publie pas du tout, par nom
// de type, avec la DATE de la decision et son motif.
//
// POURQUOI UNE SECONDE LISTE, et pourquoi `ReplayChampsNonServis` ne suffit pas. La revue
// adversariale du 2026-09-05 a montre que l'echappatoire ne couvrait PAS le cas majoritaire :
// un calque ajoute a la cuisson arrive presque toujours avec son propre type
// (`Translocation`, `AbilityImpulse`, `AbilityCharge`, `BombEvent`, `BombPlayerStats` sont
// tous nes ainsi). Ecrire `ReplayChampsNonServis["ReplayDocument.Medals"]` ne suffisait pas : le
// controle « jumeau servi » se fait AU NIVEAU DU TYPE, en amont, et echouait toujours sur
// `MedalEvent`. Il n'existait donc aucune facon d'ajouter un type stocke sans le servir — ce
// qui contredisait frontalement la promesse du lot (« ajouter un calque a la cuisson ne
// touche plus le contrat public ») et ne laissait au mainteneur que deux mauvais choix :
// miroiter dans `replaydoc` un type que le contrat ne veut pas, ou affaiblir le garde-rail.
//
// Un type inscrit ici est VISITE (pour que la decision soit lue et datee) mais le parcours
// ne DESCEND PAS dedans : les types atteignables par lui seul ne sont pas servis non plus,
// et ce n'est pas un oubli.
var ReplayTypesNonServis = map[string]string{
	"FrozenRound": "2026-11-22 (decodeur v1.5.0) : type de `players[].frozenRounds`, non servi (voir `PlayerScore.FrozenRounds`).",
	"FrozenRoundsCoverage": "2026-11-22 (decodeur v1.5.0) : type de `coverage.score.frozenRounds`, non servi " +
		"(voir `ScoreCoverage.FrozenRounds`).",
}

// motifDate : un motif commence par la date de la decision.
var motifDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)

// ReplayServisSeulsIncoherents confronte les deux listes a ce qu'un garde a VU du document servi
// (`champsVus` par `Type.Champ`, `typesVus` par nom de type) et rend, triees, les raisons de refuser :
// une entree sans motif ou dont le motif ne commence pas par une date, une entree qui ne designe plus
// ni champ ni type du document servi. Vide quand tout est coherent.
func ReplayServisSeulsIncoherents(champsVus, typesVus map[string]bool) []string {
	var out []string
	verifier := func(liste string, entrees map[string]string, vus map[string]bool) {
		for cle, motif := range entrees {
			switch {
			case strings.TrimSpace(motif) == "":
				out = append(out, fmt.Sprintf("%s : l'entree %q n'a AUCUNE justification", liste, cle))
			case !motifDate.MatchString(motif):
				out = append(out, fmt.Sprintf("%s : le motif de %q ne commence pas par une date (AAAA-MM-JJ)", liste, cle))
			}
			if !vus[cle] {
				out = append(out, fmt.Sprintf("%s : l'entree %q ne designe plus rien dans le document servi — "+
					"une exception qui survit a son champ finit par en couvrir un autre", liste, cle))
			}
		}
	}
	verifier("ReplayChampsServisSeuls", ReplayChampsServisSeuls, champsVus)
	verifier("ReplayTypesServisSeuls", ReplayTypesServisSeuls, typesVus)
	sort.Strings(out)
	return out
}
