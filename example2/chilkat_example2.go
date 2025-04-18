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

// ProgressInfo callback
func myProgressInfo(name, value string, userData interface{}) {
	fmt.Println("(", userData.(string), ") ", name, ": ", value)
}

// PercentDone callback
// Return true to abort the current operation, return false to continue.
func myPercentDone(pctDone int, userData interface{}) bool {
	fmt.Println("(", userData.(string), ") ", pctDone, "% done")
	// returning true will cause the current operation to abort.
	return false
}

// AbortCheck callback
// Return true to abort the current operation, return false to continue.
// Called periodically according to the Chilkat object's HeartbeatMs property setting.
// A PercentDone callback counts as an AbortCheck, so if PercentDone callbacks
// happen frequently enough, you may not see any AbortCheck callbacks.
func myAbortCheck(userData interface{}) bool {
	fmt.Println("(", userData.(string), ") AbortCheck")
	// returning true will cause the current operation to abort.
	return false
}

// Unlock all of Chilkat
func unlockChilkat() bool {
	glob := chilkat.NewGlobal()
	success := glob.UnlockBundle("30-day trial")
	if success != true {
		fmt.Println(glob.LastErrorText)
		return false
	}
	fmt.Println("---- Chilkat unlocked for trial.")
	return success
}

// Create a zip archive with callbacks
func zipWithCallbacks() bool {

	zip := chilkat.NewZip()

	// Setup callbacks
	// The callback methods are the same for all Chilkat classes that have callbacks.
	// Any class that has network communications such as Ssh, Ftp2, Imap, Socket, etc. will definitely have callbacks.
	// The PercentDone callback only fires when it's possible to know the percentage completion for a particular method call.
	zip.SetCallbackUserData("user_data")
	zip.SetProgressInfo(myProgressInfo)
	zip.SetPercentDone(myPercentDone)
	//zip.SetAbortCheck(myAbortCheck)

	success := zip.NewZip("output/test.zip")
	success = zip.AppendFiles("data", true)
	if success != true {
		fmt.Println(zip.LastErrorText)
		return false
	}
	fmt.Println("--")

	// Take a peek at an entry..
	entry := zip.GetEntryByName("starfish.jpg")
	fmt.Println("starfish.jpg entry type: ", entry.EntryType())
	fmt.Println("entry filename: ", entry.FileName())
	entry.DisposeZipEntry()

	success = zip.WriteZipAndClose()
	if success != true {
		fmt.Println(zip.LastErrorText)
		return false
	}

	zip.DisposeZip()
	fmt.Println("---- zipWithCallbacks successful.")
	return true
}

// Passing and returning object.
// Demonstrates methods that return objects, and methods that have object(s) in the argument list.
// Also demonstrate passing and returning byte arrays.
func passAndReturnObjects() bool {

	zip := chilkat.NewZip()

	// Open the zip we previously created.
	success := zip.OpenZip("output/test.zip")
	if success != true {
		fmt.Println(zip.LastErrorText)
		return false
	}

	// Get the "starfish.jpg" entry within the zip
	entry := zip.GetEntryByName("starfish.jpg")
	if entry == nil {
		fmt.Println("starfish.jpg entry not found in the zip archive")
		return false
	}

	// Note: JPG files are not compressed by default...
	fmt.Println("starfish.jpg contains ", entry.UncompressedLength(), " bytes")

	// Get the uncompressed bytes of the file
	jpgBytes := entry.Inflate()
	fmt.Println("jpbBytes contains ", len(jpgBytes), " bytes")

	entry.DisposeZipEntry()

	// To demonstrate passing an object as an argument, and also
	// passing a byte array, we'll create some new entries in the zip
	// using this byte data.
	entry = zip.AppendData("starfish2.jpg", jpgBytes)
	fmt.Println("starfish2.jpg contains ", entry.UncompressedLength(), " bytes")
	entry.DisposeZipEntry()

	// To pass an object, we'll create a new BinData, load it with the JPG bytes, and then
	// add a Zip entry using the BinData.
	binData := chilkat.NewBinData()
	success = binData.AppendBinary(jpgBytes)
	entry = zip.AppendBd("starfish3.jpg", binData)
	fmt.Println("starfish3.jpg contains ", entry.UncompressedLength(), " bytes")
	entry.DisposeZipEntry()

	// Write the new zip which contains the original files plus the new ones..
	zip.SetFileName("output/test2.zip")
	success = zip.WriteZipAndClose()
	if success != true {
		fmt.Println(zip.LastErrorText())
		return false
	}

	zip.DisposeZip()
	fmt.Println("---- passAndReturnObjects successful.")
	return true
}

func testAsync() bool {

	http := chilkat.NewHttp()

	// Channels are the pipes that connect concurrent goroutines.
	// You can send values into channels from one goroutine and receive those values into another goroutine.
	// Create a new channel with make(chan val-type). Channels are typed by the values they convey.
	// This creates a channel of type chilkat.Task.
	c := make(chan *chilkat.Task)

	// Call QuickGetStr in a concurrent goroutine.
	// The task object is sent to the channel when it begins.
	// When the QuickGetStr is finished, the channel is closed.
	//
	go http.QuickGetStrAsync("https://www.chilkatsoft.com/", c)

	// Get the chilkat.Task object.
	task := <-c

	// Wait for the task to complete.
	// ok is false if there are no more values to receive and the channel is closed.
	_, ok := <-c

	fmt.Println("ok = ", ok)
	fmt.Println("task = ", task)
	fmt.Println(task.ResultErrorText())
	task.DisposeTask()
	fmt.Println("---- testAsync successful.")

	return true
}

func main() {

	success := unlockChilkat()
	if success != true {
		return
	}

	success = zipWithCallbacks()
	if success != true {
		return
	}

	success = passAndReturnObjects()
	if success != true {
		return
	}

	success = testAsync()
	if success != true {
		return
	}

}
