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
	println(adb.Device(adbutils.SerialNTransportID{Serial: "a918b5a9"}).Sync().Push("D:\\软件\\bd.apk", "/data/local/tmp/bd.apk", 0755, true))
}
