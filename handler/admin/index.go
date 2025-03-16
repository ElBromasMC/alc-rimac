package admin

import (
	"alc/handler/util"
	"alc/model/constancia"
	"alc/view/admin"
	"alc/view/component"
	"context"
	"encoding/csv"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
)

func (h *Handler) HandleIndexShow(c echo.Context) error {
	return util.Render(c, http.StatusOK, admin.Index())
}

func (h *Handler) HandleEquiposInsertion(c echo.Context) error {
	// Parsing request
	file, err := c.FormFile("EquiposData")
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Debe proporcionar los equipos"))
	}
	src, err := file.Open()
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al abrir los equipos"))
	}

	equipos, err := parseEquiposFromCSV(src)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al procesar los equipos"))
	}

	err = h.ConstanciaService.BulkInsertEquipos(context.Background(), equipos)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al subir los equipos a la base de datos: "+err.Error()))
	}
	return util.Render(c, http.StatusOK, component.InfoMessage(fmt.Sprintf("Número de equipos: %d", len(equipos))))
}

func (h *Handler) HandleClientesInsertion(c echo.Context) error {
	// Parsing request
	file, err := c.FormFile("ClientesData")
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Debe proporcionar los usuarios"))
	}
	src, err := file.Open()
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al abrir los usuarios"))
	}

	clientes, err := parseClientesFromCSV(src)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al procesar los usuarios"))
	}

	err = h.ConstanciaService.BulkInsertClientes(context.Background(), clientes)
	if err != nil {
		return util.Render(c, http.StatusOK, component.ErrorMessage("Error al subir los usuarios a la base de datos: "+err.Error()))
	}
	return util.Render(c, http.StatusOK, component.InfoMessage(fmt.Sprintf("Número de clientes: %d", len(clientes))))
}

func parseEquiposFromCSV(src io.Reader) ([]constancia.Equipo, error) {
	csvReader := csv.NewReader(src)
	csvReader.FieldsPerRecord = 6

	// Read and discard header row.
	if _, err := csvReader.Read(); err != nil {
		return nil, err
	}
	var equipos []constancia.Equipo
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break // reached end of file
		}
		if err != nil {
			return nil, err
		}

		// Create a new Equipo from the record.
		equipo := constancia.Equipo{
			TipoEquipo: record[0], // maps to "tipo_equipo_nuevo"
			Marca:      record[1], // maps to "marca_equipo_nuevo"
			MTM:        record[2], // maps to "mtm"
			Modelo:     record[3], // maps to "modelo_equipo_nuevo"
			Serie:      record[4], // maps to "serie_equipo_nuevo"
			ActivoFijo: record[5], // maps to "activo.fijo_equipo.nuevo"
		}
		equipo, err = equipo.Normalize()
		equipos = append(equipos, equipo)
	}

	return equipos, nil
}

func parseClientesFromCSV(src io.Reader) ([]constancia.Cliente, error) {
	csvReader := csv.NewReader(src)
	csvReader.FieldsPerRecord = 2

	// Read and discard header row.
	if _, err := csvReader.Read(); err != nil {
		return nil, err
	}

	var clientes []constancia.Cliente
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break // reached end of file
		}
		if err != nil {
			return nil, err
		}

		// Create a new Cliente from the record.
		cliente := constancia.Cliente{
			SapId:   record[0], // maps to "sap"
			Usuario: record[1], // maps to "user"
		}
		cliente, err = cliente.Normalize()
		clientes = append(clientes, cliente)
	}

	return clientes, nil
}
