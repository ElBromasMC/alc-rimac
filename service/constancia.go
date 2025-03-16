package service

import (
	"alc/model/constancia"
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

// GetConstanciaByID fetches a Constancia record by its primary key.
func (s Constancia) GetConstanciaByID(ctx context.Context, id int64) (constancia.Constancia, error) {
	var c constancia.Constancia
	err := s.db.QueryRow(ctx,
		`SELECT id, nro_ticket, tipo_procedimiento, responsable_usuario, codigo_empleado, fecha_hora, sede, piso, area, tipo_equipo, created_at, updated_at 
		 FROM constancias WHERE id = $1`, id).
		Scan(&c.Id, &c.NroTicket, &c.TipoProcedimiento, &c.ResponsableUsuario, &c.CodigoEmpleado, &c.FechaHora, &c.Sede, &c.Piso, &c.Area, &c.TipoEquipo, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return constancia.Constancia{}, err
	}
	return c, nil
}

// GetInventarioByID fetches an Inventario record by its primary key.
func (s Constancia) GetInventarioByID(ctx context.Context, id int64) (constancia.Inventario, error) {
	var i constancia.Inventario
	err := s.db.QueryRow(ctx,
		`SELECT id, tipo_inventario, marca, modelo, serie, estado, created_at, updated_at 
		 FROM inventario WHERE id = $1`, id).
		Scan(&i.Id, &i.TipoInventario, &i.Marca, &i.Modelo, &i.Serie, &i.Estado, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return constancia.Inventario{}, err
	}
	return i, nil
}

// Additional retrieval functions using alternate keys

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

// InsertConstanciaAndInventarios inserts a Constancia record along with a list of associated Inventario records.
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

	// Insert into constancias and retrieve the generated id.
	queryConstancia := `
		INSERT INTO constancias 
			(nro_ticket, tipo_procedimiento, responsable_usuario, codigo_empleado, fecha_hora, sede, piso, area, tipo_equipo)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryConstancia,
		c.NroTicket, c.TipoProcedimiento, c.ResponsableUsuario, c.CodigoEmpleado,
		c.FechaHora, c.Sede, c.Piso, c.Area, c.TipoEquipo,
	).Scan(&c.Id)
	if err != nil {
		return err
	}

	// Prepare the query for inserting into inventario.
	queryInventario := `
		INSERT INTO inventario 
			(tipo_inventario, marca, modelo, serie, estado, constancia_id)
		VALUES 
			($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	// Insert each inventario and retrieve its generated id.
	for idx := range inventarios {
		err = tx.QueryRow(ctx, queryInventario,
			inventarios[idx].TipoInventario, inventarios[idx].Marca, inventarios[idx].Modelo,
			inventarios[idx].Serie, inventarios[idx].Estado, c.Id,
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
