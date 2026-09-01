package main

// refresh_drifted.go — RE-VALIDER LES CARTES DONT LA SOURCE A DERIVE.
//
// # Le gel prudent, et pourquoi il saute
//
// `--only-add-spawn-points` REFUSE de toucher une carte dont le `.mvar` frais ne concorde plus
// avec le catalogue : il ne sait pas si le fichier decrit encore la meme carte, alors il saute
// et compte. Seize cartes sont dans ce cas, et elles restent SANS points d'apparition — donc
// sans origine `spawner` sur leurs rejeux.
//
// Decision utilisateur du 2026-09-01 : on accepte de re-valider automatiquement. Le `.mvar`
// frais redevient la source de verite pour les cartes DEJA au catalogue, sans arbitrage carte
// par carte.
//
// # L'automatique doit rester AUDITABLE, et c'est tout l'enjeu de ce fichier
//
// Regenerer une entree en silence changerait ce que l'application sert sur une carte deja
// jouee, sans que personne puisse dire QUOI. Ce mode produit donc un RAPPORT DE DIFF par carte :
// socles ajoutes, retires, deplaces — avec les distances. Il part au journal ET dans la note du
// catalogue, si bien qu'une relecture du fichier suffit a savoir ce qui a bouge.
//
// # Ce qu'il ne fait pas
//
// Les cartes CONCORDANTES ne sont pas touchees : leur entree reste byte-identique. Et il ne
// tourne pas tout seul — c'est une commande, lancee a la main ou par une maintenance planifiee,
// jamais un effet de bord d'un fetch de film.

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/mapcatalog"
)

// diffSocles est ce qui a change entre deux jeux de socles d'une meme carte.
type diffSocles struct {
	ajoutes, retires int
	// deplaces : les socles APPARIES (meme famille, le plus proche) qui ont bouge, avec la
	// distance. Un socle qui bouge de quelques centimetres est du bruit d'extraction ; un
	// socle qui bouge de plusieurs metres est un changement de CARTE.
	deplaces []float64
}

// pireDeplacement rend la plus grande distance observee, ou zero.
func (d diffSocles) pireDeplacement() float64 {
	pire := 0.0
	for _, v := range d.deplaces {
		if v > pire {
			pire = v
		}
	}
	return pire
}

func (d diffSocles) String() string {
	if d.ajoutes == 0 && d.retires == 0 && len(d.deplaces) == 0 {
		return "aucun changement de socle"
	}
	s := fmt.Sprintf("%d ajoute(s), %d retire(s), %d deplace(s)",
		d.ajoutes, d.retires, len(d.deplaces))
	if len(d.deplaces) > 0 {
		s += fmt.Sprintf(" (deplacement max %.2f m)", d.pireDeplacement())
	}
	return s
}

// deplacementSignificatif est le seuil au-dela duquel un socle qui bouge n'est PAS du bruit
// d'extraction. C'est `mapvar.PadSpotMergeM` : sous un metre, deux declarations sont DEJA
// considerees comme le meme emplacement par le regroupement. Au-dela, ce n'est plus le meme
// socle — et ca merite d'etre signale en tete du rapport, pas enterre dedans.
const deplacementSignificatif = mapvar.PadSpotMergeM

// deplacementAnormal est le seuil au-dela duquel un socle qui bouge n'est PLUS une mise a jour
// de carte plausible, mais la signature d'un MAUVAIS FICHIER.
//
// DIX METRES, et le chiffre vient d'une mesure, pas d'une intuition : la passe fautive de ce
// chantier — carte de BASE prise pour la VARIANTE — a rendu des deplacements de 22 a 80 m sur
// neuf cartes. Avec le bon fichier, il n'en reste que deux, a 2 et 33 m. Dix metres separe donc
// nettement le remaniement reel de l'erreur de fichier, tout en laissant passer un socle
// deplace de quelques metres par une vraie mise a jour.
const deplacementAnormal = 10.0

// accepterGrandsDeplacements leve la garde ci-dessus. Variable de paquet et non parametre :
// elle est posee UNE fois par le drapeau `--accept-large-moves` et lue au fond de la boucle,
// et la faire descendre a travers trois fonctions n'apporterait rien.
var accepterGrandsDeplacements bool

// comparerSocles apparie deux jeux de socles et dit ce qui a change.
//
// L'APPARIEMENT EST PAR FAMILLE ET PAR PROXIMITE, glouton : chaque socle de l'ancien jeu prend
// le socle NON ENCORE PRIS de meme famille le plus proche dans le nouveau. Ce qui reste sans
// partenaire est retire (cote ancien) ou ajoute (cote neuf). Sur des socles espaces de
// plusieurs metres, le resultat ne depend pas de la strategie.
func comparerSocles(avant, apres []replay.MapWeaponPadSpot) diffSocles {
	var d diffSocles
	pris := make([]bool, len(apres))
	for _, a := range avant {
		best, bd := -1, math.MaxFloat64
		for i, b := range apres {
			if pris[i] || b.Family != a.Family {
				continue
			}
			dist := distanceSocles(a, b)
			if dist < bd {
				bd, best = dist, i
			}
		}
		if best < 0 {
			d.retires++
			continue
		}
		pris[best] = true
		if bd > 0 {
			d.deplaces = append(d.deplaces, bd)
		}
	}
	for i := range apres {
		if !pris[i] {
			d.ajoutes++
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(d.deplaces)))
	return d
}

func distanceSocles(a, b replay.MapWeaponPadSpot) float64 {
	dx, dy, dz := a.Pos.X-b.Pos.X, a.Pos.Y-b.Pos.Y, a.Pos.Z-b.Pos.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// bilanRefresh compte ce qu'une passe de re-validation a fait.
type bilanRefresh struct {
	regenerees, concordantes, sansDump, echecs, refusees int
	// rapports : une ligne par carte regeneree, dans l'ordre des map_id — deterministe.
	rapports []string
	// spectaculaires : les cartes dont un socle a bouge de plus de `deplacementSignificatif`.
	// Elles remontent EN TETE du rapport plutot que de se noyer dedans.
	spectaculaires []string
}

// refreshDrifted regenere l'entree COMPLETE des cartes dont la source a derive.
func refreshDrifted(ctx context.Context, objectifs *replay.MapObjectivesCatalog,
	dumps *dumpIndex, outPath string, dryRun bool,
) {
	cat, err := replay.LoadMapWeaponPads(outPath)
	if err != nil {
		fail(ctx, "catalogue des socles existant", err)
	}
	var b bilanRefresh
	ids := make([]string, 0, len(cat.Maps))
	for id := range cat.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids) // ORDRE DETERMINISTE : deux executions rendent le meme fichier.
	for _, mapID := range ids {
		revaliderUneCarte(ctx, cat, objectifs, dumps, mapID, &b)
	}
	slog.InfoContext(ctx, "mapopads: re-validation des cartes derivees",
		"regenerees", b.regenerees, "concordantes", b.concordantes,
		"refusees_deplacement_anormal", b.refusees,
		"sans_dump", b.sansDump, "echecs", b.echecs,
		"cartes_a_deplacement_significatif", len(b.spectaculaires))
	for _, r := range b.rapports {
		slog.InfoContext(ctx, "mapopads: diff de socles", "detail", r)
	}
	if len(b.spectaculaires) > 0 {
		slog.WarnContext(ctx, "mapopads: DEPLACEMENTS DE SOCLE SIGNIFICATIFS — a relire avant "+
			"de considerer la passe comme acquise", "cartes", b.spectaculaires)
	}
	if cat.Notes == nil {
		cat.Notes = map[string]string{}
	}
	cat.Notes["refresh_drifted"] = noteRefresh(b)
	if dryRun {
		slog.InfoContext(ctx, "mapopads: dry-run, rien ecrit")
		return
	}
	if err := writeCatalog(cat, outPath); err != nil {
		fail(ctx, "ecriture du catalogue", err)
	}
	slog.InfoContext(ctx, "mapopads: catalogue ecrit", "path", outPath, "cartes", len(cat.Maps))
}

// revaliderUneCarte traite UNE carte : elle est regeneree, ou laissee telle quelle.
func revaliderUneCarte(ctx context.Context, cat *replay.MapWeaponPadsCatalog,
	objectifs *replay.MapObjectivesCatalog, dumps *dumpIndex, mapID string, b *bilanRefresh,
) {
	entry := cat.Maps[mapID]
	e, ok := objectifs.Maps[mapID]
	if !ok {
		b.sansDump++
		return
	}
	path, base, ok := dumps.resolve(mapID, e)
	if !ok {
		b.sansDump++
		return
	}
	neuf, _, err := ingestFn(mapID, e, path, base)
	if err != nil {
		slog.WarnContext(ctx, "mapopads: variante illisible, carte laissee telle quelle",
			"map_id", mapID, "err", err)
		b.echecs++
		return
	}
	if mapcatalog.DriftOf(entry, neuf) == "" {
		// CONCORDANTE : on ne la touche pas, son entree reste byte-identique.
		b.concordantes++
		return
	}
	d := comparerSocles(entry.Pads, neuf.Pads)
	nom := entry.PublicName
	if nom == "" {
		nom = base
	}
	ligne := fmt.Sprintf("%s (%s) : %d -> %d socles · %s",
		nom, mapID, len(entry.Pads), len(neuf.Pads), d)
	// LA GARDE, ET ELLE PASSE AVANT L'ECRITURE — c'est tout son interet.
	//
	// Un socle qui bouge de plus de `deplacementAnormal` n'est pas une mise a jour de carte :
	// c'est la signature du MAUVAIS FICHIER (carte de base plaquee sur la variante). La
	// premiere passe de ce chantier a rendu jusqu'a 79,87 m et allait les ecrire ; seule une
	// verification humaine l'a arretee. Desormais la carte est SAUTEE, comptee, et rapportee —
	// l'automatique reste automatique pour la derive normale, et l'anomalie exige un geste.
	if d.pireDeplacement() > deplacementAnormal && !accepterGrandsDeplacements {
		b.refusees++
		b.rapports = append(b.rapports, "REFUSEE (deplacement anormal) — "+ligne)
		b.spectaculaires = append(b.spectaculaires, fmt.Sprintf("%s (%s)", nom, mapID))
		return
	}
	b.rapports = append(b.rapports, ligne)
	if d.pireDeplacement() > deplacementSignificatif || d.ajoutes > 0 || d.retires > 0 {
		b.spectaculaires = append(b.spectaculaires, fmt.Sprintf("%s (%s)", nom, mapID))
	}
	cat.Maps[mapID] = neuf
	b.regenerees++
}

// noteRefresh redige ce que la passe laisse dans le fichier lui-meme.
func noteRefresh(b bilanRefresh) string {
	s := "Re-validation des cartes a source derivee (--refresh-drifted) : " +
		itoaSimple(b.regenerees) + " carte(s) REGENEREES depuis leur .mvar frais (socles " +
		"d'armes ET points d'apparition), " + itoaSimple(b.concordantes) + " concordantes " +
		"laissees byte-identiques, " + itoaSimple(b.sansDump) + " sans .mvar au depot, " +
		itoaSimple(b.echecs) + " en echec de lecture, " + itoaSimple(b.refusees) +
		" REFUSEE(S) pour deplacement anormal (>10 m : signature du mauvais fichier, pas d'une " +
		"mise a jour de carte — relancer avec --accept-large-moves apres verification). "
	if len(b.spectaculaires) > 0 {
		s += "ATTENTION — " + itoaSimple(len(b.spectaculaires)) + " carte(s) avec socles " +
			"ajoutes, retires ou deplaces de plus d'un metre : leurs rejeux FUTURS serviront " +
			"des socles differents. "
	}
	s += "LES ARTEFACTS DEJA CUITS NE BOUGENT PAS — ce sont des donnees figees ; seules les " +
		"cuissons futures voient les socles frais. Diff par carte : voir le journal de la passe."
	for _, r := range b.rapports {
		s += " | " + r
	}
	return s
}
