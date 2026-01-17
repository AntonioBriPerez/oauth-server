package main

import (
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
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
	signKey *rsa.PrivateKey
)

func main() {
	// 1. CARGAR CLAVE PRIVADA (RSA)
	keyPath := os.Getenv("PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = "/etc/oauth/keys/private.pem"
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

	// 3. RUTAS (Registradas UNA sola vez y envueltas con CORS)
	http.HandleFunc("/token", enableCORS(tokenHandler))
	http.HandleFunc("/public-key", enableCORS(publicKeyHandler))
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 4. ARRANCAR SERVIDOR
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 OAuth Server escuchando en puerto %s", port)
	http.ListenAndServe(":"+port, nil)
}

// --- MIDDLEWARE CORS ---
// Permite que el navegador (localhost:4000) hable con este servidor (localhost:8080)
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		next(w, r)
	}
}

// --- HANDLERS ---

func publicKeyHandler(w http.ResponseWriter, r *http.Request) {
	// Extraemos la parte pública de la clave
	pubKey := signKey.Public()
	pubASN1, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		http.Error(w, "Error interno procesando clave pública", http.StatusInternalServerError)
		return
	}
	// Convertimos a formato PEM
	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Write(pubBytes)
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

	if req.GrantType != "client_credentials" {
		http.Error(w, "grant_type no soportado", http.StatusBadRequest)
		return
	}

	// Verificamos credenciales contra PostgreSQL
	var client Client
	query := `SELECT id, client_secret FROM oauth_clients WHERE id = $1`
	err := db.QueryRow(query, req.ClientID).Scan(&client.ID, &client.ClientSecret)

	if err == sql.ErrNoRows || client.ClientSecret != req.ClientSecret {
		log.Printf("⚠️ Login fallido para: %s", req.ClientID)
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	log.Printf("🔑 Generando token para: %s", client.ID)

	// Creamos el JWT
	claims := jwt.MapClaims{
		"sub": client.ID,
		"iss": "antonio-oauth-server",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(signKey)
	if err != nil {
		log.Printf("Error firmando token: %v", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}