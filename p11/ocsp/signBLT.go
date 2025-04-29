package main

import (
	"chilkat"
	"errors"
	"fmt"
	"os"            // Added for directory creation
	"path/filepath" // Added for error message parsing
	"time"

	"github.com/spf13/viper"
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-10.1.3-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

// --- Configuration Loading ---
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

// --- Chilkat Initialization ---
func initializeChilkat() error {
	glob := chilkat.NewGlobal()
	glob.SetVerboseLogging(true) // <<< Enable Global Verbose Logging
	// NOTE: glob is often managed globally, but DisposeGlobal should be called at program end.
	// We will call DisposeGlobal in main's defer. Let's ensure UnlockBundle runs.
	// defer glob.DisposeGlobal() // Called in main

	success := glob.UnlockBundle("Anything for 30-day trial") // Use your actual unlock code
	if !success {
		errMsg := glob.LastErrorText() // Capture error before potential DisposeGlobal
		// glob.DisposeGlobal() // Dispose if unlock fails? Depends on API design. Let main handle it.
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

// --- PKCS11 Initialization ---
func initializePkcs11(libPath string) (*chilkat.Pkcs11, error) {
	if libPath == "" {
		return nil, errors.New("PKCS11 library path is empty")
	}
	pkcs11 := chilkat.NewPkcs11()
	pkcs11.SetVerboseLogging(true) // <<< Enable PKCS11 Verbose Logging
	fmt.Printf("Using PKCS11 library path: %s\n", libPath)
	pkcs11.SetSharedLibPath(libPath)

	// Initialize the library - Step 1 (replaces part of QuickSession)
	// Note: Cleanup (Dispose) should be called when done with pkcs11 object (in main defer)
	success := pkcs11.Initialize()
	if !success {
		errMsg := pkcs11.LastErrorText()
		pkcs11.DisposePkcs11() // Dispose if init fails
		return nil, fmt.Errorf("PKCS11 Initialize failed: %s", errMsg)
	}
	fmt.Println("PKCS11 Initialize successful.")
	return pkcs11, nil
}

// --- PKCS11 Session Establishment ---
func establishPkcs11Session(pkcs11 *chilkat.Pkcs11, pin string, userType int) (slotID int, err error) {
	if pin == "" {
		return -1, errors.New("HSM PIN is empty")
	}

	// --- Directly use Slot ID 0 ---
	slotID = 0
	fmt.Printf("Attempting to use hardcoded Slot ID: %d\n", slotID)

	// Remove dynamic slot finding logic
	/*
		// Find an available slot with a token present - Step 2 (replaces part of QuickSession)
		tokenPresent := true
		slotIDs := chilkat.NewStringTable()
		defer slotIDs.DisposeStringTable()
		success := pkcs11.FindSlots(tokenPresent, slotIDs) // This method might not exist
		if !success {
			return -1, fmt.Errorf("PKCS11 FindSlots failed: %s", pkcs11.LastErrorText())
		}
		if slotIDs.Count() < 1 {
			return -1, errors.New("no slots with tokens found")
		}

		// Use the first available slot
		slotIDStr := slotIDs.String(0) // This method might not exist
		slotID, err = strconv.Atoi(slotIDStr) // Use standard Go conversion
		if err != nil {
			return -1, fmt.Errorf("failed to convert slot ID '%s' to int: %w", slotIDStr, err)
		}
		fmt.Printf("Found slot with token, using Slot ID: %d\n", slotID)
	*/

	// Open the session - Step 3 (replaces part of QuickSession)
	// Note: CloseSession should be called when done with the session (in main defer)
	readWrite := true
	success := pkcs11.OpenSession(slotID, readWrite)
	if !success {
		return -1, fmt.Errorf("PKCS11 OpenSession failed for Slot ID %d: %s", slotID, pkcs11.LastErrorText())
	}
	fmt.Printf("PKCS11 OpenSession successful for Slot ID: %d\n", slotID)

	// Login to the session - Step 4 (replaces part of QuickSession)
	success = pkcs11.Login(userType, pin)
	if !success {
		// Attempt to close session even if login fails
		pkcs11.CloseSession()
		return -1, fmt.Errorf("PKCS11 Login failed for Slot ID %d: %s", slotID, pkcs11.LastErrorText())
	}
	fmt.Printf("PKCS11 Login successful for Slot ID: %d, UserType: %d\n", slotID, userType)

	return slotID, nil // Return the used slot ID if needed, and nil error
}

// --- Certificate Finding ---
func findCertificateWithPrivateKey(pkcs11 *chilkat.Pkcs11) (*chilkat.Cert, error) {
	cert := chilkat.NewCert()
	// defer cert.DisposeCert() // Dispose in the caller (main)

	success := pkcs11.FindCert("privateKey", "", cert)
	if !success {
		cert.DisposeCert() // Dispose if not found
		// It's possible no cert with a private key exists, handle this gracefully
		errMsg := pkcs11.LastErrorText()
		if errMsg == "No certificates with private keys found." || errMsg == "Did not find cert matching criteria." {
			fmt.Println("No certificates having a private key were found.")
			return nil, nil // Return nil, nil to indicate not found but not necessarily an error
		}
		return nil, fmt.Errorf("error finding certificate with private key: %s", errMsg)
	}
	fmt.Println("Found cert with potential private key association: ", cert.SubjectCN())

	// Verify the found certificate object truly has a private key linkage
	if !cert.HasPrivateKey() {
		subjectCN := cert.SubjectCN() // Get CN before disposing
		cert.DisposeCert()
		errMsg := fmt.Sprintf("Error: The certificate found (CN: %s) via pkcs11.FindCert does NOT have an associated private key according to Chilkat.", subjectCN)
		fmt.Println(errMsg)
		return nil, errors.New(errMsg)
	}
	fmt.Println("Certificate object confirmed to have an associated private key.")
	return cert, nil
}

// --- (Optional) Find HSM Handles ---
// Keeping this separate if direct handle verification is needed later
func findHsmHandles(pkcs11 *chilkat.Pkcs11) (privKeyHandle uint, certHandle uint, err error) {
	privKeyHandle = 0
	certHandle = 0

	// Find Private Key Handle
	jsonTemplateKey := chilkat.NewJsonObject()
	defer jsonTemplateKey.DisposeJsonObject()
	jsonTemplateKey.UpdateString("class", "private_key")
	jsonTemplateKey.UpdateString("key_type", "ecc")
	jsonTemplateKey.UpdateString("label", "ECC Private Key")
	fmt.Println("Searching for private key with template:", jsonTemplateKey.Emit())
	privKeyHandle = pkcs11.FindObject(jsonTemplateKey) // FindObject returns uint
	if privKeyHandle == 0 {
		fmt.Println("Warning: Failed to find the ECC private key handle:", pkcs11.LastErrorText())
		// Decide if this is a fatal error or just a warning
		// err = errors.New("failed to find ECC private key handle")
		// return // Return early if it's fatal
	} else {
		fmt.Printf("Found ECC Private Key Handle: %d\n", privKeyHandle)
	}

	// Find Certificate Handle
	jsonTemplateCert := chilkat.NewJsonObject()
	defer jsonTemplateCert.DisposeJsonObject()
	jsonTemplateCert.UpdateString("class", "certificate")
	jsonTemplateCert.UpdateString("label", "X509 Certificate")
	fmt.Println("Searching for certificate with template:", jsonTemplateCert.Emit())
	certHandle = pkcs11.FindObject(jsonTemplateCert) // FindObject returns uint
	if certHandle == 0 {
		fmt.Println("Warning: Failed to find the X509 certificate handle:", pkcs11.LastErrorText())
		// Decide if this is a fatal error or just a warning
		// err = errors.New("failed to find X509 certificate handle") // Combine errors if needed
	} else {
		fmt.Printf("Found X509 Certificate Handle: %d\n", certHandle)
	}

	// Example: Return error if either handle is missing
	if privKeyHandle == 0 || certHandle == 0 {
		err = errors.New("required key or certificate handle not found on HSM")
	}

	return privKeyHandle, certHandle, err
}

// --- Configure Signing Options ---
func configureSigningOptions(pdf *chilkat.Pdf, cert *chilkat.Cert) (*chilkat.JsonObject, error) {
	json := chilkat.NewJsonObject()
	pdf.SetSigningCert(cert)
	// Base configuration
	json.UpdateString("subFilter", "/ETSI.CAdES.detached") // Use this for PAdES base
	json.UpdateBool("signingCertificateV2", true)
	json.UpdateString("hashAlgorithm", "sha256")

	// OCSP/CRL specific settings (Keep these as they control fetching)
	json.UpdateBool("sendOcspNonce", true)
	json.UpdateString("ocspDigestAlg", "sha256")
	json.UpdateInt("ocspTimeoutMs", 30000)
	json.UpdateInt("crlTimeoutMs", 30000)
	// Removed: forceRevocationChecks, validateSignatures, validateChain, deepValidation

	// --- Settings for Step 1 (SignPdf): Focus on Timestamp, maybe chain ---
	// json.UpdateBool("ltvOcsp", true)            // Disabled for Step 1
	// json.UpdateBool("ltvCrl", true)             // Disabled for Step 1
	// json.UpdateBool("embedOcspResponses", true) // Disabled for Step 1
	// json.UpdateBool("embedCrlResponses", true)  // Disabled for Step 1
	json.UpdateBool("includeCertChain", true) // Keep this, may help AddVerificationInfo later
	// 添加證書鏈驗證
	json.UpdateBool("validateChain", false) // Keep false during debugging

	// 強制更新DSS(文件安全儲存)
	// json.UpdateBool("updateDss", true)         // Disabled for Step 1, AddVerificationInfo handles DSS

	// TSA settings (Keep these for B-T level)
	json.UpdateInt("signingTime", 1)
	json.UpdateBool("timestampToken.enabled", true)
	json.UpdateString("timestampToken.tsaUrl", "http://timestamp.digicert.com")
	json.UpdateBool("timestampToken.requestTsaCert", true)
	json.UpdateInt("timestampToken.timeoutMs", 30000)

	// Appearance settings
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "PAdES Signature (Attempting B-LT)") // Adjust text

	// Removed: addDss, updateDss, forceDssCreation, pdfSubfilter, pAdESCompliant, pAdESLevel

	fmt.Println("Configured simplified core settings for B-LT level")
	return json, nil
}

// --- PDF Loading with enhanced settings ---
func loadPdfDocument(filePath string) (*chilkat.Pdf, error) {
	if filePath == "" {
		return nil, errors.New("unsigned PDF input path is empty")
	}

	pdf := chilkat.NewPdf()
	pdf.SetVerboseLogging(true)

	// Increase signature allocation size for larger OCSP/CRL responses
	pdf.SetSigAllocateSize(100000) // Increased from 30000

	// Enhanced logging for OCSP/CRL debugging
	// 添加詳細的OCSP/CRL/DSS處理調試
	pdf.SetUncommonOptions("LOG_OCSP_HTTP,LOG_CRL_HTTP,OCSP_RESP_DETAILS,CRL_DETAILS,DSS_DEBUG,FORCE_DSS")

	// Load the PDF file
	success := pdf.LoadFile(filePath)
	if !success {
		errMsg := pdf.LastErrorText()
		pdf.DisposePdf()
		return nil, fmt.Errorf("failed to load PDF '%s': %s", filePath, errMsg)
	}

	fmt.Printf("Loaded unsigned PDF: %s\n", filePath)
	return pdf, nil
}

// --- Enhanced PDF Signing function ---
func performPdfSigning(pdf *chilkat.Pdf, cert *chilkat.Cert, jsonOptions *chilkat.JsonObject, outputPath string) error {
	if outputPath == "" {
		return errors.New("signed PDF output path is empty")
	}
	if pdf == nil || cert == nil || jsonOptions == nil {
		return errors.New("invalid parameters for PDF signing (nil PDF, Cert, or JSON)")
	}

	// Set the certificate object to use for signing
	fmt.Println("Setting signing certificate on PDF object...")
	success := pdf.SetSigningCert(cert)

	// Set additional options to improve LTV support
	pdf.SetUncommonOptions("CACHE_OCSP_RESPONSES,CACHE_CRL_RESPONSES")

	if !success {
		errMsg := pdf.LastErrorText()
		fmt.Println("--- Failed to set signing certificate --- Verbose LastErrorText: ---")
		fmt.Println(errMsg)
		fmt.Println("--- End of Verbose LastErrorText ---")
		return fmt.Errorf("failed to set signing certificate on PDF object (see log)")
	}

	fmt.Println("PKCS11 signing certificate configured successfully on PDF object.")

	// --- Step 1: Sign the PDF (aiming for B-T initially) and SAVE ---
	fmt.Println("--- Beginning PDF Signing (Step 1: Base Signature + Timestamp) ---")
	fmt.Printf("Attempting to sign PDF and save B-T stage to: %s\n", outputPath)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %s", err)
	}

	// Clear LastErrorText before SignPdf
	pdf.LastErrorText()
	successSign := pdf.SignPdf(jsonOptions, outputPath)
	errMsgSign := pdf.LastErrorText() // Capture error text immediately

	if errMsgSign != "" {
		fmt.Println("--- Verbose LastErrorText after SignPdf attempt (Step 1): ---")
		fmt.Println(errMsgSign)
		fmt.Println("--- End of Verbose LastErrorText (Step 1) ---")
	}

	if !successSign {
		return fmt.Errorf("failed to sign PDF (Step 1: SignPdf): %s", errMsgSign)
	}
	fmt.Println("PDF signed successfully (Step 1: Base Signature + Timestamp) and saved.")

	// --- IMPORTANT: Dispose the first pdf object before loading the signed file ---
	// This ensures we work with the file state, not potentially inconsistent memory state.
	pdf.DisposePdf()

	// --- Step 2: Load the signed (B-T) PDF and Add LTV Info / Fill DSS ---
	fmt.Println("--- Beginning LTV Addition (Step 2: Load B-T and AddVerificationInfo) ---")
	pdfLtv := chilkat.NewPdf()     // Create a NEW Pdf object
	defer pdfLtv.DisposePdf()      // Ensure this new object is disposed
	pdfLtv.SetVerboseLogging(true) // Enable logging for the new object too

	fmt.Printf("Loading signed PDF from: %s\n", outputPath)
	if !pdfLtv.LoadFile(outputPath) {
		return fmt.Errorf("failed to load signed PDF (Step 2: LoadFile): %s", pdfLtv.LastErrorText())
	}

	emptyJson := chilkat.NewJsonObject()
	defer emptyJson.DisposeJsonObject() // Ensure empty JSON object is disposed
	// REMOVED: emptyJson.UpdateBool("ltvOcsp", true) // AddVerificationInfo expects an EMPTY json

	// Clear LastErrorText before AddVerificationInfo
	pdfLtv.LastErrorText()
	fmt.Println("Calling AddVerificationInfo...")
	successAddVI := pdfLtv.AddVerificationInfo(emptyJson, outputPath) // Use the loaded object
	errMsgAddVI := pdfLtv.LastErrorText()                             // Capture error text immediately

	if errMsgAddVI != "" {
		fmt.Println("--- Verbose LastErrorText after AddVerificationInfo attempt (Step 2): ---")
		fmt.Println(errMsgAddVI)
		fmt.Println("--- End of Verbose LastErrorText (Step 2) ---")
	}

	if !successAddVI {
		return fmt.Errorf("failed to add LTV verification info (Step 2: AddVerificationInfo): %s", errMsgAddVI)
	}

	fmt.Println("Successfully added LTV info and updated DSS (Step 2: AddVerificationInfo). PDF should be B-LT.")

	return nil
}

// --- PKCS11 Logout ---
func pkcs11Logout(pkcs11 *chilkat.Pkcs11) error {
	if pkcs11 == nil {
		return errors.New("cannot logout, PKCS11 object is nil")
	}
	success := pkcs11.Logout()
	if !success {
		// Log as a warning, as cleanup should still proceed
		fmt.Println("Warning: PKCS11 Logout failed (non-critical):", pkcs11.LastErrorText())
		return fmt.Errorf("PKCS11 logout failed: %s", pkcs11.LastErrorText()) // Return error for info
	}
	fmt.Println("PKCS11 Logout successful.")
	return nil
}

// --- Main Application Logic ---
func main() {

	// Defer global Chilkat cleanup
	defer chilkat.NewGlobal().DisposeGlobal() // Dispose the global object at the very end

	// 1. Initialize Chilkat Global
	err := initializeChilkat()
	if err != nil {
		fmt.Println("Error during Chilkat initialization:", err)
		return
	}

	// 2. Load Configuration
	vip, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return
	}

	// Get required config values
	pkcs11LibPath := vip.GetString("pkcs11_lib_path")
	pin := vip.GetString("hsm_pin")
	unsignedPdfPath := vip.GetString("unsigned_pdf_input_path")
	// Define the target directory for looped outputs
	loopOutputDir := vip.GetString("signed_hsm_pdf_output_path") // <<< Target Directory
	baseOutputFilename := "signed_hsm_ecc"                       // <<< Base name for output files
	userType := 1                                                // Normal user, could also be in config

	// 3. Initialize PKCS11
	pkcs11, err := initializePkcs11(pkcs11LibPath)
	if err != nil {
		fmt.Println("Error initializing PKCS11:", err)
		return
	}
	// Defer PKCS11 object disposal (must happen after session closure)
	defer pkcs11.DisposePkcs11() // Dispose the object itself

	// 4. Establish PKCS11 Session and Login (includes PIN)
	_, err = establishPkcs11Session(pkcs11, pin, userType)
	if err != nil {
		fmt.Println("Error establishing PKCS11 session:", err)
		// Cleanup for pkcs11 (Dispose) is handled by defer
		return
	}
	// Defer session closing (must happen before DisposePkcs11)
	defer pkcs11.CloseSession() // Close the session

	// --- Signing Loop ---
	numberOfSignatures := 10
	fmt.Printf("\n--- Starting Signing Loop (%d iterations) ---\n", numberOfSignatures)
	for i := 1; i <= numberOfSignatures; i++ {
		fmt.Printf("\n--- Iteration %d of %d ---\n", i, numberOfSignatures)

		// 5. Load PDF Document (Moved up as per request)
		pdf, err := loadPdfDocument(unsignedPdfPath)
		if err != nil {
			fmt.Println("Error loading PDF:", err)
			// Cleanup handled by defers
			return
		}
		defer pdf.DisposePdf() // Dispose the PDF object

		// 6. Find Certificate with Private Key from HSM (Moved down, requires active session)
		cert, err := findCertificateWithPrivateKey(pkcs11)
		if err != nil {
			fmt.Println("Error finding certificate:", err)
			// Cleanup handled by defers
			return
		}
		if cert == nil {
			fmt.Println("Could not find a usable certificate with an associated private key on the HSM.")
			// Cleanup handled by defers
			return
		}
		defer cert.DisposeCert() // Dispose the certificate object

		// 7. (Optional) Find HSM Handles for verification if needed
		_, _, err = findHsmHandles(pkcs11) // Ignore handles for now, just check error
		if err != nil {
			fmt.Println("Error finding HSM handles:", err)
			// Decide if this is fatal
			// return
		}
		pdf.SetSigningCert(cert)

		// 8. Configure Signing Options (JSON) (Moved down)
		jsonOptions, err := configureSigningOptions(pdf, cert)
		if err != nil {
			fmt.Println("Error configuring signing options:", err)
			// Cleanup handled by defers
			return
		}
		defer jsonOptions.DisposeJsonObject() // Dispose the JSON object

		// Construct the output path for this iteration
		outputFilename := fmt.Sprintf("%s_%d.pdf", baseOutputFilename, i)
		iterationOutputPath := filepath.Join(loopOutputDir, outputFilename)

		// Perform the signing for this iteration
		// Note: performPdfSigning now checks/creates the directory
		err = performPdfSigning(pdf, cert, jsonOptions, iterationOutputPath)
		if err != nil {
			fmt.Printf("Error during signing iteration %d: %v\n", i, err)
			// Decide whether to continue or break on error
			// break // Uncomment to stop loop on first error
			fmt.Println("Continuing to next iteration despite error...")
		}

		// Pause for 2 seconds between iterations
		fmt.Println("Sleeping for 2 seconds...")
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("\n--- Finished Signing Loop ---\n\n")

	// --- Logout after the loop ---
	err = pkcs11Logout(pkcs11)
	if err != nil {
		// Logged as warning inside the function, main continues
		fmt.Println("Note: Logout failed, proceeding with cleanup.")
	}

	fmt.Println("Program finished successfully.")
	// All deferred cleanup functions will execute now in reverse order:
	// DisposeJsonObject, DisposePdf, DisposeCert, CloseSession, DisposePkcs11, DisposeGlobal
}
