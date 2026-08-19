package replay

// ground_weapon_objects.go — LA CHAÎNE : des lectures du film aux OBJETS AU SOL bornés et datés.
//
// PUR, ENTIÈREMENT. Les trois lectures du film et leur journal vivent dans
// `build_ground_weapons.go` ; ce fichier-ci ne touche jamais un octet de disque.
//
// L'ENTRÉE EST LE RECORD DE CRÉATION, PAS LA DISPERSION DES DELTAS. C'est l'arbitrage du
// 2026-08-17 après le gate 0 du plan : le critère de dispersion mesurait l'immobilité d'une arme
// au sol sur ses seuls échantillons delta, c'est-à-dire sur sa phase MOBILE — un objet qui se
// pose CESSE d'émettre sa position. La position publiable est celle du record de CRÉATION,
// validée par deux oracles indépendants (atterrissage 282/289, identité 937/947).
//
// DEUX POSITIONS, DEUX RÔLES, et les confondre fausserait les deux mesures :
//   - la position de CRÉATION grappe les socles (c'est là que l'arme APPARAÎT) ;
//   - la position de RÉFÉRENCE date la disparition (dernier point de la piste delta si l'objet a
//     bougé, position de création sinon : c'est là qu'il EST au moment où on le prend).

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// WorldObjectScan porte ce que le film rend sur UN archétype d'objet du monde. C'est une entrée
// de DONNÉES de l'assemblage, au même titre que les poses d'équipement.
//
// UNE SEULE FORME POUR DEUX ARCHÉTYPES (2026-08-19) : les armes au sol (`ti=42`) et les objets
// d'équipement du monde (`ti=37`, d'où sortent les socles de POWER-UP) se lisent par les mêmes
// trois lectures et se bornent par le même recensement. Deux structures jumelles auraient
// divergé au premier correctif ; c'est l'archétype qui change, pas la forme de ce qu'on en lit.
//
// LES STATISTIQUES VOYAGENT AVEC LA LISTE : elles portent les dénominateurs du balayage (ancres,
// acceptées), sans lesquels un compte d'apparitions retenues ne se juge pas.
type WorldObjectScan struct {
	// Scanned dit que le film a été BALAYÉ jusqu'au bout. Faux : il n'a pas pu l'être (chunks
	// illisibles, archétype absent des images-clés, bornes de carte absentes) — ou il n'y a pas
	// eu de film du tout (assemblage sur positions figées).
	Scanned bool
	// Creations sont les records de création acceptés par le balayage, AVANT le filtre
	// d'identité (celui-ci est une règle d'assemblage, cf. `padRule`).
	Creations []filmdec.EquipmentCreation
	Stats     filmdec.EquipmentCreationStats
	// Keyframes porte la bande de slots et le RECENSEMENT qui borne les disparitions.
	Keyframes filmdec.WorldObjectKeyframes
	// Tracks sont les pistes de position des paquets delta pour la même bande. Elles disent
	// deux choses : si une vie a bougé (le critère `at_rest`) et où elle s'est arrêtée.
	Tracks []filmdec.ProjectileTrack
}

// padRule est la RÈGLE D'IDENTITÉ d'une chaîne de socles : ce qui fait qu'un record de création
// d'objet du monde devient une apparition publiable, et sous quelle NATURE.
//
// POURQUOI ELLE EST UN OBJET, et pas deux copies de la chaîne (2026-08-19). Entre les armes au
// sol et les power-ups de socle, TOUT est commun — le bornage par le recensement, la datation
// par le passage d'un joueur, la grappe d'un mètre, le seuil de récurrence, le cycle depuis le
// ramassage. Seule diffère la question « cet identifiant, je le publie sous quel nom ? » : le
// catalogue d'ARMES du titre pour `ti=42`, les familles `eqip` du manifeste pour `ti=37`. C'est
// cette question-là, et elle seule, qui devient un paramètre.
type padRule struct {
	// Kind est la NATURE publiée (`gwPadKindWeapon` / `gwPadKindPowerup`). Elle entre dans la
	// clé de grappe : deux natures ne se mélangent jamais au même socle.
	Kind string
	// Objectives est la table des objets d'OBJECTIF du titre, écartés AVANT la question « est-ce
	// publiable ? ». L'ordre est la règle : un drapeau reconnu ne devient pas un socle parce
	// qu'un jour son identifiant entrerait au catalogue d'armes. Vide = la chaîne n'en croise
	// aucun (c'est le cas de `ti=37`, dont les objets d'objectif ne partagent pas le catalogue).
	Objectives map[uint32]Label
	// Family résout l'identité brute (le mot MPP de 32 bits) en FAMILLE de grappe. `ok` faux =
	// la création est écartée — c'est LE filtre de sélectivité de la chaîne, celui sans lequel
	// le témoin fantôme du balayage ne serait pas discriminant (cf. `gwPadsIdentity`).
	Family func(id uint32) (string, bool)
}

// weaponPadRule est la règle des ARMES AU SOL : identité résolue dans le catalogue d'armes du
// titre, nom canonique (alias repliés), objets d'objectif écartés d'abord.
func weaponPadRule(flags map[uint32]Label) padRule {
	known := loadoutFamilies()
	return padRule{
		Kind:       gwPadKindWeapon,
		Objectives: flags,
		Family: func(id uint32) (string, bool) {
			if !known[id] {
				return "", false
			}
			return gwPadsWeaponFamily(id), true
		},
	}
}

// gwPickupObject est une apparition retenue, bornée et datée.
type gwPickupObject struct {
	Key filmdec.EquipmentLifeKey
	// Appar est l'apparition au sens des socles : position de CRÉATION, classe, vie delta.
	Appar gwPadApparition
	// FamilyID est le mot MPP de 32 bits — l'identité brute de l'arme, celle que l'artefact
	// publie et que `weaponLabels` nomme. `Appar.Family` en est le nom canonique (alias
	// repliés), qui sert de clé de grappe.
	FamilyID uint32
	// Pos est la position de RÉFÉRENCE (dernier point de piste, ou création) ; Moved dit
	// laquelle des deux.
	Pos    [3]float32
	Moved  bool
	Bounds gwPickupBounds
	Picker gwPickupHit
	Status string
}

// gwPickupDateUS rend l'instant retenu de la disparition : celui du passage quand il existe, la
// borne haute sinon (règle du plan : « aucun : `unknown`, date = borne haute »).
//
// IL NE SE PUBLIE PAS À L'ARTEFACT — l'artefact publie l'INTERVALLE [borne basse, borne haute],
// que le recensement mesure. Cette date sert au CYCLE (item 2.4 : l'horloge d'un socle repart au
// ramassage, 24 socles établis sur 57 contre 4 pour l'horloge d'apparition).
func (o gwPickupObject) gwPickupDateUS() uint64 {
	if o.Picker.Found {
		return o.Picker.TUS
	}
	return o.Bounds.HighUS
}

// padObjects retient les créations dont l'identité se résout selon la RÈGLE passée, puis borne
// et date la disparition de chacune. PUR.
//
// LES CRÉATIONS SONT GROUPÉES PAR CLÉ AVANT TOUT : la REPRISE d'une clé (slot, gen) borne la vie
// de la précédente, sans quoi le recensement du suivant prouverait la survie du précédent.
//
// LE SECOND RETOUR EST LE COMPTE DES ÉCARTÉES, MESURÉ SUR LE CHEMIN DE REJET LUI-MÊME (correctif
// de revue du 2026-08-17). Il valait auparavant `Accepted − len(objs)`, une différence : elle
// équilibrait l'invariant `Kept + Rejected == Accepted` par construction, donc le test qui le
// vérifiait ne pouvait pas échouer. Compté ici, l'invariant redevient un CONTRÔLE — un futur
// `continue` qui écarterait une création sans la compter le ferait tomber.
func padObjects(
	scan WorldObjectScan, rule padRule, lives map[uint32][]equipLife,
	positions []filmdec.BipedPosition,
) ([]gwPickupObject, gwRejects) {
	byKey := map[filmdec.EquipmentLifeKey][]filmdec.EquipmentCreation{}
	kept := 0
	var rejected gwRejects
	for _, c := range scan.Creations {
		w, ok := gwPadsIdentity(c)
		// L'OBJET D'OBJECTIF EST ECARTE AVANT LA QUESTION « EST-CE PUBLIABLE ? », et l'ordre
		// est la regle : un drapeau reconnu ne devient pas un socle parce qu'un jour son
		// identifiant entrerait au catalogue d'armes. Il etait deja ecarte hier — mais par
		// ACCIDENT (hors catalogue), c'est-a-dire de la meme facon que le bruit du balayage.
		if ok && rule.Objectives[w] != (Label{}) {
			rejected.total, rejected.objectives = rejected.total+1, rejected.objectives+1
			continue
		}
		if _, named := rule.Family(w); !ok || !named {
			rejected.total++
			continue
		}
		kept++
		k := filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}
		byKey[k] = append(byKey[k], c)
	}
	filmEnd := gwFilmEndUS(scan, positions)
	tracks := gwTracksByKey(scan.Tracks)
	out := make([]gwPickupObject, 0, kept)
	for k, list := range byKey {
		sort.Slice(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i, c := range list {
			lifeEnd := filmEnd
			if i+1 < len(list) {
				lifeEnd = list[i+1].TimestampUS
			}
			w, _ := gwPadsIdentity(c)
			fam, _ := rule.Family(w)
			o := gwPickupObject{Key: k, FamilyID: w, Appar: gwPadApparition{
				Kind: rule.Kind, Family: fam,
				X: c.X, Y: c.Y, Z: c.Z, TUS: c.TimestampUS,
			}}
			o.Appar.Class = gwPadsClass(lives, o.Appar)
			gwPickupResolve(&o, c, lifeEnd, filmEnd, gwResolveInputs{
				kfTimes: scan.Keyframes.TimesUS, seen: scan.Keyframes.SeenUS[k],
				tracks: tracks[k], positions: positions,
			})
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return gwPadsLess(out[i].Appar, out[j].Appar) })
	return out, rejected
}

// gwRejects ventile les créations ÉCARTÉES de la chaîne des socles. `total` est le terme de
// l'invariant `Kept + Rejected == Accepted` ; `objectives` en est le sous-ensemble NOMMÉ — les
// objets que le manifeste du titre identifie (le drapeau), écartés parce qu'on sait ce qu'ils
// sont et non parce qu'on ne les reconnaît pas.
//
// LES DEUX SE COMPTENT SUR LEUR PROPRE CHEMIN DE REJET, jamais par différence : c'est ce qui
// fait de l'invariant un contrôle plutôt qu'une tautologie (correctif de revue du 2026-08-17).
type gwRejects struct {
	total, objectives int
}

// gwResolveInputs porte ce que le bornage et la datation d'UN objet consomment (règle des
// 5 paramètres du dépôt — ces quatre-là voyagent toujours ensemble).
type gwResolveInputs struct {
	kfTimes   []uint64
	seen      []uint64
	tracks    []filmdec.ProjectileTrack
	positions []filmdec.BipedPosition
}

// gwPickupResolve borne la disparition de l'objet puis la date.
//
// IL POSE AUSSI `HasDelta`, et c'est délibéré : la piste de la vie répond à la fois à « l'objet
// a-t-il bougé ? » et à « où est-il quand on le prend ? ». Une seule règle
// (`gwPickupLifeTrack`), un seul endroit qui l'applique.
func gwPickupResolve(
	o *gwPickupObject, c filmdec.EquipmentCreation, lifeEnd, filmEnd uint64, in gwResolveInputs,
) {
	life, moved := gwPickupLifeTrack(in.tracks, c.TimestampUS, lifeEnd)
	o.Appar.HasDelta = moved
	o.Pos, o.Moved = gwPickupRefPos(c, life, moved)
	o.Bounds = gwPickupBoundsFrom(c.TimestampUS, lifeEnd, filmEnd, in.kfTimes,
		gwPickupSeenWithin(in.seen, c.TimestampUS, lifeEnd))
	// DEUX CHEMINS VERS `never`, ET UNE SEULE RAISON : aucune image-clé ne prouve la
	// disparition. `NeverPicked` — l'objet est encore recensé à la dernière ; `NoLaterKF` —
	// il n'y a plus d'image-clé du tout après lui. Le second était ignoré jusqu'au correctif
	// de revue du 2026-08-17 : ces occupations sortaient `dated` ou `unknown` avec la fin du
	// film pour borne haute, donc publiées comme achevées sans qu'aucun recensement le dise.
	if o.Bounds.NeverPicked || o.Bounds.NoLaterKF {
		o.Status = gwPickupStatusNever
		return
	}
	o.Picker = gwPickupNearestPass(o.Pos, o.Bounds.LowUS, o.Bounds.HighUS, in.positions)
	o.Status = gwPickupStatusUnknown
	if o.Picker.Found {
		o.Status = gwPickupStatusDated
	}
}

// gwTracksByKey indexe les pistes delta par vie (slot, gen).
func gwTracksByKey(
	tracks []filmdec.ProjectileTrack,
) map[filmdec.EquipmentLifeKey][]filmdec.ProjectileTrack {
	out := map[filmdec.EquipmentLifeKey][]filmdec.ProjectileTrack{}
	for _, tr := range tracks {
		k := filmdec.EquipmentLifeKey{Slot: tr.Slot, Gen: tr.Gen}
		out[k] = append(out[k], tr)
	}
	return out
}

// gwFilmEndUS rend la fin du film au sens de ce calque : le dernier instant qu'une de ses trois
// sources porte. C'est la borne haute de dernier recours — celle d'un objet créé après la
// dernière image-clé, dont rien ne prouve la disparition.
func gwFilmEndUS(scan WorldObjectScan, positions []filmdec.BipedPosition) uint64 {
	end := scan.Keyframes.LastTimeUS()
	for _, p := range positions {
		if p.TimestampUS > end {
			end = p.TimestampUS
		}
	}
	for _, c := range scan.Creations {
		if c.TimestampUS > end {
			end = c.TimestampUS
		}
	}
	return end
}

// gwPickupPadGaps rend les écarts DISPARITION -> réapparition suivante d'un socle, et le nombre
// de réapparitions dont la disparition précédente n'est pas datée.
//
// CELLES-LÀ NE MESURENT RIEN ET NE SONT PAS REMPLACÉES par l'écart d'apparition : ce serait
// mesurer l'horloge d'apparition sous un autre nom, et c'est précisément celle que la mesure a
// écartée (4 socles établis sur 57, contre 24 depuis le ramassage).
func gwPickupPadGaps(objs []gwPickupObject, members []int) ([]float64, int) {
	ms := append([]int(nil), members...)
	sort.Slice(ms, func(i, j int) bool { return objs[ms[i]].Appar.TUS < objs[ms[j]].Appar.TUS })
	var gaps []float64
	manques := 0
	for i := 0; i+1 < len(ms); i++ {
		prev, next := objs[ms[i]], objs[ms[i+1]]
		if prev.Status != gwPickupStatusDated || prev.Picker.TUS > next.Appar.TUS {
			manques++
			continue
		}
		gaps = append(gaps, float64(next.Appar.TUS-prev.Picker.TUS)/1e6)
	}
	return gaps, manques
}
