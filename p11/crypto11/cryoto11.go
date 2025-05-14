package main

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ThalesGroup/crypto11"
	"github.com/spf13/viper"
)

// --- Configuration Loading ---
func loadConfig() (*viper.Viper, error) {
	vip := viper.New()
	vip.SetConfigName("config")
	vip.SetConfigType("json")
	vip.AddConfigPath("C:/chilkatPackage/chilkattest")
	err := vip.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
	}
	fmt.Println("Configuration file loaded successfully.")
	return vip, nil
}

// --- Initialize crypto11 ---
func initializeCrypto11(pkcs11LibPath, hsmPin string) (*crypto11.Context, error) {
	if pkcs11LibPath == "" {
		return nil, errors.New("PKCS11 library path is empty")
	}

	config := &crypto11.Config{
		Path:            pkcs11LibPath,
		Pin:             hsmPin,
		UseGCMIVFromHSM: true, // Enable GCM IV from HSM
		MaxSessions:     2,    // Limit concurrent sessions
	}

	ctx, err := crypto11.Configure(config)
	if err != nil {
		// Add more detailed error information
		if err.Error() == "pkcs11: 0x6: CKR_FUNCTION_FAILED" {
			return nil, fmt.Errorf("PKCS11 function failed. Please verify HSM connection and slot configuration. Error: %w", err)
		}
		return nil, fmt.Errorf("failed to initialize crypto11: %w", err)
	}
	fmt.Println("crypto11 initialized successfully.")
	return ctx, nil
}

// --- Find Certificate with Private Key ---
func findCertificateWithPrivateKey(ctx *crypto11.Context) (*x509.Certificate, crypto.Signer, error) {
	// Find all private keys
	privateKeys, err := ctx.FindAllKeyPairs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find key pairs: %w", err)
	}

	for _, privateKey := range privateKeys {
		// Attempt to retrieve the associated certificate
		cert, err := ctx.FindCertificate(nil, nil, nil)
		if err != nil {
			fmt.Printf("Warning: Failed to retrieve certificate for key pair: %v\n", err)
			continue
		}

		if cert != nil {
			fmt.Printf("Found certificate: %s\n", cert.Subject.CommonName)
			return cert, privateKey, nil
		}
	}

	return nil, nil, errors.New("no certificate with private key found")
}

// --- Load PDF Document ---
func loadPdfDocument(filePath string) ([]byte, error) {
	if filePath == "" {
		return nil, errors.New("unsigned PDF input path is empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load PDF '%s': %w", filePath, err)
	}

	fmt.Printf("Loaded unsigned PDF: %s\n", filePath)
	return data, nil
}

// --- Perform Signing ---
func performSigningOneStep(pdfData []byte, signer crypto.Signer, cert *x509.Certificate, outputPath string) error {
	if outputPath == "" {
		return errors.New("signed PDF output path is empty")
	}

	// Example: Hash the PDF data and sign it
	hash := crypto.SHA256.New()
	_, err := hash.Write(pdfData)
	if err != nil {
		return fmt.Errorf("failed to hash PDF data: %w", err)
	}
	hashed := hash.Sum(nil)

	signature, err := signer.Sign(nil, hashed, crypto.SHA256)
	if err != nil {
		return fmt.Errorf("failed to sign PDF data: %w", err)
	}

	// Save the signed data (this is a placeholder; adapt for your PDF signing logic)
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	err = os.WriteFile(outputPath, signature, 0644)
	if err != nil {
		return fmt.Errorf("failed to save signed PDF: %w", err)
	}

	fmt.Printf("Signed PDF saved to: %s\n", outputPath)
	return nil
}

// --- Main Application Logic ---
func main() {
	vip, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		return
	}

	pkcs11LibPath := vip.GetString("pkcs11_lib_path")
	hsmPin := vip.GetString("hsm_pin")
	unsignedPdfPath := vip.GetString("unsigned_pdf_input_path")
	outputPath := "C:/chilkatPackage/chilkattest/p11/onepiece/output/signed_onestep.pdf"

	ctx, err := initializeCrypto11(pkcs11LibPath, hsmPin)
	if err != nil {
		fmt.Println("Error initializing crypto11:", err)
		return
	}
	defer ctx.Close()

	cert, signer, err := findCertificateWithPrivateKey(ctx)
	if err != nil {
		fmt.Println("Error finding certificate:", err)
		return
	}

	pdfData, err := loadPdfDocument(unsignedPdfPath)
	if err != nil {
		fmt.Println("Error loading PDF:", err)
		return
	}

	err = performSigningOneStep(pdfData, signer, cert, outputPath)
	if err != nil {
		fmt.Println("Error during signing:", err)
		return
	}

	fmt.Println("Program finished successfully.")
}
