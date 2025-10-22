-- Temperatura (serie completa en tiempo real)
SELECT 
    strftime('%s', timestamp) as time,
    value as temperature
FROM sensores
WHERE type = 'temperature'
ORDER BY time;

-- Temperatura (promedio por hora)

SELECT 
    strftime('%s', strftime('%Y-%m-%d %H:00:00', timestamp)) as time,
    AVG(value) as avg_temperature
FROM sensores
WHERE type = 'temperature'
GROUP BY strftime('%Y-%m-%d %H:00:00', timestamp)
ORDER BY time;

-- Distancia (serie completa en tiempo real)

SELECT 
    strftime('%s', timestamp) as time,
    value as distance
FROM sensores
WHERE type = 'distance'
ORDER BY time;

-- Distancia (mínimo y máximo por hora)

SELECT 
    strftime('%s', strftime('%Y-%m-%d %H:00:00', timestamp)) as time,
    MIN(value) as min_distance,
    MAX(value) as max_distance
FROM sensores
WHERE type = 'distance'
GROUP BY strftime('%Y-%m-%d %H:00:00', timestamp)
ORDER BY time;
