package main

import (
	"chilkat"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

// --- Structs for JSON Payloads ---

type CreateRequest struct {
	Payload string `json:"payload"`
	HmacKey string `json:"hmacKey"` // Expecting base64url encoded key
}

type CreateResponse struct {
	Jws   string `json:"jws,omitempty"`
	Error string `json:"error,omitempty"`
}

type ValidateRequest struct {
	Jws     string `json:"jws"`
	HmacKey string `json:"hmacKey"` // Expecting base64url encoded key
}

type ValidateResponse struct {
	IsValid bool        `json:"isValid"`
	Payload string      `json:"payload,omitempty"`
	Header  interface{} `json:"header,omitempty"` // Use interface{} to hold parsed JSON
	Error   string      `json:"error,omitempty"`
}

// --- Global Chilkat Unlock (Consider thread safety for production) ---
// It might be safer to unlock in each handler if concurrency is high
var chilkatUnlocked bool = false

func ensureChilkatUnlocked() bool {
	if chilkatUnlocked {
		return true
	}
	glob := chilkat.NewGlobal()
	success := glob.UnlockBundle("Anything for 30-day trial")
	if success != true {
		log.Println("Chilkat unlock failed:", glob.LastErrorText())
		glob.DisposeGlobal()
		return false
	}
	glob.DisposeGlobal() // Dispose after unlock check
	chilkatUnlocked = true
	log.Println("Chilkat library unlocked successfully.")
	return true
}

// --- API Handlers ---

func createHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !ensureChilkatUnlocked() {
		writeJsonError(w, "Chilkat library not unlocked", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		writeJsonError(w, "Method Not Allowed: Please use POST", http.StatusMethodNotAllowed)
		return
	}

	var req CreateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJsonError(w, fmt.Sprintf("Invalid JSON request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Payload == "" || req.HmacKey == "" {
		writeJsonError(w, "Missing 'payload' or 'hmacKey' in request body", http.StatusBadRequest)
		return
	}

	// Create JWS Protected Header
	jwsProtHdr := chilkat.NewJsonObject()
	defer jwsProtHdr.DisposeJsonObject()
	jwsProtHdr.AppendString("typ", "JWT")
	jwsProtHdr.AppendString("alg", "HS256")

	jws := chilkat.NewJws()
	defer jws.DisposeJws()

	// Set HMAC key (assuming base64url)
	signatureIndex := 0
	success := jws.SetMacKey(signatureIndex, req.HmacKey, "base64url")
	if !success {
		// Note: SetMacKey often doesn't produce LastErrorText immediately if format is wrong
		log.Printf("Failed to set HMAC key (potential format issue?): %s", jws.LastErrorText())
		writeJsonError(w, "Failed to set HMAC key (is it valid base64url?)", http.StatusInternalServerError)
		return
	}

	// Set protected header
	success = jws.SetProtectedHeader(signatureIndex, jwsProtHdr)
	if !success {
		log.Printf("Failed to set protected header: %s", jws.LastErrorText())
		writeJsonError(w, "Failed to set protected header", http.StatusInternalServerError)
		return
	}

	// Set payload
	bIncludeBom := false
	success = jws.SetPayload(req.Payload, "utf-8", bIncludeBom)
	if !success {
		log.Printf("Failed to set payload: %s", jws.LastErrorText())
		writeJsonError(w, "Failed to set payload", http.StatusInternalServerError)
		return
	}

	// Create the JWS
	jwsCompactPtr := jws.CreateJws()
	if jws.LastMethodSuccess() != true {
		log.Printf("Failed to create JWS: %s", jws.LastErrorText())
		writeJsonError(w, "Failed to create JWS", http.StatusInternalServerError)
		return
	}
	jwsCompact := *jwsCompactPtr

	log.Printf("Successfully created JWS for payload: %.30s...", req.Payload)
	resp := CreateResponse{Jws: jwsCompact}
	json.NewEncoder(w).Encode(resp)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !ensureChilkatUnlocked() {
		writeJsonError(w, "Chilkat library not unlocked", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		writeJsonError(w, "Method Not Allowed: Please use POST", http.StatusMethodNotAllowed)
		return
	}

	var req ValidateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJsonError(w, fmt.Sprintf("Invalid JSON request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Jws == "" || req.HmacKey == "" {
		writeJsonError(w, "Missing 'jws' or 'hmacKey' in request body", http.StatusBadRequest)
		return
	}

	jws2 := chilkat.NewJws()
	defer jws2.DisposeJws()

	// Load the JWS
	success := jws2.LoadJws(req.Jws)
	if !success {
		log.Printf("Failed to load JWS string: %s", jws2.LastErrorText())
		writeJsonError(w, "Failed to load JWS string (is it valid compact JWS?)", http.StatusBadRequest)
		return
	}

	// Set the MAC key for validation
	signatureIndex := 0
	success = jws2.SetMacKey(signatureIndex, req.HmacKey, "base64url")
	if !success {
		log.Printf("Failed to set HMAC key for validation: %s", jws2.LastErrorText())
		writeJsonError(w, "Failed to set HMAC key for validation (is it valid base64url?)", http.StatusInternalServerError)
		return
	}

	// Validate the signature
	v := jws2.Validate(signatureIndex)
	if v < 0 {
		log.Printf("JWS validation failed (possible unlock issue): %s", jws2.LastErrorText())
		writeJsonError(w, "JWS validation method failed", http.StatusInternalServerError)
		return
	}

	resp := ValidateResponse{}
	if v == 0 {
		log.Printf("JWS validation failed for JWS: %.30s...", req.Jws)
		resp.IsValid = false
		resp.Error = "Invalid signature. Key incorrect or JWS modified."
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Signature is valid
	resp.IsValid = true
	log.Printf("JWS validated successfully: %.30s...", req.Jws)

	// Recover payload
	payloadPtr := jws2.GetPayload("utf-8")
	if payloadPtr != nil {
		resp.Payload = *payloadPtr
	}

	// Recover header
	joseHeader := jws2.GetProtectedHeader(signatureIndex)
	if jws2.LastMethodSuccess() == true && joseHeader != nil {
		defer joseHeader.DisposeJsonObject()
		headerJsonString := joseHeader.Emit() // Emit as compact JSON string
		// Parse the JSON string into an interface{} for flexible output
		var headerData interface{}
		if err := json.Unmarshal([]byte(*headerJsonString), &headerData); err == nil {
			resp.Header = headerData
		}
	}

	json.NewEncoder(w).Encode(resp)
}

// --- Helper Function ---

func writeJsonError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := map[string]string{"error": message} // Simple error map
	json.NewEncoder(w).Encode(response)
}

// --- Main Function ---

func main() {
	// Attempt initial unlock (optional, as handlers also check)
	// ensureChilkatUnlocked()

	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/validate", validateHandler)

	port := "8080" // Use a different port
	log.Printf("Starting JWS HMAC server on port %s...\n", port)
	log.Printf("Endpoints: POST http://localhost:%s/create, POST http://localhost:%s/validate", port, port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
