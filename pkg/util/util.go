package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
)

var byteUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

func IsNil(i interface{}) bool {
	defer func() {
		recover()
	}()
	vi := reflect.ValueOf(i)
	return vi.IsNil()
}

func HommizationSize(size int64) string {
	var sizeStr string
	unit := "B"
	units := []string{"K", "M", "G", "T", "P"}
	baseV := int64(1024)
	sizeWap := size
	for i := 0; sizeWap > baseV && len(units) > i; i++ {
		sizeWap = sizeWap / baseV
		unit = units[i]
	}
	sizeStr = fmt.Sprintf("%.2d%s", sizeWap, unit)
	return sizeStr
}

// PathExists 判断文件夹是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// 判断是否文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func GenUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return ""
	}

	uuid := fmt.Sprintf("%X%X%X%X%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}
func GenMd5(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

// 获取最后一级目录名
func GetLastDir(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}

// 获取最后一级目录之前的路径
func GetDir(path string) string {
	if strings.LastIndex(path, "/") == -1 {
		return path
	}
	return path[:strings.LastIndex(path, "/")]
}

// 替换最后一级目录之前的路径为空
func ReplaceDir(path string) string {
	return strings.Replace(path, GetDir(path), "", 1)
}

func ExistFile(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return errors.Is(err, fs.ErrExist)
	}
	return true
}
func byteUnitStr(n int64) string {
	var unit string
	size := float64(n)
	for i := 1; i < len(byteUnits); i++ {
		if size < 1000 {
			unit = byteUnits[i-1]
			break
		}

		size = size / 1000
	}

	return fmt.Sprintf("%.3g %s", size, unit)
}

func DrawTextFormatBytes(progress, total int64) string {
	return fmt.Sprintf("%s/%s", byteUnitStr(progress), byteUnitStr(total))
}
