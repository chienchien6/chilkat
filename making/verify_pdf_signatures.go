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
	// *** 重要: 請確保此路徑指向您想要驗證的 PDF 檔案 ***
	pdfToVerifyPath := viper.GetString("signed_hsm_pdf_output_path") // 讀取要驗證的 PDF 路徑 (請根據需要修改此鍵名)

	// 基本驗證
	if pdfToVerifyPath == "" {
		log.Fatalf("設定檔 config.json 缺少必要欄位 (例如 pdf_output_path 或您指定的鍵名)\n")
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

	// Load the PDF file to be verified (使用設定檔中的路徑)
	success = pdf.LoadFile(pdfToVerifyPath)
	if !success {
		fmt.Printf("載入待驗證的 PDF (%s) 失敗: %s\n", pdfToVerifyPath, pdf.LastErrorText())
		return
	}

	fmt.Printf("Verifying signatures in: %s\n", pdfToVerifyPath)

	// 取得 PDF 中的簽章數量
	numSignatures := pdf.NumSignatures()
	if numSignatures < 0 {
		fmt.Println("無法取得簽章數量:", pdf.LastErrorText())
		return
	}

	if numSignatures == 0 {
		fmt.Println("指定的 PDF 文件中沒有任何簽章。")
		return
	}

	fmt.Printf("發現 %d 個簽章。\n", numSignatures)

	// 建立一個 JsonObject 來重複接收每次驗證的詳細資訊
	// 每次呼叫 VerifySignature 都會覆寫其內容
	sigInfo := chilkat.NewJsonObject()
	defer sigInfo.DisposeJsonObject() // 確保 sigInfo 在 main 函數結束時被釋放
	sigInfo.SetEmitCompact(false)     // 設定為輸出易讀的 JSON 格式

	// 迭代驗證每一個簽章
	for i := 0; i < numSignatures; i++ {
		fmt.Printf("--- 驗證簽章索引 %d ---\n", i)

		// 驗證指定索引的簽章，並將詳細結果存入 sigInfo
		verified := pdf.VerifySignature(i, sigInfo)

		if verified {
			fmt.Printf("簽章 %d: 有效\n", i)
		} else {
			fmt.Printf("簽章 %d: 無效\n", i)
			// 無效時，LastErrorText 通常包含主要原因
			fmt.Printf("LastErrorText (簽章 %d):\n%s\n", i, pdf.LastErrorText())
		}

		// 輸出 sigInfo 的 JSON 內容，包含詳細的驗證資訊
		// 使用 * 取消對 Emit() 返回的字串指標的引用
		fmt.Printf("驗證詳情 JSON (簽章 %d):\n%s\n", i, *sigInfo.Emit())

		// (可選) 獲取簽署者資訊
		// signerCert := pdf.GetSignerCert(i)
		// if signerCert != nil {
		// 	fmt.Printf("簽署者 (簽章 %d): %s\n", i, signerCert.SubjectCN())
		// 	signerCert.DisposeCert() // 釋放 GetSignerCert 返回的 Cert 物件
		// }

		fmt.Println("---")
	}

	fmt.Println("簽章驗證完成。")

	// pdf.DisposePdf() // defer 會處理
}
