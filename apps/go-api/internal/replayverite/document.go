package replayverite

// document.go — LA FORME LUE DE L'ARTEFACT PUBLIE, ET ELLE SEULE.
//
// Chaque structure ne porte que les cles que le banc lit (cf. doc.go : pas d'import du type
// producteur). Une cle renommee cote producteur sort ici a sa valeur zero ; `forme_test.go`
// l'attrape avant, en relisant `document_shape.golden`.

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Document est l'artefact de rejeu publie, reduit aux cles lues.
type Document struct {
	SchemaVersion   int     `json:"schemaVersion"`
	MatchID         string  `json:"matchId"`
	FrameCount      int     `json:"frameCount"`
	FrameIntervalMs int     `json:"frameIntervalMs"`
	Bounds          *Bornes `json:"bounds"`
	Tracks          []Piste `json:"tracks"`

	Shots            []Action `json:"shots"`
	Grenades         []Action `json:"grenades"`
	Pickups          []Action `json:"pickups"`
	WeaponChanges    []Action `json:"weaponChanges"`
	Abilities        []Action `json:"abilities"`
	EquipmentChanges []Action `json:"equipmentChanges"`

	GroundWeapons       []Objet `json:"groundWeapons"`
	EquipmentPlacements []Objet `json:"equipmentPlacements"`
	WeaponPads          []Objet `json:"weaponPads"`

	Vehicles       []Vehicule       `json:"vehicles"`
	Translocations []Translocation  `json:"translocations"`
	Roster         []Inscrit        `json:"roster"`
	Objectives     []ActionObjectif `json:"objectives"`
	ScoreTimeline  *ChronoScore     `json:"scoreTimeline"`
	FlagCarries    []Drapeau        `json:"flagCarries"`
	SkullCarries   []Portage        `json:"skullCarries"`
	BombCarries    []Portage        `json:"bombCarries"`
	BombStats      *StatsBombe      `json:"bombStats"`
	Identity       *Identite        `json:"identity"`
	Coverage       Couverture       `json:"coverage"`
}

// Bornes est l'enveloppe publiee des positions (`doc.bounds`).
type Bornes struct {
	MinX float64 `json:"minX"`
	MinY float64 `json:"minY"`
	MaxX float64 `json:"maxX"`
	MaxY float64 `json:"maxY"`
}

// Piste est UNE VIE d'un corps : un slot, un xuid (absent = vie anonyme), des points.
type Piste struct {
	Slot       int     `json:"slot"`
	XUID       string  `json:"xuid"`
	Points     []Point `json:"points"`
	StartFrame *int    `json:"startFrame"`
	EndFrame   int     `json:"endFrame"`
}

// Debut rend l'image de debut de la vie : `startFrame` quand il est publie, sinon le premier point.
func (p Piste) Debut() int {
	if p.StartFrame != nil {
		return *p.StartFrame
	}
	if len(p.Points) > 0 {
		return p.Points[0].T
	}
	return p.EndFrame
}

// Point est une position datee a l'image ; `z` est OPTIONNEL (des pistes le taisent).
type Point struct {
	T int      `json:"t"`
	X float64  `json:"x"`
	Y float64  `json:"y"`
	Z *float64 `json:"z"`
}

// Action est un evenement date porte par un slot de corps (tir, grenade, ramassage...).
type Action struct {
	T    int    `json:"t"`
	Slot *int   `json:"slot"`
	XUID string `json:"xuid"`
}

// Objet est un objet pose sur la carte (arme au sol, equipement, presentoir).
type Objet struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Vehicule porte ses echantillons de position et ses trajets.
type Vehicule struct {
	Slot    int           `json:"slot"`
	Samples []Echantillon `json:"samples"`
	Rides   []Trajet      `json:"rides"`
}

// Echantillon est une position de vehicule a l'image.
type Echantillon struct {
	T int      `json:"t"`
	X float64  `json:"x"`
	Y float64  `json:"y"`
	Z *float64 `json:"z"`
}

// Trajet est un passager (slot de corps) a bord, sur un intervalle d'images.
type Trajet struct {
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
	Slot int    `json:"slot"`
	XUID string `json:"xuid"`
}

// Translocation est une teleportation legitime publiee.
type Translocation struct {
	T int `json:"t"`
}

// Inscrit est une entree du roster et ses intervalles de presence.
type Inscrit struct {
	XUID     string     `json:"xuid"`
	Presence []Presence `json:"presence"`
}

// Presence est un intervalle de presence ; `toMax` borne un depart incertain.
type Presence struct {
	From  int  `json:"from"`
	To    int  `json:"to"`
	ToMax *int `json:"toMax"`
}

// Fin rend la borne haute la plus large de l'intervalle.
func (p Presence) Fin() int {
	if p.ToMax != nil && *p.ToMax > p.To {
		return *p.ToMax
	}
	return p.To
}

// ActionObjectif est une action d'objectif nommee (capture, prise, kill...).
type ActionObjectif struct {
	T    int    `json:"t"`
	XUID string `json:"xuid"`
	Stat string `json:"stat"`
}

// ChronoScore porte les courbes de score des camps et des joueurs.
type ChronoScore struct {
	Teams   []ScoreEquipe `json:"teams"`
	Players []ScoreJoueur `json:"players"`
}

// ScoreEquipe est la courbe d'un camp ; `teamId` absent = camp non situe.
type ScoreEquipe struct {
	TeamID *int  `json:"teamId"`
	Total  []Pas `json:"total"`
}

// ScoreJoueur porte les compteurs statborg d'un joueur.
type ScoreJoueur struct {
	XUID    string `json:"xuid"`
	Score   Serie  `json:"score"`
	Kills   Serie  `json:"kills"`
	Deaths  Serie  `json:"deaths"`
	Assists Serie  `json:"assists"`
}

// Serie est une courbe en escalier.
type Serie struct {
	Total []Pas `json:"total"`
}

// Finale rend la derniere valeur de la courbe (0 si vide).
func (s Serie) Finale() int { return derniere(s.Total) }

// Pas est une marche de courbe.
type Pas struct {
	T int `json:"t"`
	V int `json:"v"`
}

func derniere(p []Pas) int {
	if len(p) == 0 {
		return 0
	}
	return p[len(p)-1].V
}

// Drapeau porte les etats d'un drapeau ; seuls les etats `carried` nomment un porteur.
type Drapeau struct {
	Spans []EtatDrapeau `json:"spans"`
}

// EtatDrapeau est un etat de drapeau sur un intervalle.
type EtatDrapeau struct {
	State string  `json:"state"`
	T0    int     `json:"t0"`
	T1    int     `json:"t1"`
	XUID  *string `json:"xuid"`
}

// Portage est un portage de crane ou de bombe.
type Portage struct {
	XUID string `json:"xuid"`
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
}

// StatsBombe porte les stats de bombe par joueur.
type StatsBombe struct {
	Players []StatsBombeJoueur `json:"players"`
}

// StatsBombeJoueur : les detonations d'un joueur ; nil = non lues.
type StatsBombeJoueur struct {
	XUID        string `json:"xuid"`
	Detonations *int   `json:"detonations"`
}

// Identite porte la methode de nommage de chaque slot statborg.
type Identite struct {
	StatborgSlots []SlotStatborg `json:"statborgSlots"`
}

// SlotStatborg est un slot statborg d'une manche et la METHODE qui l'a nomme.
type SlotStatborg struct {
	Slot int    `json:"slot"`
	XUID string `json:"xuid"`
	Link Lien   `json:"link"`
}

// Lien nomme la source et la methode d'une identite.
type Lien struct {
	Source string `json:"source"`
	Method string `json:"method"`
}

// Couverture porte les compteurs de couverture lus par le banc.
type Couverture struct {
	ContinuousFire *CouvTirContinu   `json:"continuousFire"`
	Keyframes      *CouvImagesCles   `json:"keyframes"`
	Verdict        map[string]string `json:"verdict"`
	Bridge         *CouvPont         `json:"bridge"`
	Seats          *CouvSieges       `json:"seats"`
	Teams          *CouvEquipes      `json:"teams"`
	Score          *CouvScore        `json:"score"`
	Fallbacks      []Repli           `json:"fallbacks"`
}

// CouvTirContinu : paquets atteints et fermes au bit pres (vue C lue jusqu'a son terminateur).
type CouvTirContinu struct {
	Packets int `json:"packets"`
	Closed  int `json:"closed"`
}

// CouvImagesCles : preuves contradictoires de la marche d'image-cle.
type CouvImagesCles struct {
	Refutations         int `json:"refutations"`
	ContradictoryProofs int `json:"contradictoryProofs"`
}

// CouvPont : desaccords d'identite entre la lecture directe et le pont par morts.
type CouvPont struct {
	Discordant     int `json:"discordant"`
	SlotCollisions int `json:"slotCollisions"`
}

// CouvSieges : recouvrements de sieges.
type CouvSieges struct {
	Chevauchements int `json:"chevauchements"`
}

// CouvEquipes : l'equipe lue dans le film contre celle de la base.
type CouvEquipes struct {
	Accord        int `json:"accord"`
	Contradiction int `json:"contradiction"`
	Silence       int `json:"silence"`
}

// CouvScore : la methode qui a rattache les slots d'equipe aux camps.
type CouvScore struct {
	TeamIdentity string `json:"teamIdentity"`
}

// Repli est un repli declenche et son nombre de declenchements.
type Repli struct {
	Name string `json:"name"`
	Hits int    `json:"hits"`
}

// ErrDocumentVide : l'artefact ne porte ni match ni piste — rien a juger, et le dire.
var ErrDocumentVide = errors.New("replayverite : artefact sans match ni piste")

// LireDocument desserialise un artefact publie.
func LireDocument(blob []byte) (*Document, error) {
	var d Document
	if err := json.Unmarshal(blob, &d); err != nil {
		return nil, fmt.Errorf("replayverite : artefact illisible : %w", err)
	}
	if d.MatchID == "" && len(d.Tracks) == 0 {
		return nil, ErrDocumentVide
	}
	if d.FrameIntervalMs <= 0 {
		return nil, fmt.Errorf("replayverite : frameIntervalMs = %d, un pas d'image positif est requis", d.FrameIntervalMs)
	}
	return &d, nil
}
