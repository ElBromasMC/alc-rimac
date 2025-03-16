package constancia

import (
	"alc/handler/util"
	"alc/model/constancia"
	view "alc/view/constancia"
	"context"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

func (h *Handler) HandleUsuarioFetch(c echo.Context) error {
	userSAP := c.FormValue("sap")
	userSAP = strings.ToLower(strings.ReplaceAll(userSAP, " ", ""))
	cliente, err := h.ConstanciaService.GetClienteBySapId(context.Background(), userSAP)
	if err != nil {
		return util.Render(c, http.StatusOK, view.UsuarioForm(constancia.Cliente{}, "Usuario no encontrado"))
	}
	return util.Render(c, http.StatusOK, view.UsuarioForm(cliente, ""))
}

func (h *Handler) HandleEquipoFetch(c echo.Context) error {
	serie := c.FormValue("PORTATIL-serie")
	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))
	equipo, err := h.ConstanciaService.GetEquipoBySerie(context.Background(), serie)
	if err != nil {
		return util.Render(c, http.StatusOK, view.PortatilForm(constancia.Equipo{}, "Equipo no encontrado"))
	}
	return util.Render(c, http.StatusOK, view.PortatilForm(equipo, ""))
}
