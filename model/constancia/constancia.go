package constancia

import (
	"strings"
	"time"
)

type TipoProcedimiento string

const (
	ProcedimientoAsignacion   TipoProcedimiento = "ASIGNACION"
	ProcedimientoRecuperacion TipoProcedimiento = "RECUPERACION"
)

type TipoEquipo string

const (
	EquipoPC     TipoEquipo = "PC"
	EquipoLaptop TipoEquipo = "LAPTOP"
)

type TipoInventario string

const (
	InventarioMouse    TipoInventario = "MOUSE"
	InventarioPortatil TipoInventario = "PORTATIL"
	InventarioCargador TipoInventario = "CARGADOR"
	InventarioMochila  TipoInventario = "MOCHILA"
	InventarioCadena   TipoInventario = "CADENA"
)

type Equipo struct {
	Id         int64
	TipoEquipo string
	Marca      string
	MTM        string
	Modelo     string
	Serie      string
	ActivoFijo string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Cliente struct {
	Id        int64
	SapId     string
	Usuario   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Constancia struct {
	Id                 int64
	NroTicket          string
	TipoProcedimiento  TipoProcedimiento
	ResponsableUsuario string
	CodigoEmpleado     string
	FechaHora          time.Time
	Sede               string
	Piso               string
	Area               string
	TipoEquipo         TipoEquipo
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Inventario struct {
	Id             int64
	TipoInventario TipoInventario
	Marca          string
	Modelo         string
	Serie          string
	Estado         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Normalization functions

func (e Equipo) Normalize() (Equipo, error) {
	e.Serie = strings.ToUpper(strings.ReplaceAll(e.Serie, " ", ""))
	return e, nil
}

func (c Cliente) Normalize() (Cliente, error) {
	c.SapId = strings.ToLower(strings.ReplaceAll(c.SapId, " ", ""))
	return c, nil
}
