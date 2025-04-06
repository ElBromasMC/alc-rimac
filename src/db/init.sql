CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- User administration
CREATE TYPE user_role AS ENUM ('ADMIN', 'NORMAL');

CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    hashed_password TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'NORMAL',
    dni VARCHAR(25) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE TYPE tipo_procedimiento_enum AS ENUM ('ASIGNACION', 'RECUPERACION');
CREATE TYPE tipo_equipo_enum AS ENUM ('PC', 'LAPTOP');
CREATE TYPE tipo_inventario_enum AS ENUM ('MOUSE', 'PORTATIL', 'CARGADOR', 'MOCHILA', 'CADENA');

CREATE TABLE equipos (
    id BIGSERIAL PRIMARY KEY,
    tipo_equipo VARCHAR(100) NOT NULL,
    marca VARCHAR(100) NOT NULL,
    mtm VARCHAR(100) NOT NULL,
    modelo VARCHAR(100) NOT NULL,
    serie VARCHAR(100) UNIQUE NOT NULL,
    activo_fijo VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE clientes (
    id BIGSERIAL PRIMARY KEY,
    sap_id VARCHAR(50) UNIQUE NOT NULL,
    usuario VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE constancias (
    id BIGSERIAL PRIMARY KEY,
    issued_by UUID NOT NULL,
    nro_ticket VARCHAR(50) NOT NULL DEFAULT '',
    tipo_procedimiento tipo_procedimiento_enum NOT NULL,
    responsable_usuario VARCHAR(255) NOT NULL,
    codigo_empleado VARCHAR(255) NOT NULL,
    fecha_hora TIMESTAMPTZ NOT NULL,
    sede VARCHAR(255) NOT NULL,
    piso VARCHAR(50) NOT NULL,
    area VARCHAR(255) NOT NULL,
    tipo_equipo tipo_equipo_enum NOT NULL,
    usuario_sap VARCHAR(100) NOT NULL,
    usuario_nombre VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (issued_by) REFERENCES users(user_id) ON DELETE RESTRICT
);

CREATE TABLE inventario (
    id BIGSERIAL PRIMARY KEY,
    tipo_inventario tipo_inventario_enum NOT NULL,
    marca VARCHAR(100) NOT NULL,
    modelo VARCHAR(100) NOT NULL,
    serie VARCHAR(100) NOT NULL,
    estado VARCHAR(100) NOT NULL,
    inventario VARCHAR(100) NOT NULL,
    constancia_id BIGINT NOT NULL REFERENCES constancias(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

--
-- Sync 1
--

ALTER TYPE tipo_inventario_enum ADD VALUE 'CABLERED';

--
-- Sync 2
--

-- Add new column
ALTER TABLE constancias ADD COLUMN serie VARCHAR(100);

-- Populate series
UPDATE constancias c
SET serie = (
    SELECT i.serie
    FROM inventario i
    WHERE i.constancia_id = c.id AND i.tipo_inventario = 'PORTATIL'
    LIMIT 1
)
WHERE EXISTS (
    SELECT 1 FROM inventario i WHERE i.constancia_id = c.id AND i.tipo_inventario = 'PORTATIL'
);

-- Identify duplicates (for inspection)
SELECT serie, ARRAY_AGG(id ORDER BY id DESC) AS duplicate_ids
FROM constancias
WHERE serie IS NOT NULL
GROUP BY serie
HAVING COUNT(*) > 1;

-- Delete duplicates from inventario (keeping the constancia with the highest id)
WITH duplicates AS (
    SELECT serie, MAX(id) AS keep_id, ARRAY_AGG(id ORDER BY id DESC) AS duplicate_ids
    FROM constancias
    WHERE serie IS NOT NULL
    GROUP BY serie
    HAVING COUNT(*) > 1
)
DELETE FROM inventario
WHERE constancia_id IN (
    SELECT UNNEST(duplicate_ids[2:])  -- Skip the first element (the highest id)
    FROM duplicates
);

-- Delete duplicates from constancias (keeping the last entry based on id)
WITH duplicates AS (
    SELECT serie, MAX(id) AS keep_id, ARRAY_AGG(id ORDER BY id DESC) AS duplicate_ids
    FROM constancias
    WHERE serie IS NOT NULL
    GROUP BY serie
    HAVING COUNT(*) > 1
)
DELETE FROM constancias
WHERE id IN (
    SELECT UNNEST(duplicate_ids[2:])  -- Skip the highest id and delete the rest
    FROM duplicates
);

-- Add constraints (UNIQUE and NOT NULL)
ALTER TABLE constancias 
ALTER COLUMN serie SET NOT NULL,
ADD CONSTRAINT unique_serie UNIQUE (serie);

--
-- Sync 3
--

ALTER TYPE tipo_inventario_enum ADD VALUE 'PORTATILOLD';
ALTER TYPE tipo_inventario_enum ADD VALUE 'CARGADOROLD';

ALTER TABLE constancias ADD COLUMN observacion TEXT NOT NULL DEFAULT '';

