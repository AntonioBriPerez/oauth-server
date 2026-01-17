#!/bin/bash

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}>>> 🚀 INICIANDO DESPLIEGUE DE OAUTH SERVER (ANTONIO-AUTH) <<<${NC}"

# 1. GESTIÓN DE CLAVES (RSA)
echo -e "${YELLOW}>>> 1. Generando nuevas claves RSA (Entorno Seguro)...${NC}"
mkdir -p keys
# Usamos Docker para no ensuciar el host
docker run --rm -v "$(pwd)/keys:/keys" -w /keys alpine/openssl \
  genrsa -out private.pem 2048

echo -e "${YELLOW}>>> 2. Configurando Secretos en Kubernetes...${NC}"
# Borramos para asegurar limpieza
kubectl delete secret oauth-keys --ignore-not-found
# Creamos el secreto genérico
kubectl create secret generic oauth-keys --from-file=private.pem=./keys/private.pem

# 2. INFRAESTRUCTURA DE DATOS (POSTGRES)
echo -e "${YELLOW}>>> 3. Desplegando Base de Datos PostgreSQL...${NC}"
kubectl apply -f k8s/postgres/postgres.yaml

echo -e "${YELLOW}>>> 4. Esperando a que la DB esté lista (esto puede tardar unos segundos)...${NC}"
# Esperamos a que el pod esté en estado Ready
kubectl rollout status statefulset/auth-db --timeout=60s

echo -e "${YELLOW}>>> 5. Inicializando Esquema SQL y Datos Semilla...${NC}"
# Espera de seguridad extra para que el proceso de Postgres acepte conexiones
sleep 5
# Copiamos el SQL al pod y lo ejecutamos
kubectl cp k8s/postgres/init.sql auth-db-0:/tmp/init.sql
kubectl exec -it auth-db-0 -- psql -U admin -d oauth_db -f /tmp/init.sql

# 3. APLICACIÓN GO (OAUTH SERVER)
echo -e "${YELLOW}>>> 6. Construyendo imagen Docker del OAuth Server...${NC}"
docker build -t oauth-server:latest .

echo -e "${YELLOW}>>> 7. Importando imagen a K3s (Containerd)...${NC}"
docker save oauth-server:latest | sudo k3s ctr images import -

echo -e "${YELLOW}>>> 8. Desplegando Servidor OAuth...${NC}"
# Aseguramos que el deployment use la imagen :latest
kubectl apply -f k8s/app.yaml

echo -e "${YELLOW}>>> 9. Reiniciando Pods para aplicar cambios...${NC}"
kubectl rollout restart deployment oauth-deployment

# 4. VERIFICACIÓN FINAL
echo -e "${GREEN}>>> ✅ DESPLIEGUE FINALIZADO <<<${NC}"
echo "Esperando 5 segundos para estado final..."
sleep 5
kubectl get pods

echo -e "${GREEN}---------------------------------------------------${NC}"
echo -e "Para probar, abre el túnel en otra terminal:"
echo -e "  kubectl port-forward svc/oauth-service 8080:80"
echo -e ""
echo -e "Luego prueba el login:"
echo -e "  curl -X POST http://localhost:8080/token -d '...'"
echo -e "${GREEN}---------------------------------------------------${NC}"