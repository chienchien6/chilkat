package main

import (
	"chilkat"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-10.1.3-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {
	// --- Load Configuration using Viper (ONCE) ---
	vip := viper.New()
	vip.SetConfigName("config")
	vip.SetConfigType("json")
	vip.AddConfigPath("C:/chilkatPackage/chilkattest")
	err := vip.ReadInConfig()
	if err != nil {
		log.Fatalf("Fatal error reading config file: %v", err)
	}
	fmt.Println("Configuration file loaded successfully.")

	// --- Get required paths and password from config (ONCE) ---
	pfxFilePath := vip.GetString("pfx_file_path")
	pfxPassword := vip.GetString("pfx_password")
	inputPdfPath := vip.GetString("pdf_input_path") // Assuming this is the unsigned PDF
	outputDirectory := `C:\chilkatPackage\chilkattest\p11\output\test_one`

	// Validate essential config values (ONCE)
	if pfxFilePath == "" {
		log.Fatal("Error: 'pfx_file_path' not found or empty in config.json")
	}
	if pfxPassword == "" {
		fmt.Println("Warning: 'pfx_password' is empty.")
	}
	if inputPdfPath == "" {
		log.Fatal("Error: 'pdf_input_path' not found or empty in config.json")
	}

	// --- Create the output directory if it doesn't exist (ONCE) ---
	err = os.MkdirAll(outputDirectory, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory '%s': %v", outputDirectory, err)
	}
	fmt.Printf("Ensured output directory exists: %s\n", outputDirectory)

	// --- Chilkat Initialization and Unlock (ONCE) ---
	glob := chilkat.NewGlobal()
	// Defer global cleanup AFTER loop
	defer glob.DisposeGlobal()
	success := glob.UnlockBundle("Anything for 30-day trial")
	if !success {
		log.Fatalf("Failed to unlock Chilkat: %s\n", glob.LastErrorText())
	}
	status := glob.UnlockStatus()
	if status != 2 {
		fmt.Println("Chilkat unlocked in trial mode.")
	} else {
		fmt.Println("Chilkat unlocked using purchased unlock code.")
	}

	// --- Load Certificate and Private Key from PFX (ONCE) ---
	cert := chilkat.NewCert()
	// Defer cert cleanup AFTER loop
	defer cert.DisposeCert()
	if _, err := os.Stat(pfxFilePath); os.IsNotExist(err) {
		log.Fatalf("Error: PFX file not found at %s\n", pfxFilePath)
	}
	success = cert.LoadPfxFile(pfxFilePath, pfxPassword)
	if !success {
		log.Fatalf("Failed to load PFX file '%s': %s\n", pfxFilePath, cert.LastErrorText())
	}
	if !cert.HasPrivateKey() {
		log.Printf("Warning: Loaded certificate does not have a private key context.")
	}
	fmt.Printf("Successfully loaded PFX. Certificate SubjectCN: %s\n", cert.SubjectCN())

	// --- Load Original PDF Document (ONCE) ---
	pdf := chilkat.NewPdf()
	// Defer PDF cleanup AFTER loop
	defer pdf.DisposePdf()
	if _, err := os.Stat(inputPdfPath); os.IsNotExist(err) {
		log.Fatalf("Error: Input PDF file not found at %s\n", inputPdfPath)
	}
	success = pdf.LoadFile(inputPdfPath)
	if !success {
		log.Fatalf("Failed to load input PDF '%s': %s\n", inputPdfPath, pdf.LastErrorText())
	}
	fmt.Printf("Loaded original unsigned PDF: %s\n", inputPdfPath)

	// --- Set Signing Certificate (ONCE) ---
	success = pdf.SetSigningCert(cert)
	if !success {
		log.Fatalf("Failed to set signing certificate on PDF object: %s\n", pdf.LastErrorText())
	}
	fmt.Println("Signing certificate set successfully on PDF object.")

	// --- Configure Signing JSON (ONCE) ---
	json := chilkat.NewJsonObject()
	// Defer JSON cleanup AFTER loop
	defer json.DisposeJsonObject()
	json.UpdateString("subFilter", "/ETSI.CAdES.detached")
	json.UpdateBool("signingCertificateV2", true)
	json.UpdateString("hashAlgorithm", "sha256")
	json.UpdateString("ocspDigestAlg", "sha256")
	json.UpdateInt("ocspTimeoutMs", 30000)
	json.UpdateInt("crlTimeoutMs", 30000)
	json.UpdateBool("ltvOcsp", true)
	json.UpdateBool("embedOcspResponses", true)
	json.UpdateBool("includeCertChain", true)
	json.UpdateBool("validateChain", false)
	json.UpdateBool("updateDss", true) // Request DSS update
	json.UpdateInt("signingTime", 1)
	json.UpdateBool("timestampToken.enabled", true)
	json.UpdateString("timestampToken.tsaUrl", "http://timestamp.digicert.com")
	json.UpdateBool("timestampToken.requestTsaCert", true)
	json.UpdateInt("timestampToken.timeoutMs", 30000)
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "PAdES Signature (Attempting B-LT)")

	// --- Signing Loop ---
	numberOfSignatures := 10
	fmt.Printf("\n--- Starting Signing Loop (%d iterations) ---\n", numberOfSignatures)
	for i := 1; i <= numberOfSignatures; i++ {
		// --- Generate unique output filename for this iteration ---
		outputFilename := fmt.Sprintf("signed_output_%d.pdf", i)
		outputFilePath := filepath.Join(outputDirectory, outputFilename)

		fmt.Printf("\n--- Iteration %d ---\n", i)
		fmt.Printf("Attempting to sign and save to: %s\n", outputFilePath)

		// --- Sign the PDF (using the single loaded pdf object) ---
		// Clear LastErrorText before the operation for cleaner logs per iteration
		pdf.LastErrorText()
		success = pdf.SignPdf(json, outputFilePath)
		lastErr := pdf.LastErrorText() // Capture error text immediately

		if !success {
			log.Printf("ERROR: Failed to sign PDF for iteration %d: %s\n", i, lastErr)
			// Optional: break here if one failure should stop the test
		} else {
			fmt.Printf("Successfully signed PDF for iteration %d.\n", i)
			if lastErr != "" {
				// It succeeded but there might be warnings (like OCSP/CRL issues)
				log.Printf("INFO: SignPdf succeeded for iteration %d, but LastErrorText contained warnings:\n%s\n", i, lastErr)
				log.Println("      External verification needed to confirm B-LT level.")
			}
			// --- Optional: Add LTA Step Here ---
			// if you want LTA for each signed file:
			// errLta := addLtaTimestamp(outputFilePath, "path/to/output_lta_" + outputFilename)
			// if errLta != nil {
			//     log.Printf("ERROR: Failed to add LTA timestamp for iteration %d file %s: %v\n", i, outputFilePath, errLta)
			// } else {
			//     fmt.Printf("Successfully added LTA timestamp for iteration %d.\n", i)
			// }
		}

		// --- Sleep between iterations ---
		fmt.Printf("Sleeping for 3 seconds...\n")
		time.Sleep(3 * time.Second)
	}

	fmt.Printf("\nFinished signing loop (%d iterations).\n", numberOfSignatures)
	fmt.Println("Program finished successfully.")
	// Deferred cleanup functions will run now
}

// Optional helper function for LTA
// func addLtaTimestamp(inputSignedPdfPath string, outputLtaPdfPath string) error {
// 	pdfLta := chilkat.NewPdf()
// 	defer pdfLta.DisposePdf()
// 	// Assuming global is still unlocked

// 	if !pdfLta.LoadFile(inputSignedPdfPath) {
// 		return fmt.Errorf("LTA: failed to load signed PDF '%s': %s", inputSignedPdfPath, pdfLta.LastErrorText())
// 	}

// 	tsaUrl := "http://timestamp.digicert.com" // Or your TSA URL
// 	if !pdfLta.AddLtaTimestamp(tsaUrl, outputLtaPdfPath) {
// 		return fmt.Errorf("LTA: failed to add archive timestamp: %s", pdfLta.LastErrorText())
// 	}
// 	return nil
// }
