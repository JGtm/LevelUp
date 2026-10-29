package haloadapter

// medal_provider.go — métadonnées des médailles (GameCMS `medals/metadata.json`), commande
// `medals` du CLI refresh-metadata.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/metadata"
)

// FetchMedalsMetadata récupère les métadonnées des médailles : entrées de métadonnées, corps brut
// (pour l'empreinte) et erreur éventuelle. Jetons lus dans le contexte.
func (p *Provider) FetchMedalsMetadata(ctx context.Context, titleID string) ([]metadata.MedalEntry, []byte, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, nil, fmt.Errorf("FetchMedalsMetadata: tokens absents du contexte")
	}
	dto, body, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider(), infinite.CleGameCMS).FetchMedalsMetadata(ctx, authDe(tokens))
	if err != nil {
		return nil, body, err
	}
	entries := make([]metadata.MedalEntry, 0, len(dto.Medals))
	for _, m := range dto.Medals {
		// La ligne brute est la médaille telle que reçue ; un échec d'encodage (impossible sur ces
		// champs scalaires) laisse une ligne brute vide, journalisé.
		rowJSON, mErr := json.Marshal(m)
		if mErr != nil {
			slog.WarnContext(ctx, "medal_provider: ligne brute d'une médaille non encodée", "medal_id", m.MedalID, "err", mErr)
		}
		entries = append(entries, metadata.MedalEntry{
			TitleID:   titleID,
			MedalID:   m.MedalID,
			Label:     m.NameID,
			Category:  m.Type,
			Rarity:    m.Difficulty,
			SpriteIdx: m.SpriteIndex,
			RawJSON:   string(rowJSON),
		})
	}
	return entries, body, nil
}
