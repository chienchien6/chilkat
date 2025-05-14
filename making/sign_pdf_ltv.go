package main

import (
	"chilkat"
	"fmt"
	"log" // 使用 log 進行嚴重錯誤記錄

	"github.com/spf13/viper" // 匯入 viper
)

/*
#cgo CFLAGS: -I C:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {
	// --- 使用 Viper 載入設定 ---
	viper.SetConfigName("config") // 設定檔名稱 (不含副檔名)
	viper.SetConfigType("json")   // 如果設定檔名不含副檔名，則必須指定類型
	// 指定設定檔的路徑 (這裡使用絕對路徑，根據您的環境調整)
	viper.AddConfigPath("C:/chilkatPackage/chilkattest")

	err := viper.ReadInConfig() // 尋找並讀取設定檔
	if err != nil {             // 處理讀取設定檔時的錯誤
		log.Fatalf("讀取設定檔時發生嚴重錯誤 (請確認 C:/chilkatPackage/chilkattest/config.json 存在且格式正確): %s \n", err)
	}

	// 存取設定值 (LTV簽名需要這些)
	pdfInputPath := viper.GetString("pdf_input_path")
	pfxPath := viper.GetString("pfx_path")
	pfxPassword := viper.GetString("pfx_password")
	// 注意：config.json 目前沒有專門的 LTV 輸出路徑，暫時使用 pdf_output_path
	// 建議在 config.json 中新增一個例如 "ltv_pdf_output_path" 的欄位以區分
	pdfOutputPath := viper.GetString("pdf_output_path")

	// 基本驗證
	if pdfInputPath == "" || pfxPath == "" || pfxPassword == "" || pdfOutputPath == "" {
		log.Fatalf("設定檔 config.json 缺少必要欄位 (pdf_input_path, pfx_path, pfx_password, pdf_output_path)\n")
	}
	// --- 設定載入結束 ---

	// --- Chilkat Global Unlock ---
	glob := chilkat.NewGlobal()
	glob.SetVerboseLogging(true)                            // 啟用詳細日誌
	fmt.Println("Chilkat Library Version:", glob.Version()) // 印出版本

	success := glob.UnlockBundle("Anything for 30-day trial") // 請替換成您的有效解鎖碼
	if !success {
		log.Fatalf("Chilkat 解鎖失敗: %s\n", glob.LastErrorText())
	}
	// --- Global Unlock 結束 ---

	// This example requires the Chilkat API to have been previously unlocked.
	// See Global Unlock Sample for sample code.

	pdf := chilkat.NewPdf()
	defer pdf.DisposePdf()      // 使用 defer 確保釋放
	pdf.SetVerboseLogging(true) // 啟用 PDF 物件的詳細日誌

	// Load a PDF to be signed. (使用設定檔中的路徑)
	success = pdf.LoadFile(pdfInputPath)
	if !success {
		fmt.Println("載入 PDF 失敗:", pdf.LastErrorText())
		// pdf.DisposePdf() // defer 會處理
		return
	}

	// Options for signing are specified in JSON.
	json := chilkat.NewJsonObject()
	defer json.DisposeJsonObject() // 使用 defer 確保釋放

	// In most cases, the signingCertificateV2 and signingTime attributes are required.
	json.UpdateInt("signingCertificateV2", 1)
	json.UpdateInt("signingTime", 1)

	// Add the "ltvOcsp" instruction to the JSON passed to SignPdf.
	// This is what causes Chilkat to create an LTV-enabled signature.
	//
	// If we are signing a PDF that already has signatures, then the existing signatures
	// are automatically verified, and Chilkat will do OCSP certificate status checking (if possible)
	// for those certs in existing signatures (including certs in the certificate chains)
	// that do not yet have a valid OCSP response in the DSS (Document Security Store).
	// Chilkat will add the OCSP responses to the /OCSPs in the Document Security Store (/DSS).
	// Also, and certificates from existing signatures not yet in the DSS are added to the /Certs
	// in the DSS.
	//
	// Also, the "ltvOcsp" causes Chilkat to add the pdfRevocationInfoArchival authenticated attribute
	// to the CMS signature.  The pdfRevocationInfoArchival attribute (1.2.840.113583.1.1.8)
	// contains OCSP responses and the CRL for the issuer of the signing certificate.
	// Therefore, Chilkat will send an OCSP request to the signing certificate's OCSP URl (if one exists)
	// and will download the CRL from the issuer certificate's CRL Distribution Point (if one exists).
	json.UpdateBool("ltvOcsp", true)

	// -----------------------------------------------------------------------------------
	// Note: If Chilkat produces a signed PDF, but the signature is not LTV-enabled,
	// the cause might be related to a failure to download CRL's or OCSP requests.
	// See Possible Solution for Failure to Produce LTV-enabled PDF Signature
	// -----------------------------------------------------------------------------------
	// You can add the following to UncommonOptions to get detailed information about the CRL and OCSP requests
	// You shouldn't set the following logging options unless there is a need, because it adds a large amount of information to the LastErrorText.
	pdf.SetUncommonOptions("LOG_OCSP_HTTP,LOG_CRL_HTTP")

	// Define the appearance of the signature.
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "bottom")
	json.UpdateString("appearance.x", "right")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "This is an LTV-enabled signature.")
	// json.UpdateString("contactInfo", "Signer: John Doe\nEmail: john.doe@example.com\nPhone: +123456789")

	// Load the signing certificate. (使用設定檔中的路徑和密碼)
	cert := chilkat.NewCert()
	defer cert.DisposeCert() // 使用 defer 確保釋放

	success = cert.LoadPfxFile(pfxPath, pfxPassword)
	if !success {
		fmt.Println("載入 PFX 憑證失敗:", cert.LastErrorText())
		// 清理已建立的物件 (defer 會處理)
		// pdf.DisposePdf()
		// json.DisposeJsonObject()
		// cert.DisposeCert()
		return
	}

	// Tell the pdf object to use the certificate for signing.
	success = pdf.SetSigningCert(cert)
	if !success {
		fmt.Println("設定簽名憑證失敗:", pdf.LastErrorText())
		// 清理已建立的物件 (defer 會處理)
		// pdf.DisposePdf()
		// json.DisposeJsonObject()
		// cert.DisposeCert()
		return
	}

	// Sign the PDF (使用設定檔中的輸出路徑)
	success = pdf.SignPdf(json, pdfOutputPath)
	if !success {
		fmt.Println("簽署 PDF 失敗:", pdf.LastErrorText())
		// 清理已建立的物件 (defer 會處理)
		// pdf.DisposePdf()
		// json.DisposeJsonObject()
		// cert.DisposeCert()
		return
	}

	fmt.Println("The PDF has been successfully cryptographically signed with long-term validation.")
	fmt.Printf("Signed PDF saved to: %s\n", pdfOutputPath)

	// If you open the Signature Panel in Adobe Acrobat, it will indicate that the signature is LTV enabled
	// as shown here:
	// [Image showing LTV enabled status in Acrobat]

	// pdf.DisposePdf() // defer 會處理
	// json.DisposeJsonObject() // defer 會處理
	// cert.DisposeCert() // defer 會處理
}
