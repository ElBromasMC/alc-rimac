package service

import (
	"alc/model/constancia"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
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
			(issued_by, nro_ticket, tipo_procedimiento, responsable_usuario, codigo_empleado, fecha_hora, sede, piso, area, tipo_equipo, usuario_sap, usuario_nombre, serie, observacion)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
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
		c.Observacion,
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
            observacion = $13,
			updated_at = NOW()
		WHERE serie = $14
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
		c.Observacion,
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
//func (s Constancia) BulkInsertEquipos(ctx context.Context, equipos []constancia.Equipo) error {
//	if len(equipos) == 0 {
//		return nil // nothing to insert
//	}
//
//	// Prepare rows for CopyFrom: tipo_equipo, marca, mtm, modelo, serie, activo_fijo.
//	rows := make([][]interface{}, len(equipos))
//	for i, eq := range equipos {
//		rows[i] = []interface{}{
//			eq.TipoEquipo,
//			eq.Marca,
//			eq.MTM,
//			eq.Modelo,
//			eq.Serie,
//			eq.ActivoFijo,
//		}
//	}
//
//	// Perform the bulk insert using CopyFrom.
//	_, err := s.db.CopyFrom(
//		ctx,
//		pgx.Identifier{"equipos"},
//		[]string{"tipo_equipo", "marca", "mtm", "modelo", "serie", "activo_fijo"},
//		pgx.CopyFromRows(rows),
//	)
//	return err
//}

// BulkInsertEquipos performs a bulk insert of a list of Equipo into the equipos table.
// It avoids conflict errors by first inserting into a temporary staging table.
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

	// Begin a transaction so that the entire process is atomic.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	// Rollback the transaction if any error occurs.
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Create a temporary staging table without constraints.
	tempTableSQL := `
		CREATE TEMP TABLE temp_equipos (
			tipo_equipo VARCHAR(100),
			marca VARCHAR(100),
			mtm VARCHAR(100),
			modelo VARCHAR(100),
			serie VARCHAR(100),
			activo_fijo VARCHAR(100)
		) ON COMMIT DROP;
	`
	if _, err = tx.Exec(ctx, tempTableSQL); err != nil {
		return err
	}

	// Bulk load into the temporary table using CopyFrom.
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"temp_equipos"},
		[]string{"tipo_equipo", "marca", "mtm", "modelo", "serie", "activo_fijo"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return err
	}

	// Insert data from the staging table to the main equipos table.
	// ON CONFLICT (serie) DO NOTHING ensures that duplicate 'serie'
	// values are ignored, whether duplicates exist in the input slice or in the database already.
	upsertSQL := `
		INSERT INTO equipos (tipo_equipo, marca, mtm, modelo, serie, activo_fijo)
		SELECT tipo_equipo, marca, mtm, modelo, serie, activo_fijo
		FROM temp_equipos
		ON CONFLICT (serie) DO NOTHING;
	`
	if _, err = tx.Exec(ctx, upsertSQL); err != nil {
		return err
	}

	// Commit the transaction.
	return tx.Commit(ctx)
}

// BulkInsertClientes performs a bulk insert of a list of Cliente into the clientes table.
//func (s Constancia) BulkInsertClientes(ctx context.Context, clientes []constancia.Cliente) error {
//	if len(clientes) == 0 {
//		return nil // nothing to insert
//	}
//
//	// Prepare rows for CopyFrom: sap_id, usuario.
//	rows := make([][]interface{}, len(clientes))
//	for i, cl := range clientes {
//		rows[i] = []interface{}{
//			cl.SapId,
//			cl.Usuario,
//		}
//	}
//
//	// Perform the bulk insert using CopyFrom.
//	_, err := s.db.CopyFrom(
//		ctx,
//		pgx.Identifier{"clientes"},
//		[]string{"sap_id", "usuario"},
//		pgx.CopyFromRows(rows),
//	)
//	return err
//}

// BulkInsertClientes performs a bulk insert of a list of Cliente into the clientes table.
// It avoids conflict errors by first inserting into a temporary staging table.
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

	// Start a transaction so that all operations are atomic.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		// Ensure the transaction is rolled back in case of an error.
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Create a temporary staging table with the same structure (only needed columns) but no UNIQUE constraints.
	tempTableSQL := `
        CREATE TEMP TABLE temp_clientes (
            sap_id VARCHAR(50),
            usuario VARCHAR(255)
        ) ON COMMIT DROP;
    `
	if _, err = tx.Exec(ctx, tempTableSQL); err != nil {
		return err
	}

	// Bulk copy into the temporary table.
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"temp_clientes"},
		[]string{"sap_id", "usuario"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return err
	}

	// Upsert from the temporary table into the main clientes table.
	// Using ON CONFLICT DO NOTHING ensures that duplicate sap_id entries are ignored.
	upsertSQL := `
        INSERT INTO clientes (sap_id, usuario)
        SELECT sap_id, usuario
        FROM temp_clientes
        ON CONFLICT (sap_id) DO NOTHING;
    `
	if _, err = tx.Exec(ctx, upsertSQL); err != nil {
		return err
	}

	// Commit the transaction.
	return tx.Commit(ctx)
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

	if c.Observacion != "" {
		err = addText("1", "Observaciones: "+c.Observacion, 58, 100)
	}

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

// ExportConstanciasWithInventariosCSV writes a CSV report with all constancias,
// their associated inventarios, and the MTM from the equipos table.
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
            c.observacion,
            e.mtm,
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
            cadena.inventario,
            -- Inventario columns for PORTATILOLD
            portatil_old.marca,
            portatil_old.modelo,
            portatil_old.serie,
            portatil_old.estado,
            portatil_old.inventario,
            -- Inventario columns for CARGADOROLD
            cargador_old.marca,
            cargador_old.modelo,
            cargador_old.serie,
            cargador_old.estado,
            cargador_old.inventario
        FROM constancias c
        JOIN users u ON u.user_id = c.issued_by
        LEFT JOIN equipos e ON e.serie = c.serie
        LEFT JOIN inventario portatil ON portatil.constancia_id = c.id AND portatil.tipo_inventario = 'PORTATIL'
        LEFT JOIN inventario mouse ON mouse.constancia_id = c.id AND mouse.tipo_inventario = 'MOUSE'
        LEFT JOIN inventario cargador ON cargador.constancia_id = c.id AND cargador.tipo_inventario = 'CARGADOR'
        LEFT JOIN inventario mochila ON mochila.constancia_id = c.id AND mochila.tipo_inventario = 'MOCHILA'
        LEFT JOIN inventario cadena ON cadena.constancia_id = c.id AND cadena.tipo_inventario = 'CADENA'
        LEFT JOIN inventario portatil_old ON portatil_old.constancia_id = c.id AND portatil_old.tipo_inventario = 'PORTATILOLD'
        LEFT JOIN inventario cargador_old ON cargador_old.constancia_id = c.id AND cargador_old.tipo_inventario = 'CARGADOROLD'
        ORDER BY c.id;
    `

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	csvWriter := csv.NewWriter(w)

	header := []string{
		"id", "issued_by", "nro_ticket", "tipo_procedimiento", "responsable_usuario", "codigo_empleado",
		"fecha_hora", "sede", "piso", "area", "tipo_equipo", "usuario_sap", "usuario_nombre",
		"created_at", "updated_at",
		"mtm",
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
		// PORTATILOLD inventario fields:
		"portatil.old_marca", "portatil.old_modelo", "portatil.old_serie", "portatil.old_estado", "portatil.old_inventario",
		// CARGADOROLD inventario fields:
		"cargador.old_marca", "cargador.old_modelo", "cargador.old_serie", "cargador.old_estado", "cargador.old_inventario",
		// Observaciones
		"observaciones",
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
			observacion        sql.NullString
			mtm                sql.NullString
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
			// PORTATILOLD
			PortatilOldMarca, PortatilOldModelo, PortatilOldSerie, PortatilOldEstado, PortatilOldInventario sql.NullString
			// CARGADOROLD
			CargadorOldMarca, CargadorOldModelo, CargadorOldSerie, CargadorOldEstado, CargadorOldInventario sql.NullString
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
			&observacion,
			&mtm,
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
			// PORTATILOLD inventario columns.
			&PortatilOldMarca, &PortatilOldModelo, &PortatilOldSerie, &PortatilOldEstado, &PortatilOldInventario,
			// CARGADOROLD inventario columns.
			&CargadorOldMarca, &CargadorOldModelo, &CargadorOldSerie, &CargadorOldEstado, &CargadorOldInventario,
		)
		if err != nil {
			// Consider logging the error here as well
			return err
		}

		// Format time fields into strings (using desired format, e.g., "YYYY-MM-DD").
		loc, err := time.LoadLocation("America/Lima") // Use appropriate location
		if err != nil {
			// Handle location loading error (maybe fallback to UTC or log)
			return err // Or log and continue with UTC?
		}

		fechaHoraStr := fechaHora.In(loc).Format("2006-01-02")
		createdAtStr := createdAt.In(loc).Format("2006-01-02")
		updatedAtStr := updatedAt.In(loc).Format("2006-01-02")

		idStr := strconv.FormatInt(id, 10)

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
			nullToString(mtm),
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
			// PORTATILOLD inventario values.
			nullToString(PortatilOldMarca),
			nullToString(PortatilOldModelo),
			nullToString(PortatilOldSerie),
			nullToString(PortatilOldEstado),
			nullToString(PortatilOldInventario),
			// CARGADOROLD inventario values.
			nullToString(CargadorOldMarca),
			nullToString(CargadorOldModelo),
			nullToString(CargadorOldSerie),
			nullToString(CargadorOldEstado),
			nullToString(CargadorOldInventario),
			// Observaciones
			nullToString(observacion),
		}

		if err := csvWriter.Write(row); err != nil {
			// Consider logging the error here as well
			return err
		}
	}

	// Check for errors during row iteration (e.g., connection closed)
	if err := rows.Err(); err != nil {
		return err
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// UpdateEquipoActivoFijoBySerie updates the activo_fijo for an equipo identified by its serie.
func (s Constancia) UpdateEquipoActivoFijoBySerie(ctx context.Context, serie string, activoFijo string) error {
	// Normalize inputs (optional here, but good practice if needed)
	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))
	activoFijo = strings.TrimSpace(activoFijo)

	if serie == "" || activoFijo == "" {
		return errors.New("serie and activo fijo cannot be empty")
	}

	// Prepare the SQL UPDATE statement
	sql := `UPDATE equipos 
			SET activo_fijo = $1, updated_at = NOW() 
			WHERE serie = $2`

	// Execute the command
	commandTag, err := s.db.Exec(ctx, sql, activoFijo, serie)
	if err != nil {
		return fmt.Errorf("database error updating equipo with serie %s: %w", serie, err)
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// GetInventarioPortatilOldBySerie fetches the first inventario record matching
// tipo_inventario = 'PORTATILOLD' and the given serie.
func (s Constancia) GetInventarioPortatilOldBySerie(ctx context.Context, serie string) (constancia.Inventario, error) {
	var inv constancia.Inventario
	// Normalize serie before query
	serie = strings.ToUpper(strings.ReplaceAll(serie, " ", ""))

	query := `SELECT id, tipo_inventario, marca, modelo, serie, estado, inventario, created_at, updated_at, constancia_id
			  FROM inventario
			  WHERE tipo_inventario = $1 AND serie = $2
			  ORDER BY id ASC -- Get the first one if duplicates exist
			  LIMIT 1`

	err := s.db.QueryRow(ctx, query, constancia.InventarioPortatilOld, serie).
		Scan(&inv.Id, &inv.TipoInventario, &inv.Marca, &inv.Modelo, &inv.Serie, &inv.Estado, &inv.Inventario, &inv.CreatedAt, &inv.UpdatedAt, &inv.ConstanciaID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return constancia.Inventario{}, pgx.ErrNoRows // Explicitly return not found
		}
		return constancia.Inventario{}, fmt.Errorf("error fetching inventario PORTATILOLD by serie %s: %w", serie, err)
	}
	return inv, nil
}

// UpdateInventarioPortatilOld updates an existing PORTATILOLD inventario record,
// identified by its originalSerie. It updates the serie, inventario (RIMAC), marca, and modelo.
func (s Constancia) UpdateInventarioPortatilOld(ctx context.Context, constanciaID int64, newSerie, inventarioRimac, marca, modelo string) error {
	// Normalize inputs
	newSerie = strings.ToUpper(strings.ReplaceAll(newSerie, " ", ""))
	inventarioRimac = strings.TrimSpace(strings.ToUpper(inventarioRimac))
	marca = strings.TrimSpace(strings.ToUpper(marca))
	modelo = strings.TrimSpace(strings.ToUpper(modelo))

	if newSerie == "" {
		return errors.New("campos obligatorios faltantes para la actualización del inventario")
	}

	var inventarioID int64
	findQuery := `SELECT id FROM inventario WHERE tipo_inventario = $1 AND constancia_id = $2 ORDER BY id ASC LIMIT 1`
	err := s.db.QueryRow(ctx, findQuery, constancia.InventarioPortatilOld, constanciaID).Scan(&inventarioID)
	if err != nil {
		return fmt.Errorf("error buscando Inventario para actualizar: %w", err)
	}

	// Now update using the found ID
	updateQuery := `UPDATE inventario
					SET serie = $1, inventario = $2, marca = $3, modelo = $4, updated_at = NOW()
					WHERE id = $5 AND tipo_inventario = $6`

	commandTag, err := s.db.Exec(ctx, updateQuery, newSerie, inventarioRimac, marca, modelo, inventarioID, constancia.InventarioPortatilOld)
	if err != nil {
		// Handle potential unique constraint violation on the newSerie if it already exists for another PORTATILOLD
		// Or other database errors
		return fmt.Errorf("error actualizando inventario (ID %d, nueva serie %s): %w", inventarioID, newSerie, err)
	}

	if commandTag.RowsAffected() == 0 {
		// Should not happen if ID was found, but check anyway
		return fmt.Errorf("no se pudo actualizar el inventario (ID %d), fila no encontrada o sin cambios", inventarioID)
	}

	return nil
}

// CreateBorradoSeguro inserts a new record into the borrados_seguros table.
func (s Constancia) CreateBorradoSeguro(ctx context.Context, borrado constancia.BorradoSeguro) (int64, error) {
	var recordID int64

	// Normalize data before insertion/update
	normBorrado, err := borrado.Normalize() // Assuming you have a Normalize method
	if err != nil {
		return 0, fmt.Errorf("invalid borrado seguro data: %w", err)
	}

	// SQL query using INSERT ON CONFLICT (Upsert)
	// This requires the unique constraint `unique_borrados_seguros_serie` on the `serie` column.
	query := `
		INSERT INTO borrados_seguros
			(serie, inventario_rimac, serie_disco, marca, modelo, certificado_path, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (serie) DO UPDATE SET
			inventario_rimac = EXCLUDED.inventario_rimac,
			serie_disco = EXCLUDED.serie_disco,
			marca = EXCLUDED.marca,
			modelo = EXCLUDED.modelo,
			certificado_path = EXCLUDED.certificado_path,
			updated_at = NOW()  -- Explicitly update 'updated_at' on conflict
		RETURNING id -- Return the ID of the inserted or updated row
	`

	err = s.db.QueryRow(ctx, query,
		normBorrado.Serie,
		normBorrado.InventarioRimac,
		normBorrado.SerieDisco,
		normBorrado.Marca,
		normBorrado.Modelo,
		normBorrado.CertificadoPath, // Assumes path is already determined
	).Scan(&recordID)

	if err != nil {
		// Handle potential database errors during insert/update
		return 0, fmt.Errorf("error inserting or updating registro de borrado seguro para serie '%s': %w", normBorrado.Serie, err)
	}

	// Return the ID of the affected (inserted or updated) row
	return recordID, nil
}

// --- File Handling Helper (could be in a separate utility package) ---

// SaveUploadedFile saves the multipart file to the specified directory.
// Returns the full path of the saved file or an error.
func SaveUploadedFile(file *multipart.FileHeader, serie string, inventario string, storagePath string) (string, error) {
	if storagePath == "" {
		return "", errors.New("PDF_STORAGE_PATH no está configurado")
	}

	// Ensure storage directory exists
	err := os.MkdirAll(storagePath, os.ModePerm) // Use appropriate permissions
	if err != nil {
		return "", fmt.Errorf("no se pudo crear el directorio de almacenamiento '%s': %w", storagePath, err)
	}

	// Sanitize serie for filename (optional, but good practice)
	safeSerie := strings.ReplaceAll(strings.ToUpper(serie), " ", "")
	safeInv := strings.ReplaceAll(strings.ToUpper(inventario), " ", "")
	filename := fmt.Sprintf("%s-%s%s", safeSerie, safeInv, filepath.Ext(file.Filename))
	filePath := filepath.Join(storagePath, filename)

	// Open source file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("no se pudo abrir el archivo cargado: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("no se pudo crear el archivo de destino '%s': %w", filePath, err)
	}
	defer dst.Close()

	// Copy file content
	if _, err = io.Copy(dst, src); err != nil {
		// Attempt to remove partially created file on copy error
		os.Remove(filePath)
		return "", fmt.Errorf("no se pudo copiar el archivo a '%s': %w", filePath, err)
	}

	return filePath, nil
}

// GetInventarioPortatilOldSerieByConstanciaSerie finds the serie from the 'inventario' table
// (where tipo_inventario = 'PORTATILOLD') associated with a 'constancia' record,
// where the constancia is identified by its own unique 'serie'.
func (s Constancia) GetInventarioPortatilOldSerieByConstanciaSerie(ctx context.Context, constanciaSerie string) (string, error) {
	var inventarioSerie string

	// Normalize the input constancia serie
	constanciaSerie = strings.ToUpper(strings.ReplaceAll(constanciaSerie, " ", ""))
	if constanciaSerie == "" {
		return "", errors.New("la serie de la constancia no puede estar vacía")
	}

	// SQL query using JOIN to link constancias and inventario
	query := `
		SELECT i.serie
		FROM inventario i
		JOIN constancias c ON i.constancia_id = c.id
		WHERE c.serie = $1
		  AND i.tipo_inventario = $2
		LIMIT 1 -- Ensures only one row is returned, assumes one PORTATILOLD per constancia
	`

	err := s.db.QueryRow(ctx, query, constanciaSerie, constancia.InventarioPortatilOld).Scan(&inventarioSerie)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// This means either the constancia with the given serie wasn't found,
			// or it was found but had no associated PORTATILOLD inventario record.
			return "", pgx.ErrNoRows // Return specific error for "not found"
		}
		// Handle other potential database errors
		return "", fmt.Errorf("error buscando serie de inventario PORTATILOLD por serie de constancia '%s': %w", constanciaSerie, err)
	}

	// Return the found inventario serie
	return inventarioSerie, nil
}

// GetAllBorradosSeguros fetches all records from the borrados_seguros table.
func (s Constancia) GetAllBorradosSeguros(ctx context.Context) ([]constancia.BorradoSeguro, error) {
	var borrados []constancia.BorradoSeguro

	query := `SELECT id, serie, inventario_rimac, serie_disco, marca, modelo, certificado_path, created_at, updated_at
			  FROM borrados_seguros
			  ORDER BY created_at DESC` // Order by creation date, newest first

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying borrados_seguros: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var b constancia.BorradoSeguro
		// Ensure UpdatedAt is included in the struct and Scan if you added the column
		err := rows.Scan(
			&b.Id,
			&b.Serie,
			&b.InventarioRimac,
			&b.SerieDisco,
			&b.Marca,
			&b.Modelo,
			&b.CertificadoPath,
			&b.CreatedAt,
			&b.UpdatedAt, // Add this if the column and field exist
		)
		if err != nil {
			// Log the specific error for debugging
			// log.Printf("Error scanning borrado seguro row: %v", err)
			// Decide if you want to skip the row or return the error
			return nil, fmt.Errorf("error scanning borrado seguro row: %w", err)
		}
		borrados = append(borrados, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating borrados_seguros rows: %w", err)
	}

	return borrados, nil
}

// GetEquiposWithActivoFijo fetches all records from the equipos table that have a non-empty activo_fijo.
func (s Constancia) GetEquiposWithActivoFijo(ctx context.Context) ([]constancia.Equipo, error) {
	var equipos []constancia.Equipo

	// Filter where activo_fijo is not NULL and not an empty string
	query := `SELECT id, tipo_equipo, marca, mtm, modelo, serie, activo_fijo, created_at, updated_at
			  FROM equipos
			  WHERE activo_fijo IS NOT NULL AND activo_fijo <> ''
			  ORDER BY updated_at DESC` // Order by update date, newest first

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying equipos with activo_fijo: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e constancia.Equipo
		err := rows.Scan(
			&e.Id,
			&e.TipoEquipo,
			&e.Marca,
			&e.MTM,
			&e.Modelo,
			&e.Serie,
			&e.ActivoFijo,
			&e.CreatedAt,
			&e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning equipo row: %w", err)
		}
		equipos = append(equipos, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating equipos rows: %w", err)
	}

	return equipos, nil
}
