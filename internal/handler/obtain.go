// Handler de la vue d'obtention : GET /v1/cars/{id}/obtain. Lecture seule,
// agrégat construit côté store. Mapping DB → contrat ; les voies absentes
// restent listes vides / champs omis (rien d'inventé).
package handler

import (
	"context"
	"net/http"

	"github.com/zinackes/forza-open-api/internal/oas"
)

// GetCarObtain implémente GET /v1/cars/{id}/obtain : les voies d'obtention de la
// voiture, ou un 404 RFC 9457 si l'identifiant est inconnu.
func (h *Handler) GetCarObtain(ctx context.Context, params oas.GetCarObtainParams) (oas.GetCarObtainRes, error) {
	o, err := h.store.GetCarObtain(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return &oas.GetCarObtainNotFound{
			Title:  oas.NewOptString(http.StatusText(http.StatusNotFound)),
			Status: oas.NewOptInt(http.StatusNotFound),
			Detail: oas.NewOptString("no car with the given id"),
		}, nil
	}

	out := oas.CarObtain{
		CarId:          o.Car.ID,
		Game:           oas.Game(o.Car.Game),
		ObtainMethod:   optString(o.Car.ObtainMethod),
		ValueCr:        optInt64(o.Car.ValueCr),
		DlcPacks:       make([]oas.DlcPack, 0, len(o.DlcPacks)),
		JournalTiers:   make([]oas.JournalTier, 0, len(o.JournalTiers)),
		MasteryUnlocks: make([]oas.CarObtainMasteryUnlock, 0, len(o.MasteryUnlocks)),
	}
	for _, d := range o.DlcPacks {
		out.DlcPacks = append(out.DlcPacks, mapDlcPack(d))
	}
	if o.BarnFind != nil {
		out.BarnFind = oas.NewOptBarnFind(mapBarnFind(*o.BarnFind))
	}
	if o.TreasureCar != nil {
		out.TreasureCar = oas.NewOptTreasureCar(mapTreasureCar(*o.TreasureCar))
	}
	for _, j := range o.JournalTiers {
		out.JournalTiers = append(out.JournalTiers, mapJournalTier(j))
	}
	for _, m := range o.MasteryUnlocks {
		out.MasteryUnlocks = append(out.MasteryUnlocks, oas.CarObtainMasteryUnlock{
			PerkId:     m.PerkID,
			OwnerCarId: m.OwnerCarID,
			PerkName:   optString(m.PerkName),
		})
	}
	return &out, nil
}
