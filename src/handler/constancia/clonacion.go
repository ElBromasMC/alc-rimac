package constancia

import (
	"alc/handler/util"
	"alc/model/constancia"
	"alc/view/component"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"

	view "alc/view/constancia"
	"context"
	"github.com/labstack/echo/v4"
	"strings"
)

func (h *Handler) HandleClonacionEquipoFetch(c echo.Context) error {
	serie := c.FormValue("Serie")
	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))
	equipo, err := h.ConstanciaService.GetEquipoBySerie(context.Background(), serie)
	if err != nil {
		return util.Render(c, http.StatusOK, view.ClonacionEquipoAuto(constancia.Equipo{}, "Equipo no encontrado"))
	}

	// Check if "equipo"
	msg := ""
	if equipo.ActivoFijo != "" {
		msg = "El equipo ya tiene un activo fijo registrado."
	}

	return util.Render(c, http.StatusOK, view.ClonacionEquipoAuto(equipo, msg))
}

func (h *Handler) HandleClonacionInsert(c echo.Context) error {
	serie := c.FormValue("Serie")
	activoFijo := c.FormValue("ActivoFijo")

	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))
	activoFijo = strings.TrimSpace(activoFijo)

	if serie == "" {
		return util.Render(c, http.StatusBadRequest, component.ErrorMessage("El campo 'Serie' es obligatorio."))
	}
	if activoFijo == "" {
		return util.Render(c, http.StatusBadRequest, component.ErrorMessage("El campo 'Activo fijo' es obligatorio."))
	}

	err := h.ConstanciaService.UpdateEquipoActivoFijoBySerie(c.Request().Context(), serie, activoFijo)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return util.Render(c, http.StatusNotFound, component.ErrorMessage("No se encontró ningún equipo con la serie proporcionada."))
		}
		c.Logger().Errorf("Error updating activo_fijo for serie '%s': %v", serie, err)
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage("Ocurrió un error interno al actualizar el equipo."))
	}

	c.Response().Header().Set("HX-Redirect", "/clonacion")
	return c.NoContent(http.StatusOK)
}
