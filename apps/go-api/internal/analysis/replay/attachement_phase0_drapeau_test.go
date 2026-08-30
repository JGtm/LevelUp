package replay

// attachement_phase0_drapeau_test.go — ITEM 0.2, SECOND VOLET : LE DRAPEAU EN TANT QU'OBJET.
//
// LA PISTE VIENT DU LOT « ARMES AU SOL », ET ELLE EST PRÉCISE. Les records de CRÉATION
// `ti=42` portent, dans leur bloc `object-multiplayer-properties`, un mot de 32 bits qui est
// le GlobalID du tag de l'objet ; le croisement d'identité l'a validé pour les ARMES, et il
// ÉCARTE tout ce qui n'est pas au catalogue d'armes. Le drapeau, objet porté qui n'est dans
// aucun catalogue d'armes, doit être exactement là : parmi les écartées.
//
// LES TROIS QUESTIONS SONT POSÉES DANS CET ORDRE, et chacune est une réfutation possible.
// COMBIEN d'écartées ? OÙ naissent-elles (distance au `flag_spawn` de la carte — un drapeau
// naît à son socle) ? QUAND (écart aux prises et aux fins de portage de l'oracle — un drapeau
// lâché hors zone renaît à son socle) ? Une écartée qui naît loin des socles et jamais près
// d'un événement de drapeau n'est pas le drapeau, et le dire coûte trois distances.
//
// LE QUATRIÈME VOLET EST LA JONCTION AVEC i10 : si l'une de ces entités est le drapeau, son
// i10 doit passer à porte ouverte pendant les fenêtres de portage et rester fermé en dehors.
// Les deux taux sont publiés ensemble — un taux de portage seul ne prouve rien.

import (
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// attDrapeauRayonM : au-delà, une création n'est plus « au socle ». Trois mètres est la
// tolérance déjà employée par la chaîne des poses pour rattacher un objet à son poseur.
const attDrapeauRayonM = 3.0

// TestAttachementPhase0DrapeauObjet — ITEM 0.2, second volet.
func TestAttachementPhase0DrapeauObjet(t *testing.T) {
	root := attRequireRoot(t)
	joues := 0
	for _, id := range objCTFFilms {
		if _, ok := objOpenFilm(t, root, id); !ok {
			continue
		}
		joues++
		attDrapeauObjetFilm(t, root, id)
	}
	if joues == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", attFilmEnv, root)
	}
}

// attDrapeauObjetFilm mesure UN film.
func attDrapeauObjetFilm(t *testing.T, root, id string) {
	t.Helper()
	cre, socles, ok := attCreationsEcartees(t, root, id, "flag_spawn")
	if !ok {
		return
	}
	o := attOracleCTF(t, root, id)
	t.Logf("%s : %d créations ti=42 ÉCARTÉES du catalogue d'armes · %d socles `flag_spawn` "+
		"au catalogue de carte · %d fenêtres de portage",
		id, len(cre.ecartees), len(socles), len(o.Fenetres))
	t.Logf("%s : dénominateurs du balayage — %d ancres, %d acceptées, %d au catalogue d'armes, "+
		"%d écartées, %d mots de 32 bits distincts parmi les écartées",
		id, cre.st.Anchors, cre.st.Accepted, len(cre.connues), len(cre.ecartees), len(cre.mots))
	attLogDistances(t, id, cre.ecartees, socles, o)
	attLogI10SurEcartees(t, root, id, cre.ecartees, o)
}

// attCreations porte le tri d'un balayage de créations `ti=42`.
type attCreations struct {
	connues, ecartees []filmdec.EquipmentCreation
	mots              map[uint32]int
	st                filmdec.EquipmentCreationStats
}

// attCreationsEcartees balaye les créations `ti=42` d'un film et les trie par le croisement
// d'identité (mot de 32 bits du bloc MPP contre le catalogue d'armes).
//
// LE RÔLE DE SOCLE EST UN PARAMÈTRE DEPUIS D4 (2026-08-27) : la recette est la même pour tout
// objet d'objectif PORTÉ — le drapeau naît à son `flag_spawn`, le crâne à son `oddball_spawn`.
// Seul change le rôle interrogé au catalogue de carte. Une seconde copie de ce balayage aurait
// divergé au premier correctif, et le dépôt l'interdit.
func attCreationsEcartees(t *testing.T, root, id, roleSocle string) (
	attCreations, []PointObjective, bool) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, _, ok := attBornes(t, root, id)
	if !ok {
		t.Logf("%s : bornes de carte indisponibles — volet objet non mesurable sur ce film", id)
		return attCreations{}, nil, false
	}
	cres, st, err := filmdec.ScanFilmGroundWeaponCreations(objChunkDir(root, id), &wr)
	if err != nil {
		t.Logf("%s : balayage des créations ti=42 : %v", id, err)
		return attCreations{}, nil, false
	}
	cat := loadoutFamilies()
	out := attCreations{mots: map[uint32]int{}, st: st}
	for _, c := range cres {
		if !c.MPPPresent[filmdec.MPPWord32] {
			continue
		}
		mot := uint32(c.MPPVal[filmdec.MPPWord32])
		if cat[mot] {
			out.connues = append(out.connues, c)
			continue
		}
		out.ecartees = append(out.ecartees, c)
		out.mots[mot]++
	}
	return out, attMarqueurs(t, root, id, roleSocle), true
}

// attLogDistances publie, pour chaque mot de 32 bits écarté, sa distance minimale à un socle
// et son écart temporel minimal à un événement de drapeau.
func attLogDistances(t *testing.T, id string, ecartees []filmdec.EquipmentCreation,
	socles []PointObjective, o attOracle) {
	t.Helper()
	parMot := map[uint32][]filmdec.EquipmentCreation{}
	for _, c := range ecartees {
		m := uint32(c.MPPVal[filmdec.MPPWord32])
		parMot[m] = append(parMot[m], c)
	}
	mots := make([]uint32, 0, len(parMot))
	for m := range parMot {
		mots = append(mots, m)
	}
	sort.Slice(mots, func(i, j int) bool {
		if len(parMot[mots[i]]) != len(parMot[mots[j]]) {
			return len(parMot[mots[i]]) > len(parMot[mots[j]])
		}
		return mots[i] < mots[j]
	})
	for i, m := range mots {
		if i >= 10 {
			t.Logf("%s :   ... %d autres mots écartés", id, len(mots)-10)
			break
		}
		auSocle, dMin, tMin := attResumeMot(parMot[m], socles, o)
		t.Logf("%s :   mot 0x%08X — %d créations, %d à moins de %.0f m d'un socle "+
			"(distance min %.1f m), écart temporel min à un événement de drapeau %d ms",
			id, m, len(parMot[m]), auSocle, attDrapeauRayonM, dMin, tMin)
	}
}

// attResumeMot rend, pour les créations d'un même mot : combien naissent au socle, la
// distance minimale à un socle, et l'écart temporel minimal à un événement de drapeau.
func attResumeMot(cs []filmdec.EquipmentCreation, socles []PointObjective, o attOracle) (
	auSocle int, dMin float64, tMin int64) {
	dMin, tMin = math.MaxFloat64, int64(math.MaxInt64)
	for _, c := range cs {
		d := attDistSocleMin(c, socles)
		if d < dMin {
			dMin = d
		}
		if d <= attDrapeauRayonM {
			auSocle++
		}
		if e := attEcartEvenementMin(c, o); e < tMin {
			tMin = e
		}
	}
	return auSocle, dMin, tMin
}

// attDistSocleMin rend la distance horizontale minimale d'une création à un socle.
//
// LA DISTANCE EST PRISE EN PLAN, et c'est délibéré : la hauteur d'un socle du fichier de
// carte et celle de l'objet répliqué ne se réfèrent pas au même point (pied contre centre),
// et mêler les deux ferait passer un écart de convention pour un écart de position.
func attDistSocleMin(c filmdec.EquipmentCreation, socles []PointObjective) float64 {
	best := math.MaxFloat64
	for _, s := range socles {
		if d := math.Hypot(float64(c.X)-float64(s.Center.X), float64(c.Y)-float64(s.Center.Y)); d < best {
			best = d
		}
	}
	return best
}

// attEcartEvenementMin rend l'écart temporel minimal (ms) entre une création et un événement
// de l'oracle — prise ou fin de portage.
func attEcartEvenementMin(c filmdec.EquipmentCreation, o attOracle) int64 {
	at := int64(c.TimestampUS/1000) - o.Bridge.OffsetMS
	best := int64(math.MaxInt64)
	for _, w := range o.Fenetres {
		for _, v := range []int64{w.T0, w.T1} {
			if d := attAbs(at - v); d < best {
				best = d
			}
		}
	}
	return best
}

// attLogI10SurEcartees confronte les lectures d'i10 portées par les SLOTS des créations
// écartées à la frontière de portage de l'oracle.
func attLogI10SurEcartees(t *testing.T, root, id string,
	ecartees []filmdec.EquipmentCreation, o attOracle) {
	t.Helper()
	slots := map[uint32]bool{}
	for _, c := range ecartees {
		slots[c.Slot] = true
	}
	lectures, _ := attScanOf(t, root, id)
	parXUID := map[uint64][]objWindow{}
	for _, w := range o.Fenetres {
		parXUID[w.XUID] = append(parXUID[w.XUID], w)
	}
	var dedans, dehors, ouvDedans, ouvDehors int
	for _, l := range lectures {
		if l.TI != uint32(filmdec.GroundWeaponTypeIndex) || !slots[l.Slot] {
			continue
		}
		if attPorteEnCours(parXUID, attMatchMS(l, o.Bridge)) {
			dedans++
			if l.St.Attached {
				ouvDedans++
			}
			continue
		}
		dehors++
		if l.St.Attached {
			ouvDehors++
		}
	}
	t.Logf("%s : i10 sur les %d slots des créations écartées — pendant un portage %d lectures "+
		"dont %d à porte ouverte (%.1f %%) ; hors portage %d lectures dont %d ouvertes (%.1f %%)",
		id, len(slots), dedans, ouvDedans, 100*attPart(ouvDedans, dedans),
		dehors, ouvDehors, 100*attPart(ouvDehors, dehors))
}

// attPorteEnCours dit si UN joueur quelconque portait le drapeau à cet instant. Le drapeau
// est un objet unique : la question posée à son sujet n'est pas « qui » mais « quelqu'un ».
func attPorteEnCours(parXUID map[uint64][]objWindow, at int64) bool {
	for _, ws := range parXUID {
		if objDansFenetre(ws, at) {
			return true
		}
	}
	return false
}
