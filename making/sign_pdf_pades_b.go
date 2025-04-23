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
	viper.SetConfigName("config")                        // 設定檔名稱 (不含副檔名)
	viper.SetConfigType("json")                          // 如果設定檔名不含副檔名，則必須指定類型
	viper.AddConfigPath("C:/chilkatPackage/chilkattest") // 注意: 修正了上一版本中 víper 的拼寫錯誤

	err := viper.ReadInConfig() // 尋找並讀取設定檔
	if err != nil {             // 處理讀取設定檔時的錯誤
		log.Fatalf("讀取設定檔時發生嚴重錯誤 (請確認 C:/chilkatPackage/chilkattest/config.json 存在且格式正確): %s \n", err)
	}

	// 存取設定值
	pdfInputPath := viper.GetString("pdf_input_path")
	pfxPath := viper.GetString("pfx_path")
	pfxPassword := viper.GetString("pfx_password")
	// 注意：config.json 目前沒有專門的 PAdES-B 輸出路徑
	// 建議在 config.json 中新增一個例如 "pades_b_pdf_output_path" 的欄位
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

	// ---------------------------------------------------------------------
	// PAdES (PDF Advanced Electronic Signatures) B-Level 相關設定
	// 這些屬性決定了產生的簽章類型，符合 PAdES B-Level 基本要求
	// - subFilter 指定使用 ETSI CADES 標準的 detached 簽章格式
	// - signingCertificateV2 和 signingTime 是常見的基礎屬性
	// - hashAlgorithm 指定雜湊演算法
	// 註: 要產生更高層級的 PAdES (如 LTV)，需要加入 ltvOcsp 或 timestampToken 等屬性
	// ---------------------------------------------------------------------
	json.UpdateString("subFilter", "/ETSI.CAdES.detached")
	json.UpdateBool("signingCertificateV2", true)
	// json.UpdateString("signingAlgorithm", "pkcs") // 通常不需要特別指定，Chilkat 會自動處理
	json.UpdateString("hashAlgorithm", "sha256")
	json.UpdateInt("signingTime", 1) // 加入簽署時間

	// -----------------------------------------------------------
	// 設定簽章外觀
	json.UpdateInt("page", 1)
	json.UpdateString("appearance.y", "top")
	json.UpdateString("appearance.x", "left")
	json.UpdateString("appearance.fontScale", "10.0")
	json.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	json.UpdateString("appearance.text[1]", "current_dt")
	json.UpdateString("appearance.text[2]", "PAdES B-Level Signature") // 更新顯示文字

	// --------------------------------------------------------------
	// 載入簽章憑證 (使用設定檔中的路徑和密碼)
	cert := chilkat.NewCert()
	defer cert.DisposeCert() // 使用 defer 確保釋放
	success = cert.LoadPfxFile(pfxPath, pfxPassword)
	if !success {
		fmt.Println("載入 PFX 憑證失敗:", cert.LastErrorText())
		return
	}

	// 告知 pdf 物件使用此憑證進行簽署
	success = pdf.SetSigningCert(cert)
	if !success {
		fmt.Println("設定簽名憑證失敗:", pdf.LastErrorText())
		return
	}

	// 簽署 PDF，建立輸出檔案 (使用設定檔中的輸出路徑)
	success = pdf.SignPdf(json, pdfOutputPath)
	if !success {
		fmt.Println("簽署 PDF 失敗:", pdf.LastErrorText())
		return
	}

	fmt.Println("PDF 已成功簽署 (PAdES B-Level).")
	fmt.Printf("Signed PDF saved to: %s\n", pdfOutputPath)

	// pdf.DisposePdf() // defer 會處理
	// json.DisposeJsonObject() // defer 會處理
	// cert.DisposeCert() // defer 會處理
}
