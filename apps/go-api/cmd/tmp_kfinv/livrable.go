package main

import (
	"encoding/json"
	"os"
)

// livrable.go — mise en forme du bloc `inv` destiné au POC.
//
// RÈGLE DE TRAÇABILITÉ (née du défaut réparé le 2026-07-27) : chaque état porte l'identifiant
// du film dont il provient. Un document composite qui ne trace pas ses origines est un piège à
// contradictions.

type outGrenade struct {
	Rang int    `json:"rang"`
	Nom  string `json:"nom"`
	N    uint32 `json:"n"`
}

type outSel struct {
	Rang    int    `json:"rang"`
	Nom     string `json:"nom"`
	Methode string `json:"methode"`
}

type outAbility struct {
	Index          int    `json:"index"`
	Nom            string `json:"nom"`
	Connu          bool   `json:"connu"`
	Utilisations   *int   `json:"utilisations"`
	EtatUtilisions string `json:"etat_utilisations"`
}

type outWeapon struct {
	Emplacement int      `json:"emplacement"`
	Nom         string   `json:"nom"`
	Famille     string   `json:"famille"`
	Degainee    bool     `json:"degainee"`
	Chargeur    *uint32  `json:"chargeur"`
	Reserve     *uint32  `json:"reserve"`
	Jauge       *float64 `json:"jauge_de_charge"`
	Munitions   string   `json:"munitions_etat"`
}

type outState struct {
	Film       string       `json:"film"`
	Frame      int          `json:"f"`
	TimeMS     int          `json:"ms"`
	Slot       uint32       `json:"s"`
	Grenades   []outGrenade `json:"grenades"`
	Selected   *outSel      `json:"grenade_selectionnee"`
	Capacite   *outAbility  `json:"capacite"`
	Armes      []outWeapon  `json:"armes"`
	Degaine    *int         `json:"emplacement_degaine"`
	DegaineTxt string       `json:"emplacement_degaine_etat"`
	ParseUniq  bool         `json:"munitions_parse_unique"`
	ParseN     int          `json:"munitions_candidats"`
}

type livrable struct {
	Bloc       string      `json:"bloc"`
	Film       string      `json:"film"`
	Carte      string      `json:"carte"`
	Source     string      `json:"source"`
	Horloge    string      `json:"horloge"`
	Tables     any         `json:"tables_appliquees"`
	Couverture any         `json:"couverture"`
	Reserves   []string    `json:"reserves"`
	Etats      []outState  `json:"etats"`
	Controle   any         `json:"controle_terrain"`
	_          interface{} `json:"-"`
}

func buildLivrable(states []InvState, film string, couverture any, controle any) livrable {
	var out []outState
	for _, s := range states {
		o := outState{Film: s.Film, Frame: s.Frame, TimeMS: s.TimeMS, Slot: s.Slot,
			ParseUniq: s.Parsed, ParseN: s.ParseN}
		hasGren := false
		for i := 0; i < 4; i++ {
			if s.GrenNames[i] != "" {
				hasGren = true
			}
		}
		if hasGren {
			for i := 0; i < 4; i++ {
				o.Grenades = append(o.Grenades, outGrenade{Rang: i, Nom: grenadeNames[i], N: s.Gren[i]})
			}
			if s.GrenSel >= 0 {
				o.Selected = &outSel{Rang: s.GrenSel, Nom: grenadeNames[s.GrenSel], Methode: s.GrenSelBy}
			}
		}
		if s.AbilIdx >= 0 {
			_, known := abilityNames[uint32(s.AbilIdx)]
			o.Capacite = &outAbility{Index: s.AbilIdx, Nom: s.AbilName, Connu: known,
				Utilisations: nil, EtatUtilisions: "non localise dans le film"}
		}
		for k, w := range s.Weapons {
			if k > 1 {
				break
			}
			aw := outWeapon{Emplacement: k, Nom: w, Famille: s.WeaponFam[k],
				Degainee: s.Drawn == k, Munitions: "non lues"}
			if k < len(s.Ammo) {
				a := s.Ammo[k]
				switch {
				case a.Mag != nil:
					aw.Chargeur, aw.Reserve, aw.Munitions = a.Mag, a.Res, "chargeur"
				case a.Gauge != nil:
					aw.Jauge, aw.Munitions = a.Gauge, "jauge de charge"
				default:
					aw.Munitions = "aucune (arme sans chargeur ni jauge serialisee)"
				}
			}
			o.Armes = append(o.Armes, aw)
		}
		switch s.DrawnRaw {
		case -1:
			o.DegaineTxt = "non lu"
		case 2:
			o.DegaineTxt = "aucune arme degainee"
		default:
			d := s.DrawnRaw
			o.Degaine, o.DegaineTxt = &d, "lu"
		}
		out = append(out, o)
	}
	return livrable{
		Bloc:  "inv",
		Film:  film,
		Carte: "Cliffhanger (module ridgeline)",
		Source: "records de biped (archetype 35) des paquets d'image-cle type 2 du film — " +
			"decodage HORS LIGNE, aucune capture Cheat Engine (celle-ci ne couvre que le film 9e8fb31b)",
		Horloge: "image = (horodatage du paquet - 4521507487 us) / 100000 ; meme origine et meme pas que les blocs tracks/shots/loadouts du POC",
		Tables: map[string]any{
			"grenades": map[string]any{"0": "Fragmentation", "1": "Plasma", "2": "Dynamo", "3": "Spike",
				"source": "RECETTE_LOADOUT_2026-07-27 §1 et §8 — deux chaines independantes, question close"},
			"capacites": map[string]any{"3": "mur portatif", "4": "grappin", "5": "propulseur", "6": "capteur de menace",
				"source":  "RECETTE_LOADOUT_2026-07-27 §2",
				"reserve": "TABLE PARTIELLE : 4 index observes, 11 capacites existent dans le jeu. Un index hors table doit s'afficher INCONNU, jamais etre devine."},
			"munitions": map[string]any{
				"grammaire": "i30/i33 = union : b0=R(1), si 0 chargeur R(8) ; b1=R(1), si 0 jauge dequant(R(12),0,1). i31/i34 = reserve R(11). i32/i35 = 2 bits drapeaux + 7 bits surchauffe.",
				"source":    "RECETTE_LOADOUT_2026-07-27 §7 (carte memoire lue dans le binaire)"},
			"selecteur": map[string]any{"0": "emplacement 0 degaine", "1": "emplacement 1 degaine", "2": "aucune arme degainee",
				"source": "RECETTE_LOADOUT_2026-07-27 §3"},
		},
		Couverture: couverture,
		Reserves: []string{
			"Le COMPTEUR D'UTILISATIONS de capacite n'est PAS localise : 36 006 positions testees sur 6 ancres, 0 reproduit le releve. Le champ est publie a null avec l'etat 'non localise'.",
			"Le sens de i22 -> type de grenade vient d'un autre film (9e8fb31b) mais est confirme deux fois par le binaire ; il est ici RE-VERIFIE sur 000d5950 contre le releve Theater (8/8).",
			"L'emplacement degaine vaut 'aucune' pour les huit joueurs a la premiere image-cle (image 163) : le match n'a pas encore commence, les armes sont rangees. La confrontation au releve 'arme en main' echoue donc a cette image-cle, et c'est mesure, pas masque.",
			"Le marteau a gravite et l'epee n'emettent NI chargeur NI jauge : leur charge est portee par un composant non identifie. Publie comme 'aucune', jamais comme 0.",
			"51 records sur 150 ont plusieurs parses possibles du bloc munitions ; le plus long est retenu et le nombre de candidats est publie (munitions_candidats).",
		},
		Etats:    out,
		Controle: controle,
	}
}

func writeJSON(path string, v any) error {
	blob, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0o644)
}
