package main

import (
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

// --- ESTRUCTURAS ---
type TokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

type Client struct {
	ID           string
	ClientSecret string
}

// --- VARIABLES GLOBALES ---
var (
	db      *sql.DB
	signKey *rsa.PrivateKey // Aquí guardaremos la clave cargada en memoria
)

func main() {
	// 1. CARGAR CLAVE PRIVADA (RSA)
	// K8s montará el secreto en /etc/oauth/keys/private.pem
	keyPath := os.Getenv("PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = "/etc/oauth/keys/private.pem" // Ruta por defecto en K8s
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("FATAL: No se pudo leer la clave privada en %s: %v", keyPath, err)
	}

	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("FATAL: La clave privada no es válida: %v", err)
	}
	log.Println("✅ Clave RSA cargada correctamente.")

	// 2. CONEXIÓN A DB
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	// (Opcional: db.Ping() aquí para asegurar conexión)

	// 3. RUTAS
	http.HandleFunc("/token", tokenHandler)

	// 4. ARRANCAR
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 OAuth Server escuchando en puerto %s", port)
	http.ListenAndServe(":"+port, nil)
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// VALIDACIÓN 1: Grant Type (Solo soportamos client_credentials por ahora)
	if req.GrantType != "client_credentials" {
		http.Error(w, "grant_type no soportado", http.StatusBadRequest)
		return
	}

	// VALIDACIÓN 2: Consultar BD
	var client Client
	query := `SELECT id, client_secret FROM oauth_clients WHERE id = $1`
	err := db.QueryRow(query, req.ClientID).Scan(&client.ID, &client.ClientSecret)
	
	if err == sql.ErrNoRows || client.ClientSecret != req.ClientSecret {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	// GENERACIÓN DE JWT (La parte nueva)
	log.Printf("🔑 Generando token para ClientID: %s", client.ID)

	claims := jwt.MapClaims{
		"sub": client.ID,                        // Subject: La App
		"iss": "antonio-oauth-server",           // Issuer: Nosotros
		"iat": time.Now().Unix(),                // Issued At
		"exp": time.Now().Add(time.Hour).Unix(), // Expires: 1 hora
		// En el futuro aquí meteremos "scopes" (permisos)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(signKey)
	if err != nil {
		log.Printf("Error firmando token: %v", err)
		http.Error(w, "Error interno generando token", http.StatusInternalServerError)
		return
	}

	// RESPUESTA OAUTH 2.0 STANDARD
	resp := map[string]interface{}{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}