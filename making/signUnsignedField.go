package main

import (
	"chilkat"
	"fmt"
	"log" // 使用 log 進行嚴重錯誤記錄

	"github.com/spf13/viper" // 匯入 viper
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-10.1.3-x64/c_includes
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {

	// --- 使用 Viper 載入設定 ---
	viper.SetConfigName("config")                        // 設定檔名稱 (不含副檔名)
	viper.SetConfigType("json")                          // 如果設定檔名不含副檔名，則必須指定類型
	viper.AddConfigPath("C:/chilkatPackage/chilkattest") // 指定設定檔的絕對路徑

	err := viper.ReadInConfig() // 尋找並讀取設定檔
	if err != nil {             // 處理讀取設定檔時的錯誤
		log.Fatalf("讀取設定檔時發生嚴重錯誤 (請確認 C:/chilkatPackage/chilkattest/config.json 存在且格式正確): %s\n", err)
	}

	// 存取設定值
	// 注意：此處的 pdf_input_path 應指向一個 *包含* 未簽署簽名欄位的 PDF
	pdfInputPath := viper.GetString("pdf_input_path_unsigned")
	pfxPath := viper.GetString("pfx_path")
	pfxPassword := viper.GetString("pfx_password")
	// 新增：未簽署欄位名稱（可選，如果 PDF 只有一個未簽署欄位，則不需指定）
	unsignedFieldName := viper.GetString("unsigned_field_name")
	// 新增：簽署未簽署欄位後的輸出路徑
	pdfOutputPath := viper.GetString("unsigned_pdf_output_path")

	// 基本驗證
	if pdfInputPath == "" || pfxPath == "" || pfxPassword == "" || pdfOutputPath == "" {
		log.Fatalf("設定檔 config.json 缺少必要欄位 (pdf_input_path_unsigned, pfx_path, pfx_password, unsigned_pdf_output_path)\n")
	}
	// --- 設定載入結束 ---

	// Chilkat Global Unlock
	glob := chilkat.NewGlobal()
	glob.SetVerboseLogging(true)                            // 啟用詳細日誌 (可選)
	fmt.Println("Chilkat Library Version:", glob.Version()) // 印出版本

	success := glob.UnlockBundle("Anything for 30-day trial") // 請替換成您的有效解鎖碼
	if !success {
		log.Fatalf("Chilkat 解鎖失敗: %s\n", glob.LastErrorText())
	}

	pdf := chilkat.NewPdf()
	defer pdf.DisposePdf()      // 使用 defer 確保釋放
	pdf.SetVerboseLogging(true) // 啟用 PDF 物件的詳細日誌 (可選)

	// 載入包含未簽署簽名欄位的 PDF
	success = pdf.LoadFile(pdfInputPath)
	if !success {
		fmt.Println("載入 PDF 失敗:", pdf.LastErrorText())
		return
	}

	// 簽署選項以 JSON 格式指定
	jsonOptions := chilkat.NewJsonObject()
	defer jsonOptions.DisposeJsonObject() // 使用 defer 確保釋放

	// 基本簽署屬性
	jsonOptions.UpdateInt("signingCertificateV2", 1)
	jsonOptions.UpdateInt("signingTime", 1)

	// --- 關鍵設定：簽署現有未簽署欄位 ---
	// 不需要指定頁面 (page) 或位置 (x, y)，Chilkat 會自動尋找欄位。
	// 只需要設定 fillUnsignedSignatureField 為 true。
	jsonOptions.UpdateBool("appearance.fillUnsignedSignatureField", true)

	// (可選) 如果 PDF 中有多個未簽署欄位，且您知道要簽署哪一個，
	// 可以透過名稱指定 (從 config.json 讀取)
	if unsignedFieldName != "" {
		jsonOptions.UpdateString("unsignedSignatureField", unsignedFieldName)
		fmt.Printf("指定簽署欄位: %s\n", unsignedFieldName)
	} else {
		fmt.Println("未指定欄位名稱，將嘗試簽署第一個找到的未簽署欄位。")
	}

	// --- 外觀設定 (將自動縮放以符合欄位大小) ---
	// 簽名外觀文字行
	jsonOptions.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	jsonOptions.UpdateString("appearance.text[1]", "Date: current_dt")
	jsonOptions.UpdateString("appearance.text[2]", "The crazy brown fox jumps over the lazy dog.")

	// (可選) 加入內建圖示 (例如綠色勾勾)
	jsonOptions.UpdateString("appearance.image", "green-check-grey-circle")
	jsonOptions.UpdateString("appearance.imagePlacement", "left") // 圖示放在文字左側

	// 載入簽名憑證 (使用設定檔中的路徑和密碼)
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

	// 簽署 PDF，填充未簽署的簽名欄位
	success = pdf.SignPdf(jsonOptions, pdfOutputPath)
	if !success {
		fmt.Println("簽署 PDF (填充欄位) 失敗:", pdf.LastErrorText())
		return
	}

	fmt.Printf("PDF 已成功簽署 (填充欄位) 並儲存至 %s\n", pdfOutputPath)
}
