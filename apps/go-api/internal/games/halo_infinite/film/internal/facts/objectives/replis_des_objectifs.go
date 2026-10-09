package objectives

// replis_des_objectifs.go — LES COMPTES DES REPLIS DU LECTEUR D OBJECTIFS, EN DONNEES (lot J8.7 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision 1 du superviseur, 2026-09-27).
//
// # POURQUOI CE PAQUET NE DECLENCHE PAS LE COMPTEUR LUI-MEME
//
// Meme raison que `killsource` : il pourrait importer le registre `facts/fallback`, il ne le doit pas —
// l empreinte de sa couche (`objectives.Rev`) porterait alors chaque modification du registre, et
// toute entree neuve rendrait les faits du parc « a redecoder » (ADR 0034 amende, DU-2 (c)). Les
// replis se comptent donc ICI, en entiers nommes que CE QUE LE PAQUET REND DEJA transporte : le
// balayage du statborg ([StatRecordsAvecReplis]) et le resolveur d identite par manche
// ([RoundIdentity.ComptesDesReplis]). La cuisson les verse par sa table
// (`replay/versement_des_replis.go`).
//
// # UN REPLI SE COMPTE A LA CONSULTATION (lot J8.7-bis, 2026-09-28)
//
// `repli_emission_hors_domaine_jetee` (series nommees, `named_series.go`) se DECLENCHE A LA
// CONSULTATION, dans des lectures que plusieurs calques refont : il se compte par EVENEMENT
// DISTINCT dans un enregistreur partage par le document ([ReplisALaConsultation]), dont
// [ReplisALaConsultation.ComptesDesReplis] rend le dernier champ ci-dessous.

// ComptesDesReplis compte les declenchements des replis d `objectives` portes par un resultat.
type ComptesDesReplis struct {
	// EnregistrementsAbandonnes : `repli_enregistrement_statborg_abandonne` — en-tetes
	// d enregistrement reconnus dont aucun composant ne se decode ou dont un compteur sort du
	// domaine, abandonnes (`grammar/signaux/statborg.go`, porte par [StatRecordsAvecReplis]).
	EnregistrementsAbandonnes int
	// ComposantsArretes : `repli_composants_statborg_arretes` — enregistrements GARDES dont la
	// lecture des composants s est arretee avant le dernier annonce (`grammar/signaux/statborg.go`).
	ComposantsArretes int
	// TablesIdentiteVides : `repli_table_identite_vide` — ponts par instants de mort sans aucune
	// mort, rendus vides (`slotidentity_deaths.go`).
	TablesIdentiteVides int
	// DebutsDeMancheAuMinimum : `repli_debut_de_manche_au_minimum` — manches dont le debut vient du
	// minimum des instants declares, faute de consensus (`slotidentity_rounds.go`).
	DebutsDeMancheAuMinimum int
	// SlotsAbandonnes : `repli_slot_abandonne_au_premier_arrive` — attributions de la feuille
	// abandonnees parce que le slot ou le joueur etait deja pris (`slotidentity_rounds.go`).
	SlotsAbandonnes int
	// EmissionsHorsDomaineJetees : `repli_emission_hors_domaine_jetee` — emissions DISTINCTES
	// (serie, instant) jetees par le filtre de domaine des series nommees (`named_series.go`),
	// relevees par un [ReplisALaConsultation].
	EmissionsHorsDomaineJetees int
}

// Plus rend la somme champ a champ des deux rapports. Elle nomme chaque champ :
// `TestPlusSommeChaqueChampDesComptes` le tient par reflexion.
func (c ComptesDesReplis) Plus(d ComptesDesReplis) ComptesDesReplis {
	return ComptesDesReplis{
		EnregistrementsAbandonnes:  c.EnregistrementsAbandonnes + d.EnregistrementsAbandonnes,
		ComposantsArretes:          c.ComposantsArretes + d.ComposantsArretes,
		TablesIdentiteVides:        c.TablesIdentiteVides + d.TablesIdentiteVides,
		DebutsDeMancheAuMinimum:    c.DebutsDeMancheAuMinimum + d.DebutsDeMancheAuMinimum,
		SlotsAbandonnes:            c.SlotsAbandonnes + d.SlotsAbandonnes,
		EmissionsHorsDomaineJetees: c.EmissionsHorsDomaineJetees + d.EmissionsHorsDomaineJetees,
	}
}

// ComptesDesReplis rend les comptes de replis accumules par la CONSTRUCTION de ce resolveur
// (pont par instants de mort, bornes de manche, completion par la feuille). Resolveur construit a
// la main ([FlatRoundIdentity]) : zero.
func (ri RoundIdentity) ComptesDesReplis() ComptesDesReplis { return ri.replis }
