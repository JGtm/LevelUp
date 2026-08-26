package filmdec

// inventory_delta_ammo.go — LES MUNITIONS SUIVIES DANS LES PAQUETS DELTA. Prolonge
// inventory_delta.go (seuil de taille du dépôt, CLAUDE.md n°5) : mêmes ancre, même marche,
// même règle — c'est le déserialiseur qui publie.
//
// LE VRAI GISEMENT DU CHANTIER EST ICI, et pas dans les grenades. Sur le film témoin,
// i22 ne transmet que 120 fois quand les chargeurs en transmettent 1 156 et les réserves 99 :
// un facteur 10 sur la grandeur la plus visible d'une fiche joueur (étude du 2026-08-24 §2.4).
//
//	weapon-state-ammo              (i30/i33/i36/i39)  porte active-bas puis R(8) chargeur,
//	                                                  porte active-bas puis R(12) fraction.
//	weapon-state-rounds-inventory  (i31/i34/i37/i40)  R(11) réserve, sans porte.
//
// CE QUE LA CALIBRATION DIT, ET CE QU'ELLE NE DIT PAS. Le chargeur est un R(8) : le FORMAT
// autorise 0..255. Les 1 156 lectures du film témoin tiennent TOUTES dans 1..80, sans une
// exception ; la réserve est un R(11) (0..2047) et ses 99 lectures tiennent dans 0..240. Un
// curseur mal placé produirait une loi UNIFORME sur toute la plage du champ : la probabilité
// d'observer 1 156 valeurs consécutives sous 81 par hasard est de l'ordre de (81/256)^1156,
// rigoureusement nulle. C'est ce qui CALIBRE ces composants, que le handoff du 2026-07-27
// classait « largeurs mesurées, valeurs NON calibrées ».
//
// LES BORNES CI-DESSOUS SONT DES MESURES, PAS DES LOIS DU FORMAT — et elles sont écrites plus
// larges que la mesure exprès. Une arme au chargeur de 90 balles existerait sans rien invalider ;
// ce qui invaliderait, c'est une distribution qui remplit le champ. Les garde-rails vérifient
// donc une ENVELOPPE, pas un maximum observé.
//
// CE QUE CE CANAL NE DONNE PAS : l'IDENTITÉ de l'arme (i43/i44, 14 et 9 annonces sur 171 851
// records du film témoin). Les munitions suivies ici sont celles d'une arme nommée par le
// canal des images-clés — jamais par celui-ci.

// Les étiquettes de registre des deux composants de munitions. Résolues par NOM, comme les
// composants de grenade.
const (
	invDeltaAmmoName   = "weapon-state-ammo"
	invDeltaRoundsName = "weapon-state-rounds-inventory"
)

// invDeltaWeaponSlots est le nombre d'emplacements d'arme décrits par l'archétype biped. Seuls
// les DEUX premiers portent une arme — les deux autres sont structurellement vides, et cette
// vacuité sert de critère de parse au canal des images-clés. Les quatre sont lus ici : les
// écarter en amont supposerait acquis ce que la mesure doit pouvoir contredire.
const invDeltaWeaponSlots = 4

// invDeltaAmmoAcc est ce qu'un `weapon-state-ammo` a publié pour UN emplacement, sur UN record.
type invDeltaAmmoAcc struct {
	// Read dit que le composant a été consommé sur ce record. Faux = non transmis : les autres
	// champs ne veulent alors rien dire.
	Read bool
	// HasMag / Mag : le chargeur. HasMag faux = la porte était fermée, le film n'écrit RIEN
	// pour ce champ. Ce n'est PAS zéro — publier 0 affirmerait un chargeur vide.
	HasMag bool
	Mag    uint32
	// HasFrac / FracQ : le quantum brut de fraction de charge.
	HasFrac bool
	FracQ   uint32
}

// InventoryDeltaAmmo est l'état de munitions d'UN emplacement d'arme, tel qu'un paquet delta
// le transmet. Les trois grandeurs sont indépendamment optionnelles : elles viennent de DEUX
// composants distincts, qu'un record peut annoncer séparément.
type InventoryDeltaAmmo struct {
	// WeaponSlot est le rang de l'emplacement dans l'archétype (0 et 1 portent une arme).
	WeaponSlot int
	// Mag est le chargeur, ou nil si le film n'écrit rien pour ce champ.
	Mag *uint32
	// FracQ est le quantum R(12) BRUT de la fraction, ou nil. Brut, parce que la
	// déquantification appartient à la couche qui sait ce qu'elle affiche.
	//
	// ELLE COMPTE CE QUI A ÉTÉ CONSOMMÉ, pas ce qui reste — deux témoins concordants, cf.
	// AmmoSlot.Gauge (replay/inventory.go). Un client qui dessine une charge RESTANTE doit
	// donc afficher le complément.
	FracQ *uint32
	// Res est la réserve (R(11)), ou nil si `weapon-state-rounds-inventory` n'était pas au
	// masque de ce record.
	Res *uint32
}

// captureAmmo range la dernière publication d'un `weapon-state-ammo` sous son emplacement.
func (sc *invDeltaScanner) captureAmmo(slot int) {
	if slot < 0 || slot >= invDeltaWeaponSlots || !sc.lastAmmo.Read {
		return
	}
	acc := sc.ammo[slot]
	acc.Read, acc.HasMag, acc.Mag = true, sc.lastAmmo.HasMag, sc.lastAmmo.Mag
	acc.HasFrac, acc.FracQ = sc.lastAmmo.HasFrac, sc.lastAmmo.FracQ
	sc.ammo[slot] = acc
	sc.lastAmmo = invDeltaAmmoAcc{}
}

// captureRounds range la dernière réserve publiée sous son emplacement.
func (sc *invDeltaScanner) captureRounds(slot int) {
	if slot < 0 || slot >= invDeltaWeaponSlots || !sc.lastRoundsRead {
		return
	}
	sc.rounds[slot] = sc.lastRounds
	sc.roundsRead[slot] = true
	sc.lastRoundsRead = false
}

// collectAmmo assemble les emplacements lus sur ce record, et applique les garde-rails
// d'enveloppe. Rend vrai si au moins un emplacement porte quelque chose.
func (sc *invDeltaScanner) collectAmmo(rec *InventoryDelta) bool {
	for k := 0; k < invDeltaWeaponSlots; k++ {
		acc, hasRes := sc.ammo[k], sc.roundsRead[k]
		if !acc.Read && !hasRes {
			continue
		}
		out := InventoryDeltaAmmo{WeaponSlot: k}
		if acc.Read {
			sc.st.AmmoRead++
			if acc.HasMag {
				sc.st.MagRead++
				// LA CORROBORATION : sur ce record, i22 a-t-il rendu une lecture PLAUSIBLE ?
				// Si oui, le curseur a passé un test indépendant AVANT d'arriver ici, et le
				// chargeur qui suit hérite de cette caution. Séparer les deux populations est
				// la seule façon de savoir si un dépassement d'enveloppe est une arme
				// inhabituelle ou un curseur perdu.
				corroborated := rec.Grenades != nil
				if corroborated {
					sc.st.MagCorroborated++
				}
				if acc.Mag > invDeltaMagEnvelope {
					sc.st.MagOutOfEnvelope++
					if corroborated {
						sc.st.MagOutOfEnvelopeCorroborated++
					}
				} else {
					m := acc.Mag
					out.Mag = &m
				}
			}
			if acc.HasFrac {
				f := acc.FracQ
				out.FracQ = &f
			}
		}
		if hasRes {
			sc.st.RoundsRead++
			if sc.rounds[k] > invDeltaResEnvelope {
				sc.st.ResOutOfEnvelope++
			} else {
				r := sc.rounds[k]
				out.Res = &r
			}
		}
		if out.Mag == nil && out.FracQ == nil && out.Res == nil {
			continue
		}
		rec.Ammo = append(rec.Ammo, out)
	}
	return len(rec.Ammo) > 0
}

// refuseAmmoIfContaminated est LA PORTE PAR FILM du canal munitions, et la décision la plus
// importante de ce fichier.
//
// CE QUE LA MESURE MULTI-FILMS A TROUVÉ, et qui n'était pas prévu. Sur le film témoin
// `000d5950`, le balayage reproduit l'étude du 2026-08-24 AU CHIFFRE PRÈS : 563 lectures à
// l'emplacement 0 et 593 au 1, toutes dans 1..80. Étendu à 25 films, le tableau est BIMODAL :
//
//	16 films PROPRES      max 36, 59, 60 ou 80 — jamais au-delà, p90 <= 71
//	 9 films CONTAMINÉS   max 134 à 250, dont 5 avec p90 entre 147 et 220
//
// Sur ces cinq-là, 10 à 20 % des chargeurs lus sont hors de toute borne de jeu : ce n'est pas
// une arme inhabituelle, c'est une marche qui a dérivé. ET C'EST CE QUI INTERDIT DE FILTRER
// VALEUR PAR VALEUR : sur un film où le curseur dérive, les lectures qui TOMBENT sous
// l'enveloppe n'en sont pas plus vraies — elles sont juste indiscernables. Publier les unes en
// jetant les autres fabriquerait des chargeurs plausibles et faux, ce qui est pire qu'un trou.
//
// La porte est donc TOUT OU RIEN, par film : au-delà du taux de contamination, le canal
// munitions de ce film est refusé en bloc et le rejeu retombe sur les images-clés — la
// dégradation gracieuse que l'ADR impose, jamais une donnée devinée.
//
// LES GRENADES NE SONT PAS CONCERNÉES : elles portent leur propre test réfutable (compteur
// == 4) et leur contrôle croisé i22 <-> i47 (100,0 % sur 1 925 records), qui tiennent sur les
// mêmes films. Un film peut donc rendre ses grenades et refuser ses munitions.
func (sc *invDeltaScanner) refuseAmmoIfContaminated() {
	if sc.st.MagRead < invDeltaAmmoMinSample {
		return
	}
	rate := float64(sc.st.MagOutOfEnvelope) / float64(sc.st.MagRead)
	if rate < invDeltaAmmoRefusalRate {
		return
	}
	sc.st.AmmoRefused = true
	for i := range sc.out {
		sc.out[i].Ammo = nil
	}
	// Les lectures qui ne portaient QUE des munitions n'ont plus rien à dire.
	kept := sc.out[:0]
	for _, r := range sc.out {
		if r.Grenades != nil || r.SelRead {
			kept = append(kept, r)
		}
	}
	sc.out = kept
	sc.st.Emitted = len(sc.out)
}

const (
	// invDeltaAmmoMinSample : en deçà, le taux mesure le hasard, pas la contamination.
	invDeltaAmmoMinSample = 200
	// invDeltaAmmoRefusalRate : le taux de dépassement au-delà duquel le canal munitions du
	// film est refusé. LE SEUIL N'EST PAS SENSIBLE — sur le corpus de 25 films les taux se
	// séparent en deux paquets nettement disjoints (films propres à 0,00 %, films contaminés
	// bien au-dessus de 1 %), et toute valeur entre les deux donne le MÊME classement. On
	// prend 1 %, la valeur ronde la plus basse qui laisse passer les films propres.
	invDeltaAmmoRefusalRate = 0.01
)

// LES DEUX ENVELOPPES, et pourquoi elles ne valent pas les maxima mesurés.
//
//	chargeur  champ R(8),  0..255   mesuré 1..80   enveloppe 120
//	réserve   champ R(11), 0..2047  mesuré 0..240  enveloppe 400
//
// Le point n'est pas de rejeter la valeur 81 : c'est de rejeter une DISTRIBUTION qui remplit
// le champ. L'enveloppe est posée bien au-dessus du maximum observé pour qu'une arme inconnue
// du corpus passe, et bien en dessous du plafond du champ pour qu'un curseur perdu rougisse.
// Le taux de dépassement est publié (`MagOutOfEnvelope`, `ResOutOfEnvelope`) : c'est LUI qu'on
// surveille, pas une seule valeur.
const (
	invDeltaMagEnvelope = 120
	invDeltaResEnvelope = 400
)
