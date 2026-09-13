# TI=11 — 10 feuilles restantes : grammaire, verif structurelle, spec de cablage

## Table complete
TABLE COMPLETE ti=11 (managed-objective) — 34 composants i0..i33. dst = *(ctx+0x10) = bloc d'etat objectif partage. Chaque ligne : i | nom de registre (ecs_table.tsv) | deser Ghidra | grammaire (largeur bits) | offset dst | taille champ. [A]=ancre deja cablee, [N]=feuille a cabler (les 10). Tous les faits [N] re-verifies par decompile/disassemble ce jour.

i0 [A]  managed-objective-timers-component            | FUN_142ed5a6c | 2xR(7) via FUN_1410d9088 (boucle RBX de +0x0 a +0x8, pas 4) | +0x00,+0x04 | 2xu32
i1 [A]  managed-objective-color-component             | FUN_142ed544c | 4xR(8) quantifie f32 RGBA via FUN_1406d84b4 (scale DAT_143cd8374) -> MOVSS +0x8/+0xc/+0x10/+0x14 | +0x08..+0x17 | 4xf32
i2 [N]  managed-objective-formatted-text-component    | FUN_1411385b0 -> FUN_14080b034(dst+0x18) | R(1) presence [bit==1 present]; si present: R(32) valeur + R(3) count(0..7) + count x element (R(3) tag; tag0 rien; tag1 R(1)[si0:R(5)]; tag2 R(1)[si0:R(32) sinon R(24)]; tag3..7 R(32)). LONGUEUR VARIABLE | +0x18..+0x3f | u32 @0x18 + 4x{u32,u8} @0x1c..0x3b
i3 [A]  managed-objective-object-reference-component  | FUN_142ed5550 | R(32) inline (CMP EAX,0x20) -> [RBX+0x40] | +0x40 | u32
i4 [N]  managed-objective-interaction-filter-component| FUN_140dbe170 -> FUN_140dbe400(dst+0x48) | R(4) masque -> +0x148 ; scalaire GATE version (level>1: R(1), sinon R(32)) -> +0x14c ; boucle 4 bits bas du masque, slot arme = R(4) selecteur + dispatch FUN_141e98e10 (sel0 rien; sel1..5 = R(1) commun + charge {sel1 R(1); sel2 R(32); sel3 R(4); sel4 R(9); sel5 R(3)count + recursion}; sel6..15 assert). VARIABLE/RECURSIF | slots +0x48/+0x88/+0xc8/+0x108 (0x40 o); masque +0x148; scalaire +0x14c | 4x0x40 + 2xu32
i5 [A]  managed-objective-type-component              | FUN_1410fc4a4 | R(32) -> +0x150 | +0x150 | u32
i6 [N]  managed-objective-enabled-component           | FUN_1411615f8 | R(1) via FUN_1406cf008 -> byte[RBX+0x154] | +0x154 | u8
i7 [N]  managed-objective-priority-component          | FUN_14116d2b8 | R(8) inline (CMP EAX,0x8) -> dword[RBX+0x158] | +0x158 | u32
i8 [N]  managed-objective-message-type-component      | FUN_14116c844 | R(4) inline (CMP EAX,0x4) -> dword[RBX+0x15c] | +0x15c | u32
i9 [N]  managed-objective-secondary-formatted-text-component | FUN_142ed592c -> FUN_14080b034(dst+0x160) | IDENTIQUE a i2 (meme corps, seul l'offset differe: ADD RCX,0x160) | +0x160..+0x187 | u32 @0x160 + 4x{u32,u8}
i10 [N] managed-objective-is-new-and-unseen-component | FUN_142ed5510 | R(1) via FUN_1406cf008 -> byte[RBX+0x188] | +0x188 | u8
i11 [N] managed-objective-is-only-one-item-unlocked-component | FUN_142ed5530 | R(1) via FUN_1406cf008 -> byte[RBX+0x189] | +0x189 | u8
i12 [A] managed-objective-progress-component          | FUN_142ed575c | R(32) -> +0x18c | +0x18c | u32
i13 [A] managed-objective-required-progress-component | FUN_142ed5844 | R(32) -> +0x190 | +0x190 | u32
i14 [A] managed-objective-state-component             | FUN_142ed5948 | R(3) -> +0x194 | +0x194 | u32 (3 bits utiles)
i15 [A] managed-objective-parent-objective-component  | FUN_142ed5674 | R(32) -> +0x198 | +0x198 | u32
i16..i31 [A] managed-objective-sub-objective-entities-component (16 copies, MEME nom) | FUN_142ed5974 | R(32) GlobalID; ecrit [RDI + idx*4 + 0x19c], idx = MOVSXD [entry+0x8] | +0x19c..+0x1d8 (idx 0..15) | 16xu32
i32 [N] managed-objective-outro-phase-duration-component | FUN_142ed5634 | R(8) QUANTIFIE f32 via FUN_1406d84b4(min=0.0 XORPS, max=3.0 DAT_143cd84b8=0x40400000, width=8, flagA=0, flagB=1); pas=3/255 -> MOVSS [RBX+0x1dc] | +0x1dc | f32
i33 [N] managed-objective-forced-update-component     | FUN_142ed54f0 | R(1) via FUN_1406cf008 -> byte[RBX+0x1e0] | +0x1e0 | u8

Autorite vtable (re-confirmee par disassemble ce jour) : le corps partage i2/i9 = FUN_14080b034 (i2 ADD RCX,0x18 / i9 ADD RCX,0x160, meme CALL). i4 FUN_140dbe170 : SETA R8B sur CMP R9D,0x1 (gate version), ADD RCX,0x48. i32 : MOV [RSP+0x20],0x8 + MOVSS depuis [0x143cd84b8]=3.0f. read_memory 0x143cd84b8 = 00 00 40 40.

## Verification structurelle
PAVAGE de la struct dst, 0x00 -> 0x1e0, verifie contigu sans recouvrement :
0x00-0x07  i0    (2xu32)
0x08-0x17  i1    (4xf32 RGBA)   -> borne haute 0x18 = debut i2 : OK
0x18-0x3f  i2    (u32 valeur @0x18 + 4 slots {u32,u8} @0x1c..0x3b, pad->0x40)
0x40-0x43  i3    (u32)
0x44-0x47  PAD   (4 o, aucune feuille) — alignement avant la region 0x40-taille d'i4
0x48-0x147 i4    (4 slots x 0x40)
0x148-0x14b i4 masque (u32)
0x14c-0x14f i4 scalaire (u32) -> 0x150 = debut i5 : OK
0x150-0x153 i5   (u32)
0x154       i6   (u8) ; 0x155-0x157 PAD
0x158-0x15b i7   (u32)
0x15c-0x15f i8   (u32)
0x160-0x187 i9   (u32 valeur @0x160 + 4 slots, pad->0x188)
0x188       i10  (u8)
0x189       i11  (u8) ; 0x18a-0x18b PAD
0x18c-0x18f i12  (u32)
0x190-0x193 i13  (u32)
0x194-0x197 i14  (u32, 3 bits)
0x198-0x19b i15  (u32)
0x19c-0x1db i16-31 (16xu32 = 64 o ; 0x19c + 15*4 = 0x1d8, dernier champ 0x1d8..0x1db)
0x1dc-0x1df i32  (f32)
0x1e0       i33  (u8)

JONCTIONS CRITIQUES demandees, toutes CONFIRMEES sur pieces :
- i16-31 finit +0x1d8 (dernier idx=15), i32 a +0x1dc : contigu, aucun trou, aucun chevauchement (disasm FUN_142ed5974 ecrit +idx*4+0x19c ; FUN_142ed5634 ecrit +0x1dc).
- i1 RGBA +0x8..0x14 (4 MOVSS), i2 debute +0x18 : contigu (disasm FUN_142ed544c).
- i32 f32 [0x1dc..0x1df], i33 u8 +0x1e0 : contigu (disasm FUN_142ed5634/FUN_142ed54f0).
- i2 (fin 0x40) -> i3 (+0x40) et i4 (fin scalaire 0x14f) -> i5 (+0x150) : contigus.
- i9 (fin 0x188) -> i10 (+0x188), i10 (+0x188) i11 (+0x189) : contigus.

CORRESPONDANCE largeur-lue / taille-champ : R(1)->u8 (i6,i10,i11,i33) OK ; R(3)->u32 (i14) OK zero-etendu ; R(4)->u32 (i8) OK ; R(8)->u32 (i7) OK ; R(8) quantifie->f32 (i1 par canal, i32) OK ; R(32)->u32 (i3,i5,i12,i13,i15,i16-31) OK ; variable->sous-struct (i2,i9,i4) OK. AUCUNE incoherence de largeur.

TROUS = uniquement du padding d'alignement C (0x44-0x47, 0x155-0x157, 0x18a-0x18b) : jamais lus, jamais ecrits, non imputables a une feuille manquante. VERDICT : la struct pave proprement, i0 (nouvellement borne a +0x0/+0x4) ferme le bas, i33 (+0x1e0) ferme le haut. Aucun recouvrement, aucune largeur douteuse.

## Analyse du deficit
SOMME DES LARGEURS des 10 feuilles vs deficits mesures (Oddball 103 b, KOTH 66 b).

Feuilles FIXES (largeur constante quand PRESENTES au masque) :
  i6=1, i7=8, i8=4, i10=1, i11=1, i32=8, i33=1  ->  SOUS-TOTAL FIXE = 24 bits (invariant, identique tous modes).
Feuilles VARIABLES (data-dependantes) :
  i2 : 1 b (absent, bit presence=0) / >=36 b present (1+32+3, count 0) + count x element
  i9 : idem i2
  i4 : >=5 b (masque 4 + scalaire 1 si version>1, 0 slot arme) / >=36 b (scalaire 32 si version<=1) + slots armes (sel1 6b, sel2 37b, sel3 9b, sel4 14b, sel5 variable/recursif)

KOTH = 66 b. 66 - 24 (7 fixes presentes) = 42 b pour {i2,i9,i4}.
  Lecture PROPRE et EXACTE : i2 present count 0 (36) + i9 branche absente (1) + i4 minimal version>1 sans slot (5) = 42.  => 24 + 42 = 66. CORRESPONDANCE EXACTE.
  (Interpretation coherente : KOTH a UN texte d'objectif principal, pas de texte secondaire, filtre d'interaction vide — la colline a un nom a afficher.)

Oddball = 103 b. 103 - 24 = 79 b pour {i2,i9,i4}.
  Compatible avec les deux textes presents (i2+i9 ~72) + contenu de filtre/formatage (~7) ; ou un texte plus riche. Le decoupage EXACT est DATA-DEPENDANT et NON pinnable en statique (c'est la nature meme des feuilles variables).

DIAGNOSTIC : le coeur fixe (24 b) est verrouille et commun aux deux modes. L'ECART inter-mode 103-66 = 37 b vit ENTIEREMENT dans les 3 feuilles a longueur variable (i2 texte principal, i9 texte secondaire, i4 filtre). C'est la SIGNATURE d'un contenu variable : aucun stub a largeur fixe ne peut satisfaire 66 ET 103 simultanement — seule la vraie grammaire variable fera lander les deux. Le fait que KOTH tombe EXACTEMENT sur 66 avec {tous fixes + 1 texte + filtre minimal} ferme la comptabilite : AUCUN indice d'une 11e feuille manquante.

CA COLLE (fort indice que le cablage fera lander), avec une reserve honnete : le 24 fixe suppose les 7 fixes toutes au masque du record. Si un mode omet une fixe de son masque, l'arithmetique se decale (mais la conclusion qualitative — l'ecart est dans les variables — tient). La re-mesure Go DOIT etre par-record et respecter le masque (ne lire que les composants presents), exactement comme le traverseur actuel.

## SPEC DE CABLAGE Go
SPEC DE CABLAGE Go — dispatch consumeByName (traverse.go:194), ordre i0..i33. Lecteurs : br.ReadBit() = R(1) bool ; br.ReadBits(n) = R(n). Les 10 case a AJOUTER (ancres i0/i1/i3/i5/i12-15/i16-31 supposees deja presentes dans le worktree cadre) :

// i2 (variable) — REUTILISE consumeObjectiveFormattedText (components_batch3.go:22), deja bit-exact avec FUN_14080b034
case "managed-objective-formatted-text-component":
    consumeObjectiveFormattedText(br)
    return variant, nil, true

// i9 (variable) — MEME deser que i2
case "managed-objective-secondary-formatted-text-component":
    consumeObjectiveFormattedText(br)
    return variant, nil, true

// i4 (variable/recursif) — nouveau helper (voir ci-dessous), passe `level` (= version param_4 EXE)
case "managed-objective-interaction-filter-component":
    consumeObjectiveInteractionFilter(br, level)
    return variant, nil, true

// i6
case "managed-objective-enabled-component":
    br.ReadBit(); return variant, nil, true
// i7
case "managed-objective-priority-component":
    br.ReadBits(8); return variant, nil, true
// i8
case "managed-objective-message-type-component":
    br.ReadBits(4); return variant, nil, true
// i10
case "managed-objective-is-new-and-unseen-component":
    br.ReadBit(); return variant, nil, true
// i11
case "managed-objective-is-only-one-item-unlocked-component":
    br.ReadBit(); return variant, nil, true
// i32 (R(8) quantifie ; pour le bit-count = 8 bits ; deq f32 = 3.0*n/255 si valeur voulue)
case "managed-objective-outro-phase-duration-component":
    br.ReadBits(8); return variant, nil, true
// i33
case "managed-objective-forced-update-component":
    br.ReadBit(); return variant, nil, true

NOUVEAU HELPER (i4), miroir de FUN_140dbe400 :
func consumeObjectiveInteractionFilter(br *BitReader, level uint32) {
    mask := br.ReadBits(4)                 // FUN_140dbe598
    if level > 1 { br.ReadBits(1) } else { br.ReadBits(32) } // scalaire gate version
    for k := uint(0); k < 4; k++ {
        if (mask>>k)&1 == 0 { continue }
        sel := br.ReadBits(4)              // selecteur du slot arme
        switch sel {
        case 0:                            // rien
        case 1: br.ReadBit(); br.ReadBit()          // commun + FUN_141e9d6d0 R(1)
        case 2: br.ReadBit(); br.ReadBits(32)       // commun + FUN_141e9d120 32xR(1)
        case 3: br.ReadBit(); br.ReadBits(4)        // commun + FUN_1407ef804 R(4) biais -1
        case 4: br.ReadBit(); br.ReadBits(9)        // commun + FUN_141e9d670 R(9)
        case 5:                                     // commun + FUN_141e9a9c0 R(3)count + recursion
            br.ReadBit()
            cnt := br.ReadBits(3)
            for i := uint64(0); i < cnt; i++ { consumeObjectiveInteractionFilterNode(br) }
        default:                            // sel 6..15 = assert EXE (FUN_141e98c70), flux invalide
        }
    }
}

MAINTENANCE : consumeObjectiveFormattedText porte //nolint:unused + un commentaire "GARDEE SANS APPELANT" (batch3.go:15-21) dont la CONDITION DE RETRAIT est justement "branchee ou supprimee quand ti=11 sera decode". Le cabler dans les 2 case ci-dessus SATISFAIT cette condition : retirer le nolint et mettre a jour ce commentaire dans le meme commit. Garde-rail G1 (ecs_table_guard_test.go) : passer les lignes i2/i9 de statut "deser_non_cable" a "porte", et les 8 autres de "non_porte" a "porte", sinon le test G1 echoue (code <-> table).

Le gate version d'i4 : `level` doit valoir le param_4 (version composant) que l'EXE passe a FUN_140dbe170. KOTH landant a i4=5 b corrobore version>1 (branche R(1)). Filet defensif : si la re-mesure sous-lit de 31 b sur i4, basculer sur la branche R(32) (version<=1).

## Bloquants / vigilance
- AUCUN bloquant dur : les 10 feuilles sont TOUTES resolues (statut NON_RESOLUE = zero) et re-verifiees sur pieces ce jour (disassemble/decompile + read_memory). Le cablage peut partir.
- RESIDU NON BLOQUANT (i4 sel=5) : la grammaire du sous-noeud RECURSIF (consumeObjectiveInteractionFilterNode, via FUN_141e9a9c0 -> *vtable+0x28) n'a pas ete re-verifiee independamment ici — je la reprends de la resolution i4 (R(3) count + recursion). Elle ne fire PAS dans les frames minimales : KOTH lande avec i4=5 b (0 slot arme) et Oddball(79 b) se boucle sans sel5. A confirmer SEULEMENT si la re-mesure sous-lit encore sur i4 apres cablage (signe qu'un slot sel=5 est present).
- RESERVE de methode (pas un bloquant) : le sous-total fixe de 24 b suppose les 7 feuilles fixes presentes au masque du record. La re-mesure doit rester PAR-RECORD et ne lire que les composants presents au masque (comportement actuel du traverseur, traverse.go). Un masque qui omet une fixe decale l'arithmetique sans invalider le diagnostic (l'ecart inter-mode reste dans i2/i9/i4).
- POINT DE VIGILANCE cablage : ce worktree (LevelUp-go-migration) n'a AUCUN case ti=11 dans consumeByName (grep negatif). Les ancres i0/i1/i3/i5/i12-15/i16-31 vivent dans le worktree LevelUp-wt-ti11-cadre. Verifier, cote cadre, que ces ancres sont bien cablees AVANT de mesurer le deficit des 10 — sinon le deficit englobe aussi des ancres et la comptabilite 24/42 ne tient plus.
- DEPENDANCE gate G1 : le test ecs_table_guard exige que toute etiquette case de consumeByName soit dans ecs_table.tsv avec le bon statut. Mettre a jour les 10 lignes ti=11 (statut -> porte) dans le meme commit, sinon build de test rouge (rappel : je ne compile pas, une autre session le fait).