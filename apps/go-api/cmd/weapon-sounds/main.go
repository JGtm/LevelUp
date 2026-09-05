// Command weapon-sounds — extraction des sons de tir par arme depuis les tags `sbnk`.
//
// POURQUOI CET OUTIL. Les `.pck` du jeu livrent 90 170 `.wem` anonymes : aucune bank
// Wwise sur disque (0 bank sur 1645 `.pck`) et aucun nom de tag dans les modules
// (`stringsSize` = 0 sur les 132 modules). Un pack par arme identifie l'ARME de facon
// certaine, mais rien n'y designe le TIR parmi 80 a 360 sons. La mesure qui debloque :
// les banks ont ete converties en tags `sbnk` (1305 dans `pc/globals/globals-rtx-new`),
// et les IDs `.wem` d'une arme s'y retrouvent.
//
// Plan de rattachement : `.ai/V7.5/PLAN_EXTRACTION_SONS_ARMES.md`.
//
// MODES :
//
//	probe    (etape 1) statue sur le format des `sbnk` (bank Wwise verbatim ?)
//	map      (etape 2) d'un `.pck` d'arme vers ses evenements Wwise et leurs `.wem`
//	noms     (etape 3) dumpe les chunks `STID`, seule source de noms en clair
//	lien     (etape 3) relie les evenements aux tags `snd!` puis aux tags `weap`
//	banks    (etape 3) histogramme des `sbnk` references par les `snd!`
//	sndscan  (etape 3) cherche des identifiants arbitraires dans les `snd!`
//
// LA CHAINE D'EQUIPEMENT (2026-08-18, PLAN_EQUIPEMENTS_MANQUANTS_SONS) est une SECONDE
// FAMILLE de modes, distincte de celle des armes parce que la source l'est : un equipement
// n'a pas de tag `weap` ni de champ « Weapon Fire Sounds ». Detail dans `eqip.go`.
//
//	eqip-sons   passe 1, module `any/globals` : eqip -> effe -> snd! -> sbnk
//	eqip-banks  passe 2, module `pc/globals`  : sbnk -> .wem -> pack nomme
//	banks-noms  module `pc/globals` : NOMME les 1305 banques par hachage FNV-1 de leur
//	            identifiant Wwise (chunk BKHD). C'est ce qui permet de trouver une banque
//	            qui n'a AUCUN pack sur le disque — les banques de mode (drapeau, bastion,
//	            extraction) et 14 des 17 banques d'equipement. Detail dans `banks_noms.go`.
//	tir-vehi    module `any/globals` : le son de TIR d'une ARME DE VEHICULE, par le champ
//	            nomme « Weapon Fire Sound ». Detail dans `tir_vehicules.go` — la chaine
//	            `lot`/`lot-tir` ne balaie que les packs d'armes et de tourelles, jamais les
//	            chassis `sb_010_veh_*`, et la banque du chassis ne porte pas le tir.
//	remonter-banque  module `any/globals` : la chaine A L'ENVERS, banque -> `snd!` -> ... ->
//	            `vehi`. Detail dans `remonter_banque.go` — c'est ce qui dit QUEL vehicule
//	            joue QUELLE banque d'explosion.
//	pck-banques module `pc/globals` : rattache N `.pck` a leur `sbnk` par intersection des
//	            `.wem`, en UNE charge du module. Detail dans `pck_banques.go` — la
//	            convention de nommage FNV-1 tombe sur les familles covenant/banished.
//	pck-dump    AUCUN module : extrait les `.wem` COMPLETS (streames) d'un `.pck` AKPK.
//	            Detail dans `pck_dump.go` — la version embarquee dans une banque est un
//	            prefetch tronque, le media complet vit dans le pack.
//	eqip-arbre  module `pc/globals` : la STRUCTURE des evenements d'une banque
//	            (couches simultanees vs variantes, gains, delais, couverture). Detail
//	            dans `eqip_arbre.go` — c'est ce qui manque pour RECONSTRUIRE un son.
//
// ATTENTION MEMOIRE : `himodule.Open` lit le module ENTIER en memoire. Le module qui porte
// les `sbnk` fait 7,24 Go, celui qui porte les `snd!`/`weap` 0,62 Go. Ne jamais charger les
// deux dans le meme processus : les modes s'echangent leurs resultats par le fichier JSON.
//
// SORTIE JSON — CHAMP `variation` (plan `.ai/V7.5/PLAN_SONS_REJEU_INAPP.md`, etape 2).
//
// Les `.wav` extraits sont purs ; le jeu, lui, deplace volume et hauteur A CHAQUE LECTURE
// dans une fourchette declaree par le paquet RANGED. Cette fourchette est desormais lue et
// remontee dans les rapports, pour que l'app la rejoue (elle n'est jamais cuite dans les
// fichiers). Elle apparait a trois niveaux, toujours sous la meme forme
// (`{"volume_db": {"bas": …, "haut": …}, "pitch_cents": {…}, "couche": …,
// "gain_db_couche": …}`), et TOUJOURS EN OPTION — absente, le son se joue tel quel :
//
//	mode `arbre`   : par couche (`branches[].variation`) et par evenement
//	mode `lot`     : par evenement (`armes[].evenements[].variation`)
//	mode `lot-tir` : par mode de tir (`modes[].variation`) et par arme (`variation`)
//
// Unites : decibels pour le volume, centiemes de demi-ton pour la hauteur (l'app en tire un
// `playbackRate` par 2^(cents/1200)). Agregation : fourchette de la COUCHE DOMINANTE, celle
// de plus fort gain de chemin. Les modes qui parcourent des banks impriment en fin
// d'execution le releve des signes observes dans le paquet RANGED — c'est lui qui tranche
// l'interpretation des deux composantes (offsets signes ou magnitudes) sur donnees reelles.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/himap"
)

// moduleParDefaut : le module qui porte les `sbnk` (mesure : les 40 IDs `.wem` du fusil
// d'assaut y sont tous localises).
const moduleParDefaut = "pc/globals/globals-rtx-new.module"

// moduleTags : le module qui porte les `snd!` et les `weap` (14 228 et 111).
const moduleTags = "any/globals/globals-rtx-new.module"

// wemTemoins : IDs `.wem` reellement presents dans `sb_010_wea_un_assaultrifle.pck`.
// Ils servent de temoins pour reconnaitre le `sbnk` du fusil d'assaut sans nom de tag.
var wemTemoins = []uint32{14649067, 1002108249, 1004646855, 1009888121, 665681453, 253891388}

func main() {
	deploy := flag.String("deploy", "", "racine `deploy` des archives du jeu (auto-detectee si vide)")
	module := flag.String("module", moduleParDefaut, "module a sonder, relatif a la racine deploy")
	mode := flag.String("mode", "probe", "probe | map | noms | lien | banks | sndscan")
	pck := flag.String("pck", "", "chemin d'un .pck d'arme (mode map)")
	sortie := flag.String("json", "", "fichier JSON (sortie du mode map, entree des modes lien/final)")
	sortieTir := flag.String("out", "", "fichier JSON de sortie du mode final")
	// Defaut = tous : l'heuristique « une bank d'arme est petite » est FAUSSE (mesure :
	// la bank du fusil d'assaut fait 1,5 Mo, absente des 60 plus petites).
	limite := flag.Int("limite", 0, "nombre de tags sbnk decompresses par la sonde (0 = tous)")
	wem := flag.String("wem", "", "IDs recherches, separes par des virgules (defaut : fusil d'assaut)")
	sfx := flag.String("sfx", "", "dossier des .pck (deduit de -deploy si vide) ; construit l'index large")
	emb := flag.String("emb", "", "dossier ou ecrire les .wem embarques des banks (mode lot)")
	sbnkGid := flag.Uint("sbnk", 0, "identifiant d'une bank (mode embarques, alternative a -pck)")
	banksSup := flag.String("banks", "", "identifiants de banks a analyser en plus (mode lot) ou a structurer (mode eqip-arbre), separes par des virgules, en hexa ; \"all\" (mode eqip-arbre) = toutes les banques sbnk du module")
	etroit := flag.Bool("etroit", false, "valider les sons contre le seul pck de l'arme (comportement d'origine)")
	eqipIDs := flag.String("eqip", "", "identifiants de tags `eqip` cibles (hexa, virgules) ; vide = tous (modes eqip-sons/eqip-banks)")
	exclure := flag.String("exclure", "", "identifiants de banks a exclure du triage (mode eqip-durees), hexa, virgules")
	eventsIDs := flag.String("events", "", "identifiants d'evenements a dumper (mode hirc-event), hexa, virgules ; vide = tous")
	etatsSwitch := flag.String("etats", "", "etats de conteneur Switch a FORCER (mode hirc-event), DECIMAL, virgules ; vide = etat par defaut")
	dossierWav := flag.String("wav", "", "dossier des .wav decodes, nommes <wem>.wav (mode rendu-event)")
	dest := flag.String("dest", "", "dossier de sortie des rendus (mode rendu-event)")
	nomRendu := flag.String("nom", "evenement", "nom de base des fichiers rendus (mode rendu-event)")
	tirages := flag.Int("tirages", 3, "nombre de tirages complets a rendre (mode rendu-event)")
	graine := flag.Int64("graine", 1, "graine du tirage de variantes (mode rendu-event)")
	dureeBoucle := flag.Float64("duree", 0, "duree de boucle en secondes ; 0 = one-shot (mode rendu-event)")
	flag.Parse()

	racine, err := resoudreDeploy(*deploy)
	if err != nil {
		fmt.Fprintln(os.Stderr, "racine deploy introuvable:", err)
		os.Exit(1)
	}
	chemin := filepath.Join(racine, filepath.FromSlash(*module))

	temoins, err := parserWem(*wem)
	if err != nil {
		fmt.Fprintln(os.Stderr, "option -wem invalide:", err)
		os.Exit(1)
	}

	// INDEX LARGE. Sans lui, un `sourceID` n'est accepte que s'il appartient au pack de
	// l'arme — ce qui faisait disparaitre les couches partagees entre armes, precisement
	// celles qui manquaient a l'oreille. Les garde-fous STRUCTURELS (listes d'enfants et
	// d'actions validees contre les objets de la bank) restent inchanges.
	if !*etroit {
		dossier := *sfx
		if dossier == "" {
			dossier = dossierSFXParDefaut(racine)
		}
		idx, errIdx := indexTousPcks(dossier)
		if errIdx != nil {
			fmt.Fprintln(os.Stderr, "index large indisponible, validation etroite:", errIdx)
		} else {
			indexLarge = idx
			fmt.Printf("index large : %d identifiants .wem sur tous les packs\n", len(indexLarge))
		}
	}

	switch *mode {
	case "probe":
		err = sonder(chemin, temoins, *limite)
	case "map":
		if *pck == "" {
			err = fmt.Errorf("le mode map exige -pck")
			break
		}
		err = cartographier(chemin, *pck, *sortie)
	case "audit":
		err = auditFormat(chemin, *limite)
	case "noms":
		err = listerNoms(chemin)
	case "lien":
		if *sortie == "" {
			err = fmt.Errorf("le mode lien exige -json (le rapport produit par -mode map)")
			break
		}
		err = relier(chemin, *sortie)
	case "banks":
		err = histogrammeBanks(chemin, temoins[0])
	case "sndscan":
		err = sonderSnd(chemin, temoins)
	case "qui":
		err = quiRefere(chemin, temoins[0])
	case "deps":
		err = dependancesDe(chemin, temoins[0])
	case "remonter":
		err = remonter(chemin, temoins[0], *limite)
	case "tir":
		err = sonsDeTir(chemin, temoins[0])
	case "melee":
		var g uint32
		if len(temoins) > 0 && temoins[0] != wemTemoins[0] {
			g = temoins[0]
		}
		err = sonsDeMelee(chemin, g)
	case "meleefx":
		var g uint32
		if len(temoins) > 0 && temoins[0] != wemTemoins[0] {
			g = temoins[0]
		}
		err = effetsDeMelee(chemin, g)
	case "cadence":
		err = sonderCadence(chemin, temoins[0])
	case "arbre":
		profondeur := *limite
		if profondeur <= 0 {
			profondeur = 4
		}
		sortieCouches = *sortieTir
		err = arborescence(chemin, *pck, profondeur, uint32(*sbnkGid))
	case "embarques":
		err = extraireEmbarques(chemin, *pck, *sortieTir, uint32(*sbnkGid))
	case "lot":
		if *pck == "" || *sortie == "" {
			err = fmt.Errorf("le mode lot exige -pck (dossier SFX) et -json (sortie)")
			break
		}
		dossierEmbarques = *emb
		for _, s := range strings.Split(*banksSup, ",") {
			if s = strings.TrimSpace(s); s != "" {
				if v, e := strconv.ParseUint(s, 16, 32); e == nil {
					banksSupplementaires = append(banksSupplementaires, uint32(v))
				}
			}
		}
		err = cartographierLot(chemin, *pck, *sortie)
	case "lot-tir":
		if *sortie == "" || *sortieTir == "" {
			err = fmt.Errorf("le mode lot-tir exige -json (sortie du mode lot) et -out")
			break
		}
		err = livrerTirLot(chemin, *sortie, *sortieTir)
	case "final":
		if *sortie == "" {
			err = fmt.Errorf("le mode final exige -json (le rapport produit par -mode map)")
			break
		}
		err = livrerTir(chemin, *sortie, temoins[0], *sortieTir)
	case "eqip-sons":
		err = sonsDEquipement(chemin, parserHexa(*eqipIDs), *sortie)
	case "eqip-banks":
		err = banquesDEquipement(chemin, *sortie, *sortieTir, *emb)
	case "eqip-arbre":
		// "-banks all" : BALAYAGE STRUCTUREL de toutes les banques du module, hors du
		// graphe eqip deja ecoute (`.ai/V7.5/replay2d/PLAN_BALISE_MIX_WWISE.md`, phase 5.1).
		// Le booleen se decide ICI, une seule fois : `structureDesBanques` n'a pas a
		// re-interpreter la chaine source.
		toutes := strings.EqualFold(strings.TrimSpace(*banksSup), "all")
		var gids []uint32
		if !toutes {
			for id := range parserHexa(*banksSup) {
				gids = append(gids, id)
			}
			sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
		}
		err = structureDesBanques(chemin, gids, *sortie, *sortieTir, *emb, toutes)
	case "eqip-durees":
		err = triageDurees(chemin, *sortie, parserHexa(*exclure), *sortieTir, *emb, *limite)
	case "deps-ordre":
		err = dependancesEnOrdre(chemin, temoins[0], *limite)
	case "audit-modes":
		err = auditModesConteneurs(chemin, parserHexa(*banksSup))
	case "audit-actions":
		err = auditActions(chemin, parserHexa(*banksSup), *sortie)
	case "chaines":
		err = extraireChaines(chemin, temoins)
	case "audit-boucles":
		err = auditBoucles(chemin, parserHexa(*banksSup))
	case "blend":
		err = dumperBlend(chemin, uint32(*sbnkGid), parserHexa(*eqipIDs))
	case "orphelins":
		var cibles []uint32
		if strings.TrimSpace(*wem) != "" {
			cibles = temoins
		}
		var gids []uint32
		for id := range parserHexa(*banksSup) {
			gids = append(gids, id)
		}
		sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
		err = diagnostiquerOrphelins(chemin, gids, cibles)
	case "banks-noms":
		dossier := *sfx
		if dossier == "" {
			dossier = dossierSFXParDefaut(racine)
		}
		err = nommerBanques(chemin, dossier, *sortie)
	case "vehi-sons":
		err = sonsDeVehicules(chemin, *sortie)
	case "tir-vehi":
		err = tirDesVehicules(chemin, parserHexa(*eqipIDs), *sortie)
	case "remonter-banque":
		err = remonterDepuisBanques(chemin, parserHexa(*banksSup), *limite)
	case "pck-banques":
		err = banquesDesPacks(chemin, *pck, *sortie)
	case "hirc-event":
		etats := map[uint32]bool{}
		for _, s := range strings.Split(*etatsSwitch, ",") {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}
			v, e := strconv.ParseUint(s, 10, 32)
			if e != nil {
				err = fmt.Errorf("option -etats : %q : %w", s, e)
				break
			}
			etats[uint32(v)] = true
		}
		if err == nil {
			err = dumperEvenements(chemin, parserHexa(*banksSup), parserHexa(*eventsIDs), etats, *sortieTir)
		}
	case "rendu-event":
		var etatRendu uint64
		if s := strings.TrimSpace(*etatsSwitch); s != "" {
			if etatRendu, err = strconv.ParseUint(s, 10, 32); err != nil {
				err = fmt.Errorf("option -etats (mode rendu-event, une seule valeur) : %w", err)
				break
			}
		}
		err = rendreEvenements(optionsRendu{
			Plan: *sortie, Dossier: *dossierWav, Dest: *dest, Nom: *nomRendu, Sortie: *sortieTir,
			Events: parserHexa(*eventsIDs), Tirages: *tirages, Graine: *graine,
			DureeS: *dureeBoucle, Etat: uint32(etatRendu),
		})
	case "mesurer-wav":
		err = mesurerWavs(*dossierWav, temoins, strings.TrimSpace(*wem) == "", *sortieTir)
	case "pck-dump":
		filtre := map[uint32]bool{}
		if strings.TrimSpace(*wem) != "" {
			for _, id := range temoins {
				filtre[id] = true
			}
		}
		err = extrairePck(*pck, *sortieTir, filtre)
	default:
		err = fmt.Errorf("mode inconnu %q", *mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "echec:", err)
		os.Exit(1)
	}
}

// resoudreDeploy rend la racine `deploy`, explicite ou auto-detectee.
func resoudreDeploy(explicite string) (string, error) {
	if explicite != "" {
		return explicite, nil
	}
	return himap.DeployRoot()
}

// parserHexa lit une liste d'identifiants de tags en HEXADECIMAL, separes par des virgules.
// Vide rend une table vide, que les modes interpretent comme « tous ».
func parserHexa(s string) map[uint32]bool {
	out := map[uint32]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "0x"))
		if part == "" {
			continue
		}
		if v, err := strconv.ParseUint(part, 16, 32); err == nil {
			out[uint32(v)] = true
		} else {
			fmt.Fprintf(os.Stderr, "option -eqip : %q ignore (%v)\n", part, err)
		}
	}
	return out
}

// parserWem lit la liste d'identifiants recherches ; vide rend les temoins du fusil d'assaut.
func parserWem(s string) ([]uint32, error) {
	if strings.TrimSpace(s) == "" {
		return wemTemoins, nil
	}
	var out []uint32
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", part, err)
		}
		out = append(out, uint32(v))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun ID exploitable")
	}
	return out, nil
}
