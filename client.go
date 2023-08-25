package adbutils

import (
	"bytes"
	"errors"
	"fmt"
	_ "github.com/youluo1230/adbutils/binaries"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	OKAY            = "OKAY"
	FAIL            = "FAIL"
	DENT            = "DENT"
	DONE            = "DONE"
	DATA            = "DATA"
	TCP             = "tcp"
	UNIX            = "unix"
	DEV             = "dev"
	LOCAL           = "local"
	LOCALRESERVED   = "localreserved"
	LOCALFILESYSTEM = "localfilesystem"
	LOCALABSTRACT   = "localabstract"
	Windows         = "windows"
	Mac             = "darwin"
	Linux           = "linux"
	macAdbURL       = "https://cdn.mongona.com/mac/adb"
	linuxAdbURL     = "https://cdn.mongona.com/linux/adb"
	WinAdbURL       = "https://cdn.mongona.com/win"
)

func checkServer(host string, port int) bool {
	_, err := net.Dial("tcp", fmt.Sprintf("%v:%v", host, port))
	return err == nil
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func getParentDirectory(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, "/"))
}
func getCurrentFile() string {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic(errors.New("Can not get current file info"))
	}
	return getParentDirectory(file)
}

func GetFreePort() int {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	defer conn.Close()
	if err != nil {
		log.Println("getFreePort error! ", err.Error())
		return 0
	}
	ipPort := strings.Split(conn.Addr().String(), ":")
	port, _ := strconv.Atoi(ipPort[len(ipPort)-1])
	return port
}

// AdbConnection region AdbConnection

type AdbConnection struct {
	Host string
	Port int
	Conn net.Conn
}

func (adbConnection AdbConnection) safeConnect(t time.Duration) (*net.Conn, error) {
	conn, err := adbConnection.createSocket(t)
	if err != nil {
		switch reflect.TypeOf(err) {
		case reflect.TypeOf(&net.OpError{}):
			cmd := exec.Command(AdbPath(), "start-server")
			err = cmd.Start()
			if err != nil {
				return nil, err
			}
			err = cmd.Wait()
			if err != nil {
				return nil, err
			}
			conn, err = adbConnection.createSocket(t)
			if err != nil {
				return nil, err
			}
			return conn, nil
		default:
			return nil, err
		}
	}
	return conn, nil
}

func (adbConnection AdbConnection) SetTimeout(timeOut time.Duration) error {
	if timeOut != 0 {
		var err error
		err = adbConnection.Conn.SetDeadline(time.Now().Add(time.Second * timeOut))
		if err != nil {
			panic(err.Error())
			return err
		}
	}
	return nil
}

func (adbConnection AdbConnection) createSocket(t time.Duration) (*net.Conn, error) {
	dialer := net.Dialer{
		Timeout: t,
	}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%v:%d", adbConnection.Host, adbConnection.Port))
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

func (adbConnection AdbConnection) Close() {
	err := adbConnection.Conn.Close()
	if err != nil {
		return
	}
}

func (adbConnection AdbConnection) Read(n int) []byte {
	return adbConnection.readFully(n)
}

func (adbConnection AdbConnection) readFully(n int) []byte {
	t := 0
	buffer := make([]byte, n)
	result := bytes.NewBuffer(nil)
	for t < n {
		length, err := adbConnection.Conn.Read(buffer[t:n])
		result.Write(buffer[:length])
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		if length == 0 {
			break
		}
		t += length
	}
	return result.Bytes()
}

func (adbConnection AdbConnection) SendCommand(cmd string) {
	msg := fmt.Sprintf("%04x%s", len(cmd), cmd)
	_, err := adbConnection.Conn.Write([]byte(msg))
	if err != nil {
		log.Println("write error!", err.Error())
		return
	}
}

func (adbConnection AdbConnection) ReadString(n int) string {
	res := adbConnection.Read(n)
	return strings.TrimSpace(string(res))
}
func (adbConnection AdbConnection) ReadStringBlock() string {
	str := adbConnection.ReadString(4)
	size, _ := strconv.ParseUint(str, 16, 32)
	return adbConnection.ReadString(int(size))
}

func (adbConnection AdbConnection) ReadUntilClose() string {
	buf := []byte{}
	for {
		chunk := adbConnection.Read(4096)
		if len(chunk) == 0 {
			break
		}

		buf = append(buf, chunk...)
	}
	return string(buf)
}

func (adbConnection AdbConnection) CheckOkay() bool {
	data := adbConnection.ReadString(4)
	if data == OKAY {
		return true
	}
	return false
}

// end region AdbConnection

// AdbClient region AdbClient
type AdbClient struct {
	Host       string
	Port       int
	SocketTime time.Duration
}

func downloadFile(url string, localPath string, wg *sync.WaitGroup) error {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()
	var (
		buf     = make([]byte, 32*1024)
		written int64
	)
	tmpFilePath := localPath + ".download"
	client := new(http.Client)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	file, err := os.Create(tmpFilePath)
	defer file.Close()
	if err != nil {
		return err
	}
	fileSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		log.Println(err.Error())
	}
	defer resp.Body.Close()
	if resp.Body == nil {
		return errors.New("body is null")
	}
	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := file.Write(buf[0:nr])
			log.Println(fmt.Sprintf("Download %v Done:%dKb, Total:%dKb, Process:%.2f", url, written/1024, fileSize, float32(written)/float32(fileSize)))
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	if err == nil {
		err = os.Rename(tmpFilePath, localPath)
	}
	return err
}

func createDir(path string) bool {
	_exist, _err := pathExists(path)
	if _err != nil {
		return true
	}
	if _exist {
		return true
	} else {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return true
		}
	}
	return false
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func AdbPath() string {
	// so ugly
	currentPath := getCurrentFile()
	platform := runtime.GOOS
	adbPath := ""
	subPath := "mac"
	url := macAdbURL
	if platform == Linux {
		subPath = Linux
		url = linuxAdbURL
	} else if platform == Windows {
		subPath = "win"
		url = WinAdbURL
	}
	dir, _ := filepath.Abs(path.Join(currentPath, "binaries", subPath))
	exist, err := pathExists(dir)
	if err != nil {
		return ""
	}
	if !exist {
		createDir(dir)
	}
	adbPath, _ = filepath.Abs(path.Join(dir, "adb"))
	if platform == Windows {
		adbPath, _ = filepath.Abs(path.Join(dir, "adb.exe"))
	}
	exist, _ = pathExists(adbPath)
	if !exist {
		if platform == Windows {
			AdbWinApiPath, _ := filepath.Abs(path.Join(dir, "AdbWinApi.dll"))
			AdbWinUsbApiPath, _ := filepath.Abs(path.Join(dir, "AdbWinUsbApi.dll"))
			wg := &sync.WaitGroup{}
			wg.Add(3)
			err = downloadFile(url+"/adb.exe", adbPath, wg)
			err = downloadFile(url+"/AdbWinApi.dll", AdbWinApiPath, wg)
			err = downloadFile(url+"/AdbWinUsbApi.dll", AdbWinUsbApiPath, wg)
			wg.Wait()
			if err != nil {
			}
		} else {
			err = downloadFile(url, adbPath, nil)
			if err != nil {
			}
			_ = os.Chmod(adbPath, 0777)
		}
	}
	return adbPath
}

func (adb *AdbClient) connect() (*AdbConnection, error) {
	adbConnection := &AdbConnection{
		Host: adb.Host,
		Port: adb.Port,
	}
	conn, err := adbConnection.safeConnect(adb.SocketTime)
	if err != nil {
		return nil, err
	}
	adbConnection.Conn = *conn
	return adbConnection, nil

}

func (adb *AdbClient) ServerVersion() (int, error) {
	c, err := adb.connect()
	if err != nil {
		return 0, err
	}
	defer c.Close()
	c.SendCommand("host:version")
	c.CheckOkay()
	res := c.ReadStringBlock()
	l, _ := strconv.Atoi(res)
	return l + 16, nil
}

func (adb *AdbClient) ServerKill() error {
	if checkServer(adb.Host, adb.Port) {
		c, err := adb.connect()
		if err != nil {
			return err
		}
		defer c.Close()
		c.SendCommand("host:kill")
		c.CheckOkay()
	}
	return nil
}

func (adb *AdbClient) WaitFor() {
	// pass
}

func (adb *AdbClient) Connect(addr string) bool {
	//addr (str): adb remote address [eg: 191.168.0.1:5555]
	c, err := adb.connect()
	if err != nil {
		return false
	}
	defer c.Close()
	c.SendCommand("host:connect:" + addr)
	c.CheckOkay()
	if strings.Index(c.ReadStringBlock(), "由于") > -1 {
		return false
	}
	return true
}

func (adb *AdbClient) Disconnect(addr string, raiseErr bool) bool {
	//addr (str): adb remote address [eg: 191.168.0.1:5555]
	c, err := adb.connect()
	if err != nil {
		return false
	}
	defer c.Close()
	c.SendCommand("host:disconnect:" + addr)
	c.CheckOkay()
	c.ReadStringBlock()
	return true
}

type SerialNTransportID struct {
	Serial      string
	TransportID int
}

func (adb *AdbClient) Shell(serial string, command string, stream bool) interface{} {
	snNtid := SerialNTransportID{Serial: serial}
	return adb.Device(snNtid).Shell(command, stream, adb.SocketTime)
}

func (adb *AdbClient) DeviceList() []AdbDevice {
	var res []AdbDevice
	c, err := adb.connect()
	if err != nil {
		return res
	}
	defer c.Close()
	c.SendCommand("host:devices-l")
	c.CheckOkay()
	outPut := c.ReadStringBlock()
	re := regexp.MustCompile(`(\S+)\s+device\s+product:(\S+)\s+model:(\S+)\s+device:(\S+)\s+transport_id:(\d)`)
	outPuts := re.FindAllStringSubmatch(outPut, -1)
	for _, line := range outPuts {
		if len(line) != 6 {
			continue
		}
		id, _ := strconv.Atoi(line[5])
		res = append(res, AdbDevice{ShellMixin{Client: adb, Serial: line[1], TransportID: id, Model: line[3], DeviceType: line[4]}})

	}
	return res
}

func (adb *AdbClient) Device(snNtid SerialNTransportID) AdbDevice {
	if snNtid.Serial != "" || snNtid.TransportID != 0 {
		return AdbDevice{ShellMixin{Client: adb, Serial: snNtid.Serial, TransportID: snNtid.TransportID}}
	}
	serial := os.Getenv("ANDROID_SERIAL")
	if serial != "" {
		ds := adb.DeviceList()
		if len(ds) > 0 {
			return ds[0]
		}
	}
	return AdbDevice{ShellMixin{Client: adb, Serial: snNtid.Serial, TransportID: snNtid.TransportID}}
}

func NewAdb(host string, port int, timeOut time.Duration) *AdbClient {
	adb := &AdbClient{Host: host, Port: port, SocketTime: time.Second * timeOut}
	return adb
}

// end region AdbClient
