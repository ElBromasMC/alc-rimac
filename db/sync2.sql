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

