package test

import (
	"testing"
	"time"

	"github.com/youluo1230/adbutils"
)

var adb = adbutils.AdbClient{Host: "localhost", Port: 5037, SocketTime: 10}

func TestServerVersion(t *testing.T) {
	version := adb.ServerVersion()
	t.Logf("version: %d", version)
	time.Sleep(time.Second * 1000)
}

func TestConnect(t *testing.T) {
	adb := adbutils.NewAdb("localhost", 5037, 10)
	directory, _ := adb.Device(adbutils.SerialNTransportID{Serial: "a918b5a9"}).Sync().IterDirectory("/sdcard/222")
	for _, info := range *directory {
		println(info.Size)
	}
	//println(adb.Device(adbutils.SerialNTransportID{Serial: "a918b5a9"}).Sync().Stat("/data/local/tmp/output.txt").Mtime.String())
	//println(adb.Device(adbutils.SerialNTransportID{Serial: "a918b5a9"}).Sync().Push("1.txt", "/sdcard/222/1.txt", 0, true))
}
