package replay

import (
	"fmt"
	"log/slog"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// equipment_placements.go — LES POSES d'équipement sur la carte : le mur de protection, le
// capteur de menaces, et tout ce que l'archétype d'équipement porte sans qu'on le nomme.
//
// D'OÙ VIENT LA DONNÉE (2026-08-17/18, PLAN_IDENTITE_TI37 gates 0-1 puis
// PLAN_POSES_EQUIPEMENT_PUBLICATION) : le record de CRÉATION d'une entité `ti=37` porte, dans
// son bloc `object-multiplayer-properties`, le GlobalID du tag `eqip` de l'objet — les
// 21 valeurs du corpus se résolvent toutes dans le groupe `eqip` du jeu. Le MÊME record porte
// la position i0, c'est-à-dire le lieu exact de la pose. La fin de vie vient de la trajectoire
// décodée des paquets delta. `filmdec.ScanFilmEquipmentPlacements` rend le tout.
//
// CE QUE CETTE COUCHE AJOUTE, et pourquoi c'est ici : le POSEUR et son CAP. Ni l'un ni l'autre
// n'est écrit dans le record — le champ de référence d'entité du default-state est une porte
// FERMÉE sur 503 records sur 503. Le poseur se MESURE : c'est le bipède le plus proche à
// l'instant de la pose. La mesure du corpus le justifie et le borne — médiane 0,52 à 0,60 m
// sur 11 films, contre 11 à 36 m pour le témoin (un autre bipède vivant au même instant).
// Le cap est celui de la VISÉE de ce poseur au même instant (i21, déjà décodé par CaptureDirs).
//
// CE QU'ON NE PRÉTEND PAS. `poseHeading` n'est PAS l'orientation de l'objet : le record n'en
// porte aucune. C'est là où le poseur REGARDAIT quand il a posé — une mesure, dont le rendu
// peut se servir pour orienter un mur, en sachant ce qu'elle est.

// equipOwnerWindowUS est la fenêtre dans laquelle un échantillon de bipède est jugé
// contemporain de la pose : 250 ms, la même borne que celle qui sépare deux vies de projectile.
const equipOwnerWindowUS = 250_000

// equipOwnerMaxDist est la distance MAXIMALE, en mètres, entre une pose et son poseur. Le
// seuil est énoncé avant la mesure (plan, décision 2) et la mesure le confirme largement :
// médiane 0,56 m, p90 0,73 à 0,86 m sur les films d'arène. Au-delà de 3 m, la proximité ne
// veut plus rien dire — c'est le cas des objets du monde (bonus, socles), qui n'ont pas de
// poseur et ne doivent pas s'en voir attribuer un.
const equipOwnerMaxDist = 3.0

// equipHeadingWindowUS borne l'écart entre la pose et la lecture de visée qui lui donne son
// cap. 200 ms : plus court que la fenêtre du poseur, parce qu'un cap vieilli est faux là où
// une position vieillie reste juste (on tourne plus vite qu'on ne se déplace).
const equipHeadingWindowUS = 200_000

// EquipmentPlacement est UNE pose d'équipement, datée et située.
type EquipmentPlacement struct {
	// T0 / T1 bornent la présence de l'objet sur le même axe que Point.T : de la création
	// (le geste) à la disparition (le dernier point de la vie décodée).
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// X / Y : la position de la pose, en coordonnées monde (mêmes axes que Point.X/Y).
	X float32 `json:"x"`
	Y float32 `json:"y"`
	// Z : l'altitude de la pose. Gratuite (le même record la porte), publiée pour la
	// cohérence d'étage. PIÈGE omitempty accepté, même argument que GrappleLine.AZ.
	Z float32 `json:"z,omitempty"`
	// Family est la famille de rendu : `wall`, `sensor`, ou `other`. Identifiant STABLE du
	// document (même règle que NeutralDeath.Kind) — les libellés vivent dans l'i18n du
	// client, jamais ici. `other` est un RÉSULTAT, pas un défaut d'analyse : l'objet est
	// bien posé, sa nature n'est pas établie.
	Family string `json:"family"`
	// ID est le GlobalID du tag `eqip`, en hexadécimal. Publié même pour `other` : c'est
	// l'identité que le jeu donne à l'objet, et c'est ce qui permettra de nommer plus tard
	// sans re-cuire les artefacts (le client peut regrouper par identifiant).
	ID string `json:"id"`
	// Owner est le SLOT du poseur — donc une VIE, pas un joueur (même règle que les autres
	// calques : le slot migre aux réapparitions). Vaut -1 quand aucun bipède contemporain
	// n'est assez proche : c'est le cas des objets du monde, et c'est une mesure.
	Owner int `json:"owner"`
	// H est le cap de VISÉE du poseur à l'instant de la pose, en degrés [0,360[ — même
	// convention que Point.H. Absent quand aucune lecture de visée n'est contemporaine, ou
	// quand la pose n'a pas de poseur. POINTEUR, PAS float32 : un cap de zéro est une
	// valeur, et omitempty l'effacerait (même piège qu'OriginMs).
	H *float32 `json:"h,omitempty"`
}

// EquipmentPlacementCoverage dit ce que le calque a lu et ce qu'il en a publié.
//
// LA CALIBRATION Y FIGURE, et c'est le point : la largeur du bloc `object-multiplayer-
// properties` varie d'un film à l'autre et se MESURE dans le film. Publier les poses sans
// publier la mesure qui les rend lisibles laisserait croire qu'elles tombent d'une constante.
type EquipmentPlacementCoverage struct {
	// Widths est le découpage retenu (« lead/index »), vide si la calibration a échoué.
	Widths string `json:"widths,omitempty"`
	// Calibrated dit si le film a tranché. Faux : aucune pose n'est publiée, et c'est
	// délibéré — un balayage à la mauvaise largeur ne rend pas des poses, il rend du bruit.
	Calibrated bool `json:"calibrated"`
	// Lives est le nombre de vies d'objet d'équipement décodées ; Anchors le nombre
	// d'en-têtes de création reconnus (bruit compris) ; Confirmed ceux que l'oracle de
	// position a validés. L'écart entre les trois dit la sélectivité réelle.
	Lives     int `json:"lives"`
	Anchors   int `json:"anchors"`
	Confirmed int `json:"confirmed"`
	// Placements est le nombre de poses publiées ; Named celles dont la famille est établie
	// (wall ou sensor) ; Other les autres.
	Placements int `json:"placements"`
	Named      int `json:"named"`
	Other      int `json:"other"`
	// WithOwner / WithHeading : poses dont le poseur, puis le cap, ont été mesurés.
	WithOwner   int `json:"withOwner"`
	WithHeading int `json:"withHeading"`
	// ByFamily compte les poses par famille — le détail que `Named` résume.
	ByFamily map[string]int `json:"byFamily,omitempty"`
}

// equipmentFamilyOther est la famille par défaut : un objet dont la nature n'est pas établie.
const equipmentFamilyOther = "other"

// decodeFilmPlacements décode les poses du film et JOURNALISE ce qu'il en est.
//
// TROIS SORTIES, TROIS PHRASES — et la distinction est le point. Un balayage impossible (film
// illisible) n'est pas un film sans équipement, et un film dont le découpage du bloc de
// réplication n'a pas été tranché n'est ni l'un ni l'autre : il aurait des poses, mais les lire
// à une largeur devinée rendrait du bruit. Les trois se lisent au journal, et la troisième se
// relit ensuite dans l'artefact (`coverage.placements.calibrated`).
//
// HORS LIGNE — appelée par BuildFromFilm, sous LockProcessDecode.
func decodeFilmPlacements(
	filmDir string, worldRange *filmdec.Vec3Range,
) ([]filmdec.EquipmentPlacement, filmdec.EquipmentPlacementStats) {
	pl, st, err := filmdec.ScanFilmEquipmentPlacements(filmDir, worldRange)
	switch {
	case err != nil:
		slog.Warn("poses d'equipement illisibles — rejeu sans equipement pose",
			"err", err, "filmDir", filmDir)
		return nil, st
	case !st.Calibration.Widths.Valid():
		slog.Warn("poses d'equipement : le decoupage du bloc de replication n'a pas ete tranche"+
			" sur ce film — AUCUNE pose publiee plutot que du bruit",
			"filmDir", filmDir, "ancres", st.Calibration.Anchors,
			"vies", st.Calibration.Lives, "chunksLus", st.Calibration.Chunks)
	default:
		slog.Info("poses d'equipement : records de creation ti=37",
			"decoupage", st.Calibration.Widths.String(), "accords", st.Calibration.Agree,
			"ancres", st.Anchors, "acceptes", st.Accepted,
			"confirmes", st.Confirmed, "poses", st.Placements)
	}
	return pl, st
}

// logPlacementCoverage publie au journal ce que le calque a rendu — les mêmes dénominateurs
// que l'artefact, pour qu'un build se juge sans ouvrir le JSON.
func logPlacementCoverage(c *EquipmentPlacementCoverage) {
	if c == nil {
		return
	}
	slog.Info("rejeu : poses d'equipement",
		"calibre", c.Calibrated, "decoupage", c.Widths, "poses", c.Placements,
		"nommees", c.Named, "autres", c.Other,
		"avecPoseur", c.WithOwner, "avecCap", c.WithHeading)
}

// buildEquipmentPlacements assemble les poses : famille par le manifeste, poseur par
// proximité mesurée, cap par la visée du poseur.
//
// `positions` doit être TRIÉ par instant (c'est le cas de `sorted` dans BuildFromFilm) : la
// recherche du poseur est une fenêtre glissante, pas un balayage complet par pose.
func buildEquipmentPlacements(
	raw []filmdec.EquipmentPlacement, st filmdec.EquipmentPlacementStats,
	positions []filmdec.BipedPosition, clock replayClock,
) ([]EquipmentPlacement, *EquipmentPlacementCoverage) {
	cov := &EquipmentPlacementCoverage{
		Calibrated: st.Calibration.Widths.Valid(),
		Lives:      st.Lives,
		Anchors:    st.Anchors,
		Confirmed:  st.Confirmed,
		ByFamily:   map[string]int{},
	}
	if cov.Calibrated {
		cov.Widths = st.Calibration.Widths.String()
	}
	if len(raw) == 0 || clock.step == 0 {
		return nil, cov
	}
	out := make([]EquipmentPlacement, 0, len(raw))
	for _, p := range raw {
		t0 := frameOf(p.T0US, clock.origin, clock.step)
		t1 := frameOf(p.T1US, clock.origin, clock.step)
		if t1 < 0 || t0 >= clock.frames {
			continue // hors de l'axe publié : rien à dessiner
		}
		pl := EquipmentPlacement{
			T0: clampFrame(t0, clock.frames), T1: clampFrame(t1, clock.frames),
			X: p.X, Y: p.Y, Z: p.Z,
			Family: clock.families[p.GlobalID],
			ID:     fmt.Sprintf("0x%08x", p.GlobalID),
			Owner:  -1,
		}
		if pl.Family == "" {
			pl.Family = equipmentFamilyOther
		}
		if slot, h, ok := equipmentOwner(positions, p); ok {
			pl.Owner, pl.H = int(slot), h
		}
		out = append(out, pl)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].ID < out[j].ID
	})
	tallyEquipmentPlacements(out, cov)
	return out, cov
}

// replayClock porte l'axe de temps et la table de familles du document (règle des
// 5 paramètres — ces quatre-là voyagent toujours ensemble).
type replayClock struct {
	origin, step uint64
	frames       int
	families     map[uint32]string
}

func clampFrame(t, frames int) int {
	switch {
	case t < 0:
		return 0
	case t >= frames:
		return frames - 1
	}
	return t
}

func tallyEquipmentPlacements(out []EquipmentPlacement, cov *EquipmentPlacementCoverage) {
	cov.Placements = len(out)
	for _, p := range out {
		cov.ByFamily[p.Family]++
		if p.Family == equipmentFamilyOther {
			cov.Other++
		} else {
			cov.Named++
		}
		if p.Owner >= 0 {
			cov.WithOwner++
		}
		if p.H != nil {
			cov.WithHeading++
		}
	}
}

// equipmentOwner rend le bipède le plus proche de la pose dans la fenêtre temporelle, à
// condition qu'il soit à moins d'equipOwnerMaxDist mètres. Rend aussi, quand elle existe, la
// lecture de VISÉE la plus proche en temps du même slot — c'est elle qui porte le cap.
//
// UN ÉCHANTILLON PAR SLOT, LE PLUS PROCHE EN TEMPS : plusieurs records d'un même bipède
// tombent dans la fenêtre, et retenir le plus proche en ESPACE au lieu du plus proche en
// TEMPS ferait gagner le joueur qui passe par là au bon moment plutôt que celui qui pose.
func equipmentOwner(
	positions []filmdec.BipedPosition, p filmdec.EquipmentPlacement,
) (slot uint32, heading *float32, ok bool) {
	lo := sort.Search(len(positions), func(k int) bool {
		return positions[k].TimestampUS+equipOwnerWindowUS >= p.T0US
	})
	best := map[uint32]filmdec.BipedPosition{}
	aim := map[uint32]filmdec.BipedPosition{}
	for k := lo; k < len(positions) && positions[k].TimestampUS <= p.T0US+equipOwnerWindowUS; k++ {
		s := positions[k]
		if !s.HasWorld {
			continue // sans bornes de carte, la distance n'est pas une distance
		}
		if b, seen := best[s.Slot]; !seen || equipCloser(s, b, p.T0US) {
			best[s.Slot] = s
		}
		if !s.HasYaw || equipTimeGap(s.TimestampUS, p.T0US) > equipHeadingWindowUS {
			continue
		}
		if b, seen := aim[s.Slot]; !seen || equipCloser(s, b, p.T0US) {
			aim[s.Slot] = s
		}
	}
	var near filmdec.BipedPosition
	for _, s := range best {
		if d := equipDist(p, s); d <= equipOwnerMaxDist && (!ok || d < equipDist(p, near)) {
			near, ok = s, true
		}
	}
	if !ok {
		return 0, nil, false
	}
	// Le CAP vient de la lecture de visée du MÊME slot la plus proche en temps. Jamais d'un
	// autre slot, et jamais d'une lecture trop vieille : on tourne plus vite qu'on ne marche.
	if a, seen := aim[near.Slot]; seen {
		if h, valid := a.AimHeadingDeg(); valid {
			heading = &h
		}
	}
	return near.Slot, heading, true
}

// equipCloser dit si a est plus proche de `at` en TEMPS que b.
func equipCloser(a, b filmdec.BipedPosition, at uint64) bool {
	return equipTimeGap(a.TimestampUS, at) < equipTimeGap(b.TimestampUS, at)
}

func equipDist(p filmdec.EquipmentPlacement, s filmdec.BipedPosition) float32 {
	dx, dy, dz := p.X-s.X, p.Y-s.Y, p.Z-s.Z
	return float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
}

func equipTimeGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}
