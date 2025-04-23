package main

import (
	"chilkat"
	"fmt"
	"log" // 使用 log 進行嚴重錯誤記錄

	"github.com/spf13/viper" // 匯入 viper
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {

	// --- 使用 Viper 載入設定 ---
	viper.SetConfigName("config") // 設定檔名稱 (不含副檔名)
	viper.SetConfigType("json")   // 如果設定檔名不含副檔名，則必須指定類型
	viper.AddConfigPath(".")      // 設定檔搜尋路徑 (目前目錄)
	viper.AddConfigPath("C:/chilkatPackage/chilkattest")
	// 您可以多次呼叫 viper.AddConfigPath() 來新增多個搜尋路徑
	// 例如：viper.AddConfigPath("$HOME/.yourapp")

	err := viper.ReadInConfig() // 尋找並讀取設定檔
	if err != nil {             // 處理讀取設定檔時的錯誤
		// 對於關鍵的啟動錯誤，使用 log.Fatalf
		log.Fatalf("讀取設定檔時發生嚴重錯誤: %s \n", err)
	}

	// 存取設定值
	pdfInputPath := viper.GetString("pdf_input_path")
	pfxPath := viper.GetString("pfx_path")
	pfxPassword := viper.GetString("pfx_password")
	pdfOutputPath := viper.GetString("pdf_output_path") // 認證後輸出的 PDF 路徑

	// 基本驗證
	if pdfInputPath == "" || pfxPath == "" || pfxPassword == "" || pdfOutputPath == "" {
		log.Fatalf("設定檔 config.json 缺少必要欄位 (pdf_input_path, pfx_path, pfx_password, pdf_output_path)\n")
	}
	// --- 設定載入結束 ---

	// Chilkat Global Unlock
	glob := chilkat.NewGlobal()
	success := glob.UnlockBundle("Anything for 30-day trial") // 請替換成您的有效解鎖碼
	if success != true {
		// 解鎖失敗是嚴重錯誤
		log.Fatalf("Chilkat 解鎖失敗: %s\n", glob.LastErrorText())
	}
	// 注意：glob 物件通常不需要明確 Dispose，除非您有特殊需求

	pdf := chilkat.NewPdf()
	defer pdf.DisposePdf() // 確保 pdf 物件在使用完畢後被釋放

	// 載入要認證和鎖定的 PDF (使用設定檔中的路徑)
	success = pdf.LoadFile(pdfInputPath)
	if !success {
		fmt.Println("載入 PDF 失敗:", pdf.LastErrorText()) // 對於操作中的非嚴重錯誤，使用 fmt.Println
		return
	}

	// 簽署選項以 JSON 格式指定
	jsonOptions := chilkat.NewJsonObject()
	defer jsonOptions.DisposeJsonObject() // 確保 jsonOptions 物件在使用完畢後被釋放

	// 大多數情況下，signingCertificateV2 和 signingTime 是必要的
	jsonOptions.UpdateInt("signingCertificateV2", 1)
	jsonOptions.UpdateInt("signingTime", 1)

	// --- 認證與鎖定 PDF 的關鍵設定 ---
	// 與普通的批准簽名相比，認證和鎖定 PDF 的唯一程式碼差異是
	// 在 JSON 中加入 "lockAfterSigning" 和 "docMDP.add"
	jsonOptions.UpdateBool("lockAfterSigning", true)
	jsonOptions.UpdateBool("docMDP.add", true)
	// --- 關鍵設定結束 ---

	// 將簽名放在第一頁，左上角
	jsonOptions.UpdateInt("page", 1)
	jsonOptions.UpdateString("appearance.y", "top")
	jsonOptions.UpdateString("appearance.x", "left")

	// 使用 10.0 的字型縮放比例
	jsonOptions.UpdateString("appearance.fontScale", "10.0")

	// 簽名外觀文字行
	// 您可以使用關鍵字 "cert_cn" (憑證通用名稱) 和 "current_dt" (目前日期時間)
	jsonOptions.UpdateString("appearance.text[0]", "Digitally certified by: cert_cn")
	jsonOptions.UpdateString("appearance.text[1]", "Date: current_dt")
	jsonOptions.UpdateString("appearance.text[2]", "Document is certified and locked.") // 自訂文字

	// 載入簽名憑證 (使用設定檔中的路徑和密碼)
	cert := chilkat.NewCert()
	defer cert.DisposeCert() // 確保 cert 物件在使用完畢後被釋放
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

	// 認證並儲存 PDF (使用設定檔中的輸出路徑)
	success = pdf.SignPdf(jsonOptions, pdfOutputPath)
	if !success {
		fmt.Println("認證與鎖定 PDF 失敗:", pdf.LastErrorText())
		return
	}

	fmt.Printf("PDF 已成功認證、鎖定並儲存至 %s\n", pdfOutputPath)
}
