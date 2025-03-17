package constancia

import (
	"alc/model/auth"
	"errors"
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
	InventarioCableRed TipoInventario = "CABLERED"
)

func GetTipoProcedimiento(s string) (TipoProcedimiento, error) {
	if s == "ASIGNACION" {
		return ProcedimientoAsignacion, nil
	} else if s == "RECUPERACION" {
		return ProcedimientoRecuperacion, nil
	} else {
		return "", errors.New("no se encontro el tipo de procedimiento")
	}
}
func GetTipoEquipo(s string) (TipoEquipo, error) {
	if s == "PC" {
		return EquipoPC, nil
	} else if s == "LAPTOP" {
		return EquipoLaptop, nil
	} else {
		return "", errors.New("no se encontro el tipo de equipo")
	}
}
func GetTipoInventario(s string) (TipoInventario, error) {
	if s == "MOUSE" {
		return InventarioMouse, nil
	} else if s == "PORTATIL" {
		return InventarioPortatil, nil
	} else if s == "CARGADOR" {
		return InventarioCargador, nil
	} else if s == "MOCHILA" {
		return InventarioMochila, nil
	} else if s == "CADENA" {
		return InventarioCadena, nil
	} else if s == "CABLERED" {
		return InventarioCableRed, nil
	} else {
		return "", errors.New("no se encontro el tipo de inventario")
	}
}

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
	IssuedBy           auth.User
	NroTicket          string
	TipoProcedimiento  TipoProcedimiento
	ResponsableUsuario string
	CodigoEmpleado     string
	FechaHora          time.Time
	Sede               string
	Piso               string
	Area               string
	TipoEquipo         TipoEquipo
	UsuarioSAP         string
	UsuarioNombre      string
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
	Inventario     string
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

func (c Constancia) Normalize() (Constancia, error) {
	c.NroTicket = strings.TrimSpace(strings.ToUpper(c.NroTicket))
	c.ResponsableUsuario = strings.TrimSpace(strings.ToUpper(c.ResponsableUsuario))
	c.CodigoEmpleado = strings.TrimSpace(strings.ToUpper(c.CodigoEmpleado))
	c.Sede = strings.TrimSpace(strings.ToUpper(c.Sede))
	c.Piso = strings.TrimSpace(strings.ToUpper(c.Piso))
	c.Area = strings.TrimSpace(strings.ToUpper(c.Area))
	c.UsuarioSAP = strings.TrimSpace(strings.ToUpper(c.UsuarioSAP))
	c.UsuarioNombre = strings.TrimSpace(strings.ToUpper(c.UsuarioNombre))
	c.IssuedBy.Name = strings.TrimSpace(strings.ToUpper(c.IssuedBy.Name))
	if len(c.UsuarioSAP) == 0 || len(c.UsuarioNombre) == 0 {
		return Constancia{}, errors.New("invalid constancia")
	}
	return c, nil
}

func (i Inventario) Normalize() (Inventario, error) {
	i.Marca = strings.TrimSpace(strings.ToUpper(i.Marca))
	i.Modelo = strings.TrimSpace(strings.ToUpper(i.Modelo))
	i.Serie = strings.TrimSpace(strings.ToUpper(i.Serie))
	i.Estado = strings.TrimSpace(strings.ToUpper(i.Estado))
	i.Inventario = strings.TrimSpace(strings.ToUpper(i.Inventario))
	return i, nil
}
