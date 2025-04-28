package main

import (
	"chilkat"
	"errors"
	"fmt"
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

// --- Configuration Loading ---
func loadConfig() (*viper.Viper, error) {
	vip := viper.New()
	vip.SetConfigName("config")
	vip.SetConfigType("json")
	vip.AddConfigPath("C:/chilkatPackage/chilkattest") // Add other paths if needed
	// Add current directory as a fallback
	vip.AddConfigPath(".")
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
	glob.SetVerboseLogging(true)
	success := glob.UnlockBundle("Anything for 30-day trial") // Use your actual unlock code
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

// --- Certificate Loading from PFX ---
func loadCertificateFromPfx(pfxPath, password string) (*chilkat.Cert, error) {
	if pfxPath == "" {
		return nil, errors.New("PFX file path is empty")
	}
	cert := chilkat.NewCert()
	// No need to set password here, SetSigningCert handles it with Pdf.SetPfxPassword
	success := cert.LoadPfxFile(pfxPath, password)
	if !success {
		errMsg := cert.LastErrorText()
		cert.DisposeCert() // Dispose if load fails
		return nil, fmt.Errorf("failed to load certificate from PFX '%s': %s", pfxPath, errMsg)
	}
	fmt.Printf("Successfully loaded certificate from PFX: %s (SubjectCN: %s)\n", pfxPath, cert.SubjectCN())

	// Optional: Check if the loaded cert has a private key
	if !cert.HasPrivateKey() {
		errMsg := fmt.Sprintf("Error: The certificate loaded from PFX '%s' does not have an associated private key.", pfxPath)
		cert.DisposeCert()
		return nil, errors.New(errMsg)
	}
	fmt.Println("Certificate from PFX confirmed to have an associated private key.")
	return cert, nil
}

// --- PDF Loading ---
func loadPdfDocument(filePath string) (*chilkat.Pdf, error) {
	if filePath == "" {
		return nil, errors.New("unsigned PDF input path is empty")
	}
	pdf := chilkat.NewPdf()
	pdf.SetVerboseLogging(true)
	pdf.SetSigAllocateSize(30000)
	success := pdf.LoadFile(filePath)
	if !success {
		errMsg := pdf.LastErrorText()
		pdf.DisposePdf()
		return nil, fmt.Errorf("failed to load PDF '%s': %s", filePath, errMsg)
	}
	fmt.Printf("Loaded unsigned PDF: %s\n", filePath)
	return pdf, nil
}

// --- Configure Signing Options ---
func configureSigningOptions() (*chilkat.JsonObject, error) {
	json := chilkat.NewJsonObject()
	// Use standard PKCS7 detached signature for PFX signing unless specific needs require others
	json.UpdateString("subFilter", "/adbe.pkcs7.detached")
	json.UpdateBool("signingCertificateV2", true)
	json.UpdateInt("signingTime", 1)
	// Timestamping is optional for basic PFX signing, remove or keep based on needs
	// json.UpdateInt("timestamp", 1)
	// json.UpdateString("tsaUrl", "http://tsa.dutchconnect.nl/tsa")
	json.UpdateString("hashAlgorithm", "sha256")
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "Validated via PFX") // Updated text
	fmt.Println("Signing options configured.")
	return json, nil
}

// --- PDF Signing using PFX Certificate Object ---
func performPdfSigningPfx(pdf *chilkat.Pdf, cert *chilkat.Cert, pfxPassword string, jsonOptions *chilkat.JsonObject, outputPath string) error {
	if outputPath == "" {
		return errors.New("signed PDF output path is empty")
	}
	if pdf == nil || cert == nil || jsonOptions == nil {
		return errors.New("invalid parameters for PDF signing (nil PDF, Cert, or JSON)")
	}
	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		fmt.Printf("Output directory %s does not exist, creating it.\n", outputDir)
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
		}
	}

	// 設定從 PFX 載入的憑證物件
	fmt.Println("正在設定簽署憑證（來自 PFX）到 PDF 物件...")
	success := pdf.SetSigningCert(cert)
	// pdf.SetUncommonOptions("NO_VERIFY_CERT_SIGNATURES") // Keep if needed

	if !success {
		return fmt.Errorf("failed to set signing certificate (from PFX) on PDF object: %s", pdf.LastErrorText())
	}
	fmt.Println("PFX signing certificate configured successfully on PDF object.")

	// Sign the PDF
	fmt.Println("--- Beginning PDF Signing (PFX) --- (Verbose logs follow if error occurs)")
	fmt.Printf("Attempting to sign PDF and save to: %s\n", outputPath)
	success = pdf.SignPdf(jsonOptions, outputPath)
	if !success {
		errMsg := pdf.LastErrorText()
		fmt.Println("--- PDF Signing (PFX) Failed --- Verbose LastErrorText: ---")
		fmt.Println(errMsg)
		fmt.Println("--- End of Verbose LastErrorText ---")
		return fmt.Errorf("failed to sign PDF '%s' using PFX (see verbose log above)", outputPath)
	}
	fmt.Printf("PDF signed successfully using PFX: %s\n", outputPath)
	return nil
}

// --- Main Application Logic ---
func main() {
	// Defer global Chilkat cleanup
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

	// Get config values for PFX signing
	unsignedPdfPath := vip.GetString("unsigned_pdf_input_path")
	pfxFilePath := vip.GetString("pfx_file_path")                        // <<< Config key for PFX file
	pfxPassword := vip.GetString("pfx_password")                         // <<< Config key for PFX password
	loopOutputDir := "C:/chilkatPackage/chilkattest/p11/output/test_pfx" // <<< Adjusted output dir
	baseOutputFilename := "signed_pfx_hello"                             // <<< Adjusted base filename

	// --- Check required PFX config ---
	if pfxFilePath == "" {
		fmt.Println("Error: 'pfx_file_path' not found or empty in config file.")
		return
	}
	// pfxPassword can be empty if the PFX file has no password

	// --- Load resources needed for signing ---
	pdf, err := loadPdfDocument(unsignedPdfPath)
	if err != nil {
		fmt.Println("Error loading PDF:", err)
		return
	}
	defer pdf.DisposePdf()

	// Load certificate from PFX file
	cert, err := loadCertificateFromPfx(pfxFilePath, pfxPassword)
	if err != nil {
		fmt.Println("Error loading certificate from PFX:", err)
		return
	}
	if cert == nil { // Should be handled by error, but double-check
		fmt.Println("Failed to load a valid certificate from PFX.")
		return
	}
	defer cert.DisposeCert()

	jsonOptions, err := configureSigningOptions()
	if err != nil {
		fmt.Println("Error configuring signing options:", err)
		return
	}
	defer jsonOptions.DisposeJsonObject()

	// --- Signing Loop ---
	numberOfSignatures := 10
	fmt.Printf("\n--- Starting PFX Signing Loop (%d iterations) ---\n", numberOfSignatures)
	for i := 1; i <= numberOfSignatures; i++ {
		fmt.Printf("\n--- Iteration %d of %d ---\n", i, numberOfSignatures)

		outputFilename := fmt.Sprintf("%s_%d.pdf", baseOutputFilename, i)
		iterationOutputPath := filepath.Join(loopOutputDir, outputFilename)

		// Perform the signing for this iteration using PFX cert
		err = performPdfSigningPfx(pdf, cert, pfxPassword, jsonOptions, iterationOutputPath)
		if err != nil {
			fmt.Printf("Error during signing iteration %d: %v\n", i, err)
			// break // Uncomment to stop loop on first error
			fmt.Println("Continuing to next iteration despite error...")
		}

		fmt.Println("Sleeping for 2 seconds...")
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("\n--- Finished PFX Signing Loop ---\n\n")

	fmt.Println("Program finished successfully.")
	// Deferred cleanup: DisposeJsonObject, DisposeCert, DisposePdf, DisposeGlobal
}
