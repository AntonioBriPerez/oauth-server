# OAuth 2.0 Authorization Server (Go + Kubernetes)

![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)
![Kubernetes](https://img.shields.io/badge/Kubernetes-K3s-326CE5?style=flat-square&logo=kubernetes)
![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat-square&logo=postgresql)
![Security](https://img.shields.io/badge/Security-RSA%20%2F%20JWT-red?style=flat-square)

Una implementación robusta de un Servidor de Autorización OAuth 2.0 escrita en **Go**, diseñada para ejecutarse nativamente en **Kubernetes**.

Este proyecto implementa el flujo **Client Credentials Grant** (RFC 6749) para autenticación Máquina-a-Máquina (M2M), utilizando **PostgreSQL** para persistencia y **firmas digitales RSA** para la emisión de tokens JWT.

---

## 🏗 Arquitectura

El sistema sigue una arquitectura de microservicios "Cloud Native":

1.  **Core (Go):** Servicio backend optimizado, compilado mediante Docker Multi-stage (imagen final basada en Alpine).
2.  **Persistencia (PostgreSQL):** Desplegada como `StatefulSet` en Kubernetes.
3.  **Gestión de Secretos:**
    * Las claves privadas RSA se generan efímeramente durante el despliegue.
    * Se inyectan en el pod vía **Kubernetes Secrets** y se montan como volúmenes de solo lectura.
4.  **Zero Trust:** El servidor no confía en nadie. Verifica credenciales (`client_id`, `client_secret`) contra la base de datos antes de firmar un token.

---

## 📂 Estructura del Proyecto

El repositorio sigue el **Standard Go Project Layout**:

```text
.
├── cmd/api/            # Punto de entrada (Main)
├── k8s/                # Manifiestos de Kubernetes
│   ├── app.yaml        # Deployment y Service del OAuth Server
│   └── postgres/       # StatefulSet de DB e init.sql
├── start.sh            # Script de orquestación "Zero-Install"
├── Dockerfile          # Build Multi-stage
└── README.md           # Documentación
```
## 🚀 Puesta en Marcha (Quick Start)

### Requisitos
* **Docker** y **Kubectl** instalados.
* Un clúster activo (ej: K3s).
* **NO** es necesario tener Go ni PostgreSQL instalados en tu máquina local.

### Despliegue Automático

El script `start.sh` automatiza todo el ciclo de vida: genera claves criptográficas, levanta la base de datos, inyecta el esquema SQL, compila el binario de Go y despliega en el clúster.

```bash
# 1. Clonar el repositorio
git clone <URL_DEL_REPO>
cd oauth-server

# 2. Ejecutar el orquestador
chmod +x start.sh
./start.sh
```

## 🧪 Pruebas (Usage)

Una vez desplegado, puedes probar la obtención de un token de acceso.

### 1. Abrir túnel al servicio
Como el servicio es interno, exponlo a tu máquina local:

```bash
kubectl port-forward svc/oauth-service 8080:80
```
### 2. Solicitar Token (Client Credentials Flow)
```bash
curl -X POST http://localhost:8080/token \
  -H "Content-Type: application/json" \
  -d '{
        "client_id": "mi-app-python",
        "client_secret": "secreto_super_seguro",
        "grant_type": "client_credentials"
      }'
```
### 3. Respuesta Esperada
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```
## ⚙️ Configuración y Variables

El despliegue se configura mediante variables de entorno definidas en `k8s/app.yaml`:

| Variable | Descripción | Valor por defecto |
| :--- | :--- | :--- |
| `DB_HOST` | Host del servicio de base de datos | `auth-db` |
| `DB_USER` | Usuario de PostgreSQL | `admin` |
| `PRIVATE_KEY_PATH` | Ubicación de la clave privada RSA | `/etc/oauth/keys/private.pem` |
| `PORT` | Puerto de escucha del servidor | `8080` |
## 🔒 Seguridad Implementada

* **Firmas Asimétricas:** Uso de pares de claves RSA 2048-bit.
* **Secretos en K8s:** Las claves nunca se guardan en la imagen Docker.
* **Volúmenes ReadOnly:** El contenedor no puede modificar la clave privada.
* **Contenedores Efímeros:** Build y herramientas se ejecutan en contenedores temporales, manteniendo el host limpio.

---

**Autor:** Antonio  
**Stack:** Go 1.23, K3s, Docker