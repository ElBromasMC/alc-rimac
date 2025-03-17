package constancia

import (
	"alc/handler/util"
	"alc/model/auth"
	"alc/model/constancia"
	"alc/view/component"
	view "alc/view/constancia"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func (h *Handler) HandleConstanciaInsert(c echo.Context) error {
	// Parse request
	user, _ := auth.GetUser(c.Request().Context())
	tipoProcedimiento, err := constancia.GetTipoProcedimiento(c.FormValue("tipoProcedimiento"))
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	tipoEquipo, err := constancia.GetTipoEquipo(c.FormValue("tipoEquipo"))
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	fechaHoraStr := c.FormValue("fechaHora")
	loc, err := time.LoadLocation("America/Lima")
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error interno del servidor"))
	}
	fechaHora, err := time.ParseInLocation("2006-01-02T15:04", fechaHoraStr, loc)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Fecha inválida"))
	}

	// Get data
	userSAP := c.FormValue("sap")
	userSAP = strings.ToLower(strings.ReplaceAll(userSAP, " ", ""))
	cliente, err := h.ConstanciaService.GetClienteBySapId(context.Background(), userSAP)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Usuario inválido"))
	}

	serie := c.FormValue("PORTATIL-serie")
	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))
	equipo, err := h.ConstanciaService.GetEquipoBySerie(context.Background(), serie)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Portatil inválido"))
	}

	cta := constancia.Constancia{
		NroTicket:          c.FormValue("nroTicket"),
		TipoProcedimiento:  tipoProcedimiento,
		ResponsableUsuario: c.FormValue("responsableUsuario"),
		CodigoEmpleado:     c.FormValue("codigoEmpleado"),
		FechaHora:          fechaHora,
		Sede:               c.FormValue("sede"),
		Piso:               c.FormValue("piso"),
		Area:               c.FormValue("area"),
		TipoEquipo:         tipoEquipo,
		IssuedBy:           user,
		UsuarioSAP:         cliente.SapId,
		UsuarioNombre:      cliente.Usuario,
	}
	cta, err = cta.Normalize()
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	var inventarios []constancia.Inventario

	portatil := constancia.Inventario{
		TipoInventario: constancia.InventarioPortatil,
		Serie:          c.FormValue("PORTATIL-serie"),
		Estado:         c.FormValue("PORTATIL-estado"),
		Marca:          equipo.Marca,
		Modelo:         equipo.Modelo,
		Inventario:     equipo.ActivoFijo,
	}
	portatil, err = portatil.Normalize()
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}
	inventarios = append(inventarios, portatil)

	types := []string{"MOUSE", "CABLERED", "CARGADOR", "MOCHILA", "CADENA"}
	for _, t := range types {
		tipoInventario, err := constancia.GetTipoInventario(t)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}
		inv := constancia.Inventario{
			TipoInventario: tipoInventario,
			Serie:          c.FormValue(fmt.Sprintf("%s-serie", t)),
			Estado:         c.FormValue(fmt.Sprintf("%s-estado", t)),
			Marca:          c.FormValue(fmt.Sprintf("%s-marca", t)),
			Modelo:         c.FormValue(fmt.Sprintf("%s-modelo", t)),
			Inventario:     c.FormValue(fmt.Sprintf("%s-inventario", t)),
		}
		inv, err = inv.Normalize()
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}
		inventarios = append(inventarios, inv)
	}

	// Insert to database
	err = h.ConstanciaService.InsertConstanciaAndInventarios(context.Background(), cta, inventarios)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	// Generate PDF
	// Step 1: Create a temporary file to get a unique filename.
	tempFile, err := os.CreateTemp("./pdf", "output-*.pdf")
	if err != nil {
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage("Failed to create temp file"))
	}
	tempFilename := tempFile.Name()
	tempFile.Close() // Close immediately since we'll write our own copy

	// Ensure the temporary file is deleted after we're done.
	//defer os.Remove(tempFilename)

	// Step 2: Open the base PDF file.
	srcFile, err := os.Open("./pdf/constancia.pdf")
	if err != nil {
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage("Failed to open base PDF"))
	}
	defer srcFile.Close()

	// Step 3: Create (or overwrite) the temporary file and copy the base PDF into it.
	dstFile, err := os.Create(tempFilename)
	if err != nil {
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage("Failed to create destination file"))
	}
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		dstFile.Close()
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage("Failed to copy PDF"))
	}
	dstFile.Close()

	// Step 4: Modify the PDF in place using your GeneratePDF function.
	err = h.ConstanciaService.GeneratePDF(context.Background(), tempFilename, cta, inventarios)
	if err != nil {
		return util.Render(c, http.StatusInternalServerError, component.ErrorMessage(err.Error()))
	}

	// Build the URL for the download endpoint.
	// We send only the file base name as a parameter.
	downloadURL := fmt.Sprintf("/download?file=%s&serie=%s&usuario=%s",
		url.QueryEscape(filepath.Base(tempFilename)),
		url.QueryEscape(portatil.Serie),
		url.QueryEscape(cta.UsuarioNombre),
	)
	// Instead of returning the file directly, set the HX-Redirect header.
	c.Response().Header().Set("HX-Redirect", downloadURL)
	return util.Render(c, http.StatusOK, component.InfoMessage("Cargado exitosamente"))
}

// GET handler that serves the PDF file.
func (h *Handler) DownloadPDFHandler(c echo.Context) error {
	// Get the file name from the query string.
	fileName := c.QueryParam("file")
	serie := c.QueryParam("serie")
	usuario := c.QueryParam("usuario")
	if fileName == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	// Construct the full path to the file.
	filePath := filepath.Join("./pdf", fileName)

	// Clean up: remove the file after serving.
	defer os.Remove(filePath)

	// Serve the file to the user with a custom download name.
	err := c.Attachment(filePath, fmt.Sprintf("%s-%s.pdf", serie, usuario))

	return err
}
