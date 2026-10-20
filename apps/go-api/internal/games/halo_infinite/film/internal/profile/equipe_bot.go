package profile

// equipe_bot.go — OU L EQUIPE D UN BOT SE LIT DANS SON ENTREE BOT_METADATA, PAR BUILD.
//
// L entree d un bot du paquet de type 12 porte le corps des fiches de joueur, qui finit par un bloc
// de 44 octets bruts (`fiche + 0x1400`, cf. `grammar/bot_metadata.go`). L equipe y est un octet
// signe, et son jumeau un second. Leur POSITION dans le bloc change d un build a l autre : le bloc
// porte aussi une copie du nom en UTF-16 petit-boutiste (16 unites) et un mot de 4 octets
// (`a0bd7618` sur tout le parc), dont l ordre n est pas le meme avant et apres `HI_1_13_0` :
//
//	HI_1_10_0 a HI_1_12_0   [4 octets][equipe, 01, jumeau, x][a0bd7618][nom, 32 octets]
//	HI_1_13_0               [nom, 32 octets][a0bd7618][x, equipe, jumeau, 01][4 octets nuls]
//	HI_1_8_0                [nom, 32 octets][x, e, e, 01][a0bd7618][4 octets nuls]
//
// Lire l octet 0x25 sur un build ancien, c est lire l octet de poids fort d une unite du nom : 0
// pour un nom ASCII, donc une equipe 0 FAUSSE et d apparence valide (mesure du 2026-10-15).
//
// UN COMMENTAIRE DE PROVENANCE PAR LIGNE, comme pour [PersonnalisationOctets] (D-3 d ADR 0034). Les
// deux provenances sont celles de `build_profile.go` : RELU chez l ecrivain pour le build de
// l executable desassemble, MESURE ailleurs — et une mesure dit son oracle et son score. Un build
// sans position etablie n a PAS de ligne : son equipe ne se lit pas, et l appelant le compte. Il
// n herite jamais de la position d un build voisin (D-4).

// PositionEquipeBot : la position, en octets depuis le debut du bloc de 44 octets d une entree
// BOT_METADATA, de l equipe et de son jumeau.
type PositionEquipeBot struct {
	Equipe, Jumeau int
}

// PositionEquipeBotPour rend la position de l equipe d un bot pour un build ; faux quand elle n est
// pas etablie pour ce build.
//
// LA MESURE (2026-10-15, lot K-B) : 617 films du parc local dont la feuille de match porte un bot,
// bloc de 44 octets de chaque entree que la marche de l ecrivain lit dans un paquet qui ferme,
// confronte a l equipe de la feuille (cle `bid`) et a celle de l entite `ti=9` du bot quand les faits
// cuits la donnent. Les scores ci-dessous comptent des BOTS distincts (film, `bid`).
func PositionEquipeBotPour(build string) (PositionEquipeBot, bool) {
	switch build {
	case buildHI1131:
		// RELU chez l ecrivain, Ghidra 2026-10-06 : `FUN_1424d512c` recopie les 44 octets en
		// `configuration + 0xCC0`, `FUN_140ad37f8` pose l octet 0x25 (`+ 0xCE5`) sur le joueur ;
		// jumeau 0x26 (`+ 0xCE6`). Controle : feuille 891/891, entite 85/85, jumeau egal 909/909.
		return PositionEquipeBot{Equipe: 0x25, Jumeau: 0x26}, true
	case buildHI1120:
		// MESURE (l executable de ce build n est pas disponible) : octet 4, jumeau octet 6 ;
		// feuille 111/111 bots sur 60 films, entite 1/1, jumeau egal 116/116, aucune contradiction
		// entre paquets. L octet 0x25 n y donne la feuille que 59 fois sur 112 entrees.
		return PositionEquipeBot{Equipe: 4, Jumeau: 6}, true
	case buildHI1110:
		// MESURE : octet 4, jumeau octet 6 ; feuille 2/2 (un film, trois bots, le troisieme absent
		// de la feuille), jumeau egal 3/3 — la disposition du bloc est celle de HI_1_12_0.
		return PositionEquipeBot{Equipe: 4, Jumeau: 6}, true
	case buildHI1100:
		// MESURE : octet 4, jumeau octet 6 ; feuille 10/10 sur 3 films, jumeau egal 10/10.
		return PositionEquipeBot{Equipe: 4, Jumeau: 6}, true
	}
	// HI_1_9_0, HI_1_4_1 : aucun bot au parc. HI_1_8_0 : 4 bots sur 2 films, tous deux en chacun
	// pour soi, ou la feuille donne des equipes 29 et 30 hors du domaine du designateur d equipe
	// (-1..8) ; aucun oracle par entite. Position NON ETABLIE.
	return PositionEquipeBot{}, false
}
