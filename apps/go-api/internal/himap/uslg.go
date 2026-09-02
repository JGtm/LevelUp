// Package himap — uslg.go : les LIBELLÉS JOUEUR des zones nommées, lus dans les listes de
// chaînes localisées du jeu (`uslg` de globals-rtx-new.module).
//
// POURQUOI CE FICHIER EXISTE. Le catalogue des callouts joignait ses libellés par une
// extraction FIGÉE (callouts_i18n.csv, 463 string_id — le vocabulaire des 22 cartes
// intégrées). Les cartes Forge piochent dans le vocabulaire GLOBAL du jeu : sur la rotation
// du 2026-08-27, 266 string_id de lieu employés, 66 résolus (25 %). Deux tiers des zones
// étaient donc publiées SANS texte — une carte muette, inutilisable en compétitif.
//
// LA CHAÎNE, mesurée le 2026-09-02 (sonde_uslg_gamefiles_test.go pour les mesures) :
//
//	globals-rtx-new.module
//	  tag `locs` (location_name_globals_definition)
//	    -> root+0x120 : 778 StringId de 4 octets — le VOCABULAIRE des lieux, sans texte
//	  tag `uslg` x488 (multilingual_unicode_string_list)
//	    -> le tag lui-même ne porte QUE ses 18 langues (bloc de 18 x 20 octets, u32 index
//	       de langue à +0x00). AUCUN texte.
//	    -> son BLOB DE RESSOURCES porte 18 sous-fichiers `ucsh` CONCATÉNÉS, un par langue,
//	       dans l'ordre de ce bloc. Chaque sous-fichier est un tag complet :
//	         +0x38 headerSize, +0x3C dataSize    (le suivant commence à hs+ds)
//	         +0x18 deps, +0x1C blocs, +0x20 structs, +0x24 dataRefs
//	         table des blocs      @ 0x50 + deps*0x18   (0x10/entrée : size, section, offset)
//	         table des structs    @ +blocs*0x10        (0x20/entrée : type, cible, propriétaire, champ)
//	         table des dataRefs   @ +structs*0x20      (0x14/entrée : bloc cible à +0x08)
//	       racine       = la struct sans propriétaire (owner == -1)
//	       table d'index = le TagBlock au champ 0 de la racine : N x { u32 hash, u32 offset }
//	       texte         = le bloc de la 1re dataRef : chaînes UTF-8 terminées par NUL
//
// LE HASH DE LA TABLE D'INDEX EST LE string_id. Ce n'est pas une hypothèse : le lexique
// ainsi décodé reproduit les 463 string_id de callouts_i18n.csv avec un texte EN et FR
// IDENTIQUE AU CARACTÈRE PRÈS (0 absent, 0 divergence — uslg_gamefiles_test.go), et il
// résout 777/777 entrées du vocabulaire `locs`.
//
// LA LANGUE NE SE DEVINE PAS : l'index de langue est lu dans le tag (0 = anglais,
// 3 = français, vérifié sur les 18 slots par la traduction d'une chaîne connue). L'ordre
// des slots N'EST PAS croissant (0..9, 11, 12, 15, 17, 10, 13, 14, 16) — lire le slot 3
// en supposant « français » donnerait le bon résultat par accident ici et le mauvais au
// prochain ajout de langue.
//
// CE FICHIER NE SERT QUE LA PRODUCTION D'UNE DONNÉE DE RÉFÉRENCE VERSIONNÉE
// (cmd/mapcallouts-build --lexique -> reference/callouts_lexique.csv). Le rejeu, lui, ne
// touche jamais aux fichiers du jeu : il lit le catalogue. Même règle que map_objectives.
package himap

import (
	"encoding/binary"
	"fmt"
	"path/filepath"

	"levelup/go-api/internal/himodule"
)

// LangueEN et LangueFR : index de langue tels que le jeu les numérote. Les autres existent
// (18 en tout) ; le dépôt est bilingue par typage, on n'extrait que ces deux-là.
const (
	LangueEN = 0
	LangueFR = 3
)

// LibelleLieu porte le texte joueur d'un string_id de lieu.
type LibelleLieu struct {
	EN string
	FR string
}

// Offsets de l'en-tête d'un sous-fichier `ucsh` — cf. en-tête de fichier.
const (
	ucshEnteteFixe   = 0x50
	ucshOffDeps      = 0x18
	ucshOffBlocs     = 0x1C
	ucshOffStructs   = 0x20
	ucshOffDataRefs  = 0x24
	ucshOffHeaderLen = 0x38
	ucshOffDataLen   = 0x3C

	ucshDepStride    = 0x18
	ucshBlocStride   = 0x10
	ucshStructStride = 0x20
	ucshRefStride    = 0x14

	// struct : type à +0x10, bloc cible à +0x14, bloc propriétaire à +0x18, champ à +0x1C.
	ucshStructType  = 0x10
	ucshStructCible = 0x14
	ucshStructProp  = 0x18
	ucshStructChamp = 0x1C
	ucshTypeTagBloc = 1

	// dataRef : bloc cible à +0x08.
	ucshRefCible = 0x08

	// une entrée de la table d'index : { u32 hash, u32 offset }.
	ucshIndexStride = 8
)

// locsOffVocabulaire / locsStride : le tableau de StringId du tag `locs`. Mesuré : root
// block de 64 octets, un unique bloc enfant de 778 entrées de 4 octets à 0x120, et
// 0x120 + 778*4 = 3 400 = la taille exacte du tag.
const (
	locsOffVocabulaire = 0x120
	locsStride         = 4
)

// cheminGlobals rend le module des tags globaux d'une installation.
func cheminGlobals(racineDeploy string) string {
	return filepath.Join(racineDeploy, "any", "globals", "globals-rtx-new.module")
}

// VocabulaireLieux rend les StringId de lieu déclarés par le tag `locs` — le vocabulaire
// GLOBAL du jeu, celui dans lequel une carte Forge pioche ses zones nommées.
func VocabulaireLieux(racineDeploy string) ([]uint32, error) {
	m, err := himodule.Open(cheminGlobals(racineDeploy))
	if err != nil {
		return nil, fmt.Errorf("himap: globals : %w", err)
	}
	fichiers := m.Files("locs")
	if len(fichiers) != 1 {
		return nil, fmt.Errorf("himap: %d tag(s) locs, 1 attendu", len(fichiers))
	}
	tag, err := m.Extract(fichiers[0])
	if err != nil {
		return nil, fmt.Errorf("himap: extraire locs : %w", err)
	}
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return nil, fmt.Errorf("himap: struct-table de locs : %w", err)
	}
	liens := liensBlocs(ti)
	if len(liens) != 1 {
		return nil, fmt.Errorf("himap: locs porte %d TagBlock, 1 attendu", len(liens))
	}
	n := compteChamp(ti, liens[0])
	abs, taille := ti.blockAbs(liens[0].target)
	if n <= 0 || abs != locsOffVocabulaire || taille != n*locsStride {
		return nil, fmt.Errorf("himap: bloc de locs inattendu (n=%d abs=0x%X taille=%d)", n, abs, taille)
	}
	out := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, binary.LittleEndian.Uint32(tag[abs+i*locsStride:]))
	}
	return out, nil
}

// LexiqueLieux rend le texte joueur EN/FR de CHAQUE entrée de la liste de chaînes qui
// porte les noms de lieu.
//
// La liste n'est pas désignée par un indice écrit en dur : on prend, parmi les 488 tags
// `uslg`, celui qui couvre le mieux le vocabulaire `locs`, et on EXIGE qu'il le couvre
// entièrement. Un indice en dur casserait à la prochaine mise à jour du jeu sans que rien
// ne le signale ; ce critère-là, lui, est vérifiable et il échoue bruyamment.
func LexiqueLieux(racineDeploy string) (map[uint32]LibelleLieu, error) {
	vocab, err := VocabulaireLieux(racineDeploy)
	if err != nil {
		return nil, err
	}
	attendus := make(map[uint32]bool, len(vocab))
	for _, v := range vocab {
		attendus[v] = true
	}
	m, err := himodule.Open(cheminGlobals(racineDeploy))
	if err != nil {
		return nil, fmt.Errorf("himap: globals : %w", err)
	}
	var meilleur map[uint32]LibelleLieu
	meilleureCouverture := -1
	for _, f := range m.Files("uslg") {
		lex, err := lexiqueDUnTag(m, f)
		if err != nil {
			return nil, fmt.Errorf("himap: tag uslg #%d : %w", f.Index, err)
		}
		couverture := 0
		for v := range attendus {
			if _, ok := lex[v]; ok {
				couverture++
			}
		}
		if couverture > meilleureCouverture {
			meilleureCouverture, meilleur = couverture, lex
		}
	}
	if meilleur == nil {
		return nil, fmt.Errorf("himap: aucun tag uslg dans globals")
	}
	if meilleureCouverture < len(attendus) {
		return nil, fmt.Errorf("himap: la meilleure liste de chaînes ne couvre que %d des %d "+
			"noms de lieu déclarés par locs — format changé ?", meilleureCouverture, len(attendus))
	}
	return meilleur, nil
}

// lexiqueDUnTag décode un tag `uslg` : ses 18 langues, puis les sous-fichiers `ucsh` de son
// blob de ressources. Ne rend que l'anglais et le français.
func lexiqueDUnTag(m *himodule.Module, f himodule.File) (map[uint32]LibelleLieu, error) {
	tag, err := m.Extract(f)
	if err != nil {
		return nil, fmt.Errorf("extraire : %w", err)
	}
	langues, err := ordreDesLangues(tag)
	if err != nil {
		return nil, err
	}
	blob, err := m.ResourceBlob(f)
	if err != nil {
		return nil, fmt.Errorf("blob de ressources : %w", err)
	}
	out := map[uint32]LibelleLieu{}
	p := 0
	for slot := 0; p+ucshEnteteFixe <= len(blob); slot++ {
		table, suivant, err := decodeSousFichierUcsh(blob, p)
		if err != nil {
			return nil, err
		}
		if slot < len(langues) {
			for h, s := range table {
				l := out[h]
				switch langues[slot] {
				case LangueEN:
					l.EN = s
				case LangueFR:
					l.FR = s
				default:
					continue
				}
				out[h] = l
			}
		}
		p = suivant
	}
	return out, nil
}

// ordreDesLangues lit, dans le tag `uslg`, l'index de langue de chacun de ses sous-fichiers.
func ordreDesLangues(tag []byte) ([]int, error) {
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return nil, fmt.Errorf("struct-table : %w", err)
	}
	liens := liensBlocs(ti)
	if len(liens) == 0 {
		return nil, fmt.Errorf("aucun TagBlock (bloc des langues attendu)")
	}
	n := compteChamp(ti, liens[0])
	abs, taille := ti.blockAbs(liens[0].target)
	if n <= 0 || taille < n*4 {
		return nil, fmt.Errorf("bloc des langues inattendu (n=%d taille=%d)", n, taille)
	}
	stride := taille / n
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, int(binary.LittleEndian.Uint32(tag[abs+i*stride:])))
	}
	return out, nil
}

// enteteUcsh porte la disposition d'un sous-fichier : ses trois tables et ses longueurs.
type enteteUcsh struct {
	debut, headerLen, dataLen     int
	nBlocs, nStructs, nRefs       int
	tabBlocs, tabStructs, tabRefs int
}

// litEnteteUcsh lit et VALIDE l'en-tête d'un sous-fichier à l'offset p.
func litEnteteUcsh(blob []byte, p int) (enteteUcsh, error) {
	if p+ucshEnteteFixe > len(blob) || string(blob[p:p+4]) != "ucsh" {
		return enteteUcsh{}, fmt.Errorf("offset 0x%X : magie `ucsh` attendue", p)
	}
	u := func(o int) int { return int(binary.LittleEndian.Uint32(blob[p+o:])) }
	e := enteteUcsh{
		debut: p, headerLen: u(ucshOffHeaderLen), dataLen: u(ucshOffDataLen),
		nBlocs: u(ucshOffBlocs), nStructs: u(ucshOffStructs), nRefs: u(ucshOffDataRefs),
	}
	if e.headerLen <= 0 || e.dataLen < 0 || p+e.headerLen+e.dataLen > len(blob) {
		return enteteUcsh{}, fmt.Errorf("offset 0x%X : bornes (header %d, data %d)", p, e.headerLen, e.dataLen)
	}
	if e.nBlocs < 0 || e.nStructs < 0 || e.nRefs < 0 {
		return enteteUcsh{}, fmt.Errorf("offset 0x%X : comptes de tables négatifs", p)
	}
	e.tabBlocs = p + ucshEnteteFixe + u(ucshOffDeps)*ucshDepStride
	e.tabStructs = e.tabBlocs + e.nBlocs*ucshBlocStride
	e.tabRefs = e.tabStructs + e.nStructs*ucshStructStride
	if e.tabBlocs < p || e.tabRefs+e.nRefs*ucshRefStride > p+e.headerLen {
		return enteteUcsh{}, fmt.Errorf("offset 0x%X : tables hors de l'en-tête", p)
	}
	return e, nil
}

// bloc rend les bornes absolues du i-ième data-block, ou false s'il n'existe pas / déborde.
func (e enteteUcsh) bloc(blob []byte, i int) (int, int, bool) {
	if i < 0 || i >= e.nBlocs {
		return 0, 0, false
	}
	b := e.tabBlocs + i*ucshBlocStride
	taille := int(binary.LittleEndian.Uint32(blob[b:]))
	abs := e.debut + int(binary.LittleEndian.Uint64(blob[b+8:]))
	if binary.LittleEndian.Uint16(blob[b+6:]) != 0 {
		abs += e.headerLen
	}
	if taille < 0 || abs < 0 || abs+taille > len(blob) {
		return 0, 0, false
	}
	return abs, taille, true
}

// decodeSousFichierUcsh décode un sous-fichier du blob à l'offset p et rend sa table
// hash -> texte ainsi que l'offset du sous-fichier suivant.
func decodeSousFichierUcsh(blob []byte, p int) (map[uint32]string, int, error) {
	e, err := litEnteteUcsh(blob, p)
	if err != nil {
		return nil, 0, err
	}
	idxBloc, err := blocDeLIndex(blob, e.tabStructs, e.nStructs)
	if err != nil {
		return nil, 0, fmt.Errorf("offset 0x%X : %w", p, err)
	}
	txtBloc := -1
	if e.nRefs > 0 {
		txtBloc = int(int32(binary.LittleEndian.Uint32(blob[e.tabRefs+ucshRefCible:])))
	}
	idxAbs, idxTaille, okIdx := e.bloc(blob, idxBloc)
	txtAbs, txtTaille, okTxt := e.bloc(blob, txtBloc)
	if !okIdx || !okTxt {
		// Liste vide : licite (certains sous-fichiers ne portent aucune chaîne).
		return map[uint32]string{}, p + e.headerLen + e.dataLen, nil
	}
	return litTableDIndex(blob, idxAbs, idxTaille, txtAbs, txtTaille), p + e.headerLen + e.dataLen, nil
}

// litTableDIndex parcourt les paires { u32 string_id, u32 offset } et découpe les chaînes
// terminées par NUL du bloc de texte. Une entrée dont l'offset sort du bloc est IGNORÉE :
// mieux vaut une zone muette qu'un texte pris à côté.
func litTableDIndex(blob []byte, idxAbs, idxTaille, txtAbs, txtTaille int) map[uint32]string {
	table := make(map[uint32]string, idxTaille/ucshIndexStride)
	for i := 0; i+ucshIndexStride <= idxTaille; i += ucshIndexStride {
		h := binary.LittleEndian.Uint32(blob[idxAbs+i:])
		o := int(binary.LittleEndian.Uint32(blob[idxAbs+i+4:]))
		if o < 0 || o >= txtTaille {
			continue
		}
		fin := txtAbs + o
		for fin < txtAbs+txtTaille && blob[fin] != 0 {
			fin++
		}
		table[h] = string(blob[txtAbs+o : fin])
	}
	return table
}

// blocDeLIndex rend l'indice du bloc de la table d'index : le TagBlock porté par le champ 0
// de la structure racine (celle sans propriétaire). Rend -1 quand la liste est vide, ce qui
// est licite — certains sous-fichiers ne portent aucune chaîne.
func blocDeLIndex(blob []byte, tabStructs, nStructs int) (int, error) {
	racine := -2
	for k := 0; k < nStructs; k++ {
		b := tabStructs + k*ucshStructStride
		if int32(binary.LittleEndian.Uint32(blob[b+ucshStructProp:])) == -1 {
			racine = int(int32(binary.LittleEndian.Uint32(blob[b+ucshStructCible:])))
			break
		}
	}
	if racine == -2 {
		return 0, fmt.Errorf("aucune structure racine (propriétaire -1)")
	}
	for k := 0; k < nStructs; k++ {
		b := tabStructs + k*ucshStructStride
		if binary.LittleEndian.Uint16(blob[b+ucshStructType:]) != ucshTypeTagBloc {
			continue
		}
		if int(int32(binary.LittleEndian.Uint32(blob[b+ucshStructProp:]))) != racine {
			continue
		}
		if binary.LittleEndian.Uint32(blob[b+ucshStructChamp:]) != 0 {
			continue
		}
		return int(int32(binary.LittleEndian.Uint32(blob[b+ucshStructCible:]))), nil
	}
	return -1, nil
}
