package main

import (
	"chilkat"
	"fmt"
	// "os" // Keep os package for file check - Removed as os.Stat is no longer needed
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

// go run .\example1.go > signature.xml
func main() {

	glob := chilkat.NewGlobal()
	success := glob.UnlockBundle("Anything for 30-day trial")
	if success != true {
		fmt.Println(glob.LastErrorText())
		return
	}

	zipFile := chilkat.NewBinData()
	localKeyFile := "private-key.zip"

	// Try to load the key from the local file first.
	success = zipFile.LoadFile(localKeyFile)
	if success != true {
		fmt.Printf("Local key file '%s' not found or failed to load. Falling back to URL download.\n", localKeyFile)
		// fmt.Println(glob.LastErrorText()) // Optional: print error if needed

		// Fallback to downloading from URL
		http := chilkat.NewHttp()
		keyUrl := "https://www.chilkatsoft.com/exampleData/secp256r1-key.zip"
		fmt.Printf("Downloading key from %s...\n", keyUrl)
		success = http.QuickGetBd(keyUrl, zipFile)
		if success != true {
			fmt.Println("Failed to download key from URL:")
			fmt.Println(http.LastErrorText())
			return
		}
		fmt.Println("Key downloaded successfully.")
	} else {
		fmt.Printf("Loaded key from local file '%s'.\n", localKeyFile)
	}

	// Proceed with the zip data (either loaded locally or downloaded)
	zip := chilkat.NewZip()
	success = zip.OpenBd(zipFile)
	if success != true {
		fmt.Println("Failed to open zip data:", zip.LastErrorText())
		return
	}

	zipEntry := zip.FirstMatchingEntry("*.pem")
	if zipEntry == nil {
		fmt.Println("No .pem file found inside the zip data.")
		zip.DisposeZip() // Dispose zip object
		return
	}

	ecKey := chilkat.NewPrivateKey()
	// Use UnzipToString to get PEM content
	pemContent := zipEntry.UnzipToString(0, "utf-8")
	if pemContent == nil {
		fmt.Println("Failed to unzip PEM content:", zipEntry.LastErrorText())
		zipEntry.DisposeZipEntry()
		zip.DisposeZip()
		return
	}

	success = ecKey.LoadPem(*pemContent)
	if success != true {
		fmt.Println("Failed to load PEM key:")
		fmt.Println(ecKey.LastErrorText())
		zipEntry.DisposeZipEntry()
		zip.DisposeZip()
		return
	}

	zipEntry.DisposeZipEntry()
	zip.DisposeZip() // Dispose zip object after use

	//  ----------------------------------------------------------------------------
	gen := chilkat.NewXmlDSigGen()

	// Provide the ECDSA key to the XML Digital Signature generator
	gen.SetPrivateKey(ecKey)

	// Add an enveloped reference to the content to be signed.
	sbContent := chilkat.NewStringBuilder()
	sbContent.Append("This is the content that is signed.")
	gen.AddEnvelopedRef("abc123", sbContent, "sha256", "C14N", "")

	// Generate the XML digital signature.
	sbXml := chilkat.NewStringBuilder()
	success = gen.CreateXmlDSigSb(sbXml)
	if success != true {
		fmt.Println(gen.LastErrorText())
		ecKey.DisposePrivateKey() // Dispose key object
		return
	}

	// Examine the enveloped signature
	fmt.Println("Generated XML Signature:")
	fmt.Println(*sbXml.GetAsString())
	fmt.Println("\nsuccess")

	// Dispose remaining objects
	ecKey.DisposePrivateKey()
	gen.DisposeXmlDSigGen()
	sbContent.DisposeStringBuilder()
	sbXml.DisposeStringBuilder()
	glob.DisposeGlobal()

}
