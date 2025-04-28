package main

import (
	"chilkat"
	"errors"        // <<< Added for custom errors
	"fmt"           // <<< Added for directory creation
	"path/filepath" // <<< Added for path joining
	"time"          // <<< Added for sleep

	// <<< Added for converting int to string if needed later
	// <<< Added for converting int to string if needed later
	"github.com/spf13/viper" // Viper for configuration
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
	jsonTemplateKey.UpdateString("key_type", "rsa")
	jsonTemplateKey.UpdateString("label", "RSA Private Key")
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
	jsonTemplateCert.UpdateString("label", "X509 RSA Certificate")
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

// --- PDF Loading ---
func loadPdfDocument(filePath string) (*chilkat.Pdf, error) {
	if filePath == "" {
		return nil, errors.New("unsigned PDF input path is empty")
	}
	pdf := chilkat.NewPdf()
	pdf.SetVerboseLogging(true)   // <<< Enable PDF Verbose Logging
	pdf.SetSigAllocateSize(30000) // <<< Use the setter method

	// defer pdf.DisposePdf() // Dispose in the caller (main)
	success := pdf.LoadFile(filePath)
	if !success {
		errMsg := pdf.LastErrorText()
		pdf.DisposePdf() // Dispose if load fails
		return nil, fmt.Errorf("failed to load PDF '%s': %s", filePath, errMsg)
	}
	fmt.Printf("Loaded unsigned PDF: %s\n", filePath)
	return pdf, nil
}

// --- Configure Signing Options ---
func configureSigningOptions() (*chilkat.JsonObject, error) {
	json := chilkat.NewJsonObject()
	// defer json.DisposeJsonObject() // Dispose in the caller (main)

	json.UpdateString("subFilter", "/ETSI.CAdES.detached") // Or /adbe.pkcs7.detached
	// Defaulting to no subFilter specified, let Chilkat decide or use SetSignatureSigningTime/HashAlg directly if needed
	json.UpdateBool("signingCertificateV2", true)
	json.UpdateInt("signingTime", 1)
	json.UpdateInt("timestamp", 1)               // Add signing time attribute
	json.UpdateString("hashAlgorithm", "sha256") // Use SHA-256

	// Appearance settings
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "Validated via HSM") // Example text

	fmt.Println("Signing options configured.")
	return json, nil
}

// --- PDF Signing ---
func performPdfSigning(pdf *chilkat.Pdf, cert *chilkat.Cert, jsonOptions *chilkat.JsonObject, outputPath string) error {
	if outputPath == "" {
		return errors.New("signed PDF output path is empty")
	}
	if pdf == nil || cert == nil || jsonOptions == nil {
		return errors.New("invalid parameters for PDF signing (nil PDF, Cert, or JSON)")
	}

	// Set the certificate object to use for signing.
	// Chilkat automatically uses the private key associated within the HSM session.
	fmt.Println("Setting signing certificate on PDF object...")
	success := pdf.SetSigningCert(cert)
	pdf.SetUncommonOptions("NO_VERIFY_CERT_SIGNATURES") // <<< Use the setter method

	if !success {
		return fmt.Errorf("failed to set signing certificate on PDF object: %s", pdf.LastErrorText())
	}
	fmt.Println("PKCS11 signing certificate configured successfully on PDF object.")

	// Sign the PDF
	fmt.Println("--- Beginning PDF Signing --- (Verbose logs follow if error occurs)") // <<< Added Log
	fmt.Printf("Attempting to sign PDF and save to: %s\n", outputPath)
	success = pdf.SignPdf(jsonOptions, outputPath)
	if !success {
		errMsg := pdf.LastErrorText()                                        // Capture the potentially long error message
		fmt.Println("--- PDF Signing Failed --- Verbose LastErrorText: ---") // <<< Added Log
		fmt.Println(errMsg)                                                  // Print the full verbose error
		fmt.Println("--- End of Verbose LastErrorText ---")                  // <<< Added Log
		return fmt.Errorf("failed to sign PDF (see verbose log above)")      // Modify error message slightly
	}
	fmt.Println("PDF signed successfully!")
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
	baseOutputFilename := "signed_hsm_rsa"                       // <<< Base name for output files
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

		// 8. Configure Signing Options (JSON) (Moved down)
		jsonOptions, err := configureSigningOptions()
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
