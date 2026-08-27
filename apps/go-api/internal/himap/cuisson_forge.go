// Package himap — cuisson_forge.go : LA CHAINE DE PRODUCTION D'UNE CARTE FORGE.
//
// LA DIFFERENCE N'EST PAS « Forge ou pas ». Cliffhanger aussi a ete concue dans Forge, mais 343
// l'a CUITE dans un module dedie : 10 223 instances de geometrie contre 443 objets de variante.
// Vagabond, elle, n'est pas cuite — 788 instances dans son canevas contre 4 709 objets dans son
// `.mvar`. Sa carte est le RACK D'OBJETS, pas le module.
//
// LA CHAINE, etablie sur pieces le 2026-08-10 (sondes `sonde_forge_gamefiles_test.go`) :
//
//	objet .mvar --type_id--> tag `food` (GlobalID, forge_objects-rtx-new.module)
//	           --refs inline--> tags `rtgo` (les MEMES maillages que la chaine sbsp)
//	           --Pos/Up/Forward--> repere monde (Left = Up x Forward, base orthonormee)
//
// CE QUI FONDE CHAQUE MAILLON, mesure et non suppose :
//   - type_id -> food : 467/468 type_id de Vagabond se resolvent ;
//   - food -> rtgo : les deps des tags food sont VIDES (457/467) et root+0x08 est
//     l'auto-reference — le lien est INLINE. 374 type_id portent au moins une ref rtgo,
//     couvrant 3 558 des 4 697 objets (75,7 %). Les 93 restants passent par `bloc`/`scen`/`mach`
//     (963/173/9 objets) : le SAUT `bloc`/`scen`/`mach` -> `hlmt` -> `rtgo` les resout (lot B,
//     2026-08-13, mesure : `sonde_forge_saut_gamefiles_test.go`) ;
//   - echelle : AUCUNE dans le `.mvar` de Vagabond. MESURE, pas suppose — le champ objet [6]
//     n'existe pas et [9] est une struct VIDE sur 4 709/4 709 objets. Le piege de l'echelle
//     d'instance sbsp, paye deux jours, a ete verifie ici sur pieces.
//
// CE QUE CETTE CHAINE NE FAIT PAS, et c'est assume : elle ne rend pas la TOILE du canevas sous
// les objets (`fo08_wetland`), et n'applique ni frontiere de mort ni eau — les cartes Forge
// declarent leurs limites dans leurs propres objets, pas dans un tag `sddt`. Au registre.
//
// Les DECLARATIONS des cartes (map_id, canevas, `.mvar`) vivent dans cartes_forge.go.
package himap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
)

// TypesVolumesDeMort : empreinte fonctionnelle des volumes de mort, etablie sur 101 `.mvar`
// (INVESTIGATION_VOLUMES_MORT_MVAR_2026-08-10.md §2).
//
// Ils sont exclus PAR CONSTRUCTION (ils n'ont aucune ref rtgo) et COMPTES explicitement, pour
// que l'exclusion soit un fait mesure et non une consequence heureuse.
var TypesVolumesDeMort = map[int32]string{
	-588988541:  "volume de mort principal",
	176825834:   "plancher de mort",
	937132837:   "volume de mort sous-sol",
	-1751270658: "mur de limite",
}

// ErrSansObjetForge signale un `.mvar` sans objet exploitable.
var ErrSansObjetForge = errors.New("himap: aucun objet Forge a poser")

// OptionsCuissonForge decrit une carte Forge a cuire.
type OptionsCuissonForge struct {
	// RacineDeploy est la racine `deploy` de l'installation.
	RacineDeploy string
	// Objets sont les objets poses de la variante de carte (`.mvar` deja lu).
	Objets []mapvar.Object
	// Ancres sont les positions monde des objectifs de la carte.
	Ancres [][3]float64
	// CheminModuleCanevas est le `.module` du canevas (ex. fo08_wetland), ajoute a l'index
	// pour que ses `rtgo` propres soient resolus. Vide = canevas inconnu, on s'en passe.
	CheminModuleCanevas string
	// Cle est le nom sous lequel l'asset sera publie (cf. BilanCuisson.Module) : le map_id
	// de la carte (cartes_forge.go).
	Cle string
	// Echelle est le cote d'un pixel du fond, en metres. ZERO = `EchelleFondCarte`.
	// Meme reglage et memes consequences que `OptionsCuisson.Echelle` — voir cuisson.go.
	Echelle float64
	// CibleCadrePx : meme role que OptionsCuisson.CibleCadrePx.
	CibleCadrePx int
	// LES LEVIERS DE RENDU, jusqu ici reserves a la chaine native (2026-08-27). Une carte
	// Forge n en avait AUCUN : declarer un ecretage pour Isolation ne changeait pas un octet
	// de son image, et rien ne le disait. Ils ont la meme semantique que dans OptionsCuisson.
	EcreteToits            bool
	PlafondArene           float64
	SubstitutionSansPortee bool
	BoiteUtile             [4]float64
	// RogneAuxVolumesDeMort borne la matiere a l emprise des VOLUMES DE MORT de la variante.
	// C est l equivalent Forge du rognage aux zones de callout : les callouts disent ou l on
	// joue, les volumes de mort disent ou l on meurt, et les 22 cartes a callouts sont toutes
	// natives. Sans effet si la variante n en declare aucun avec une forme.
	RogneAuxVolumesDeMort bool
	// TypesExclus ecarte des TYPES d objet Forge du dessin. Dernier recours, quand un modele
	// balaie la carte et qu aucune coupe geometrique ne peut l atteindre — voir le diagnostic
	// « types les plus etendus » journalise a chaque cuisson.
	TypesExclus map[int32]bool
	// PlancherTranche / PlafondTranche : memes roles que dans OptionsCuisson.
	PlancherTranche float64
	PlafondTranche  float64
}

// CuitCarteForge rend le fond de carte d'une carte Forge en posant les modeles de ses objets.
func CuitCarteForge(ctx context.Context, opts OptionsCuissonForge) (*Rendu, BilanCuisson, error) {
	b := BilanCuisson{Module: opts.Cle, Ancres: len(opts.Ancres), ObjetsForge: len(opts.Objets)}
	if len(opts.Ancres) == 0 {
		return nil, b, ErrSansAncre
	}
	if len(opts.Objets) == 0 {
		return nil, b, ErrSansObjetForge
	}
	idx, forge, err := indexForge(opts)
	if err != nil {
		return nil, b, err
	}

	r := CadreSurAncresEchelle(opts.Ancres, EchellePourCadre(opts.Ancres, opts.Echelle, opts.CibleCadrePx))
	// LA TRANCHE, TRANSLATEE AU SOL DES ANCRES. Le sol de Vagabond vit vers z=52 : c'est ici
	// qu'on a compris qu'une tranche absolue n'avait pas de sens, et la chaine native applique
	// desormais la meme regle (cf. `TrancheDeJeu`).
	zJeu := MedianeZ(opts.Ancres) - AncrageDecalageSol
	b.NiveauDeJeu = zJeu
	minT, maxT := TrancheDeJeu(zJeu)
	if opts.PlancherTranche < 0 {
		minT = zJeu + opts.PlancherTranche
	}
	if opts.PlafondTranche > 0 {
		maxT = zJeu + opts.PlafondTranche
	}
	r.Tranche(minT, maxT)
	r.NiveauDeJeu(zJeu)
	// MEME regle que la chaine native : la voie de reference contre les toits
	// (rendu_reference.go). Une carte Forge a ciel ouvert reste sous le seuil et n'est pas
	// touchee ; la regle est universelle, pas une affaire de chaine.
	s := NewSurfaceReference(opts.Ancres)
	r.ArmeReference(s)

	poseObjetsForge(ctx, r, &b, opts.Objets, idx, forge, opts.TypesExclus)
	if b.ObjetsDessines == 0 {
		return nil, b, fmt.Errorf("aucun des %d objets Forge n'a de modele rtgo", len(opts.Objets))
	}
	if opts.EcreteToits {
		b.TauxCouverture, b.CellulesSubstituees, b.CellulesEcretees = r.EcretteToits(s, opts.PlafondArene)
		b.CarteCouverte = b.TauxCouverture > SeuilCarteCouverte
	} else {
		b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s, opts.SubstitutionSansPortee)
	}
	borneALaBoite(ctx, r, &b, boiteForge(ctx, opts))
	if b.VolumesDeMort == 0 {
		b.degrade(ctx, "aucun volume de mort reconnu — l'empreinte des types a peut-etre bouge")
	}
	JugeParLesAncres(r, &b, opts.Ancres)
	slog.InfoContext(ctx, "carte Forge cuite", "cle", b.Module,
		"objets", fmt.Sprintf("%d/%d", b.ObjetsDessines, b.ObjetsForge),
		"sansModele", b.ObjetsSansModele, "volumesDeMort", b.VolumesDeMort,
		"couverture", fmt.Sprintf("%.1f%%", 100*b.TauxCouverture), "couverte", b.CarteCouverte,
		"ancres", fmt.Sprintf("%d/%d", b.AncresAvecSol, b.AncresDansLeCadre))
	return r, b, nil
}

// indexForge construit l'index des tags : le module des objets Forge d'abord (les `food` et
// leurs `rtgo`), puis le canevas et les globaux.
func indexForge(opts OptionsCuissonForge) (*ModuleIndex, *himodule.Module, error) {
	principal := filepath.Join(opts.RacineDeploy, "any", "globals", "forge", "forge_objects-rtx-new.module")
	chemins := []string{principal}
	if p := filepath.Join(opts.RacineDeploy, "pc", "globals", "forge", "forge_objects-rtx-new.module"); existeFichier(p) {
		chemins = append(chemins, p)
	}
	if opts.CheminModuleCanevas != "" && existeFichier(opts.CheminModuleCanevas) {
		chemins = append(chemins, opts.CheminModuleCanevas)
	}
	globs, _ := filepath.Glob(filepath.Join(opts.RacineDeploy, "pc", "globals", "*.module"))
	chemins = append(chemins, globs...)
	// Les definitions d'objet du SAUT (`bloc`/`scen`/`mach`, lot B) vivent pour partie dans
	// les globals de la variante `any` (globals-rtx-new, common-rtx-new) : sans eux, 17 food
	// de Vagabond ne resolvent aucune definition (sonde du 2026-08-13).
	globsAny, _ := filepath.Glob(filepath.Join(opts.RacineDeploy, "any", "globals", "*.module"))
	chemins = append(chemins, globsAny...)

	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		return nil, nil, fmt.Errorf("index des modules Forge : %w", err)
	}
	forge, err := himodule.Open(principal)
	if err != nil {
		return nil, nil, fmt.Errorf("module des objets Forge : %w", err)
	}
	return idx, forge, nil
}

// poseObjetsForge resout le modele de chaque type d'objet, puis pose ses triangles.
func poseObjetsForge(ctx context.Context, r *Rendu, b *BilanCuisson,
	objets []mapvar.Object, idx *ModuleIndex, forge *himodule.Module, exclus map[int32]bool) {
	modeleDuType := modeleParType(ctx, objets, idx, forge)
	assets := map[uint32]*RuntimeGeoAsset{}
	// DIAGNOSTIC DES TYPES ETENDUS. Le « gribouillis » d Isolation ne cede ni a l ecretage ni
	// au bornage : il vit A HAUTEUR DE SOL. Reste une cause possible — un TYPE d objet dont le
	// modele balaie la carte. On mesure donc, une fois par type, l emprise du premier exemplaire
	// pose ; le classement se lit dans les logs et sert a remplir le reglage typesExclus.
	etendues := map[int32]float64{}
	comptes := map[int32]int{}
	defer func() { journaliseTypesEtendus(ctx, b, etendues, comptes) }()
	for _, o := range objets {
		if _, mort := TypesVolumesDeMort[o.TypeID]; mort {
			b.VolumesDeMort++
			continue
		}
		m, ok := modeleDuType[o.TypeID]
		if !ok {
			b.ObjetsSansModele++
			continue
		}
		a, deja := assets[m.id]
		if !deja {
			a = ouvreAsset(ctx, idx, m.id, m.groupe)
			assets[m.id] = a
		}
		if a == nil {
			b.ObjetsSansModele++
			continue
		}
		in := InstanceForge(o)
		if _, vu := etendues[o.TypeID]; !vu {
			etendues[o.TypeID] = etendueMondeDe(a, in)
		}
		comptes[o.TypeID]++
		if exclus[o.TypeID] {
			b.ObjetsExclus++
			continue
		}
		for mi := 0; mi < a.MeshCount(); mi++ {
			if mesh := a.Mesh(mi); mesh != nil {
				r.AddMesh(mesh, in)
			}
		}
		b.ObjetsDessines++
	}
}

// refModele designe le tag de geometrie d'un type d'objet : son GlobalID et son groupe —
// `rtgo` (lecture directe) ou `mode` (render_model, lot B), qui ne s'ouvrent pas pareil.
type refModele struct {
	id     uint32
	groupe string
}

// modeleParType etablit, une fois par type d'objet, le tag de geometrie de son modele : la
// ref `rtgo` directe du `food` d'abord, le saut `bloc`/`scen`/`mach` -> `hlmt` sinon.
func modeleParType(ctx context.Context, objets []mapvar.Object,
	idx *ModuleIndex, forge *himodule.Module) map[int32]refModele {
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	out := map[int32]refModele{}
	vus := map[int32]bool{}
	for _, o := range objets {
		if vus[o.TypeID] {
			continue
		}
		vus[o.TypeID] = true
		f, ok := foodParID[uint32(o.TypeID)]
		if !ok {
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			slog.DebugContext(ctx, "tag food illisible", "typeID", o.TypeID, "err", err)
			continue
		}
		if refs := refsInlineDuGroupe(tag, idx, GroupeRtgo); len(refs) > 0 {
			out[o.TypeID] = refModele{id: refs[0], groupe: GroupeRtgo}
			continue
		}
		if m, ok := modeleParSaut(ctx, idx, tag); ok {
			out[o.TypeID] = m
		}
	}
	return out
}

// GroupeHlmt / GroupeMode : le tag de modele (`model`) et le render_model, maillons du saut.
const (
	GroupeHlmt = "hlmt"
	GroupeMode = "mode"
)

// groupesSautForge : les groupes de definition d'objet Forge SANS ref rtgo directe dans leur
// `food`, dans l'ordre de frequence mesure sur Vagabond — 963 objets via `bloc`, 173 via
// `scen`, 9 via `mach` (sonde F1CouvertureRtgo, 2026-08-10).
var groupesSautForge = []string{"bloc", "scen", "mach"}

// modeleParSaut resout le modele d'un `food` sans ref rtgo directe : la definition d'objet
// passe par un tag `bloc`/`scen`/`mach`, qui reference son modele `hlmt`, lequel porte la
// geometrie — un `rtgo`, ou un `mode` (mesure 2026-08-13 : les 125 hlmt du saut de Vagabond
// ne referencent QUE des `mode`). Meme mecanique a chaque maillon — le scan des octets
// contre l'index, la methode qui a ferme F1 (`sonde_forge_saut_gamefiles_test.go`).
func modeleParSaut(ctx context.Context, idx *ModuleIndex, tagFood []byte) (refModele, bool) {
	for _, groupe := range groupesSautForge {
		for _, hObjet := range refsInlineDuGroupe(tagFood, idx, groupe) {
			objet, err := idx.Extract(hObjet)
			if err != nil {
				slog.DebugContext(ctx, "tag de saut illisible", "groupe", groupe, "id", hObjet, "err", err)
				continue
			}
			if m, ok := modeleDuHlmt(ctx, idx, objet); ok {
				return m, true
			}
		}
	}
	return refModele{}, false
}

// modeleDuHlmt rend la premiere ref de geometrie (`rtgo`, sinon `mode`) portee par les
// modeles `hlmt` d'un tag d'objet.
func modeleDuHlmt(ctx context.Context, idx *ModuleIndex, objet []byte) (refModele, bool) {
	for _, hModele := range refsInlineDuGroupe(objet, idx, GroupeHlmt) {
		hlmt, err := idx.Extract(hModele)
		if err != nil {
			slog.DebugContext(ctx, "tag hlmt illisible", "id", hModele, "err", err)
			continue
		}
		if refs := refsInlineDuGroupe(hlmt, idx, GroupeRtgo); len(refs) > 0 {
			return refModele{id: refs[0], groupe: GroupeRtgo}, true
		}
		if refs := refsInlineDuGroupe(hlmt, idx, GroupeMode); len(refs) > 0 {
			return refModele{id: refs[0], groupe: GroupeMode}, true
		}
	}
	return refModele{}, false
}

// refsInlineDuGroupe rend les GlobalID du groupe donne references dans les octets d'un tag.
func refsInlineDuGroupe(tag []byte, idx *ModuleIndex, groupe string) []uint32 {
	return RefsInline(tag, func(h uint32) bool {
		g, _, ok := idx.Lookup(h)
		return ok && g == groupe
	})
}

// RefsInline rend les GlobalID retenus par le predicat dans les octets d'un tag, par pas de
// 4, dans l'ordre du tag. Le premier est la variante principale — convention etablie au lot 2.
func RefsInline(tag []byte, retient func(uint32) bool) []uint32 {
	var out []uint32
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(u32(tag, o))
		if !vus[h] && retient(h) {
			vus[h] = true
			out = append(out, h)
		}
	}
	return out
}

// InstanceForge construit l'instance de pose d'un objet `.mvar` : base orthonormee
// Forward/Left/Up avec Left = Up x Forward, echelle unitaire (mesuree ABSENTE du format).
func InstanceForge(o mapvar.Object) Instance {
	f := normalise([3]float64{o.Forward.X, o.Forward.Y, o.Forward.Z})
	u := normalise([3]float64{o.Up.X, o.Up.Y, o.Up.Z})
	l := [3]float64{
		u[1]*f[2] - u[2]*f[1],
		u[2]*f[0] - u[0]*f[2],
		u[0]*f[1] - u[1]*f[0],
	}
	return Instance{
		Scale:    [3]float64{1, 1, 1},
		Position: [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z},
		Forward:  f,
		Left:     l,
		Up:       u,
	}
}

func existeFichier(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// BoiteDesVolumesDeMort rend l'emprise XY [minX, minY, maxX, maxY] des volumes de mort d'une
// variante, et le nombre de volumes qui l'ont formee.
//
// POURQUOI. Une carte native se rogne a ses ZONES DE CALLOUT : elles disent ou l'on joue, et
// c'est le levier qui a sauve Streets, Prism ou Forbidden. Une carte FORGE n'en a aucune — les
// 22 cartes a callouts sont toutes natives. Son equivalent est ici : les volumes de mort
// bornent le terrain par l'autre bout, en declarant ou l'on MEURT. Ils sont deja reconnus et
// comptes par la cuisson (`TypesVolumesDeMort`, empreinte etablie sur 101 `.mvar`), et
// jusqu'ici seulement pour etre ECARTES du dessin ; leur position n'avait jamais servi.
//
// La forme d'un objet donne des DEMI-EXTENTS (shape.go) : l'emprise d'un volume est donc son
// centre plus ou moins sa demi-largeur. Un volume sans forme est ignore — il ne borne rien.
func BoiteDesVolumesDeMort(objets []mapvar.Object) (boite [4]float64, n int) {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, o := range objets {
		if _, mort := TypesVolumesDeMort[o.TypeID]; !mort {
			continue
		}
		s := o.Shape()
		if s == nil {
			continue
		}
		demi := [2]float64{}
		switch {
		case s.Radius != nil:
			demi = [2]float64{*s.Radius, *s.Radius}
		case s.HalfX != nil && s.HalfY != nil:
			// La boite est orientee par Forward ; on prend le PIRE cas, la demi-diagonale,
			// plutot que de projeter — un bornage trop large ne retire rien a tort.
			d := math.Hypot(*s.HalfX, *s.HalfY)
			demi = [2]float64{d, d}
		default:
			continue
		}
		c := [2]float64{float64(o.Pos.X), float64(o.Pos.Y)}
		for k := 0; k < 2; k++ {
			lo[k] = math.Min(lo[k], c[k]-demi[k])
			hi[k] = math.Max(hi[k], c[k]+demi[k])
		}
		n++
	}
	if n == 0 {
		return [4]float64{}, 0
	}
	return [4]float64{lo[0], lo[1], hi[0], hi[1]}, n
}

// boiteForge rend le rectangle monde auquel borner la matiere : celui declare a la main s'il
// l'est, sinon l'emprise des volumes de mort si on la demande, sinon aucun.
func boiteForge(ctx context.Context, opts OptionsCuissonForge) [4]float64 {
	if opts.BoiteUtile[2] > opts.BoiteUtile[0] && opts.BoiteUtile[3] > opts.BoiteUtile[1] {
		return opts.BoiteUtile
	}
	if !opts.RogneAuxVolumesDeMort {
		return [4]float64{}
	}
	boite, n := BoiteDesVolumesDeMort(opts.Objets)
	slog.InfoContext(ctx, "mapfond: bornage aux volumes de mort", "carte", opts.Cle,
		"volumes", n, "boite", fmt.Sprintf("[%.1f %.1f %.1f %.1f]", boite[0], boite[1], boite[2], boite[3]))
	return boite
}

// etendueMondeDe rend le plus grand cote XY, en metres, de l'emprise MONDE du premier
// exemplaire d'un modele. Une valeur de l'ordre du metre est un objet de decor ; plusieurs
// dizaines de metres sur une arene de cent, c'est un modele qui balaie la carte.
func etendueMondeDe(a *RuntimeGeoAsset, in Instance) float64 {
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	for mi := 0; mi < a.MeshCount(); mi++ {
		m := a.Mesh(mi)
		if m == nil {
			continue
		}
		for _, v := range m.Vertices {
			w := in.LocalToWorld(v)
			for k := 0; k < 2; k++ {
				lo[k] = math.Min(lo[k], w[k])
				hi[k] = math.Max(hi[k], w[k])
			}
		}
	}
	if math.IsInf(lo[0], 1) {
		return 0
	}
	return math.Max(hi[0]-lo[0], hi[1]-lo[1])
}

// journaliseTypesEtendus classe les types par emprise et journalise les dix premiers. C'est la
// SEULE sortie qui permette de designer un type fautif : les objets Forge n'ont pas de nom.
func journaliseTypesEtendus(ctx context.Context, b *BilanCuisson, etendues map[int32]float64, comptes map[int32]int) {
	type ligne struct {
		typeID  int32
		etendue float64
		n       int
	}
	var l []ligne
	for t, e := range etendues {
		l = append(l, ligne{t, e, comptes[t]})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].etendue > l[j].etendue })
	if len(l) > 10 {
		l = l[:10]
	}
	var detail []string
	for _, x := range l {
		detail = append(detail, fmt.Sprintf("%d:%.0fm x%d", x.typeID, x.etendue, x.n))
	}
	slog.InfoContext(ctx, "mapfond: types les plus etendus", "carte", b.Module,
		"types", len(etendues), "top", strings.Join(detail, " "))
}
