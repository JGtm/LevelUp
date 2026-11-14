package handlers

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/observability"
)

// ReplayColumnsEncoding est la valeur du paramètre `encoding` qui demande la forme en colonnes.
const ReplayColumnsEncoding = "columns"

// replayColumnsRefusedCounter compte les réponses servies en forme normale alors que la forme en
// colonnes était demandée (clé réservée, corps hors modèle).
const replayColumnsRefusedCounter = "replay_columns_refused_total"

// ReplayColumnEncoder réécrit le corps JSON déjà encodé du rejeu en forme en colonnes. Il est
// injecté au câblage (`replayview.EncodeColumns`) : un handler n'importe pas la couche service.
type ReplayColumnEncoder func(body []byte) ([]byte, error)

// columnsMiddleware est le middleware d'opération de la route du rejeu : quand `encoding=columns`
// est demandé, il retient le corps que Huma vient d'écrire (donc assaini et sérialisé comme la
// forme normale) et le réécrit par l'encodeur. Un statut autre que 200 (erreur, validation) passe
// sans toucher au corps ; un refus de l'encodeur sert le corps normal, journalisé et compté.
func (h *ReplayHandler) columnsMiddleware(ctx huma.Context, next func(huma.Context)) {
	u := ctx.URL()
	if u.Query().Get("encoding") != ReplayColumnsEncoding || h.encodeColumns == nil {
		next(ctx)
		return
	}
	held := &heldBodyContext{underlyingContext: ctx}
	next(held)
	body := held.buf.Bytes()
	if held.status == http.StatusOK {
		enc, err := h.encodeColumns(body)
		if err != nil {
			slog.WarnContext(ctx.Context(), "rejeu servi en forme normale : encodage en colonnes refusé", "err", err)
			observability.IncCounter(replayColumnsRefusedCounter)
		} else {
			body = enc
		}
	}
	if held.status != 0 {
		ctx.SetStatus(held.status)
	}
	if _, err := ctx.BodyWriter().Write(body); err != nil {
		slog.WarnContext(ctx.Context(), "écriture du rejeu interrompue", "err", err)
	}
}

// underlyingContext évite qu'un champ nommé Context ne masque la méthode Context() de l'interface.
type underlyingContext huma.Context

// heldBodyContext retient le statut et le corps de la réponse jusqu'à la fin de l'opération ; les
// en-têtes passent directement.
type heldBodyContext struct {
	underlyingContext
	buf    bytes.Buffer
	status int
}

func (c *heldBodyContext) SetStatus(code int) { c.status = code }
func (c *heldBodyContext) Status() int        { return c.status }
func (c *heldBodyContext) BodyWriter() io.Writer {
	return &c.buf
}
