-- k8s/postgres/init.sql
-- 1. Tabla de Usuarios
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Tabla de Clientes OAuth (Apps)
CREATE TABLE IF NOT EXISTS oauth_clients (
    id VARCHAR(100) PRIMARY KEY, -- Client ID
    client_secret VARCHAR(255) NOT NULL,
    redirect_uri VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. Datos semilla (Seed Data)
-- Usuario Admin
INSERT INTO users (email, password_hash, role) 
VALUES ('antonio@devops.com', 'hash_falso_123', 'admin')
ON CONFLICT DO NOTHING;

-- Cliente App Python (Para probar)
INSERT INTO oauth_clients (id, client_secret, redirect_uri) 
VALUES ('mi-app-python', 'secreto_super_seguro', 'http://localhost:3000/callback')
ON CONFLICT DO NOTHING;