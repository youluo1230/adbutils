package test

import (
	"testing"
	"time"

	"github.com/youluo1230/adbutils"
)

var adb = adbutils.AdbClient{Host: "localhost", Port: 5037, SocketTime: 10}

func TestServerVersion(t *testing.T) {
	serverVersion, _ := adb.ServerVersion()
	t.Logf("version: %d", serverVersion)
	time.Sleep(time.Second * 1000)
}

func TestConnect(t *testing.T) {
	adb := adbutils.NewAdb("localhost", 5037, time.Second*100)
	//println(adb.DeviceList()[0].Sync())
	//println(adbutils.AdbPath())
	//println(adb.Device(adbutils.SerialNTransportID{Serial: "a918b5a9"}).Sync().Stat("/data/local/tmp/output.txt").Mtime.String())
	pull, err := adb.Device(adbutils.SerialNTransportID{Serial: "a918b5a9"}).Sync().Pull("/sdcard/2232/aaa.apk", "bd.apk")
	if err != nil {
		println(err.Error())
	}
	println(pull)
}
