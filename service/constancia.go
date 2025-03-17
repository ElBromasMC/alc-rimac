package service

import (
	"alc/model/constancia"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pdfcpu/pdfcpu/pkg/api"
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
			(issued_by, nro_ticket, tipo_procedimiento, responsable_usuario, codigo_empleado, fecha_hora, sede, piso, area, tipo_equipo, usuario_sap, usuario_nombre)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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

func (s Constancia) GeneratePDF(ctx context.Context, c constancia.Constancia, inventarios []constancia.Inventario) error {
	descSimpleText := "points:8, scale:1 abs, pos:bl, offset: %.2f %.2f, rot:0, mo:0, c: 0 0 0"
	descSmallText := "points:6, scale:1 abs, pos:bl, offset: %.2f %.2f, rot:0, mo:0, c: 0 0 0"
	descX := "points:8, scale:1 abs, pos:bl, offset: %.2f %.2f, rot:0, mo:2, c: 0 0 0, strokecolor: 0 0 0"
	addText := func(page, text string, x, y float64) error {
		err := api.AddTextWatermarksFile(
			"output.pdf",
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
		err := api.AddTextWatermarksFile(
			"output.pdf",
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
			"output.pdf",
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
		pos := float64(getPosition(item.TipoInventario))
		err = addX("1", crdXTable, crdYTable-pos*spaceYTable)
		err = addSText("1", item.Marca, startX, crdYTable-pos*spaceYTable)
		err = addSText("1", item.Modelo, startX+spaceXTable, crdYTable-pos*spaceYTable)
		err = addSText("1", item.Serie+" | "+item.Inventario, startX+2*spaceXTable, crdYTable-pos*spaceYTable)
		err = addSText("1", item.Estado, startX+3*spaceXTable, crdYTable-pos*spaceYTable)
	}

	err = addText("2", c.UsuarioNombre, 105, 675.5)
	err = addText("2", c.IssuedBy.Name, 105, 627)

	return err
}
