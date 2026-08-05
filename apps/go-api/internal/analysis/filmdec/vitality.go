package filmdec

// Extraction des VALEURS de vitalité (i4 santé, i5 bouclier) des records d'objet.
//
// Le paquet est écrit en « sauteurs de bits » : `consumeXxx(br)` avance le curseur sans
// rien rendre. Pour ouvrir la boîte sans casser ce contrat, chaque composant visé reçoit
// un `decodeXxx(br) T` qui lit EXACTEMENT les mêmes bits et rend la valeur ; le
// `consumeXxx` correspondant se réduit à un appel de ce decode. La grammaire ne vit donc
// qu'à UN endroit — pas de deuxième copie qui puisse diverger.
//
// Désérialiseurs et arguments relus au désassemblage le 2026-07-26 (cf. quantize_endpoint.go) :
//
//	i4 FUN_140fb8978 : DequantEndpoint(min=-1, max=+1, W=8, excl=1, endpointExact=1) -> +0x58
//	                   puis 3 x R(1) -> +0x5c, +0x5d, +0x5e.            Total 11 bits.
//	i5 FUN_140d50cbc : DequantEndpoint(min=0, max=4, W=8, excl=0, endpointExact=1)   -> +0x60
//	                   R(1) présence -> +0x6e ; si présent 2 x [R(1) ; si 1 -> R(12)]
//	                   R(16) inline -> +0x64
//	                   4 x R(1) -> +0x66, +0x67, +0x69, +0x68 (ordre NON séquentiel).
//	                                                          Total 25 / 27 / 39 / 51 bits.

// BodyVitality porte les champs du composant i4 object-body-vitality.
type BodyVitality struct {
	// Health est la valeur déquantifiée, dans [-1, +1]. Ce N'EST PAS une fraction de vie :
	// la zone négative existe dans la sérialisation et sa signification n'est pas établie.
	// Utiliser HealthFraction pour l'affichage.
	Health float32
	// Q est le quantum brut sur 8 bits, conservé pour les témoins (il ne dépend d'aucune
	// hypothèse de déquantification).
	Q uint8
	// F5c, F5d, F5e sont les trois drapeaux qui suivent (+0x5c/+0x5d/+0x5e). Sémantique
	// non établie.
	F5c, F5d, F5e bool
}

// ShieldVitality porte les champs du composant i5 object-shield-vitality.
type ShieldVitality struct {
	// Shield est la valeur déquantifiée dans [0, 4]. 0..4 est la plage de SÉRIALISATION ;
	// rien dans le binaire ne dit qu'un bouclier x4 soit atteignable en jeu.
	Shield float32
	// Q est le quantum brut sur 8 bits.
	Q uint8
	// RegenPresent = bit +0x6e ; Regen0/Regen1 les deux champs R(12) optionnels qui le
	// suivent (présents seulement si HasRegen0/HasRegen1).
	RegenPresent         bool
	HasRegen0, HasRegen1 bool
	Regen0, Regen1       uint16
	// Block64 est le mot R(16) inline (+0x64).
	Block64 uint16
	// F66, F67, F69, F68 sont les quatre drapeaux de queue, DANS L'ORDRE DU BINAIRE
	// (+0x69 avant +0x68) — ne pas « corriger » en séquentiel.
	F66, F67, F69, F68 bool
}

// HealthFraction convertit la valeur i4 en fraction de vie affichable [0, 1].
// La zone négative est repliée sur 0 : elle n'est PAS interprétée (aucune preuve, dans le
// binaire, qu'elle encode un sur-dégât plutôt qu'autre chose).
func HealthFraction(v float32) float32 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

// ShieldFraction convertit la valeur i5 en fraction de bouclier standard [0, 1] ;
// ShieldOverfill rend ce qui dépasse (le « surbouclier » éventuel), également en fractions
// de bouclier standard. SUPPOSÉ, et dit comme tel : que 1.0 soit le bouclier plein. La
// plage 0..4 est une plage de sérialisation, pas une échelle de jeu établie.
func ShieldFraction(v float32) float32 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

// ShieldOverfill rend la part au-delà du bouclier standard (0 si aucune).
func ShieldOverfill(v float32) float32 {
	if v <= 1 {
		return 0
	}
	return v - 1
}

// decodeObjectBodyVitality lit i4 (FUN_140fb8978) et rend ses champs.
func decodeObjectBodyVitality(br *BitReader) BodyVitality {
	q := uint8(br.ReadBits(vitalityBodyBits))
	return BodyVitality{
		Health: DequantEndpoint(uint64(q), VitalityBodyMin, VitalityBodyMax, vitalityBodyBits, true, true),
		Q:      q,
		F5c:    br.ReadBit(),
		F5d:    br.ReadBit(),
		F5e:    br.ReadBit(),
	}
}

// decodeObjectShieldVitality lit i5 (FUN_140d50cbc) et rend ses champs.
func decodeObjectShieldVitality(br *BitReader) ShieldVitality {
	q := uint8(br.ReadBits(vitalityShieldBits))
	out := ShieldVitality{
		Shield: DequantEndpoint(uint64(q), VitalityShieldMin, VitalityShieldMax, vitalityShieldBits, false, true),
		Q:      q,
	}
	if out.RegenPresent = br.ReadBit(); out.RegenPresent { // +0x6e
		if out.HasRegen0 = br.ReadBit(); out.HasRegen0 { // FUN_1411b1ac0 #1 (+0x6a)
			out.Regen0 = uint16(br.ReadBits(12))
		}
		if out.HasRegen1 = br.ReadBit(); out.HasRegen1 { // FUN_1411b1ac0 #2 (+0x6c)
			out.Regen1 = uint16(br.ReadBits(12))
		}
	}
	out.Block64 = uint16(br.ReadBits(16)) // +0x64
	out.F66 = br.ReadBit()
	out.F67 = br.ReadBit()
	out.F69 = br.ReadBit() // ordre du binaire : +0x69 AVANT +0x68
	out.F68 = br.ReadBit()
	return out
}

// RespawnTimer porte les champs de player-respawn-timer-component (ti=5 i1).
//
// FUN_140f3fb8c relu au désassemblage le 2026-07-26 :
//
//	R(1)  -> byte [state+0x01]   (actif)
//	R(10) -> word [state+0x02]   (SHR R9,0x36 = les 10 bits de tête)
//	R(10) -> word [state+0x06]
//
// Total 21 bits. AUCUNE déquantification : ce sont des entiers BRUTS (le sérialiseur
// keyframe FUN_142f0857c écrit les mêmes mots sans passer par FUN_1406d84b4). L'UNITÉ
// n'est donc PAS donnée par le binaire — cf. le protocole de calibration dans cmd/tmp_vitals.
type RespawnTimer struct {
	Active bool
	T0, T1 uint16
}

// decodePlayerRespawnTimer lit ti=5 i1 (FUN_140f3fb8c).
func decodePlayerRespawnTimer(br *BitReader) RespawnTimer {
	return RespawnTimer{
		Active: br.ReadBit(),
		T0:     uint16(br.ReadBits(10)),
		T1:     uint16(br.ReadBits(10)),
	}
}

// RoundTimer porte les champs de game-engine-round-timer-component (ti=0 i5).
//
// FUN_1407ee790 -> FUN_140d580d0 relus au désassemblage le 2026-07-26 :
//
//	2 x DequantEndpoint(min=0, max=DAT_143cd8a84=36000.0f, W=16, excl=0, endpointExact=1)
//	    -> [state+0x20] et [state+0x24]
//	puis FUN_1407f0354 (queue R(5), déjà portée dans le dispatch).
//
// Résolution 36000/65534 ≈ 0,549 s. L'ATTRIBUTION « écoulé vs restant » n'est PAS donnée
// par le binaire : c'est la PENTE mesurée sur le film qui la tranche (cf. cmd/tmp_vitals).
type RoundTimer struct {
	A, B   float32 // secondes, déquantifiées
	QA, QB uint16  // quanta bruts
	Tail   uint8   // queue R(5)
}

// decodeGameEngineRoundTimer lit ti=0 i5 (FUN_1407ee790).
func decodeGameEngineRoundTimer(br *BitReader) RoundTimer {
	qa := uint16(br.ReadBits(roundTimerBits))
	qb := uint16(br.ReadBits(roundTimerBits))
	return RoundTimer{
		A:    DequantEndpoint(uint64(qa), 0, RoundTimerMax, roundTimerBits, false, true),
		B:    DequantEndpoint(uint64(qb), 0, RoundTimerMax, roundTimerBits, false, true),
		QA:   qa,
		QB:   qb,
		Tail: uint8(br.ReadBits(5)),
	}
}
