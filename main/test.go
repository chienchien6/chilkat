package main

import (
	"chilkat"
	"fmt"
)

/*
#cgo CFLAGS: -IC:/Users/admin/chilkatsoft.com/chilkat-9.5.0-x64/include
#cgo LDFLAGS: -LC:/Users/admin/chilkatsoft.com/native_c_lib -lchilkatExt -lstdc++ -lws2_32
*/
import "C"

func main() {
	// 初始化 Chilkat Global 物件
	glob := chilkat.NewGlobal()
	defer glob.DisposeGlobal()

	// 解鎖 Chilkat 元件
	success := glob.UnlockBundle("Anything for 30-day trial")
	if !success {
		fmt.Println(glob.LastErrorText())
		return
	}

	// 檢查解鎖狀態
	status := glob.UnlockStatus()
	if status == 2 {
		fmt.Println("Unlocked using purchased unlock code.")
	} else {
		fmt.Println("Unlocked in trial mode.")
	}

	// 輸出解鎖詳細資訊
	fmt.Println(glob.LastErrorText())
}
