package filmdec

// Déquantificateur À EXTRÉMITÉS EXACTES — port de FUN_1406d84b4.
//
// CE N'EST PAS celui de quantize.go. `ReadQuantizedVec3` (FUN_140c1e978) fait le calcul
// simple « milieu de bucket » v = min + step*(q+0.5) ; FUN_1406d84b4 ajoute DEUX règles que
// la formule simple n'a pas, et qui changent la valeur rendue :
//
//   - `excl` retire un niveau (levels = 2^W - 1 au lieu de 2^W) ET force le quantum
//     médian à la valeur milieu EXACTE, calculée en double ;
//   - `endpointExact` mappe q=0 sur min et q=levels-1 sur max EXACTEMENT, et étale les
//     quanta intermédiaires sur levels-2 pas.
//
// VÉRIFIÉ LE 2026-07-26 : les 113 instructions de 0x1406d84b4..0x1406d8647 ont été relues
// au désassemblage pendant cette session (pas reprises d'un handoff — le paquet a déjà
// porté un « CONFIRMED bit-exact » faux). Correspondance instruction -> ligne de code :
//
//	1406d84f1  ADD [R11+0x2c],EBX ; SHL/SHR  -> q = R(W), MSB-first
//	1406d8513  SHL EDI,CL ; LEA ECX,[RDI-1] ; CMOVZ ECX,EDI
//	                                        -> levels = excl ? (1<<W)-1 : (1<<W)
//	1406d851e  CMP byte [RSP+0x38],0 ; JNZ   -> aiguillage endpointExact
//	1406d8525  (endpointExact == 0)          -> v = (q*step + min) + step*0.5
//	1406d8644  MOVAPS XMM0,XMM2              -> q == 0        : v = min
//	1406d863c  MOVAPS XMM0,XMM3              -> q == levels-1 : v = max
//	1406d858c  DIVSS/(levels-2) ...          -> v = (q-1)*step2 + (step2*0.5 + min)
//	1406d8558  DEC ECX ; LEA EAX,[R9+R9] ; CMP ; JZ 1406d85be
//	                                        -> si excl && 2q == levels-1 :
//	1406d85be  CVTSS2SD x2 ; ADDSD ; MULSD [143cd8910]=0.5d ; CVTPD2PS
//	                                        -> v = float32((min+max) * 0.5 en double)
//
// Constantes lues en mémoire cette session : 143cd84b0 = 0x3F000000 = 0.5f ;
// 143cd8910 = 0x3FE0000000000000 = 0.5 (double).
//
// L'ORDRE DES OPÉRATIONS EST REPRODUIT À L'IDENTIQUE (associations comprises) : une
// réécriture « équivalente » en algèbre ne l'est pas en float32, et c'est justement le
// point milieu de la santé qui sert de détecteur (cf. quantize_endpoint_test.go).

// Bornes de déquantification des composants de vitalité, lues en mémoire le 2026-07-26 aux
// adresses citées par les désérialiseurs (et non recopiées d'un document).
const (
	// VitalityBodyMin / VitalityBodyMax : DAT_143cd84ec = 0x000080BF = -1.0f et
	// DAT_143cd8374 = 0x0000803F = +1.0f, arguments de FUN_140fb8978 (i4 body-vitality).
	// La zone NÉGATIVE existe donc dans la sérialisation ; ce qu'elle signifie
	// (sur-dégât ? marge ?) n'est PAS établi par le binaire.
	VitalityBodyMin float32 = -1
	VitalityBodyMax float32 = 1
	// VitalityShieldMin / VitalityShieldMax : XORPS XMM2 (= 0.0f) et
	// DAT_143cd893c = 0x00008040 = 4.0f, arguments de FUN_140d50cbc (i5 shield-vitality).
	// 0..4 est une plage de SÉRIALISATION : elle ne prouve pas qu'un bouclier x4 soit
	// atteignable en jeu.
	VitalityShieldMin float32 = 0
	VitalityShieldMax float32 = 4
	// RoundTimerMax : DAT_143cd8a84 = 0x470CA000 = 36000.0f, borne haute des deux
	// scalaires R(16) de game-engine-round-timer (via FUN_140d580d0). Résolution
	// 36000/65534 ≈ 0,549 s.
	RoundTimerMax float32 = 36000
)

// Largeurs des champs déquantifiés par FUN_1406d84b4 sur les composants portés ici.
const (
	vitalityBodyBits   uint = 8
	vitalityShieldBits uint = 8
	roundTimerBits     uint = 16
)

// DequantEndpoint applique FUN_1406d84b4 à un quantum DÉJÀ LU sur w bits (MSB-first).
//
// La lecture des bits est laissée à l'appelant : le paquet est écrit en « sauteurs de
// bits » et les désérialiseurs lisent déjà q correctement — ce qui manquait était
// uniquement la conversion en valeur.
//
// Hors domaine : si q > levels-1 (quantum impossible pour la largeur annoncée), la formule
// est appliquée telle quelle, comme le fait le moteur — aucun clamp n'est ajouté, sans quoi
// un décodage faux passerait pour une valeur plausible.
func DequantEndpoint(q uint64, min, max float32, w uint, excl, endpointExact bool) float32 {
	levels := uint64(1) << w
	if excl {
		levels-- // 1406d8518 LEA ECX,[RDI-1] / CMOVZ
	}
	var v float32
	switch {
	case endpointExact && q == 0: // 1406d8644 MOVAPS XMM0,XMM2
		v = min
	case endpointExact && q == levels-1: // 1406d863c MOVAPS XMM0,XMM3
		v = max
	case endpointExact: // 1406d858c
		step2 := (max - min) / float32(levels-2)
		v = float32(q-1)*step2 + (step2*0.5 + min)
	default: // 1406d8525
		step := (max - min) / float32(levels)
		v = (float32(q)*step + min) + step*0.5
	}
	// 1406d8558 : la règle du point milieu s'applique aux DEUX branches, et seulement
	// quand excl est armé. C'est elle qui rend « santé q=127 -> 0.0 » exact ; en float32
	// la branche endpointExact seule donnerait ~1e-8 (mesuré dans le test).
	if excl && 2*q == levels-1 {
		v = float32((float64(min) + float64(max)) * 0.5) // 1406d85be, en DOUBLE
	}
	return v
}
