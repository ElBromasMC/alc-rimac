package constancia

import (
	"alc/handler/util"
	"alc/model/constancia"
	"alc/view/component"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"

	view "alc/view/constancia"
	"context"
	"encoding/csv"
	"github.com/labstack/echo/v4"
	"strconv"
	"strings"
	"time"
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

// HandleEquiposReportDownload handles the CSV download for equipos with activo_fijo.
func (h *Handler) HandleEquiposReportDownload(c echo.Context) error {
	ctx := c.Request().Context()
	equipos, err := h.ConstanciaService.GetEquiposWithActivoFijo(ctx)
	if err != nil {
		c.Logger().Errorf("Failed to get equipos with activo_fijo for report: %v", err)
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage("Error al obtener los datos para el reporte de equipos."))
	}

	// Set headers for CSV download
	fileName := fmt.Sprintf("reporte_equipos_con_activofijo_%s.csv", time.Now().Format("2006-01-02"))
	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+fileName+"\"")

	// Create CSV writer
	wr := csv.NewWriter(c.Response().Writer)

	// Write header row
	header := []string{
		"ID", "Tipo Equipo", "Marca", "MTM", "Modelo", "Serie", "Activo Fijo",
		"Fecha Creación", "Fecha Actualización",
	}
	if err := wr.Write(header); err != nil {
		c.Logger().Errorf("Error writing CSV header for equipos report: %v", err)
		return nil
	}

	// Write data rows
	for _, e := range equipos {
		row := []string{
			strconv.FormatInt(e.Id, 10),
			e.TipoEquipo,
			e.Marca,
			e.MTM,
			e.Modelo,
			e.Serie,
			e.ActivoFijo,
			e.CreatedAt.Format("2006-01-02 15:04:05"), // Format timestamp
			e.UpdatedAt.Format("2006-01-02 15:04:05"), // Format timestamp
		}
		if err := wr.Write(row); err != nil {
			c.Logger().Errorf("Error writing CSV row for equipos report (ID %d): %v", e.Id, err)
			return nil
		}
	}

	// Flush ensures all data is written
	wr.Flush()
	if err := wr.Error(); err != nil {
		c.Logger().Errorf("Error flushing CSV writer for equipos report: %v", err)
	}

	return nil
}
