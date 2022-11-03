package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	path := "/Users/keke/fsdownload/1667447238420163000/mains"
	//file, err := os.Stat(path)
	//if err == nil {
	//	fmt.Println("error", err.Error())
	//	//return
	//}
	////isnotexist来判断，是不是不存在的错误
	//if os.IsNotExist(err) { //如果返回的错误类型使用os.isNotExist()判断为true，说明文件或者文件夹不存在
	//	fmt.Println("文件不存在")
	//	//return
	//}
	//if !pExists {
	val := CreateDateDir(path)
	//}
	fmt.Println("val=============", val)
}

func CreateDateDir(Path string) string {
	folderName := time.Now().Format("20060102")
	folderPath := filepath.Join(Path, folderName)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		os.MkdirAll(folderPath, 0777)
		os.Chmod(folderPath, 0777)
	}
	return folderPath
}
