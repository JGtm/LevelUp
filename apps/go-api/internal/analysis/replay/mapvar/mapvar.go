// Package mapvar — mapvar.go : grammaire métier d'une variante de carte (.mvar).
//
// Grammaire ÉTABLIE sur pièces (cf. tests sur fixtures breaker/cliffhanger) :
//
//	root[1]                  en-tête de carte (root[1][0][0] = identifiant de niveau)
//	root[3]                  liste des objets placés
//	root[6]                  tables de regroupement (non exploitées ici)
//	root[10][1]              table des chaînes lisibles (paires nom d'instance / préfab)
//	root[11]                 surcharges de propriétés indexées (non exploitées ici)
//
// Par objet (root[3][i]) :
//
//	[2][0]     type_id int32  (identifiant de type d'objet Forge)
//	[3]        position  vec3 (repère monde, mètres — même repère que les positions joueur)
//	[4]        vecteur up
//	[5]        vecteur forward
//	[7]        flags uint8
//	[8]        sac de propriétés — voir gameplayBag ci-dessous
//	[10][0]    identifiant d'instance int32
//
// Sac de propriétés #8, sous-arbre gameplay ([0] → liste de 1 struct) :
//
//	[1]        catégorie int
//	[8]        index d'équipe (0 / 1) — ABSENT si l'objet n'est pas d'équipe
//	[9]        liste de labels ; chaque entrée est un struct{0: hash int32}
//
// Le hash de label est un murmur3_x86_32(seed=0) du nom snake_case du label
// (CRAQUÉ : murmur3("stockpile_socket") == 2110778921, valeur observée telle quelle
// dans ctf_breaker.mvar). Voir objectives.go pour la table de résolution.
package mapvar

import "fmt"

// Vec3 est une position ou une direction en repère monde.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// TeamUnset marque un objet sans index d'équipe.
const TeamUnset = -1

// Object est un objet placé par la variante de carte.
type Object struct {
	Index      int
	TypeID     int32
	Pos        Vec3
	Up         Vec3
	Forward    Vec3
	Flags      uint8
	Category   int
	TeamIndex  int
	Labels     []int32
	InstanceID int32
}

// Variant est une variante de carte décodée.
type Variant struct {
	Objects []Object
	Names   []string // root[10][1] : chaînes lisibles, ordre du fichier
	LevelID int32    // root[1][0][0]
}

// Parse décode un .mvar complet.
func Parse(buf []byte) (*Variant, error) {
	root, err := DecodeRoot(buf)
	if err != nil {
		return nil, fmt.Errorf("mapvar: %w", err)
	}
	v := &Variant{}
	if hdr, ok := root.Field(1); ok {
		if sub, ok2 := hdr.Field(0); ok2 {
			if id, ok3 := sub.Field(0); ok3 {
				v.LevelID = int32(id.Int)
			}
		}
	}
	if names, ok := root.Field(10); ok {
		if lst, ok2 := names.Field(1); ok2 {
			for _, it := range lst.Items {
				v.Names = append(v.Names, it.Str)
			}
		}
	}
	objs, ok := root.Field(3)
	if !ok {
		return nil, fmt.Errorf("mapvar: champ racine 3 (objets) absent")
	}
	v.Objects = make([]Object, 0, len(objs.Items))
	for i, raw := range objs.Items {
		v.Objects = append(v.Objects, parseObject(i, raw))
	}
	return v, nil
}

func parseObject(index int, raw Value) Object {
	o := Object{Index: index, TeamIndex: TeamUnset, Category: -1}
	if t, ok := raw.Field(2); ok {
		if id, ok2 := t.Field(0); ok2 {
			o.TypeID = int32(id.Int)
		}
	}
	o.Pos = readVec(raw, 3)
	o.Up = readVec(raw, 4)
	o.Forward = readVec(raw, 5)
	if f, ok := raw.Field(7); ok {
		o.Flags = uint8(f.Uint)
	}
	if inst, ok := raw.Field(10); ok {
		if v, ok2 := inst.Field(0); ok2 {
			o.InstanceID = int32(v.Int)
		}
	}
	if bag, ok := raw.Field(8); ok {
		readGameplayBag(&o, bag)
	}
	return o
}

// readGameplayBag lit le sous-arbre gameplay du sac #8 : catégorie, index d'équipe
// et labels de mode de jeu.
func readGameplayBag(o *Object, bag Value) {
	sub, ok := bag.Field(0)
	if !ok || len(sub.Items) == 0 {
		return
	}
	gp := sub.Items[0]
	if c, ok := gp.Field(1); ok {
		o.Category = int(c.Int)
	}
	if t, ok := gp.Field(8); ok {
		o.TeamIndex = int(t.Int)
	}
	if lbls, ok := gp.Field(9); ok {
		for _, entry := range lbls.Items {
			if h, ok2 := entry.Field(0); ok2 {
				o.Labels = append(o.Labels, int32(h.Int))
			}
		}
	}
}

func readVec(raw Value, field uint16) Vec3 {
	v, ok := raw.Field(field)
	if !ok {
		return Vec3{}
	}
	var out Vec3
	if c, ok := v.Field(0); ok {
		out.X = c.Float
	}
	if c, ok := v.Field(1); ok {
		out.Y = c.Float
	}
	if c, ok := v.Field(2); ok {
		out.Z = c.Float
	}
	return out
}
