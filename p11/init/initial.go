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
	pkcs11 := chilkat.NewPkcs11()

	// You'll need to know in advance the name and possibly the full path to the smart card vendor's shared library.
	// On Windows systems it is a .dll.   On Linux it is a .so.  On Mac OS X it will be a .dylib.
	// On Windows, if you set the SharedLibPath equal to just the name of the DLL, then it is assumed to be located in the Windows system directory
	// which contains dynamic-link libraries and drivers. The Windows system directory is typically C:\Windows\System32

	// In this example we will pass just the name of the DLL because it is located in C:\Windows\System32.
	// On non-Windows systems you should specify the full path to the shared lib.
	// Also use the full path on Windows systems where the smart card vendor's DLL does not install to C:\Windows\System32.
	pkcs11.SetSharedLibPath("C:\\OpenAPI GatewayRT\\Go\\lib\\V4.55.0.0\\Windows\\x86-64\\cs_pkcs11_R3.dll")

	successInitialize := pkcs11.Initialize()
	if successInitialize == false {
		fmt.Println(pkcs11.LastErrorText())
	} else {
		fmt.Println("PKCS11 successfully initialized.")
	}

	pkcs11.DisposePkcs11()

}
