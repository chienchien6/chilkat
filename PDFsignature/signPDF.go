package main

import (
	"chilkat"
	"fmt"
	"log" // Use log for fatal errors

	"github.com/spf13/viper" // Import viper
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

// No need for custom Config struct or loadConfig function with viper

func main() {

	// --- Load configuration using Viper ---
	viper.SetConfigName("config") // Name of config file (without extension)
	viper.SetConfigType("json")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // Path to look for the config file in (current directory)
	// viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
	// viper.AddConfigPath("/etc/appname/")  // path to look for the config file in

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		// Use log.Fatalf for critical startup errors
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	// Access config values
	pdfInputPath := viper.GetString("pdf_input_path")
	pfxPath := viper.GetString("pfx_path")
	pfxPassword := viper.GetString("pfx_password")
	pdfOutputPath := viper.GetString("pdf_output_path")

	// Basic validation
	if pdfInputPath == "" || pfxPath == "" || pfxPassword == "" || pdfOutputPath == "" {
		log.Fatalf("Config file config.json is missing required fields (pdf_input_path, pfx_path, pfx_password, pdf_output_path)\n")
	}
	// --- End Configuration Loading ---

	glob := chilkat.NewGlobal()
	success := glob.UnlockBundle("Anything for 30-day trial")
	if success != true {
		// Use log.Fatalf as unlocking is critical
		log.Fatalf("Chilkat unlock failed: %s\n", glob.LastErrorText())
	}

	pdf := chilkat.NewPdf()
	defer pdf.DisposePdf() // Ensure PDF object is disposed

	// Load a PDF to be signed using path from config
	success = pdf.LoadFile(pdfInputPath)
	if success == false {
		fmt.Println("Failed to load PDF:", pdf.LastErrorText()) // Keep fmt for non-fatal errors during operation
		return
	}

	// Options for signing are specified in JSON.
	jsonOptions := chilkat.NewJsonObject()
	defer jsonOptions.DisposeJsonObject() // Ensure JSON object is disposed

	// In most cases, the signingCertificateV2 and signingTime attributes are required.
	jsonOptions.UpdateInt("signingCertificateV2", 1)
	jsonOptions.UpdateInt("signingTime", 1)

	// Put the signature on page 1, top left
	jsonOptions.UpdateInt("page", 1)
	jsonOptions.UpdateString("appearance.y", "top")
	jsonOptions.UpdateString("appearance.x", "left")

	// Use a font scale of 10.0
	jsonOptions.UpdateString("appearance.fontScale", "10.0")

	// Appearance text lines
	jsonOptions.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	jsonOptions.UpdateString("appearance.text[1]", "current_dt")
	jsonOptions.UpdateString("appearance.text[2]", "The crazy brown fox jumps over the lazy dog.")

	// Load the signing certificate using path and password from config.
	cert := chilkat.NewCert()
	defer cert.DisposeCert() // Ensure Cert object is disposed
	success = cert.LoadPfxFile(pfxPath, pfxPassword)
	if success == false {
		fmt.Println("Failed to load PFX certificate:", cert.LastErrorText())
		return
	}

	// Tell the pdf object to use the certificate for signing.
	success = pdf.SetSigningCert(cert)
	if success == false {
		fmt.Println("Failed to set signing certificate:", pdf.LastErrorText())
		return
	}

	// Sign the PDF using output path from config
	success = pdf.SignPdf(jsonOptions, pdfOutputPath)
	if success == false {
		fmt.Println("Failed to sign PDF:", pdf.LastErrorText())
		return
	}

	fmt.Printf("The PDF has been successfully signed and saved to %s\n", pdfOutputPath)

	// glob is disposed implicitly by program exit, or could be added with defer if needed elsewhere
}
