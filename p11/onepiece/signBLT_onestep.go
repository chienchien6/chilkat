package main

import (
	"chilkat"
	"encoding/base64" // <-- Import standard base64 encoding library
	"errors"
	"fmt"
	"os" // Added for directory creation
	"path/filepath"
	"strings" // Added for error message parsing

	"github.com/spf13/viper"
)

/*
#cgo CFLAGS: -I C:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

// --- Configuration Loading (Copied from p11/ocsp/signBLT.go) ---
func loadConfig() (*viper.Viper, error) {
	vip := viper.New()
	vip.SetConfigName("config")
	vip.SetConfigType("json")
	vip.AddConfigPath("C:/chilkatPackage/chilkattest") // Add other paths if needed
	err := vip.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}
	fmt.Println("Configuration file loaded successfully.")
	return vip, nil
}

// --- Chilkat Initialization (Copied from p11/ocsp/signBLT.go) ---
func initializeChilkat() error {
	glob := chilkat.NewGlobal()
	glob.SetVerboseLogging(true)

	success := glob.UnlockBundle("Anything for 30-day trial")
	if !success {
		errMsg := glob.LastErrorText()
		return fmt.Errorf("failed to unlock Chilkat: %s", errMsg)
	}

	status := glob.UnlockStatus()
	if status == 2 {
		fmt.Println("Chilkat unlocked using purchased unlock code.")
	} else {
		fmt.Println("Chilkat unlocked in trial mode.")
	}
	return nil
}

// --- PKCS11 Initialization (Copied from p11/ocsp/signBLT.go) ---
func initializePkcs11(libPath string) (*chilkat.Pkcs11, error) {
	if libPath == "" {
		return nil, errors.New("PKCS11 library path is empty")
	}
	pkcs11 := chilkat.NewPkcs11()
	pkcs11.SetVerboseLogging(true)
	fmt.Printf("Using PKCS11 library path: %s\n", libPath)
	pkcs11.SetSharedLibPath(libPath)

	success := pkcs11.Initialize()
	if !success {
		errMsg := pkcs11.LastErrorText()
		pkcs11.DisposePkcs11()
		return nil, fmt.Errorf("PKCS11 Initialize failed: %s", errMsg)
	}
	fmt.Println("PKCS11 Initialize successful.")
	return pkcs11, nil
}

// --- PKCS11 Session Establishment (Copied from p11/ocsp/signBLT.go) ---
func establishPkcs11Session(pkcs11 *chilkat.Pkcs11, pin string, userType int) (slotID int, err error) {
	if pin == "" {
		return -1, errors.New("HSM PIN is empty")
	}
	slotID = 0 // Hardcoded Slot ID
	fmt.Printf("Attempting to use hardcoded Slot ID: %d\n", slotID)

	readWrite := true
	success := pkcs11.OpenSession(slotID, readWrite)
	if !success {
		return -1, fmt.Errorf("PKCS11 OpenSession failed for Slot ID %d: %s", slotID, pkcs11.LastErrorText())
	}
	fmt.Printf("PKCS11 OpenSession successful for Slot ID: %d\n", slotID)

	success = pkcs11.Login(userType, pin)
	if !success {
		pkcs11.CloseSession()
		return -1, fmt.Errorf("PKCS11 Login failed for Slot ID %d: %s", slotID, pkcs11.LastErrorText())
	}
	fmt.Printf("PKCS11 Login successful for Slot ID: %d, UserType: %d\n", slotID, userType)
	return slotID, nil
}

// --- Certificate Finding (Copied from p11/ocsp/signBLT.go) ---
func findCertificateWithPrivateKey(pkcs11 *chilkat.Pkcs11) (*chilkat.Cert, error) {
	cert := chilkat.NewCert()
	success := pkcs11.FindCert("privateKey", "", cert)
	if !success {
		cert.DisposeCert()
		errMsg := pkcs11.LastErrorText()
		if errMsg == "No certificates with private keys found." || errMsg == "Did not find cert matching criteria." {
			fmt.Println("No certificates having a private key were found.")
			return nil, nil
		}
		return nil, fmt.Errorf("error finding certificate with private key: %s", errMsg)
	}
	fmt.Println("Found cert with potential private key association: ", cert.SubjectCN())

	if !cert.HasPrivateKey() {
		subjectCN := cert.SubjectCN()
		cert.DisposeCert()
		errMsg := fmt.Sprintf("Error: The certificate found (CN: %s) via pkcs11.FindCert does NOT have an associated private key according to Chilkat.", subjectCN)
		fmt.Println(errMsg)
		return nil, errors.New(errMsg)
	}
	fmt.Println("Certificate object confirmed to have an associated private key.")
	return cert, nil
}

// --- Configure Signing Options for ONE-STEP B-LT Attempt ---
func configureSigningOptionsOneStep(pdf *chilkat.Pdf, cert *chilkat.Cert) (*chilkat.JsonObject, error) {
	json := chilkat.NewJsonObject()
	pdf.SetSigningCert(cert) // Set cert early

	// Base configuration
	json.UpdateString("subFilter", "/ETSI.CAdES.detached")
	json.UpdateBool("signingCertificateV2", true)
	json.UpdateString("hashAlgorithm", "sha256")

	// OCSP/CRL specific settings
	json.UpdateBool("sendOcspNonce", true)
	json.UpdateString("ocspDigestAlg", "sha256")
	json.UpdateInt("ocspTimeoutMs", 30000)
	json.UpdateInt("crlTimeoutMs", 30000)

	// --- LTV / DSS Core Settings for One-Step Attempt ---
	json.UpdateBool("ltvOcsp", true)            // Enable LTV via OCSP
	json.UpdateBool("ltvCrl", true)             // Enable LTV via CRL (Try enabling both)
	json.UpdateBool("embedOcspResponses", true) // Embed OCSP responses found
	json.UpdateBool("embedCrlResponses", true)  // Embed CRL responses found
	json.UpdateBool("includeCertChain", true)   // Ensure cert chain is included for validation
	json.UpdateBool("validateChain", false)     // Keep false for initial debugging
	json.UpdateBool("updateDss", true)          // Request DSS update/creation

	// TSA settings (Required for B-T and above)
	json.UpdateInt("signingTime", 1)
	json.UpdateBool("timestampToken.enabled", true)
	json.UpdateString("timestampToken.tsaUrl", "http://timestamp.digicert.com") // Replace with your TSA
	json.UpdateBool("timestampToken.requestTsaCert", true)
	json.UpdateInt("timestampToken.timeoutMs", 30000)

	// --- Manually add Intermediate CA Certificate to JSON options ---
	// (Copied logic from previous attempt)
	intermediateCert := chilkat.NewCert()
	defer intermediateCert.DisposeCert() // Dispose intermediate cert object
	// IMPORTANT: Replace with the ACTUAL path to your intermediate CA cert file
	intermediateCertPath := "C:/chilkatPackage/chilkattest/certs/intermediateCA.cer" // Example path, ADJUST AS NEEDED
	fmt.Printf("Attempting to load intermediate CA certificate from: %s\n", intermediateCertPath)
	if intermediateCert.LoadFromFile(intermediateCertPath) {
		// Assuming GetEncoded returns *string with raw DER data
		derStringPtr := intermediateCert.GetEncoded()
		// Check if pointer is nil or the string it points to is empty
		if derStringPtr != nil && *derStringPtr != "" {
			// Convert the string to []byte, then Base64 encode
			derBytes := []byte(*derStringPtr)
			base64Der := base64.StdEncoding.EncodeToString(derBytes)

			// Ensure the array exists using UpdateNewArray
			// If it already exists as an array, this should be harmless.
			json.UpdateNewArray("certsToEmbedBase64")

			// Get the JsonArray object first
			certsArray := json.ArrayOf("certsToEmbedBase64")
			if certsArray != nil {
				// Correct according to documentation: Use AddStringAt with index -1 to append
				successAdd := certsArray.AddStringAt(-1, base64Der)
				certsArray.DisposeJsonArray() // Dispose the array object reference

				if successAdd {
					fmt.Printf("Successfully added intermediate cert (Subject: %s) to certsToEmbedBase64 option.\n", intermediateCert.SubjectCN())
				} else {
					fmt.Println("Warning: Failed to add intermediate cert Base64 using JsonArray.AddStringAt.")
					// Check json.LastErrorText() or certsArray.LastErrorText()
					fmt.Println("LastErrorText after AddStringAt failure:", json.LastErrorText())
				}
			} else {
				fmt.Println("Warning: Failed to get JsonArray object for certsToEmbedBase64.")
			}

		} else {
			fmt.Println("Warning: Failed to get DER data (*string) from intermediate certificate:", intermediateCert.LastErrorText())
		}
	} else {
		fmt.Printf("Warning: Failed to load intermediate CA certificate from '%s': %s\n", intermediateCertPath, intermediateCert.LastErrorText())
		// Consider returning an error if intermediate is mandatory
		// return nil, fmt.Errorf("failed to load required intermediate CA certificate from %s", intermediateCertPath)
	}
	// ---------------------------------------------------------

	// Appearance settings
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "PAdES Signature (One-Step B-LT Attempt)")

	fmt.Println("Configured comprehensive settings for one-step B-LT attempt.")
	return json, nil
}

// --- PDF Loading with enhanced settings (Copied from p11/ocsp/signBLT.go) ---
func loadPdfDocument(filePath string) (*chilkat.Pdf, error) {
	if filePath == "" {
		return nil, errors.New("unsigned PDF input path is empty")
	}
	pdf := chilkat.NewPdf()
	pdf.SetVerboseLogging(true)
	pdf.SetSigAllocateSize(100000)
	// Add all potentially useful debugging options
	pdf.SetUncommonOptions("LOG_OCSP_HTTP,LOG_CRL_HTTP,OCSP_RESP_DETAILS,CRL_DETAILS,DSS_DEBUG,FORCE_DSS,CACHE_OCSP_RESPONSES,CACHE_CRL_RESPONSES")

	success := pdf.LoadFile(filePath)
	if !success {
		errMsg := pdf.LastErrorText()
		pdf.DisposePdf()
		return nil, fmt.Errorf("failed to load PDF '%s': %s", filePath, errMsg)
	}
	fmt.Printf("Loaded unsigned PDF: %s\n", filePath)
	return pdf, nil
}

// --- PDF Signing Function for ONE-STEP Attempt ---
func performSigningOneStep(pdf *chilkat.Pdf, cert *chilkat.Cert, jsonOptions *chilkat.JsonObject, outputPath string) error {
	if outputPath == "" {
		return errors.New("signed PDF output path is empty")
	}
	if pdf == nil || cert == nil || jsonOptions == nil {
		return errors.New("invalid parameters for PDF signing (nil PDF, Cert, or JSON)")
	}

	// Certificate should already be set in configureSigningOptions

	// --- Attempt One-Step Sign ---
	fmt.Println("--- Beginning PDF Signing (One-Step B-LT Attempt) --- (Verbose logs follow)")
	fmt.Printf("Attempting to sign PDF and save to: %s\n", outputPath)

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %s", err)
	}

	// Clear LastErrorText before SignPdf
	pdf.LastErrorText()
	success := pdf.SignPdf(jsonOptions, outputPath)
	errMsg := pdf.LastErrorText() // Capture error text immediately

	if errMsg != "" {
		fmt.Println("--- Verbose LastErrorText after SignPdf attempt (One-Step): ---")
		fmt.Println(errMsg)
		fmt.Println("--- End of Verbose LastErrorText (One-Step) ---")
	} else {
		fmt.Println("SignPdf attempt completed, LastErrorText is empty.")
	}

	if !success {
		return fmt.Errorf("failed to sign PDF (One-Step SignPdf): %s", errMsg)
	}

	fmt.Println("PDF SignPdf call completed successfully (One-Step). Check actual signature level and DSS content.")

	// --- Optional: Get and print DSS content after signing for verification ---
	dssJson := chilkat.NewJsonObject()
	defer dssJson.DisposeJsonObject()
	fmt.Println("Attempting to get DSS content after one-step signing...")
	gotDss := pdf.GetDss(dssJson)
	if gotDss {
		dssContent := dssJson.Emit()
		fmt.Println("Successfully retrieved DSS content:")
		fmt.Println(*dssContent) // Dereference to print the string value
		// Analyze dssContent here or externally to see if it's complete
		if *dssContent == "{}" || !strings.Contains(*dssContent, "VRI") { // Basic check
			fmt.Println("WARNING: DSS content appears empty or incomplete based on basic check.")
		}
	} else {
		fmt.Println("Failed to get DSS content after signing:")
		fmt.Println(pdf.LastErrorText())
	}
	// ---------------------------------------------------------------------

	return nil // Return nil if SignPdf succeeded, analysis of level is separate
}

// --- PKCS11 Logout (Copied from p11/ocsp/signBLT.go) ---
func pkcs11Logout(pkcs11 *chilkat.Pkcs11) error {
	if pkcs11 == nil {
		return errors.New("cannot logout, PKCS11 object is nil")
	}
	success := pkcs11.Logout()
	if !success {
		fmt.Println("Warning: PKCS11 Logout failed (non-critical):", pkcs11.LastErrorText())
		return fmt.Errorf("PKCS11 logout failed: %s", pkcs11.LastErrorText())
	}
	fmt.Println("PKCS11 Logout successful.")
	return nil
}

// --- Main Application Logic (Adapted for One-Step) ---
func main() {
	defer chilkat.NewGlobal().DisposeGlobal()

	err := initializeChilkat()
	if err != nil {
		fmt.Println("Error during Chilkat initialization:", err)
		return
	}

	vip, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return
	}

	pkcs11LibPath := vip.GetString("pkcs11_lib_path")
	pin := vip.GetString("hsm_pin")
	unsignedPdfPath := vip.GetString("unsigned_pdf_input_path")
	// Define output for the one-piece attempt
	onepieceOutputDir := "C:/chilkatPackage/chilkattest/p11/onepiece/output" // <<< New Output Directory
	baseOutputFilename := "signed_onestep_ecc"
	userType := 1

	pkcs11, err := initializePkcs11(pkcs11LibPath)
	if err != nil {
		fmt.Println("Error initializing PKCS11:", err)
		return
	}
	defer pkcs11.DisposePkcs11()

	_, err = establishPkcs11Session(pkcs11, pin, userType)
	if err != nil {
		fmt.Println("Error establishing PKCS11 session:", err)
		return
	}
	defer pkcs11.CloseSession()

	// --- Signing Loop ---
	numberOfSignatures := 1 // Let's try one first for focused debugging
	fmt.Printf("\n--- Starting One-Step Signing Loop (%d iterations) ---\n", numberOfSignatures)
	for i := 1; i <= numberOfSignatures; i++ {
		fmt.Printf("\n--- Iteration %d of %d ---\n", i, numberOfSignatures)

		pdf, err := loadPdfDocument(unsignedPdfPath)
		if err != nil {
			fmt.Println("Error loading PDF:", err)
			return
		}
		// Defer disposal within the loop iteration
		defer pdf.DisposePdf()

		cert, err := findCertificateWithPrivateKey(pkcs11)
		if err != nil {
			fmt.Println("Error finding certificate:", err)
			return
		}
		if cert == nil {
			fmt.Println("Could not find a usable certificate with an associated private key on the HSM.")
			return
		}
		// Defer disposal within the loop iteration
		defer cert.DisposeCert()

		// No need to call pdf.SetSigningCert here, configureSigningOptionsOneStep handles it

		jsonOptions, err := configureSigningOptionsOneStep(pdf, cert)
		if err != nil {
			fmt.Println("Error configuring signing options:", err)
			return
		}
		// Defer disposal within the loop iteration
		defer jsonOptions.DisposeJsonObject()

		outputFilename := fmt.Sprintf("%s_%d.pdf", baseOutputFilename, i)
		iterationOutputPath := filepath.Join(onepieceOutputDir, outputFilename)

		err = performSigningOneStep(pdf, cert, jsonOptions, iterationOutputPath)
		if err != nil {
			fmt.Printf("Error during one-step signing iteration %d: %v\n", i, err)
			// break // Stop loop on first error
		} else {
			fmt.Printf("One-step signing process completed for iteration %d. Please verify the output file: %s\n", i, iterationOutputPath)
		}

		// Optional: Pause if running multiple iterations
		// fmt.Println("Sleeping for 2 seconds...")
		// time.Sleep(2 * time.Second)
	}
	fmt.Printf("\n--- Finished One-Step Signing Loop ---\n\n")

	err = pkcs11Logout(pkcs11)
	if err != nil {
		fmt.Println("Note: Logout failed, proceeding with cleanup.")
	}

	fmt.Println("Program finished.")
}
