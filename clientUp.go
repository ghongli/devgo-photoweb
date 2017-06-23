/**
client 模拟文件上传
客户端通过 multipart.Write 把文件的文本流写入一个缓存中
调用 http.post 把缓存传到服务器
*/
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
)

func postFile(fileName, targetUrl string) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("uploadFile", fileName)
	if err != nil {
		fmt.Println("error writing to buffer")
		return err
	}

	// 打开文件句柄操作
	fh, err := os.Open(fileName)
	if err != nil {
		fmt.Println("error opening file")
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetUrl, contentType, bodyBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(resp.Status, resp.StatusCode)
	fmt.Println(string(respBody))

	return nil
}

func main() {
	targetUrl := "http://localhost:3002/upload"
	fileName := "./tmp/sheep.png"
	fmt.Println(postFile(fileName, targetUrl))
}
