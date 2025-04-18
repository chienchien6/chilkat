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

// Response structure for JSON output
type SignResponse struct {
	Signature string `json:"signature,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Handles the /sign request
func signHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	glob := chilkat.NewGlobal()
	// It's important to dispose of Chilkat objects
	defer glob.DisposeGlobal()

	success := glob.UnlockBundle("Anything for 30-day trial")
	if success != true {
		log.Println("Chilkat unlock failed:", glob.LastErrorText())
		writeError(w, "Failed to unlock Chilkat library", http.StatusInternalServerError)
		return
	}

	zipFile := chilkat.NewBinData()
	defer zipFile.DisposeBinData()
	localKeyFile := "private-key.zip"

	// Try to load the key from the local file first.
	success = zipFile.LoadFile(localKeyFile)
	if success != true {
		log.Printf("Local key file '%s' not found or failed to load. Falling back to URL download.\n", localKeyFile)

		// Fallback to downloading from URL
		httpChilkat := chilkat.NewHttp()
		defer httpChilkat.DisposeHttp()
		keyUrl := "https://www.chilkatsoft.com/exampleData/secp256r1-key.zip"
		log.Printf("Downloading key from %s...\n", keyUrl)
		success = httpChilkat.QuickGetBd(keyUrl, zipFile)
		if success != true {
			errMsg := fmt.Sprintf("Failed to download key from URL: %s", httpChilkat.LastErrorText())
			log.Println(errMsg)
			writeError(w, errMsg, http.StatusInternalServerError)
			return
		}
		log.Println("Key downloaded successfully.")
	} else {
		log.Printf("Loaded key from local file '%s'.\n", localKeyFile)
	}

	// Proceed with the zip data
	zip := chilkat.NewZip()
	defer zip.DisposeZip()
	success = zip.OpenBd(zipFile)
	if success != true {
		errMsg := fmt.Sprintf("Failed to open zip data: %s", zip.LastErrorText())
		log.Println(errMsg)
		writeError(w, errMsg, http.StatusInternalServerError)
		return
	}

	zipEntry := zip.FirstMatchingEntry("*.pem")
	if zipEntry == nil {
		log.Println("No .pem file found inside the zip data.")
		writeError(w, "No .pem file found inside the zip data.", http.StatusInternalServerError)
		return
	}
	defer zipEntry.DisposeZipEntry()

	ecKey := chilkat.NewPrivateKey()
	defer ecKey.DisposePrivateKey()
	pemContentPtr := zipEntry.UnzipToString(0, "utf-8")
	if pemContentPtr == nil {
		errMsg := fmt.Sprintf("Failed to unzip PEM content: %s", zipEntry.LastErrorText())
		log.Println(errMsg)
		writeError(w, errMsg, http.StatusInternalServerError)
		return
	}
	pemContent := *pemContentPtr

	success = ecKey.LoadPem(pemContent)
	if success != true {
		errMsg := fmt.Sprintf("Failed to load PEM key: %s", ecKey.LastErrorText())
		log.Println(errMsg)
		writeError(w, errMsg, http.StatusInternalServerError)
		return
	}

	// XML Signature Generation
	gen := chilkat.NewXmlDSigGen()
	defer gen.DisposeXmlDSigGen()
	gen.SetPrivateKey(ecKey)

	sbContent := chilkat.NewStringBuilder()
	defer sbContent.DisposeStringBuilder()
	sbContent.Append("This is the content that is signed.")
	gen.AddEnvelopedRef("abc123", sbContent, "sha256", "C14N", "")

	sbXml := chilkat.NewStringBuilder()
	defer sbXml.DisposeStringBuilder()
	success = gen.CreateXmlDSigSb(sbXml)
	if success != true {
		errMsg := fmt.Sprintf("Failed to generate XML signature: %s", gen.LastErrorText())
		log.Println(errMsg)
		writeError(w, errMsg, http.StatusInternalServerError)
		return
	}

	// Prepare successful response
	response := SignResponse{
		Signature: *sbXml.GetAsString(),
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		// Log error if encoding fails, but likely headers are already sent
		log.Printf("Failed to encode JSON response: %v", err)
	}
	log.Println("Successfully generated and returned signature.")
}

// Helper function to write JSON error responses
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := SignResponse{Error: message}
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/sign", signHandler)

	port := "8080"
	log.Printf("Starting server on port %s...\n", port)
	log.Printf("Access the API at: http://localhost:%s/sign", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
