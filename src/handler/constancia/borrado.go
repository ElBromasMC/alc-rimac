package constancia

import (
	"alc/handler/util"
	"alc/model/constancia"
	"alc/service"
	"alc/view/component"
	view "alc/view/constancia"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HandleBorradoFormShow renders the initial secure erase form page.
func (h *Handler) HandleBorradoFormShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, view.Borrado())
}

// HandleBorradoInventarioFetch handles the HTMX request to autocomplete form fields.
func (h *Handler) HandleBorradoInventarioFetch(c echo.Context) error {
	serie := c.QueryParam("Serie")
	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))

	if serie == "" {
		return util.Render(c, http.StatusOK, view.BorradoAutocomplete(constancia.Inventario{}, true, ""))
	}

	inv, err := h.ConstanciaService.GetInventarioPortatilOldBySerie(c.Request().Context(), serie)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return util.Render(c, http.StatusOK, view.BorradoAutocomplete(constancia.Inventario{}, true, serie))
		}
		c.Logger().Errorf("Error fetching inventario PORTATILOLD by serie '%s': %v", serie, err)
		return util.Render(c, http.StatusOK, view.BorradoAutocomplete(constancia.Inventario{}, true, serie))
	}

	return util.Render(c, http.StatusOK, view.BorradoAutocomplete(inv, false, serie))
}

// HandleBorradoInsert handles the POST submission of the secure erase form.
func (h *Handler) HandleBorradoInsert(c echo.Context) error {
	ctx := c.Request().Context()

	// --- 1. Get data ---
	serieAntiguo := strings.ToUpper(strings.ReplaceAll(c.FormValue("Serie"), " ", ""))
	inventarioRimac := strings.ToUpper(strings.TrimSpace(c.FormValue("InventarioRIMAC")))
	serieDisco := strings.ToUpper(strings.TrimSpace(c.FormValue("SerieDisco")))

	serieEquipoNuevo := strings.ToUpper(strings.ReplaceAll(c.FormValue("SerieEquipoNuevo"), " ", ""))
	marca := strings.ToUpper(strings.TrimSpace(c.FormValue("Marca")))
	modelo := strings.ToUpper(strings.TrimSpace(c.FormValue("Modelo")))

	// --- 2. Handle File Upload ---
	file, err := c.FormFile("certificado")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return util.Render(c, http.StatusOK, component.ErrorMessage("Falta el archivo del certificado."))
		}
		c.Logger().Errorf("Error getting uploaded file 'certificado': %v", err)
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al procesar el archivo cargado."))
	}

	if filepath.Ext(strings.ToLower(file.Filename)) != ".pdf" {
		return util.Render(c, http.StatusOK, component.ErrorMessage("El certificado debe ser un archivo PDF."))
	}

	// --- 4. Validate Mandatory Fields ---
	if serieAntiguo == "" || inventarioRimac == "" || serieDisco == "" {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Faltan campos obligatorios (Serie, RIMAC, Serie Disco)."))
	}

	// --- 5. Perform Data Correction in 'inventario' if needed ---
	if serieEquipoNuevo != "" {
		serieIncorrecta, err := h.ConstanciaService.GetInventarioPortatilOldSerieByConstanciaSerie(ctx, serieEquipoNuevo)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage("Serie Equipo Nuevo no válido"))
		}
		err = h.ConstanciaService.UpdateInventarioPortatilOld(ctx, serieIncorrecta, serieAntiguo, inventarioRimac, marca, modelo)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(fmt.Sprintf("Error al corregir datos del inventario antiguo: %v", err)))
		}
	} else {
		_, err := h.ConstanciaService.GetInventarioPortatilOldBySerie(c.Request().Context(), serieAntiguo)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return util.Render(c, http.StatusOK, view.BorradoAutocomplete(constancia.Inventario{}, true, serieAntiguo))
			}
			return util.Render(c, http.StatusOK, view.BorradoAutocomplete(constancia.Inventario{}, true, serieAntiguo))
		}
	}

	// --- 6. Save PDF Certificate ---
	pdfStoragePath := os.Getenv("PDF_STORAGE_PATH")
	savedFilePath, err := service.SaveUploadedFile(file, serieAntiguo, inventarioRimac, pdfStoragePath)
	if err != nil {
		c.Logger().Errorf("Error saving uploaded PDF for serie %s: %v", serieAntiguo, err)
		return util.Render(c, http.StatusOK, component.ErrorMessage(fmt.Sprintf("Error al guardar el archivo PDF: %v", err)))
	}

	// --- 7. Create Borrado Seguro Log Record ---
	borrado := constancia.BorradoSeguro{
		Serie:           serieAntiguo,
		InventarioRimac: inventarioRimac,
		SerieDisco:      serieDisco,
		Marca:           marca,
		Modelo:          modelo,
		CertificadoPath: savedFilePath,
	}

	_, err = h.ConstanciaService.CreateBorradoSeguro(ctx, borrado)
	if err != nil {
		c.Logger().Errorf("Error creating borrado_seguro record for serie %s: %v", serieAntiguo, err)
		errRemove := os.Remove(savedFilePath)
		if errRemove != nil {
			c.Logger().Errorf("Failed to cleanup saved PDF '%s' after DB error: %v", savedFilePath, errRemove)
		}
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al guardar el registro de borrado seguro en la base de datos."))
	}

	// --- 8. Success Response ---
	c.Response().Header().Set("HX-Redirect", "/borrado")
	return c.NoContent(http.StatusOK)
}

func (h *Handler) HandleBorradosReportDownload(c echo.Context) error {
	ctx := c.Request().Context()
	borrados, err := h.ConstanciaService.GetAllBorradosSeguros(ctx)
	if err != nil {
		c.Logger().Errorf("Failed to get borrados seguros for report: %v", err)
		// Decide how to show error - maybe redirect back with flash message or render error page
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al obtener los datos para el reporte de borrados."))
	}

	// Set headers for CSV download
	fileName := fmt.Sprintf("reporte_borrados_seguros_%s.csv", time.Now().Format("2006-01-02"))
	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+fileName+"\"")

	// Create CSV writer
	wr := csv.NewWriter(c.Response().Writer)

	// Write header row (adjust column names as needed)
	header := []string{
		"ID", "Serie", "Inventario RIMAC", "Serie Disco", "Marca", "Modelo",
		"Ruta Certificado", "Fecha Creación", "Fecha Actualización",
	}
	if err := wr.Write(header); err != nil {
		c.Logger().Errorf("Error writing CSV header for borrados report: %v", err)
		// Hard to return specific error to client after headers are sent
		return nil // Or maybe return an error status if possible before writing starts
	}

	// Write data rows
	for _, b := range borrados {
		row := []string{
			strconv.FormatInt(b.Id, 10),
			b.Serie,
			b.InventarioRimac,
			b.SerieDisco,
			b.Marca,
			b.Modelo,
			b.CertificadoPath,
			b.CreatedAt.Format("2006-01-02 15:04:05"), // Format timestamp
			b.UpdatedAt.Format("2006-01-02 15:04:05"), // Format timestamp
		}
		if err := wr.Write(row); err != nil {
			c.Logger().Errorf("Error writing CSV row for borrados report (ID %d): %v", b.Id, err)
			// Stop processing if a write error occurs?
			return nil
		}
	}

	// Flush ensures all data is written to the underlying writer (response)
	wr.Flush()
	if err := wr.Error(); err != nil {
		c.Logger().Errorf("Error flushing CSV writer for borrados report: %v", err)
	}

	return nil // Indicates success to Echo
}
