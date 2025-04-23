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
	viper.SetConfigName("config") // 設定檔名稱 (不含副檔名)
	viper.SetConfigType("json")   // 如果設定檔名不含副檔名，則必須指定類型
	// 指定設定檔的絕對路徑
	viper.AddConfigPath("C:/chilkatPackage/chilkattest")

	err := viper.ReadInConfig() // 尋找並讀取設定檔
	if err != nil {             // 處理讀取設定檔時的錯誤
		log.Fatalf("讀取設定檔時發生嚴重錯誤 (請確認 C:/chilkatPackage/chilkattest/config.json 存在且格式正確): %s \n", err)
	}

	// 存取設定值
	pdfInputPath := viper.GetString("pdf_input_path")
	pfxPath := viper.GetString("pfx_path")
	pfxPassword := viper.GetString("pfx_password")
	pdfOutputPath := viper.GetString("image_pdf_output_path")
	jpgImagePath := viper.GetString("jpg_image_path") // 新增：讀取 JPG 圖片路徑

	// 基本驗證 (包含新的 jpg_image_path)
	if pdfInputPath == "" || pfxPath == "" || pfxPassword == "" || pdfOutputPath == "" || jpgImagePath == "" {
		log.Fatalf("設定檔 config.json 缺少必要欄位 (pdf_input_path, pfx_path, pfx_password, pdf_output_path, jpg_image_path)\n")
	}
	// --- 設定載入結束 ---

	// Chilkat Global Unlock
	glob := chilkat.NewGlobal()
	glob.SetVerboseLogging(true) // <--- 在 Global 物件上啟用詳細日誌
	// ---> 加入這行來印出版本 <---
	fmt.Println("Chilkat Library Version:", glob.Version())

	success := glob.UnlockBundle("Anything for 30-day trial") // 請替換成您的有效解鎖碼
	if !success {
		log.Fatalf("Chilkat 解鎖失敗: %s\n", glob.LastErrorText())
	}

	pdf := chilkat.NewPdf()
	defer pdf.DisposePdf()      // 使用 defer 確保釋放
	pdf.SetVerboseLogging(true) // <--- 啟用詳細日誌紀錄

	// 載入要簽署的 PDF (使用設定檔中的路徑)
	success = pdf.LoadFile(pdfInputPath)
	if !success {
		fmt.Println("載入 PDF 失敗:", pdf.LastErrorText())
		return
	}

	// 簽署選項以 JSON 格式指定
	jsonOptions := chilkat.NewJsonObject()
	defer jsonOptions.DisposeJsonObject() // 使用 defer 確保釋放

	// 基本簽署選項
	jsonOptions.UpdateInt("signingCertificateV2", 1)
	jsonOptions.UpdateInt("signingTime", 1)
	jsonOptions.UpdateInt("page", 1)
	jsonOptions.UpdateString("appearance.y", "top")
	jsonOptions.UpdateString("appearance.x", "right")
	jsonOptions.UpdateString("appearance.fontScale", "10.0")

	// 簽名外觀文字行
	jsonOptions.UpdateString("appearance.text[0]", "Digitally signed by: cert_cn")
	jsonOptions.UpdateString("appearance.text[1]", "Date: current_dt")
	jsonOptions.UpdateString("appearance.text[2]", "The crazy brown fox jumps over the lazy dog.")

	// --- 載入並設定簽名圖片 ---
	jpgData := chilkat.NewBinData()
	defer jpgData.DisposeBinData() // 使用 defer 確保釋放

	// 從設定檔讀取的路徑載入 JPG 圖片
	success = jpgData.LoadFile(jpgImagePath)
	if !success {
		fmt.Printf("載入 JPG 圖片失敗 (%s): %s\n", jpgImagePath, glob.LastErrorText()) // 使用 glob 物件取得錯誤訊息
		return
	}

	// 將圖片資料設定給 PDF 物件
	success = pdf.SetSignatureJpeg(jpgData)
	fmt.Println("設定簽名圖片成功:", pdf.LastErrorText())
	if !success {
		fmt.Println("設定簽名圖片失敗:", pdf.LastErrorText())
		return
	}

	// 在 JSON 選項中指定圖片顯示方式
	jsonOptions.UpdateString("appearance.image", "signature") //custom-jpg//result-pass//signature//document-check//green-check-grey-circle
	// jsonOptions.UpdateString("appearance.width", "100")  // Width in points (adjust as needed)
	// jsonOptions.UpdateString("appearance.height", "200") // Height in points (adjust as needed)
	fmt.Println("設定簽名圖片成功2:", pdf.LastErrorText())
	jsonOptions.UpdateString("appearance.imagePlacement", "center") // 圖片置中
	jsonOptions.UpdateString("appearance.imageOpacity", "50")       // 設定 30% 透明度//0 表示完全透明，1 表示完全不透明//0.3 表示 30% 不透明度，即 70% 透明度
	// --- 圖片設定結束 ---

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

	// 簽署 PDF (使用設定檔中的輸出路徑)
	success = pdf.SignPdf(jsonOptions, pdfOutputPath)
	if !success {
		fmt.Println("簽署 PDF 失敗:", pdf.LastErrorText())
		return
	}

	fmt.Printf("PDF 已成功簽署 (包含圖片) 並儲存至 %s\n", pdfOutputPath)
}
