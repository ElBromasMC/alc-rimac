package service

import (
	"alc/model/constancia"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"io"
	"strconv"
	"strings"
	"time"
)

type Constancia struct {
	db *pgxpool.Pool
}

func NewConstanciaService(db *pgxpool.Pool) Constancia {
	return Constancia{
		db: db,
	}
}

// GetEquipoByID fetches an Equipo record by its primary key.
func (s Constancia) GetEquipoByID(ctx context.Context, id int64) (constancia.Equipo, error) {
	var equipo constancia.Equipo
	err := s.db.QueryRow(ctx,
		`SELECT id, tipo_equipo, marca, mtm, modelo, serie, activo_fijo, created_at, updated_at 
		 FROM equipos WHERE id = $1`, id).
		Scan(&equipo.Id, &equipo.TipoEquipo, &equipo.Marca, &equipo.MTM, &equipo.Modelo, &equipo.Serie, &equipo.ActivoFijo, &equipo.CreatedAt, &equipo.UpdatedAt)
	if err != nil {
		return constancia.Equipo{}, err
	}
	return equipo, nil
}

// GetClienteByID fetches a Cliente record by its primary key.
func (s Constancia) GetClienteByID(ctx context.Context, id int64) (constancia.Cliente, error) {
	var cliente constancia.Cliente
	err := s.db.QueryRow(ctx,
		`SELECT id, sap_id, usuario, created_at, updated_at 
		 FROM clientes WHERE id = $1`, id).
		Scan(&cliente.Id, &cliente.SapId, &cliente.Usuario, &cliente.CreatedAt, &cliente.UpdatedAt)
	if err != nil {
		return constancia.Cliente{}, err
	}
	return cliente, nil
}

// GetEquipoBySerie fetches an Equipo record by its unique Serie.
func (s Constancia) GetEquipoBySerie(ctx context.Context, serie string) (constancia.Equipo, error) {
	var equipo constancia.Equipo
	err := s.db.QueryRow(ctx,
		`SELECT id, tipo_equipo, marca, mtm, modelo, serie, activo_fijo, created_at, updated_at 
		 FROM equipos WHERE serie = $1`, serie).
		Scan(&equipo.Id, &equipo.TipoEquipo, &equipo.Marca, &equipo.MTM, &equipo.Modelo, &equipo.Serie, &equipo.ActivoFijo, &equipo.CreatedAt, &equipo.UpdatedAt)
	if err != nil {
		return constancia.Equipo{}, err
	}
	return equipo, nil
}

// GetClienteBySapId fetches a Cliente record by its unique SapId.
func (s Constancia) GetClienteBySapId(ctx context.Context, sapId string) (constancia.Cliente, error) {
	var cliente constancia.Cliente
	err := s.db.QueryRow(ctx,
		`SELECT id, sap_id, usuario, created_at, updated_at 
		 FROM clientes WHERE sap_id = $1`, sapId).
		Scan(&cliente.Id, &cliente.SapId, &cliente.Usuario, &cliente.CreatedAt, &cliente.UpdatedAt)
	if err != nil {
		return constancia.Cliente{}, err
	}
	return cliente, nil
}

// GetConstanciaBySerie retrieves a constancia record based on its serie.
func (s Constancia) GetConstanciaBySerie(ctx context.Context, serie string) (constancia.Constancia, error) {
	query := `
		SELECT 
			id, 
			issued_by, 
			nro_ticket, 
			tipo_procedimiento, 
			responsable_usuario, 
			codigo_empleado, 
			fecha_hora, 
			sede, 
			piso, 
			area, 
			tipo_equipo, 
			usuario_sap, 
			usuario_nombre, 
			serie, 
			created_at, 
			updated_at
		FROM constancias
		WHERE serie = $1
	`
	var c constancia.Constancia
	err := s.db.QueryRow(ctx, query, serie).Scan(
		&c.Id,
		&c.IssuedBy.Id, // assuming that IssuedBy is an auth.User with an Id field
		&c.NroTicket,
		&c.TipoProcedimiento,
		&c.ResponsableUsuario,
		&c.CodigoEmpleado,
		&c.FechaHora,
		&c.Sede,
		&c.Piso,
		&c.Area,
		&c.TipoEquipo,
		&c.UsuarioSAP,
		&c.UsuarioNombre,
		&c.Serie,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return constancia.Constancia{}, err
	}
	return c, nil
}

// InsertConstanciaAndInventarios inserts a Constancia record along with its associated Inventario records.
// All inserts are performed within a transaction so that they either all succeed or all fail.
func (s Constancia) InsertConstanciaAndInventarios(ctx context.Context, c constancia.Constancia, inventarios []constancia.Inventario) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	// Ensure the transaction is either committed or rolled back.
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	queryConstancia := `
		INSERT INTO constancias 
			(issued_by, nro_ticket, tipo_procedimiento, responsable_usuario, codigo_empleado, fecha_hora, sede, piso, area, tipo_equipo, usuario_sap, usuario_nombre, serie)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryConstancia,
		c.IssuedBy.Id,
		c.NroTicket,
		c.TipoProcedimiento,
		c.ResponsableUsuario,
		c.CodigoEmpleado,
		c.FechaHora,
		c.Sede,
		c.Piso,
		c.Area,
		c.TipoEquipo,
		c.UsuarioSAP,
		c.UsuarioNombre,
		c.Serie,
	).Scan(&c.Id)
	if err != nil {
		return err
	}

	queryInventario := `
		INSERT INTO inventario 
			(tipo_inventario, marca, modelo, serie, estado, inventario, constancia_id)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	for idx := range inventarios {
		err = tx.QueryRow(ctx, queryInventario,
			inventarios[idx].TipoInventario,
			inventarios[idx].Marca,
			inventarios[idx].Modelo,
			inventarios[idx].Serie,
			inventarios[idx].Estado,
			inventarios[idx].Inventario,
			c.Id,
		).Scan(&inventarios[idx].Id)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConstanciaExists checks if a constancia with the given serie exists.
func (s Constancia) ConstanciaExists(ctx context.Context, serie string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM constancias WHERE serie = $1)`
	if err := s.db.QueryRow(ctx, query, serie).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// UpdateConstanciaAndInventarios updates an existing constancia identified by its serie,
// and recreates its associated inventario records.
func (s Constancia) UpdateConstanciaAndInventarios(ctx context.Context, c constancia.Constancia, inventarios []constancia.Inventario) error {
	// Start a transaction.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	// Ensure the transaction is either committed or rolled back.
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	// Update the constancia record. Note that we use the unique 'serie' to identify the row.
	updateConstanciaQuery := `
		UPDATE constancias 
		SET 
			issued_by = $1,
			nro_ticket = $2,
			tipo_procedimiento = $3,
			responsable_usuario = $4,
			codigo_empleado = $5,
			fecha_hora = $6,
			sede = $7,
			piso = $8,
			area = $9,
			tipo_equipo = $10,
			usuario_sap = $11,
			usuario_nombre = $12,
			updated_at = NOW()
		WHERE serie = $13
		RETURNING id
	`
	err = tx.QueryRow(ctx, updateConstanciaQuery,
		c.IssuedBy.Id,
		c.NroTicket,
		c.TipoProcedimiento,
		c.ResponsableUsuario,
		c.CodigoEmpleado,
		c.FechaHora,
		c.Sede,
		c.Piso,
		c.Area,
		c.TipoEquipo,
		c.UsuarioSAP,
		c.UsuarioNombre,
		c.Serie,
	).Scan(&c.Id)
	if err != nil {
		return err
	}

	// Delete all existing inventario records for this constancia.
	deleteInventarioQuery := `DELETE FROM inventario WHERE constancia_id = $1`
	_, err = tx.Exec(ctx, deleteInventarioQuery, c.Id)
	if err != nil {
		return err
	}

	// Insert new inventario records.
	insertInventarioQuery := `
		INSERT INTO inventario 
			(tipo_inventario, marca, modelo, serie, estado, inventario, constancia_id)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	for idx := range inventarios {
		err = tx.QueryRow(ctx, insertInventarioQuery,
			inventarios[idx].TipoInventario,
			inventarios[idx].Marca,
			inventarios[idx].Modelo,
			inventarios[idx].Serie,
			inventarios[idx].Estado,
			inventarios[idx].Inventario,
			c.Id,
		).Scan(&inventarios[idx].Id)
		if err != nil {
			return err
		}
	}

	return nil
}

// BulkInsertEquipos performs a bulk insert of a list of Equipo into the equipos table.
func (s Constancia) BulkInsertEquipos(ctx context.Context, equipos []constancia.Equipo) error {
	if len(equipos) == 0 {
		return nil // nothing to insert
	}

	// Prepare rows for CopyFrom: tipo_equipo, marca, mtm, modelo, serie, activo_fijo.
	rows := make([][]interface{}, len(equipos))
	for i, eq := range equipos {
		rows[i] = []interface{}{
			eq.TipoEquipo,
			eq.Marca,
			eq.MTM,
			eq.Modelo,
			eq.Serie,
			eq.ActivoFijo,
		}
	}

	// Perform the bulk insert using CopyFrom.
	_, err := s.db.CopyFrom(
		ctx,
		pgx.Identifier{"equipos"},
		[]string{"tipo_equipo", "marca", "mtm", "modelo", "serie", "activo_fijo"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// BulkInsertClientes performs a bulk insert of a list of Cliente into the clientes table.
func (s Constancia) BulkInsertClientes(ctx context.Context, clientes []constancia.Cliente) error {
	if len(clientes) == 0 {
		return nil // nothing to insert
	}

	// Prepare rows for CopyFrom: sap_id, usuario.
	rows := make([][]interface{}, len(clientes))
	for i, cl := range clientes {
		rows[i] = []interface{}{
			cl.SapId,
			cl.Usuario,
		}
	}

	// Perform the bulk insert using CopyFrom.
	_, err := s.db.CopyFrom(
		ctx,
		pgx.Identifier{"clientes"},
		[]string{"sap_id", "usuario"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func (s Constancia) GeneratePDF(ctx context.Context, filename string, c constancia.Constancia, inventarios []constancia.Inventario) error {
	descSimpleText := "points:8, scale:1 abs, pos:bl, offset: %.2f %.2f, rot:0, mo:0, c: 0 0 0"
	descSmallText := "points:6, scale:1 abs, pos:bl, offset: %.2f %.2f, rot:0, mo:0, c: 0 0 0"
	descX := "points:8, scale:1 abs, pos:bl, offset: %.2f %.2f, rot:0, mo:2, c: 0 0 0, strokecolor: 0 0 0"
	addText := func(page, text string, x, y float64) error {
		if text == "" {
			return nil
		}
		err := api.AddTextWatermarksFile(
			filename,
			"",
			[]string{page},
			true,
			text,
			fmt.Sprintf(descSimpleText, x, y),
			nil,
		)
		return err
	}
	addSText := func(page, text string, x, y float64) error {
		if text == "" {
			return nil
		}
		err := api.AddTextWatermarksFile(
			filename,
			"",
			[]string{page},
			true,
			text,
			fmt.Sprintf(descSmallText, x, y),
			nil,
		)
		return err
	}
	addX := func(page string, x, y float64) error {
		err := api.AddTextWatermarksFile(
			filename,
			"",
			[]string{page},
			true,
			"X",
			fmt.Sprintf(descX, x, y),
			nil,
		)
		return err
	}

	loc, err := time.LoadLocation("America/Lima")
	if err != nil {
		return err
	}
	t := c.FechaHora.In(loc)
	timeFt := t.Format("02/01/2006 15:04:05")

	crdY := 707.0
	spaceY := 23.7
	err = addText("1", c.NroTicket, 232, crdY)
	if c.TipoProcedimiento == constancia.ProcedimientoAsignacion {
		err = addX("1", 287, 684)
	} else if c.TipoProcedimiento == constancia.ProcedimientoRecuperacion {
		err = addX("1", 414.5, 684)
	}
	err = addText("1", c.ResponsableUsuario, 232, crdY-2*spaceY)
	err = addText("1", c.CodigoEmpleado, 232, crdY-3*spaceY)
	err = addText("1", timeFt, 232, crdY-4*spaceY)
	err = addText("1", c.Sede, 232, crdY-5*spaceY)
	err = addText("1", c.Piso, 232, crdY-6*spaceY)
	err = addText("1", c.Area, 232, crdY-7*spaceY)

	if c.TipoEquipo == constancia.EquipoPC {
		err = addX("1", 309, 498.5)
	} else if c.TipoEquipo == constancia.EquipoLaptop {
		err = addX("1", 411.7, 498.5)
	}

	getPosition := func(x constancia.TipoInventario) int {
		if x == constancia.InventarioMouse {
			return 3
		} else if x == constancia.InventarioCableRed {
			return 5
		} else if x == constancia.InventarioPortatil {
			return 6
		} else if x == constancia.InventarioCargador {
			return 7
		} else if x == constancia.InventarioMochila {
			return 8
		} else if x == constancia.InventarioCadena {
			return 9
		} else {
			return 0
		}
	}

	crdXTable := 101.2
	crdYTable := 465.5
	spaceXTable := 82.0
	spaceYTable := 15.9
	separationX := 130.0
	startX := crdXTable + separationX

	for _, item := range inventarios {
		if item.Marca == "" &&
			item.Modelo == "" &&
			item.Serie == "" &&
			item.Inventario == "" &&
			item.Estado == "" {
			continue
		}
		pos := float64(getPosition(item.TipoInventario))
		err = addX("1", crdXTable, crdYTable-pos*spaceYTable)
		err = addSText("1", item.Marca, startX, crdYTable-pos*spaceYTable)
		err = addSText("1", item.Modelo, startX+spaceXTable, crdYTable-pos*spaceYTable)
		if item.Serie != "" || item.Inventario != "" {
			err = addSText("1", item.Serie+" | "+item.Inventario, startX+2*spaceXTable, crdYTable-pos*spaceYTable)
		}
		err = addSText("1", item.Estado, startX+3*spaceXTable, crdYTable-pos*spaceYTable)
	}

	err = addText("2", c.UsuarioNombre, 105, 675.5)
	err = addText("2", c.IssuedBy.Name, 105, 627)

	return err
}

// ExportConstanciasWithInventariosCSV writes a CSV report with all constancias and their associated inventarios.
func (s Constancia) ExportConstanciasWithInventariosCSV(ctx context.Context, w io.Writer) error {
	query := `
		SELECT 
			c.id,
            u.name AS issued_by,
			c.nro_ticket,
			c.tipo_procedimiento,
			c.responsable_usuario,
			c.codigo_empleado,
			c.fecha_hora,
			c.sede,
			c.piso,
			c.area,
			c.tipo_equipo,
			c.usuario_sap,
			c.usuario_nombre,
			c.created_at,
			c.updated_at,
			-- Inventario columns for PORTATIL
			portatil.marca,
			portatil.modelo,
			portatil.serie,
			portatil.estado,
			portatil.inventario,
			-- Inventario columns for MOUSE
			mouse.marca,
			mouse.modelo,
			mouse.serie,
			mouse.estado,
			mouse.inventario,
			-- Inventario columns for CARGADOR
			cargador.marca,
			cargador.modelo,
			cargador.serie,
			cargador.estado,
			cargador.inventario,
			-- Inventario columns for MOCHILA
			mochila.marca,
			mochila.modelo,
			mochila.serie,
			mochila.estado,
			mochila.inventario,
			-- Inventario columns for CADENA
			cadena.marca,
			cadena.modelo,
			cadena.serie,
			cadena.estado,
			cadena.inventario
		FROM constancias c
        JOIN users u ON u.user_id = c.issued_by
		LEFT JOIN inventario portatil ON portatil.constancia_id = c.id AND portatil.tipo_inventario = 'PORTATIL'
		LEFT JOIN inventario mouse ON mouse.constancia_id = c.id AND mouse.tipo_inventario = 'MOUSE'
		LEFT JOIN inventario cargador ON cargador.constancia_id = c.id AND cargador.tipo_inventario = 'CARGADOR'
		LEFT JOIN inventario mochila ON mochila.constancia_id = c.id AND mochila.tipo_inventario = 'MOCHILA'
		LEFT JOIN inventario cadena ON cadena.constancia_id = c.id AND cadena.tipo_inventario = 'CADENA'
		ORDER BY c.id;
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	csvWriter := csv.NewWriter(w)

	// Write CSV header row.
	header := []string{
		"id", "issued_by", "nro_ticket", "tipo_procedimiento", "responsable_usuario", "codigo_empleado",
		"fecha_hora", "sede", "piso", "area", "tipo_equipo", "usuario_sap", "usuario_nombre",
		"created_at", "updated_at",
		// PORTATIL inventario fields:
		"portatil_marca", "portatil_modelo", "portatil_serie", "portatil_estado", "portatil_inventario",
		// MOUSE inventario fields:
		"mouse_marca", "mouse_modelo", "mouse_serie", "mouse_estado", "mouse_inventario",
		// CARGADOR inventario fields:
		"cargador_marca", "cargador_modelo", "cargador_serie", "cargador_estado", "cargador_inventario",
		// MOCHILA inventario fields:
		"mochila_marca", "mochila_modelo", "mochila_serie", "mochila_estado", "mochila_inventario",
		// CADENA inventario fields:
		"cadena_marca", "cadena_modelo", "cadena_serie", "cadena_estado", "cadena_inventario",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	// Helper to convert sql.NullString to plain string.
	nullToString := func(ns sql.NullString) string {
		if ns.Valid {
			return ns.String
		}
		return ""
	}

	for rows.Next() {
		// Declare variables for constancia columns.
		var (
			id                 int64
			issuedBy           string
			nroTicket          string
			tipoProcedimiento  string
			responsableUsuario string
			codigoEmpleado     string
			fechaHora          time.Time
			sede               string
			piso               string
			area               string
			tipoEquipo         string
			usuarioSAP         string
			usuarioNombre      string
			createdAt          time.Time
			updatedAt          time.Time
		)

		// Inventario fields for each type (using sql.NullString to handle possible NULLs)
		var (
			// PORTATIL
			portatilMarca, portatilModelo, portatilSerie, portatilEstado, portatilInventario sql.NullString
			// MOUSE
			mouseMarca, mouseModelo, mouseSerie, mouseEstado, mouseInventario sql.NullString
			// CARGADOR
			cargadorMarca, cargadorModelo, cargadorSerie, cargadorEstado, cargadorInventario sql.NullString
			// MOCHILA
			mochilaMarca, mochilaModelo, mochilaSerie, mochilaEstado, mochilaInventario sql.NullString
			// CADENA
			cadenaMarca, cadenaModelo, cadenaSerie, cadenaEstado, cadenaInventario sql.NullString
		)

		err := rows.Scan(
			// Constancia columns.
			&id,
			&issuedBy,
			&nroTicket,
			&tipoProcedimiento,
			&responsableUsuario,
			&codigoEmpleado,
			&fechaHora,
			&sede,
			&piso,
			&area,
			&tipoEquipo,
			&usuarioSAP,
			&usuarioNombre,
			&createdAt,
			&updatedAt,
			// PORTATIL inventario columns.
			&portatilMarca, &portatilModelo, &portatilSerie, &portatilEstado, &portatilInventario,
			// MOUSE inventario columns.
			&mouseMarca, &mouseModelo, &mouseSerie, &mouseEstado, &mouseInventario,
			// CARGADOR inventario columns.
			&cargadorMarca, &cargadorModelo, &cargadorSerie, &cargadorEstado, &cargadorInventario,
			// MOCHILA inventario columns.
			&mochilaMarca, &mochilaModelo, &mochilaSerie, &mochilaEstado, &mochilaInventario,
			// CADENA inventario columns.
			&cadenaMarca, &cadenaModelo, &cadenaSerie, &cadenaEstado, &cadenaInventario,
		)
		if err != nil {
			return err
		}

		// Format time fields into strings (using RFC3339, adjust as needed).
		loc, err := time.LoadLocation("America/Lima")
		if err != nil {
			return err
		}

		fechaHoraStr := fechaHora.In(loc).Format("2006-01-02")
		createdAtStr := createdAt.In(loc).Format("2006-01-02")
		updatedAtStr := updatedAt.In(loc).Format("2006-01-02")

		idStr := strconv.FormatInt(id, 10)

		// Build the CSV row.
		row := []string{
			idStr,
			strings.ToUpper(issuedBy),
			nroTicket,
			tipoProcedimiento,
			responsableUsuario,
			codigoEmpleado,
			fechaHoraStr,
			sede,
			piso,
			area,
			tipoEquipo,
			usuarioSAP,
			usuarioNombre,
			createdAtStr,
			updatedAtStr,
			// PORTATIL inventario values.
			nullToString(portatilMarca),
			nullToString(portatilModelo),
			nullToString(portatilSerie),
			nullToString(portatilEstado),
			nullToString(portatilInventario),
			// MOUSE inventario values.
			nullToString(mouseMarca),
			nullToString(mouseModelo),
			nullToString(mouseSerie),
			nullToString(mouseEstado),
			nullToString(mouseInventario),
			// CARGADOR inventario values.
			nullToString(cargadorMarca),
			nullToString(cargadorModelo),
			nullToString(cargadorSerie),
			nullToString(cargadorEstado),
			nullToString(cargadorInventario),
			// MOCHILA inventario values.
			nullToString(mochilaMarca),
			nullToString(mochilaModelo),
			nullToString(mochilaSerie),
			nullToString(mochilaEstado),
			nullToString(mochilaInventario),
			// CADENA inventario values.
			nullToString(cadenaMarca),
			nullToString(cadenaModelo),
			nullToString(cadenaSerie),
			nullToString(cadenaEstado),
			nullToString(cadenaInventario),
		}

		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}
