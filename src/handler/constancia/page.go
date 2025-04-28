package constancia

import (
	"alc/handler/util"
	view "alc/view/constancia"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) HandleIndexShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, view.Index())
}

func (h *Handler) HandleAccesoriosFormShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, view.Accesorios())
}

func (h *Handler) HandleDevolucionFormShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, view.Devolucion())
}

func (h *Handler) HandleClonacionFormShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, view.Clonacion())
}
