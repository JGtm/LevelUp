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
//     (963/173/9 objets) : saut supplementaire NON traite ici, c'est le lot B ;
//   - echelle : AUCUNE dans le `.mvar` de Vagabond. MESURE, pas suppose — le champ objet [6]
//     n'existe pas et [9] est une struct VIDE sur 4 709/4 709 objets. Le piege de l'echelle
//     d'instance sbsp, paye deux jours, a ete verifie ici sur pieces.
//
// CE QUE CETTE CHAINE NE FAIT PAS, et c'est assume : elle ne rend pas la TOILE du canevas sous
// les objets (`fo08_wetland`), et n'applique ni frontiere de mort ni eau — les cartes Forge
// declarent leurs limites dans leurs propres objets, pas dans un tag `sddt`. Au registre.
package himap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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

// CarteForge declare une carte Forge a cuire. Le catalogue d'objectifs ne suffit pas a la
// designer : l'entree de Vagabond y porte le module GENERIQUE `map` (comme Highpower), donc la
// selection par nom de module est ambigue. On la designe par son `map_id` (asset UGC) et on
// VERIFIE la jointure par le compte d'objets du catalogue.
type CarteForge struct {
	// Cle est le nom sous lequel l'asset est publie. C'est le module du CANEVAS, pour rester
	// dans le meme espace de noms que les cartes natives.
	//
	// CONDITION DE VALIDITE : deux cartes Forge baties sur le meme canevas entreraient en
	// collision. Il n'y en a qu'une aujourd'hui ; le jour ou il y en aura deux, la cle devra
	// devenir le `map_id`. Inscrit au registre des reports.
	Cle string
	// MapID est l'asset UGC, cle du catalogue d'objectifs.
	MapID string
	// FichierMvar est le nom du `.mvar` dans le depot de variantes de carte.
	FichierMvar string
	// ModuleCanevas est le dossier du module sur lequel la carte est batie.
	ModuleCanevas string
}

// CartesForge : les cartes Forge dont l'asset est produit.
//
// COMMENT ON SAIT QU'UNE CARTE EST FORGE, et ce n'est pas un prefixe de nom : son module de
// canevas ne porte AUCUNE instance de geometrie. La chaine native le dit d'elle-meme —
// « aucune instance dessinee sur 0 du bsp retenu » — c'est ainsi que Corpo a ete identifiee, en
// echouant. Un canevas vierge (`fo11_blank`) est vide par construction : la carte EST son rack
// d'objets.
var CartesForge = []CarteForge{
	{
		Cle:           "fo08_wetland",
		MapID:         "105f5d84-8de1-4908-af3a-1c4f3bf9d642",
		FichierMvar:   "vagabond_map.mvar",
		ModuleCanevas: "fo08_wetland",
	},
	{
		Cle:           "fo11_blank",
		MapID:         "8be179f7-8940-4868-b881-44cad1ca8711",
		FichierMvar:   "corpo_map.mvar",
		ModuleCanevas: "fo11_blank",
	},
}

// EstCarteForge dit si une cle de publication est celle d'une carte Forge declaree — donc si la
// chaine NATIVE doit la laisser tranquille.
func EstCarteForge(cle string) bool {
	for _, c := range CartesForge {
		if c.Cle == cle {
			return true
		}
	}
	return false
}

// DepotVariantesCarte : ou sont les `.mvar`, RELATIVEMENT A LA RACINE DU DEPOT.
//
// Ce dossier n'est pas versionne (`.gitignore`), au meme titre que l'installation du jeu :
// c'est une entree d'outillage hors ligne, pas une donnee de reference. Sa constitution est
// decrite dans `.ai/V7.5/` — un `.mvar` absent fait echouer la cuisson de sa carte, jamais des
// autres.
const DepotVariantesCarte = ".ai/re_dump/mapvar"

// CheminCanevasForge rend le chemin du `.module` du canevas d'une carte Forge, ou la chaine
// vide s'il n'est pas installe (la cuisson s'en passe alors, cf. OptionsCuissonForge).
func CheminCanevasForge(c CarteForge) string {
	dir, err := LevelsDir("pc")
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, c.ModuleCanevas, c.ModuleCanevas+"-rtx-new.module")
	if !existeFichier(p) {
		return ""
	}
	return p
}

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
	// Cle est le nom sous lequel l'asset sera publie (cf. BilanCuisson.Module).
	Cle string
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

	r := CadreSurAncres(opts.Ancres)
	// LA TRANCHE, TRANSLATEE AU SOL DES ANCRES. Le sol de Vagabond vit vers z=52 : c'est ici
	// qu'on a compris qu'une tranche absolue n'avait pas de sens, et la chaine native applique
	// desormais la meme regle (cf. `TrancheDeJeu`).
	zJeu := MedianeZ(opts.Ancres) - AncrageDecalageSol
	b.NiveauDeJeu = zJeu
	r.Tranche(TrancheDeJeu(zJeu))
	r.NiveauDeJeu(zJeu)
	// MEME regle que la chaine native : la voie de reference contre les toits
	// (rendu_reference.go). Une carte Forge a ciel ouvert reste sous le seuil et n'est pas
	// touchee ; la regle est universelle, pas une affaire de chaine.
	s := NewSurfaceReference(opts.Ancres)
	r.ArmeReference(s)

	poseObjetsForge(ctx, r, &b, opts.Objets, idx, forge)
	if b.ObjetsDessines == 0 {
		return nil, b, fmt.Errorf("aucun des %d objets Forge n'a de modele rtgo", len(opts.Objets))
	}
	b.TauxCouverture, b.CellulesSubstituees, b.CarteCouverte = r.AppliqueReference(s)
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
	objets []mapvar.Object, idx *ModuleIndex, forge *himodule.Module) {
	rtgoDuType := rtgoParType(ctx, objets, idx, forge)
	assets := map[uint32]*RuntimeGeoAsset{}
	for _, o := range objets {
		if _, mort := TypesVolumesDeMort[o.TypeID]; mort {
			b.VolumesDeMort++
			continue
		}
		id, ok := rtgoDuType[o.TypeID]
		if !ok {
			b.ObjetsSansModele++
			continue
		}
		a, deja := assets[id]
		if !deja {
			a = ouvreAsset(ctx, idx, id)
			assets[id] = a
		}
		if a == nil {
			b.ObjetsSansModele++
			continue
		}
		in := InstanceForge(o)
		for mi := 0; mi < a.MeshCount(); mi++ {
			if m := a.Mesh(mi); m != nil {
				r.AddMesh(m, in)
			}
		}
		b.ObjetsDessines++
	}
}

// rtgoParType etablit, une fois par type d'objet, le tag `rtgo` de son modele.
func rtgoParType(ctx context.Context, objets []mapvar.Object,
	idx *ModuleIndex, forge *himodule.Module) map[int32]uint32 {
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	estRtgo := func(h uint32) bool {
		g, _, ok := idx.Lookup(h)
		return ok && g == GroupeRtgo
	}
	out := map[int32]uint32{}
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
		if refs := RtgoRefsInline(tag, estRtgo); len(refs) > 0 {
			out[o.TypeID] = refs[0]
		}
	}
	return out
}

// RtgoRefsInline rend les GlobalID `rtgo` references dans les octets d'un tag `food`, dans
// l'ordre du tag. Le premier est la variante principale — convention etablie au lot 2.
func RtgoRefsInline(tag []byte, estRtgo func(uint32) bool) []uint32 {
	var out []uint32
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(u32(tag, o))
		if !vus[h] && estRtgo(h) {
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
