package constancia

import (
	"alc/handler/util"
	view "alc/view/constancia"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) HandleFormShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, view.Index())
}
