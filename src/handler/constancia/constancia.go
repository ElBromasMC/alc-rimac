package constancia

import (
	"alc/assets"
	"alc/handler/util"
	"alc/model/auth"
	"alc/model/constancia"
	"alc/view/component"
	view "alc/view/constancia"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
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
	// Check if constancia already exists
	exists, err := h.ConstanciaService.ConstanciaExists(context.Background(), equipo.Serie)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	msg := ""
	if exists {
		msg = "El equipo ya ha sido registrado."
	}
	return util.Render(c, http.StatusOK, view.PortatilForm(equipo, msg))
}

func generateSendPDF(h *Handler, c *echo.Context, cta constancia.Constancia, inventarios []constancia.Inventario, formulario constancia.TipoFormulario) error {
	if formulario == constancia.FormularioAccesorios {
		// Generate PDF
		// Step 1: Create a temporary file to get a unique filename.
		tempFile, err := os.CreateTemp("./pdf", "output-*.pdf")
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to create temp file"))
		}
		tempFilename := tempFile.Name()
		tempFile.Close() // Close immediately since we'll write our own copy

		// Step 2: Open the base PDF file.
		srcFile, err := assets.Assets.Open("static/pdf/constancia.pdf")
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to open base PDF"))
		}
		defer srcFile.Close()

		// Step 3: Create (or overwrite) the temporary file and copy the base PDF into it.
		dstFile, err := os.Create(tempFilename)
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to create destination file"))
		}
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			dstFile.Close()
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to copy PDF"))
		}
		dstFile.Close()

		// Step 4: Modify the PDF in place using your GeneratePDF function.
		err = h.ConstanciaService.GeneratePDF(context.Background(), tempFilename, cta, inventarios)
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage(err.Error()))
		}

		// Send the PDF
		// Read the first PDF file.
		pdfBytes, err := os.ReadFile(tempFilename)
		if err != nil {
			return (*c).String(http.StatusInternalServerError, "Error reading PDF")
		}
		// Encode to base64.
		pdfBase64 := base64.StdEncoding.EncodeToString(pdfBytes)

		(*c).Response().Header().Set("HX-Retarget", "#constancia-target")
		return util.Render(*c, http.StatusOK, view.AccesoriosDocuments(pdfBase64, fmt.Sprintf("%s-%s", cta.Serie, cta.UsuarioNombre)))

	} else if formulario == constancia.FormularioDevolucion {
		// Generate PDFs
		tempFile1, err := os.CreateTemp("./pdf", "output-*.pdf")
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to create temp file"))
		}
		tempFilename1 := tempFile1.Name()
		tempFile1.Close() // Close immediately since we'll write our own copy

		tempFile2, err := os.CreateTemp("./pdf", "output-*.pdf")
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to create temp file"))
		}
		tempFilename2 := tempFile2.Name()
		tempFile2.Close() // Close immediately since we'll write our own copy

		// Open the base PDF file.
		srcFile, err := assets.Assets.Open("static/pdf/constancia.pdf")
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to open base PDF"))
		}
		defer srcFile.Close()

		// Create (or overwrite) the temporary file and copy the base PDF into it.
		dstFile1, err := os.Create(tempFilename1)
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to create destination file"))
		}
		_, err = io.Copy(dstFile1, srcFile)
		if err != nil {
			dstFile1.Close()
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to copy PDF"))
		}
		dstFile1.Close()

		// Open the asset again for the second copy.
		srcFile2, err := assets.Assets.Open("static/pdf/constancia.pdf")
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to open base PDF for second copy"))
		}
		defer srcFile2.Close()

		dstFile2, err := os.Create(tempFilename2)
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to create destination file"))
		}
		_, err = io.Copy(dstFile2, srcFile2)
		if err != nil {
			dstFile2.Close()
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage("Failed to copy PDF"))
		}
		dstFile2.Close()

		// Step 4: Modify the PDF in place using your GeneratePDF function.
		// Asignacion (Equipo nuevo)
		cta1 := cta
		cta1.TipoProcedimiento = constancia.ProcedimientoAsignacion
		cta1.Observacion = ""
		var inventarios1 []constancia.Inventario
		for _, i := range inventarios {
			if i.TipoInventario == constancia.InventarioPortatil {
				in := i
				inventarios1 = append(inventarios1, in)
			} else if i.TipoInventario == constancia.InventarioCargador {
				in := i
				inventarios1 = append(inventarios1, in)
			}
		}
		err = h.ConstanciaService.GeneratePDF(context.Background(), tempFilename1, cta1, inventarios1)
		defer os.Remove(tempFilename1)
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage(err.Error()))
		}

		// Recuperacion (Equipo antiguo)
		cta2 := cta
		cta2.TipoProcedimiento = constancia.ProcedimientoRecuperacion
		var inventarios2 []constancia.Inventario
		for _, i := range inventarios {
			if i.TipoInventario == constancia.InventarioPortatilOld {
				in := i
				in.TipoInventario = constancia.InventarioPortatil
				inventarios2 = append(inventarios2, in)
			} else if i.TipoInventario == constancia.InventarioCargadorOld {
				in := i
				in.TipoInventario = constancia.InventarioCargador
				inventarios2 = append(inventarios2, in)
			}
		}
		err = h.ConstanciaService.GeneratePDF(context.Background(), tempFilename2, cta2, inventarios2)
		defer os.Remove(tempFilename2)
		if err != nil {
			return util.Render(*c, http.StatusInternalServerError, component.ErrorMessage(err.Error()))
		}
		// Read the first PDF file.
		pdf1Bytes, err := os.ReadFile(tempFilename1)
		if err != nil {
			return (*c).String(http.StatusInternalServerError, "Error reading PDF 1")
		}
		// Encode to base64.
		pdf1Base64 := base64.StdEncoding.EncodeToString(pdf1Bytes)

		// Read the second PDF file.
		pdf2Bytes, err := os.ReadFile(tempFilename2)
		if err != nil {
			return (*c).String(http.StatusInternalServerError, "Error reading PDF 2")
		}
		// Encode to base64.
		pdf2Base64 := base64.StdEncoding.EncodeToString(pdf2Bytes)

		(*c).Response().Header().Set("HX-Retarget", "#constancia-target")
		return util.Render(*c, http.StatusOK, view.DevolucionDocuments(pdf1Base64, pdf2Base64, fmt.Sprintf("%s-%s", cta.Serie, cta.UsuarioNombre)))
	} else {
		return util.Render(*c, http.StatusOK, component.ErrorMessage("Tipo de formulario inválido"))
	}
}

func (h *Handler) HandleConstanciaInsert(c echo.Context) error {
	// Parse request
	user, _ := auth.GetUser(c.Request().Context())
	formulario, err := constancia.GetTipoFormulario(c.FormValue("formulario"))
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}
	tipoProcedimiento := constancia.ProcedimientoAsignacion
	if formulario != constancia.FormularioDevolucion {
		tipoProcedimiento, err = constancia.GetTipoProcedimiento(c.FormValue("tipoProcedimiento"))
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}
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
		CodigoEmpleado:     cliente.Usuario,
		FechaHora:          fechaHora,
		Sede:               c.FormValue("sede"),
		Piso:               c.FormValue("piso"),
		Area:               c.FormValue("area"),
		TipoEquipo:         tipoEquipo,
		IssuedBy:           user,
		UsuarioSAP:         cliente.SapId,
		UsuarioNombre:      cliente.Usuario,
		Serie:              c.FormValue("PORTATIL-serie"),
		Observacion:        c.FormValue("observacion"),
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

	types := []string{"MOUSE", "CABLERED", "CARGADOR", "MOCHILA", "CADENA", "PORTATILOLD", "CARGADOROLD"}
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

	// Check if constancia already exists
	exists, err := h.ConstanciaService.ConstanciaExists(context.Background(), cta.Serie)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	if exists {
		ctaJSON, err := json.Marshal(cta)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}

		inventariosJSON, err := json.Marshal(inventarios)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}

		// Query previous constancia
		ctaOld, err := h.ConstanciaService.GetConstanciaBySerie(context.Background(), cta.Serie)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}

		// Send confirmation form
		return util.Render(c, http.StatusOK, view.UpdateForm(ctaOld.UsuarioNombre, cta.Serie, string(ctaJSON), string(inventariosJSON), formulario))
	} else {
		// Insert to database
		err = h.ConstanciaService.InsertConstanciaAndInventarios(context.Background(), cta, inventarios)
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}

		return generateSendPDF(h, &c, cta, inventarios, formulario)
	}
}

func (h *Handler) HandleConstanciaUpdate(c echo.Context) error {
	user, _ := auth.GetUser(c.Request().Context())
	formulario, err := constancia.GetTipoFormulario(c.FormValue("formulario"))
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}
	ctaStr := c.FormValue("cta")
	inventariosStr := c.FormValue("inventarios")

	var cta constancia.Constancia
	if err := json.Unmarshal([]byte(ctaStr), &cta); err != nil {
		return c.String(http.StatusBadRequest, "Invalid constancia data")
	}
	cta.IssuedBy = user

	var inventarios []constancia.Inventario
	if err := json.Unmarshal([]byte(inventariosStr), &inventarios); err != nil {
		return c.String(http.StatusBadRequest, "Invalid inventario data")
	}

	cta, err = cta.Normalize()
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	for i, inv := range inventarios {
		normalizedInv, err := inv.Normalize()
		if err != nil {
			return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
		}
		inventarios[i] = normalizedInv
	}

	// Update constancia
	err = h.ConstanciaService.UpdateConstanciaAndInventarios(context.Background(), cta, inventarios)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}

	return generateSendPDF(h, &c, cta, inventarios, formulario)
}

// GET handler that serves the PDF file.
func (h *Handler) DownloadPDFHandler(c echo.Context) error {
	// Get the file name from the query string.
	formulario, err := constancia.GetTipoFormulario(c.FormValue("formulario"))
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage(err.Error()))
	}
	serie := c.QueryParam("serie")
	usuario := c.QueryParam("usuario")

	if formulario == constancia.FormularioAccesorios {
		fileName := c.QueryParam("file")
		if fileName == "" {
			return c.NoContent(http.StatusBadRequest)
		}

		// Construct the full path to the file.
		filePath := filepath.Join("./pdf", fileName)

		// Clean up: remove the file after serving.
		defer os.Remove(filePath)

		// Serve the file to the user with a custom download name.
		err = c.Attachment(filePath, fmt.Sprintf("%s-%s.pdf", serie, usuario))

		return err
	} else {
		return c.NoContent(http.StatusBadRequest)
	}
}
