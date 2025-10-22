import sqlite3
import json

# Conexión a la base de datos
conn = sqlite3.connect("containers.db")
cursor = conn.cursor()

# Crear tabla si no existe
cursor.execute("""
CREATE TABLE IF NOT EXISTS sensores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sensorId TEXT NOT NULL,
    type TEXT NOT NULL,
    value REAL NOT NULL,
    unit TEXT NOT NULL,
    timestamp TEXT NOT NULL
)
""")

# Cargar datos del JSON
with open("sensores_datos.json", "r") as f:
    datos = json.load(f)

# Insertar registros
for d in datos:
    cursor.execute("""
        INSERT INTO sensores (sensorId, type, value, unit, timestamp)
        VALUES (?, ?, ?, ?, ?)
    """, (d["sensorId"], d["type"], d["value"], d["unit"], d["timestamp"]))

conn.commit()
conn.close()

print("Datos insertados correctamente en la tabla sensores.")
