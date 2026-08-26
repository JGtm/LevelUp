package replay

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// grenade_reads.go — LES GRENADES PORTÉES, sur leur propre axe, alimentées par DEUX canaux.
//
// UNE SEULE GRANDEUR, DEUX SOURCES — c'est le patron d'`abilities.go`, et c'est le même remède
// contre le même défaut (« deux canaux, une seule étiquette »).
//
//	kf     le record de biped des IMAGES-CLÉS (inventory_decode.go). Dense quand il lit —
//	       une lecture par joueur et par image-clé — mais espacé de ~20 s, et MUET sur une
//	       large part du corpus : 42 films sur 70 ne rendent aucune lecture de grenades.
//	delta  les composants i22 (compteurs) et i47 (masque + sélection) des paquets DELTA
//	       (filmdec.ScanFilmInventoryDeltas). Transmis AU CHANGEMENT — un ramassage, un
//	       lancer — donc rares (0,09 % des records) mais placés exactement là où l'état bouge.
//
// POURQUOI UN AXE À PART, ET NON DANS `Inventory`. Le client retient, pour un slot et une
// image, la lecture d'`Inventory` la plus récente ≤ T, et lit sur ELLE le chargeur, la réserve
// et l'emplacement dégainé autant que les grenades. Y verser des lectures delta — qui ne
// portent QUE des grenades — ferait masquer une lecture d'image-clé pleine par une lecture
// partielle, et la cellule de munitions se viderait. C'est très exactement le défaut que la
// version 19 vient de fermer (« une lecture vide EFFACE ») ; le reproduire sous une autre
// forme serait impardonnable. Les deux canaux se rejoignent donc sur la GRANDEUR GRENADES et
// sur elle seule, chacun disant d'où il vient.
//
// CE QUI N'EST PAS ICI, ET POURQUOI : LES MUNITIONS. Le suivi delta des chargeurs a été
// implémenté, mesuré, et REFUSÉ par sa propre mesure (lot 4.2 du 2026-08-25,
// .ai/V7.5/replay2d/LOT4_SUIVI_DELTA_2026-08-25.md) : la concordance avec les images-clés
// plafonne à 92,80 % et — c'est le point — elle DESCEND quand on rapproche les deux lectures
// (88,06 % à 0,10 s contre 93,19 % à 2 s). Un désaccord dû au tir survenu entre les deux
// mesures ferait l'inverse. Le scanner reste en place et publie ses chiffres ; il n'alimente
// pas la fiche.
//
// L'IDENTITÉ DE L'ARME N'EST PAS SUIVIE NON PLUS : i43/i44 comptent 14 et 9 annonces sur
// 171 851 records du film témoin. Elle reste une lecture d'image-clé, comme l'étude l'annonçait.

// Sources de lecture publiées sur GrenadeRead.Src. Elles ne sont pas décoratives : les deux
// canaux n'ont ni la même cadence ni la même couverture, et un lecteur qui veut juger une
// fraîcheur doit pouvoir les séparer.
const (
	// GrenadeSrcKeyframe : record de biped d'une image-clé, ~toutes les 20 s.
	GrenadeSrcKeyframe = "kf"
	// GrenadeSrcDelta : composants i22/i47 d'un paquet delta, transmis au CHANGEMENT.
	GrenadeSrcDelta = "delta"
)

// GrenadeRead est UNE lecture des grenades portées par un slot.
type GrenadeRead struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée — donc une VIE, pas un joueur.
	Slot uint32 `json:"slot"`
	// G porte le compteur de chaque type de grenade, par RANG (même ordre que
	// ReplayDocument.GrenadeLabels, exactement comme Inventory.G). Une case à 0 est une
	// mesure : « ce type, aucune en réserve ».
	G []uint32 `json:"g"`
	// Gs est le rang SÉLECTIONNÉ — le type qui partira au prochain lancer. POINTEUR : le rang
	// 0 est une valeur, et nil dit « non lu », jamais « le premier type ».
	Gs *int `json:"gs,omitempty"`
	// Src dit par quel canal la lecture est arrivée (GrenadeSrcKeyframe / GrenadeSrcDelta).
	Src string `json:"src"`
}

// buildGrenadeReads projette les deux canaux sur la grille de frames du rejeu.
//
// AUCUN ARBITRAGE, AUCUNE FUSION DE VALEURS : les deux canaux disent la même grandeur, on
// publie les deux lectures telles quelles et le client prend la plus récente. Les départager
// quand elles divergent supposerait de savoir laquelle a tort — ce qu'on ne sait pas. Ce que
// la mesure dit, c'est qu'elles divergent peu : 714 accords sur 729 couples confrontables
// (97,94 %) sur 28 films.
//
// Les lectures ANTÉRIEURES à l'origine du rejeu sont écartées : elles n'ont pas de place sur
// l'axe, et leur en inventer une les poserait sur la première image comme si elles y avaient
// été mesurées.
func buildGrenadeReads(
	kf []KeyframeInventory, deltas []filmdec.InventoryDelta, origin, step uint64,
) []GrenadeRead {
	out := make([]GrenadeRead, 0, len(kf)+len(deltas))
	for _, r := range kf {
		if r.TimestampUS < origin || !r.GrenadesRead {
			continue
		}
		g := GrenadeRead{
			T: int((r.TimestampUS - origin) / step), Slot: r.Slot,
			G: append([]uint32(nil), r.Grenades[:]...), Src: GrenadeSrcKeyframe,
		}
		if r.SelectedGrenadeRank >= 0 {
			sel := r.SelectedGrenadeRank
			g.Gs = &sel
		}
		out = append(out, g)
	}
	for _, d := range deltas {
		// Une lecture delta qui ne porte QUE la sélection n'a pas de compteurs à publier : sur
		// cet axe, la grandeur est le quadruplet. La taire vaut mieux que publier un tableau
		// vide, que le client lirait comme « plus aucune grenade ».
		if d.TimestampUS < origin || d.Grenades == nil {
			continue
		}
		g := GrenadeRead{
			T: int((d.TimestampUS - origin) / step), Slot: d.Slot,
			G: append([]uint32(nil), d.Grenades...), Src: GrenadeSrcDelta,
		}
		if d.SelRead && d.Sel != filmdec.InventoryDeltaNoSel {
			sel := d.Sel
			g.Gs = &sel
		}
		out = append(out, g)
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T != out[j].T {
			return out[i].T < out[j].T
		}
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Src < out[j].Src
	})
	return out
}

// keepGrenadeReadsOfPublishedTracks écarte les lectures dont le slot n'a pas de trajectoire
// publiée : le client n'aurait aucune fiche où les poser.
func keepGrenadeReadsOfPublishedTracks(reads []GrenadeRead, tracks []Track) []GrenadeRead {
	return keepOfPublishedTracks(reads, tracks,
		func(g GrenadeRead, published map[uint32]bool) bool { return published[g.Slot] })
}

// GrenadeReadCoverage dit ce que chaque canal a apporté. Sans ces dénominateurs, un axe
// clairsemé ne se diagnostique pas : rien ne distinguerait « le film ne transmet pas i22 » de
// « le balayage a échoué ».
//
// TÉLÉMÉTRIE PURE : n'affecte aucune valeur consommée par le client.
type GrenadeReadCoverage struct {
	// FromKeyframe / FromDelta : lectures publiées par chaque canal.
	FromKeyframe int `json:"fromKeyframe"`
	FromDelta    int `json:"fromDelta"`
	// Unpublished est le nombre de lectures retirées faute de trajectoire publiée.
	Unpublished int `json:"unpublished"`
	// AmmoRefused dit que le canal MUNITIONS de ce film a été refusé en bloc par le scanner
	// (distribution de chargeurs contaminée). Il ne change rien à cet axe — les grenades ont
	// leurs propres tests — mais il explique pourquoi ce film ne portera jamais de munitions
	// delta, et c'est une information qu'un diagnostic doit pouvoir lire.
	AmmoRefused bool `json:"ammoRefused,omitempty"`
}

// attachGrenadeReadCoverage pose la couverture de l'axe sur le document, et journalise — un
// seul endroit le fait, comme `attachInventoryCoverage` pour le calque d'inventaire.
//
// LA GARDE EST LE POINT, et c'est la même que celle des calques frères : quand l'axe est VIDE,
// aucune couverture n'est publiée. Un {0,0} affirmerait « lecture faite, rien trouvé » là où
// l'ABSENCE dit « ce film ne transmet pas de grenades » — deux choses différentes, et le
// diagnostic repose sur cette distinction.
func attachGrenadeReadCoverage(doc *ReplayDocument, built []GrenadeRead, ammoRefused bool) {
	if doc.Coverage == nil || len(built) == 0 {
		return
	}
	cov := &GrenadeReadCoverage{
		Unpublished: countUnpublished(len(built), len(doc.GrenadeReads)),
		AmmoRefused: ammoRefused,
	}
	for _, g := range doc.GrenadeReads {
		if g.Src == GrenadeSrcDelta {
			cov.FromDelta++
		} else {
			cov.FromKeyframe++
		}
	}
	doc.Coverage.GrenadeReads = cov
	slog.Info("rejeu : couverture des grenades portees",
		"imagesCles", cov.FromKeyframe, "delta", cov.FromDelta,
		"ecarteesSansPiste", cov.Unpublished, "canalMunitionsRefuse", cov.AmmoRefused)
}
