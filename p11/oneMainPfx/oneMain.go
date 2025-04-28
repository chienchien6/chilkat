package main

import (
	"chilkat"
	"fmt"
	"log"           // Using log for fatal errors
	"os"            // Required for file existence check and creating directories
	"path/filepath" // For constructing paths robustly
	"time"          // For sleeping between iterations

	"github.com/spf13/viper" // Viper for configuration
)

/*
// Ensure CGO flags point to the correct Chilkat version you are using
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-10.1.3-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {
	numberOfSignatures := 10
	fmt.Printf("\nStarting signing loop for %d iterations...\n", numberOfSignatures)

	for i := 1; i <= numberOfSignatures; i++ { // Loop from 1 to 10 for filenames
		// --- Load Configuration using Viper ---
		vip := viper.New()
		vip.SetConfigName("config")
		vip.SetConfigType("json")
		vip.AddConfigPath("C:/chilkatPackage/chilkattest") // Path to your config.json
		err := vip.ReadInConfig()
		if err != nil {
			log.Fatalf("Fatal error reading config file: %v\nEnsure config.json exists at the specified path and is valid JSON.", err)
		}
		fmt.Println("Configuration file loaded successfully.")

		// --- Get required paths and password from config ---
		pfxFilePath := vip.GetString("pfx_file_path") // Path to your PFX file
		pfxPassword := vip.GetString("pfx_password")  // Password for the PFX file
		// Use the *same* input PDF for all iterations
		inputPdfPath := vip.GetString("pdf_input_path") // Or "unsigned_pdf_input_path" if that's the one you mean

		// --- Define the specific output directory for this loop ---
		outputDirectory := `C:\chilkatPackage\chilkattest\p11\output\test_one` // Raw string literal for path

		// Validate essential config values
		if pfxFilePath == "" {
			log.Fatal("Error: 'pfx_file_path' not found or empty in config.json")
		}
		if pfxPassword == "" {
			fmt.Println("Warning: 'pfx_password' is empty in config.json. Assuming PFX has no password.")
		}
		if inputPdfPath == "" {
			log.Fatal("Error: 'pdf_input_path' (or the key you intend to use) not found or empty in config.json")
		}

		// --- Create the output directory if it doesn't exist ---
		err = os.MkdirAll(outputDirectory, 0755) // 0755 permissions are common
		if err != nil {
			log.Fatalf("Failed to create output directory '%s': %v", outputDirectory, err)
		}
		fmt.Printf("Ensured output directory exists: %s\n", outputDirectory)

		// --- Chilkat Initialization and Unlock ---
		glob := chilkat.NewGlobal()
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

		// --- Load Certificate and Private Key from PFX (Load ONCE outside the loop) ---
		cert := chilkat.NewCert()
		defer cert.DisposeCert()

		if _, err := os.Stat(pfxFilePath); os.IsNotExist(err) {
			log.Fatalf("Error: PFX file not found at %s\n", pfxFilePath)
		}
		success = cert.LoadPfxFile(pfxFilePath, pfxPassword)
		if !success {
			log.Fatalf("Failed to load PFX file '%s': %s\n", pfxFilePath, cert.LastErrorText())
		}
		if !cert.HasPrivateKey() {
			log.Printf("Warning: Loaded certificate from PFX does not have a private key context according to Chilkat.")
		}
		fmt.Printf("Successfully loaded PFX for signing. Certificate SubjectCN: %s\n", cert.SubjectCN())

		// --- PDF Handling (Load ONCE outside the loop) ---
		pdf := chilkat.NewPdf()
		defer pdf.DisposePdf()

		if _, err := os.Stat(inputPdfPath); os.IsNotExist(err) {
			log.Fatalf("Error: Input PDF file not found at %s\n", inputPdfPath)
		}
		success = pdf.LoadFile(inputPdfPath)
		if !success {
			log.Fatalf("Failed to load input PDF '%s' for signing: %s\n", inputPdfPath, pdf.LastErrorText())
		}
		fmt.Printf("Loaded unsigned PDF: %s\n", inputPdfPath)

		// --- Configure Signing JSON (Configure ONCE outside the loop) ---
		json := chilkat.NewJsonObject()
		defer json.DisposeJsonObject()

		json.UpdateString("subFilter", "/ETSI.CAdES.detached")
		json.UpdateBool("signingCertificateV2", true)
		// json.UpdateString("signingAlgorithm", "pkcs") // 通常不需要特別指定，Chilkat 會自動處理
		json.UpdateString("hashAlgorithm", "sha256")
		json.UpdateInt("signingTime", 1) // 加入簽署時間

		json.UpdateInt("page", 1)
		json.UpdateString("appearance.y", "top")
		json.UpdateString("appearance.x", "left")
		json.UpdateString("appearance.fontScale", "10.0")
		json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
		json.UpdateString("appearance.text[1]", "current_dt")
		json.UpdateString("appearance.text[2]", "Loop Signing Test (Go/PFX)")
		// json.UpdateString("signingReason", "Automated Test")
		// json.UpdateString("signingLocation", "Test Environment")

		// --- Set Signing Certificate (Set ONCE outside the loop) ---
		success = pdf.SetSigningCert(cert)
		if !success {
			log.Fatalf("Failed to set signing certificate on PDF object: %s\n", pdf.LastErrorText())
		}
		fmt.Println("Signing certificate set successfully on PDF object.")

		// --- Signing Loop ---

		// --- Generate unique output filename for this iteration ---
		outputFilename := fmt.Sprintf("signed_output_%d.pdf", i)
		// Use filepath.Join for cross-platform compatibility
		outputFilePath := filepath.Join(outputDirectory, outputFilename)

		fmt.Printf("\n--- Iteration %d ---\n", i)
		fmt.Printf("Attempting to sign and save to: %s\n", outputFilePath)

		// --- Sign the PDF ---
		// The pdf object still holds the original loaded document.
		// SetSigningCert was already called. We just call SignPdf again.
		success = pdf.SignPdf(json, outputFilePath)
		if !success {
			// Log error but continue the loop (unless you want to stop on first error)
			log.Printf("ERROR: Failed to sign PDF for iteration %d: %s\n", i, pdf.LastErrorText())
		} else {
			fmt.Printf("Successfully signed PDF for iteration %d.\n", i)
		}

		// --- Sleep between iterations ---
		fmt.Printf("Sleeping for 3 seconds...\n")
		time.Sleep(3 * time.Second)
	}

	// --- Cleanup ---
	// Cleanup is handled by defer statements
	fmt.Printf("\nFinished signing loop (%d iterations).\n", numberOfSignatures)
	fmt.Println("Program finished successfully.")
}
