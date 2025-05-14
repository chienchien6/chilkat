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
	viper.AddConfigPath("C:/chilkatPackage/chilkattest")

	err := viper.ReadInConfig() // 尋找並讀取設定檔
	if err != nil {             // 處理讀取設定檔時的錯誤
		log.Fatalf("讀取設定檔時發生嚴重錯誤 (請確認 C:/chilkatPackage/chilkattest/config.json 存在且格式正確): %s \n", err)
	}

	// 存取設定值
	pdfInputPath := viper.GetString("pdf_input_path")
	pfxPath := viper.GetString("pfx_file_path")
	pfxPassword := viper.GetString("pfx_password")
	// 注意：config.json 目前沒有專門的 LTV+Timestamp 輸出路徑
	// 建議在 config.json 中新增一個例如 "ltv_timestamp_pdf_output_path" 的欄位
	pdfOutputPath := viper.GetString("pdf_output_path") // 暫時使用通用的輸出路徑

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

	pdf := chilkat.NewPdf()
	defer pdf.DisposePdf()      // 使用 defer 確保釋放
	pdf.SetVerboseLogging(true) // 啟用 PDF 物件的詳細日誌

	// Load a PDF to be signed (使用設定檔中的路徑)
	success = pdf.LoadFile(pdfInputPath)
	if !success {
		fmt.Println("載入 PDF 失敗:", pdf.LastErrorText())
		return
	}

	// Options for signing are specified in JSON.
	json := chilkat.NewJsonObject()
	defer json.DisposeJsonObject() // 使用 defer 確保釋放

	// In most cases, the signingCertificateV2 and signingTime attributes are required.
	json.UpdateInt("signingCertificateV2", 1)
	json.UpdateInt("signingTime", 1)

	// Tell Chilkat to create an LTV-enabled (long term validation) signature.
	json.UpdateBool("ltvOcsp", true)

	// --- 加入 TSA 時間戳記設定 ---
	// Tell Chilkat to request a timestamp from a TSA server
	json.UpdateBool("timestampToken.enabled", true)

	// 指定 TSA 伺服器 URL (這裡使用免費的 DigiCert TSA，您可以替換成您自己的)
	// 也可以考慮將此 URL 放入 config.json
	json.UpdateString("timestampToken.tsaUrl", "http://timestamp.digicert.com")

	// 如果您的 TSA 伺服器需要驗證，請取消註解並填入帳號密碼
	// json.UpdateString("timestampToken.tsaUsername", "the_tsa_username")
	// json.UpdateString("timestampToken.tsaPassword", "the_tsa_password")

	// 要求 TSA 伺服器在其回應中包含其憑證
	// 這有助於 LTV 驗證過程也能包含時間戳記伺服器的憑證狀態
	json.UpdateBool("timestampToken.requestTsaCert", true)
	// --- TSA 時間戳記設定結束 ---

	// Define the appearance of the signature.
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "This is an LTV-enabled signature with a TSA timestamp.") // 更新顯示文字
	json.UpdateString("contactInfo", "0927556282")                                                    // 更新顯示文字

	// Load the signing certificate (使用設定檔中的路徑和密碼)
	cert := chilkat.NewCert()
	defer cert.DisposeCert() // 使用 defer 確保釋放
	success = cert.LoadPfxFile(pfxPath, pfxPassword)
	if !success {
		fmt.Println("載入 PFX 憑證失敗:", cert.LastErrorText())
		return
	}

	// Tell the pdf object to use the certificate for signing.
	success = pdf.SetSigningCert(cert)
	if !success {
		fmt.Println("設定簽名憑證失敗:", pdf.LastErrorText())
		return
	}

	// Sign the PDF (使用設定檔中的輸出路徑)
	success = pdf.SignPdf(json, pdfOutputPath)
	if !success {
		fmt.Println("簽署 PDF 失敗:", pdf.LastErrorText())
		return
	}

	fmt.Println("The PDF has been successfully cryptographically signed with TSA timestamp and long-term validation.")
	fmt.Printf("Signed PDF saved to: %s\n", pdfOutputPath)

	// pdf.DisposePdf() // defer 會處理
	// json.DisposeJsonObject() // defer 會處理
	// cert.DisposeCert() // defer 會處理
}
