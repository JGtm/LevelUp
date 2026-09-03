package mappings

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// loader_replay_labels.go — libelles PROPRES AU REJEU 2D d'un titre
// (config/titles/{slug}/mappings/replay_labels.toml).
//
// Trois catalogues, trois raisons d'etre ici plutot qu'en Go :
//   - les RANGS DE GRENADE etaient nommes a DEUX endroits et differemment (« Dynamo »
//     contre « Shock » pour le meme rang) ;
//   - les CAPACITES etaient en FRANCAIS DANS DU GO, ce qui interdit l'anglais autant
//     qu'un second titre ;
//   - l'EFFET DE TIR etait un catalogue Halo code en dur cote web (22 noms d'armes).
//
// Ce loader ne resout AUCUN nom d'arme : ceux-la vivent dans weapon_names.toml, keyes
// par weapon_key. Ici on ne porte que l'effet, qui n'appartient a aucune langue.

// BilingualLabel — un libelle affichable dans les deux langues du produit. en ET fr sont
// obligatoires ; quand aucun FR officiel n'est connu, le fichier met le EN dans les deux
// (jamais de FR invente — meme regle que weapon_names.toml).
//
// Icon (OPTIONNEL) : stem de la vignette du HUD du jeu sous static/weapons-assets/{slug}/
// (ex. « hud/Frag »). Sans icon, le client garde le libelle — jamais la vignette d'un
// voisin.
type BilingualLabel struct {
	En   string
	Fr   string
	Icon string
}

// AbilityPalette est UNE palette de capacites : les rangs qui la SIGNENT, et les noms
// qu'elle donne a ceux d'entre eux qui sont etablis.
//
// POURQUOI UNE PALETTE ET PAS UNE TABLE UNIQUE : le rang transmis par le film est un index
// dans un groupe de tags choisi A L'EXECUTION. Sur les 46 equipements presents dans au
// moins deux des 12 groupes mesures, 20 CHANGENT de rang — une table unique nommerait faux
// un film sur deux (RECETTE_LOADOUT §13).
type AbilityPalette struct {
	// ID nomme la palette dans les journaux et les rapports. Il ne sort jamais a l'ecran.
	ID string
	// Markers sont les rangs dont l'observation signe cette palette. Ils doivent etre
	// DISJOINTS de ceux des autres palettes du titre — sans quoi rien ne se classe.
	Markers []int
	// Ranks nomme les rangs ETABLIS. Table partielle par nature : un rang absent garde son
	// numero a l'ecran.
	Ranks map[int]BilingualLabel
}

// ReplayLabelSet porte les libelles de rejeu d'un titre.
type ReplayLabelSet struct {
	titleSlug     string
	schemaVersion int
	grenades      []BilingualLabel // index = rang lu dans le film
	palettes      []AbilityPalette
	shotEffects   map[string]string // weapon_key -> famille de rendu
	shotTints     map[string]string // weapon_key -> nature de la decharge
	equipObjects  map[uint32]string // GlobalID de tag `eqip` -> famille de pose
	// objObjects : GlobalID de tag `ti=42` -> objet d'objectif (famille + nom bilingue).
	// Table a part de la precedente : autre archetype, autre chaine d'etablissement
	// (cf. loader_replay_labels_objectives.go).
	objObjects map[uint32]ObjectiveObject
	// flagZone : la regle de retour du drapeau (cf. loader_replay_labels_flagzone.go). Zero quand
	// le titre ne la declare pas — le rejeu ne dessine alors ni cercle ni jauge.
	flagZone FlagReturnZone
}

// replayLabelsTOML — projection brute du fichier.
type replayLabelsTOML struct {
	Meta         metaSection            `toml:"meta"`
	Grenades     []bilingualEntry       `toml:"grenades"`
	Palettes     []abilityPaletteEntry  `toml:"ability_palettes"`
	ShotEffects  map[string]string      `toml:"shot_effects"`
	ShotTints    map[string]string      `toml:"shot_tints"`
	EquipObjects []equipmentObjectEntry `toml:"equipment_objects"`
	ObjObjects   []objectiveObjectEntry `toml:"objective_objects"`
	FlagZone     *flagReturnZoneTOML    `toml:"flag_return_zone"`
}

type abilityPaletteEntry struct {
	ID      string                    `toml:"id"`
	Markers []int                     `toml:"markers"`
	Ranks   map[string]bilingualEntry `toml:"ranks"`
}

type bilingualEntry struct {
	En   string `toml:"en"`
	Fr   string `toml:"fr"`
	Icon string `toml:"icon"`
}

// shotEffectFamilies — les familles de rendu ADMISES. La liste est fermee a dessein :
// une valeur libre ferait tomber l'arme sur le rendu neutre en silence, ce qui est
// indistinguable d'une arme volontairement non cataloguee.
var shotEffectFamilies = map[string]bool{
	"ballistic": true, "plasma": true, "light": true, "shock": true,
	"explosive": true, "melee": true, "needles": true,
}

// shotTintKinds — les NATURES DE DECHARGE admises. Liste fermee pour la meme raison que
// les familles de rendu : une valeur libre ferait tomber l'arme sur la teinte neutre en
// silence, indistinguable d'une arme volontairement non teintee.
//
// Elles nomment une NATURE, jamais une couleur : la couleur vit dans le theme du client
// (tokens dedies), et c'est ce qui permet aux deux themes d'en donner deux valeurs.
var shotTintKinds = map[string]bool{
	"kinetic": true, "plasma_cool": true, "plasma_hot": true, "forerunner": true,
	"electric": true, "needle": true, "blast": true,
}

// GrenadeRanks retourne les libelles de grenade DANS L'ORDRE DES RANGS (copie).
func (s *ReplayLabelSet) GrenadeRanks() []BilingualLabel {
	if s == nil {
		return nil
	}
	out := make([]BilingualLabel, len(s.grenades))
	copy(out, s.grenades)
	return out
}

// AbilityPalettes retourne les palettes de capacites (copie profonde). nil-safe.
func (s *ReplayLabelSet) AbilityPalettes() []AbilityPalette {
	if s == nil {
		return nil
	}
	out := make([]AbilityPalette, 0, len(s.palettes))
	for _, p := range s.palettes {
		cp := AbilityPalette{
			ID:      p.ID,
			Markers: append([]int(nil), p.Markers...),
			Ranks:   make(map[int]BilingualLabel, len(p.Ranks)),
		}
		for k, v := range p.Ranks {
			cp.Ranks[k] = v
		}
		out = append(out, cp)
	}
	return out
}

// ShotEffects retourne la table weapon_key -> famille de rendu (copie).
func (s *ReplayLabelSet) ShotEffects() map[string]string {
	out := make(map[string]string)
	if s == nil {
		return out
	}
	for k, v := range s.shotEffects {
		out[k] = v
	}
	return out
}

// ShotTints retourne la table weapon_key -> nature de la decharge (copie).
func (s *ReplayLabelSet) ShotTints() map[string]string {
	out := make(map[string]string)
	if s == nil {
		return out
	}
	for k, v := range s.shotTints {
		out[k] = v
	}
	return out
}

// EquipmentObjects retourne la table GlobalID de tag `eqip` -> famille de pose (copie).
//
// Table PARTIELLE par nature : le film porte des dizaines d'identifiants (bonus au sol,
// socles, objets de decor) qui partagent l'archetype d'equipement, et seuls ceux dont la
// mesure a etabli la nature y figurent. Un identifiant absent vaut `other`, ce qui se lit a
// l'ecran comme un objet pose non nomme — jamais comme un mur.
func (s *ReplayLabelSet) EquipmentObjects() map[uint32]string {
	out := make(map[uint32]string)
	if s == nil {
		return out
	}
	for k, v := range s.equipObjects {
		out[k] = v
	}
	return out
}

// ObjectiveObjects retourne la table GlobalID de tag `ti=42` -> objet d'objectif (copie).
//
// Table PARTIELLE par nature, et le plus souvent MINUSCULE : elle ne porte que les objets du
// monde dont la nature EST le mode de jeu et dont l'identifiant a ete etabli. Un identifiant
// absent n'est pas un objet d'objectif — c'est ce que la chaine des socles lit pour savoir ce
// qu'elle doit refuser de publier comme arme.
func (s *ReplayLabelSet) ObjectiveObjects() map[uint32]ObjectiveObject {
	out := make(map[uint32]ObjectiveObject)
	if s == nil {
		return out
	}
	for k, v := range s.objObjects {
		out[k] = v
	}
	return out
}

// FlagReturnZone retourne la regle de retour du drapeau. Zero quand le titre ne la declare pas :
// l'appelant ne dessine alors ni zone ni jauge (cf. loader_replay_labels_flagzone.go).
func (s *ReplayLabelSet) FlagReturnZone() FlagReturnZone {
	if s == nil {
		return FlagReturnZone{}
	}
	return s.flagZone
}

// TitleSlug retourne le slug declare.
func (s *ReplayLabelSet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schema declaree.
func (s *ReplayLabelSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadReplayLabelsFromFile lit et valide un replay_labels.toml a un chemin donne.
func LoadReplayLabelsFromFile(path string) (*ReplayLabelSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadReplayLabelsFromBytes(path, raw)
}

// LoadReplayLabelsFromBytes parse et valide un payload TOML deja en memoire.
//
// Validation STRICTE (tout-ou-rien) : un titre dont le fichier est incoherent ne doit pas
// produire un rejeu a moitie nomme — un libelle manquant se lit comme une donnee absente,
// alors que c'est une erreur de configuration.
func LoadReplayLabelsFromBytes(path string, raw []byte) (*ReplayLabelSet, error) {
	var doc replayLabelsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	grenades, err := parseGrenadeRanks(path, doc.Grenades)
	if err != nil {
		return nil, err
	}
	palettes, err := parseAbilityPalettes(path, doc.Palettes)
	if err != nil {
		return nil, err
	}
	effects, err := parseShotEffects(path, doc.ShotEffects)
	if err != nil {
		return nil, err
	}
	tints, err := parseShotTints(path, doc.ShotTints)
	if err != nil {
		return nil, err
	}
	equip, err := parseEquipmentObjects(path, doc.EquipObjects)
	if err != nil {
		return nil, err
	}
	objs, err := parseObjectiveObjects(path, doc.ObjObjects)
	if err != nil {
		return nil, err
	}
	zone, err := parseFlagReturnZone(path, doc.FlagZone)
	if err != nil {
		return nil, err
	}
	return &ReplayLabelSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		grenades:      grenades,
		palettes:      palettes,
		shotEffects:   effects,
		shotTints:     tints,
		equipObjects:  equip,
		objObjects:    objs,
		flagZone:      zone,
	}, nil
}

// parseGrenadeRanks valide les rangs : l'ORDRE est la donnee, un trou la detruirait.
func parseGrenadeRanks(path string, rows []bilingualEntry) ([]BilingualLabel, error) {
	out := make([]BilingualLabel, 0, len(rows))
	for i, e := range rows {
		lbl, err := bilingual(path, fmt.Sprintf("grenade rang %d", i), e)
		if err != nil {
			return nil, err
		}
		out = append(out, lbl)
	}
	return out, nil
}

// parseAbilityPalettes valide les palettes de capacites.
//
// TROIS INVARIANTS, tous FATAUX — un titre dont les palettes sont incoherentes ne doit pas
// produire un rejeu a moitie nomme : un identifiant vide ou duplique rendrait le journal
// illisible, une palette sans marqueur ne pourrait JAMAIS etre reconnue (elle serait du
// code mort en donnee), et deux palettes partageant un marqueur rendraient le classement
// ambigu sur tout film qui le montre — c'est-a-dire faux.
func parseAbilityPalettes(path string, rows []abilityPaletteEntry) ([]AbilityPalette, error) {
	out := make([]AbilityPalette, 0, len(rows))
	seenID := map[string]bool{}
	markerOwner := map[int]string{}
	for _, e := range rows {
		id := strings.TrimSpace(e.ID)
		if id == "" {
			return nil, fmt.Errorf("%s: palette de capacités sans id", path)
		}
		if seenID[id] {
			return nil, fmt.Errorf("%s: palette de capacités %q déclarée deux fois", path, id)
		}
		seenID[id] = true
		if len(e.Markers) == 0 {
			return nil, fmt.Errorf("%s: palette %q sans marqueur : aucun film ne pourrait la reconnaître", path, id)
		}
		for _, m := range e.Markers {
			if owner, taken := markerOwner[m]; taken {
				return nil, fmt.Errorf("%s: le rang %d marque à la fois %q et %q — le classement serait ambigu",
					path, m, owner, id)
			}
			markerOwner[m] = id
		}
		ranks, err := parseAbilityRanks(path, id, e.Ranks)
		if err != nil {
			return nil, err
		}
		out = append(out, AbilityPalette{ID: id, Markers: e.Markers, Ranks: ranks})
	}
	return out, nil
}

// parseAbilityRanks valide les noms d'une palette. Les cles sont des rangs NUMERIQUES lus
// dans le film : une cle non numerique ne designerait aucune capacite.
func parseAbilityRanks(path, palette string, rows map[string]bilingualEntry) (map[int]BilingualLabel, error) {
	out := make(map[int]BilingualLabel, len(rows))
	for rawKey, e := range rows {
		rank, err := strconv.Atoi(strings.TrimSpace(rawKey))
		if err != nil {
			return nil, fmt.Errorf("%s: rang de capacité %q non numérique (palette %q)", path, rawKey, palette)
		}
		lbl, err := bilingual(path, fmt.Sprintf("capacité %d de la palette %q", rank, palette), e)
		if err != nil {
			return nil, err
		}
		out[rank] = lbl
	}
	return out, nil
}

// parseShotEffects valide les familles de rendu contre la liste fermee.
func parseShotEffects(path string, rows map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(rows))
	for rawKey, rawFam := range rows {
		key := strings.TrimSpace(rawKey)
		fam := strings.TrimSpace(rawFam)
		if key == "" {
			return nil, fmt.Errorf("%s: weapon_key vide dans [shot_effects]", path)
		}
		if !shotEffectFamilies[fam] {
			return nil, fmt.Errorf("%s: effet %q inconnu pour %q (admis : ballistic, plasma, light, shock, explosive, melee, needles)",
				path, fam, key)
		}
		out[key] = fam
	}
	return out, nil
}

// parseShotTints valide les natures de decharge contre la liste fermee. Table OPTIONNELLE :
// un titre sans [shot_tints] rend des tirs a la teinte neutre, ce qui est un rendu entier.
func parseShotTints(path string, rows map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(rows))
	for rawKey, rawKind := range rows {
		key := strings.TrimSpace(rawKey)
		kind := strings.TrimSpace(rawKind)
		if key == "" {
			return nil, fmt.Errorf("%s: weapon_key vide dans [shot_tints]", path)
		}
		if !shotTintKinds[kind] {
			return nil, fmt.Errorf("%s: teinte %q inconnue pour %q (admises : kinetic, plasma_cool, plasma_hot, forerunner, electric, needle, blast)",
				path, kind, key)
		}
		out[key] = kind
	}
	return out, nil
}

// bilingual valide qu'un libelle porte bien ses deux langues. L'icone reste optionnelle.
func bilingual(path, what string, e bilingualEntry) (BilingualLabel, error) {
	en := strings.TrimSpace(e.En)
	fr := strings.TrimSpace(e.Fr)
	if en == "" {
		return BilingualLabel{}, fmt.Errorf("%s: %s sans en (nom EN obligatoire)", path, what)
	}
	if fr == "" {
		return BilingualLabel{}, fmt.Errorf("%s: %s sans fr (mettre le EN si aucun FR officiel)", path, what)
	}
	return BilingualLabel{En: en, Fr: fr, Icon: strings.TrimSpace(e.Icon)}, nil
}

// tagGlobalID32 lit un GlobalID de tag du jeu, tel que les manifestes l'ecrivent : un entier
// hexadecimal de 32 bits, prefixe `0x` ou non.
//
// POURQUOI UN HELPER. Le motif etait ecrit deux fois dans la table des objets d'equipement
// (l'identifiant et l'identifiant de chaine) ; la table des objets d'objectif en faisait la
// TROISIEME, c'est-a-dire la limite que le depot fixe. `quoi` nomme le champ dans le message,
// pour qu'une erreur de manifeste dise laquelle des trois cles est illisible.
func tagGlobalID32(path, raw, quoi string) (uint32, error) {
	v, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X"), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: %s %q illisible (attendu : GlobalID hexadécimal 32 bits, ex. \"0x8e2dc574\")",
			path, quoi, raw)
	}
	return uint32(v), nil
}
