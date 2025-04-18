package main

import (
	"chilkat"
	"fmt"
)

/*
#cgo CFLAGS: -I C:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {

	glob := chilkat.NewGlobal()
	success := glob.UnlockBundle("Anything for 30-day trial")
	if success != true {
		fmt.Println(glob.LastErrorText())
		return
	}

	//  Let's use the ECDSA private key at https://www.chilkatsoft.com/exampleData/secp256r1-key.zip
	//  for signing.
	http := chilkat.NewHttp()
	zipFile := chilkat.NewBinData()
	keyUrl := "https://www.chilkatsoft.com/exampleData/secp256r1-key.zip"

	success = http.QuickGetBd(keyUrl, zipFile)
	if success != true {
		fmt.Println(http.LastErrorText())
		return
	}

	zip := chilkat.NewZip()
	success = zip.OpenBd(zipFile)
	zipEntry := zip.FirstMatchingEntry("*.pem")
	ecKey := chilkat.NewPrivateKey()
	success = ecKey.LoadPem(*zipEntry.UnzipToString(0, "utf-8"))
	if success != true {
		fmt.Println(ecKey.LastErrorText())
		zipEntry.DisposeZipEntry()
		return
	}

	zipEntry.DisposeZipEntry()
	//zipEntry.DisposeZipEntry() 是用來釋放與 ZipEntry 物件相關的資源，確保不再需要時能正確清理記憶體或檔案句柄

	//  ----------------------------------------------------------------------------
	gen := chilkat.NewXmlDSigGen()

	//  Provide the ECDSA key to the XML Digital Signature generator使用設定好的 ECDSA 私密金鑰 (ecKey) 對雜湊值進行簽章。
	gen.SetPrivateKey(ecKey)

	//  Add an enveloped reference to the content to be signed.
	sbContent := chilkat.NewStringBuilder()
	sbContent.Append("This is the content that is signed.")
	gen.AddEnvelopedRef("abc123", sbContent, "sha256", "C14N", "")

	//  Generate the XML digital signature.
	//  Notice that in other examples, the sbXml passed to CreateXmlDSigSb
	//  already contains XML, and the XML signature is inserted at the location
	//  specified by the SigLocation property.  In this case, both SigLocation
	//  and sbXml are empty.  The result is that sbXml will contain just the Signature.
	sbXml := chilkat.NewStringBuilder()
	success = gen.CreateXmlDSigSb(sbXml)
	if success != true {
		fmt.Println(gen.LastErrorText())
		return
	}

	//  Examine the enveloped signature, where the data is contained within the XML Signature
	fmt.Println(*sbXml.GetAsString())
	fmt.Println("success")
	//go run .\chilkat_example1.go > signature.xml

}
