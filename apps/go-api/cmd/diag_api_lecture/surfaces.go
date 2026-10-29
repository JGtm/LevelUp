package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
)

// pageHistorique : taille de la page lue (le maximum de l'API).
const pageHistorique = 25

// langueActif : langue demandée pour le nom de l'actif de découverte (valeur du protocole, pas un
// choix de présentation).
const langueActif = "en-US"

// surface : une sous-commande.
type surface struct {
	nom      string
	args     []string // noms des arguments positionnels attendus
	resume   string
	executer func(s *session, args []string) error
}

// surfaces : les lectures du témoin T3 (les six du plan, plus le service record, repli de la surface
// « calendrier » quand le service de saisons répond 403).
var surfaces = []surface{
	{"historique", nil, "une page (25 matchs) de l'historique du compte", lireHistorique},
	{"stats", []string{"match_id"}, "les statistiques d'un match", lireStats},
	{"skill", []string{"match_id"}, "le skill (MMR, CSR) du compte sur un match", lireSkill},
	{"manifeste", []string{"match_id"}, "le manifeste d'un film (références de fragments, sans octets)", lireManifeste},
	{"actif", []string{"type", "asset_id", "version_id"}, "un actif de découverte (map|playlist|pair|game_variant)", lireActif},
	{"calendrier", nil, "le calendrier de saisons (corps JSON reçu)", lireCalendrier},
	{"servicerecord", nil, "le service record matchmade du compte (cumul, valeurs normalisées)", lireServiceRecord},
}

// run exécute la surface avec ses arguments.
func (sf surface) run(s *session, args []string) error { return sf.executer(s, args) }

// trouverSurface retrouve la sous-commande de `args` (nom + arguments positionnels exacts).
func trouverSurface(args []string) (surface, bool) {
	if len(args) == 0 {
		return surface{}, false
	}
	for _, sf := range surfaces {
		if sf.nom == args[0] && len(args)-1 == len(sf.args) {
			return sf, true
		}
	}
	return surface{}, false
}

// usage rend l'aide.
func usage() string {
	var b strings.Builder
	b.WriteString("usage: diag_api_lecture [-xuid X] <sous-commande> [arguments]\n")
	for _, sf := range surfaces {
		fmt.Fprintf(&b, "  %-11s %-34s %s\n", sf.nom, signature(sf.args), sf.resume)
	}
	return b.String()
}

// signature écrit les arguments positionnels d'une surface (`<a> <b>`, vide sans argument).
func signature(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return "<" + strings.Join(args, "> <") + ">"
}

// ecrireJSON écrit la valeur en `json.MarshalIndent` (deux espaces), sans saut de ligne final.
func (s *session) ecrireJSON(v any) error {
	blob, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage JSON: %w", err)
	}
	_, err = s.out.Write(blob)
	return err
}

func lireHistorique(s *session, _ []string) error {
	entrees, err := s.client.GetMatchHistory(s.ctx, "xuid("+s.xuid+")", "all", 0, pageHistorique)
	if err != nil {
		return err
	}
	return s.ecrireJSON(entrees)
}

func lireStats(s *session, args []string) error {
	stats, err := s.client.GetMatchStats(s.ctx, args[0])
	if err != nil {
		return err
	}
	return s.ecrireJSON(stats)
}

func lireSkill(s *session, args []string) error {
	skill, err := s.client.GetMatchSkill(s.ctx, args[0], []string{s.xuid})
	if err != nil {
		return err
	}
	return s.ecrireJSON(skill)
}

// reponseManifeste : la forme écrite pour un manifeste (absent = film expiré côté serveur).
type reponseManifeste struct {
	Trouve    bool          `json:"trouve"`
	Fragments []refFragment `json:"fragments"`
}

// refFragment : une référence de fragment du manifeste (Index, type, début, durée, adresse).
type refFragment struct {
	Index      int    `json:"index"`
	ChunkType  int    `json:"chunk_type"`
	StartMS    int    `json:"start_ms"`
	DurationMS int    `json:"duration_ms"`
	URL        string `json:"url"`
}

func lireManifeste(s *session, args []string) error {
	refs, trouve, err := s.client.GetFilmChunkURLs(s.ctx, args[0])
	if err != nil {
		return err
	}
	rep := reponseManifeste{Trouve: trouve, Fragments: make([]refFragment, 0, len(refs))}
	for _, r := range refs {
		rep.Fragments = append(rep.Fragments, refFragment{
			Index: r.Index, ChunkType: r.ChunkType, StartMS: r.StartMS, DurationMS: r.DurationMS,
			URL: sansRequete(r.URL),
		})
	}
	return s.ecrireJSON(rep)
}

// sansRequete retire la partie requête (et les identifiants) d'une adresse : une adresse de blob
// signée ne doit jamais atterrir dans un témoin.
func sansRequete(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "<adresse illisible>"
	}
	u.RawQuery, u.Fragment, u.User = "", "", nil
	return u.String()
}

// actifLu : la forme écrite pour un actif de découverte (l'adresse d'image comprise : le champ du
// client est exclu de sa propre sérialisation JSON).
type actifLu struct {
	AssetID     string `json:"asset_id"`
	VersionID   string `json:"version_id"`
	PublicName  string `json:"public_name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

func lireActif(s *session, args []string) error {
	a, err := s.provider.FetchAsset(s.ctx, haloadapter.AssetType(args[0]), title.DefaultSlug, args[1], args[2], langueActif)
	if err != nil {
		return err
	}
	return s.ecrireJSON(actifLu{
		AssetID: a.AssetID, VersionID: a.VersionID, PublicName: a.PublicName,
		Description: a.Description, ImageURL: sansRequete(a.ImageURL),
	})
}

// lireServiceRecord écrit le service record cumulé du compte tel que le client le normalise.
func lireServiceRecord(s *session, _ []string) error {
	rec, err := s.provider.FetchServiceRecord(s.ctx, s.gamertag, title.DefaultSlug)
	if err != nil {
		return err
	}
	return s.ecrireJSON(rec)
}

// lireCalendrier écrit le corps JSON reçu, octet pour octet. Le client rend aussi les saisons
// converties ; elles sont ignorées (le témoin est la réponse du service).
func lireCalendrier(s *session, _ []string) error {
	_, brut, err := s.provider.FetchSeasonCalendar(s.ctx, title.DefaultSlug)
	if len(brut) == 0 {
		if err == nil {
			err = fmt.Errorf("réponse vide")
		}
		return err
	}
	if _, werr := s.out.Write(brut); werr != nil {
		return werr
	}
	return err
}
