package main

import (
	"net/http"
	"github.com/qiniu/log"
	"io"
	"os"
	"html/template"
	"io/ioutil"
	"path"
	"strings"
)

const (
	UPLOAD_DIR = "./uploads"
	TEMPLATE_DIR = "./views"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "Get" {
		if err := renderHtml(w, "upload", nil); err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			check(err)
		}

		return
	}

	if r.Method == "POST" {
		mf, mfh, err := r.FormFile("image")
		if err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			check(err)
		}
		filename := mfh.Filename
		defer mf.Close()

		t, err := os.Create(UPLOAD_DIR + "/" + filename)
		if err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			check(err)
		}
		defer t.Close()

		if _, err := io.Copy(t, mf); err != nil {
			//http.Error(w, err.Error(), http.StatusInternalServerError)
			check(err)
		}

		http.Redirect(w, r, "/views?id=" + filename, http.StatusFound)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId

	if exists := isExist(imagePath); !exists {
		http.NotFound(w, r)
		return
	}

	// todo 准确解析出文件的 MimeType ，并将其作为 Content-Type 进行输出
	// http.DetectContentType() or package mime
	f, err := ioutil.ReadFile(imagePath)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		check(err)
	}

	w.Header().Set("Content-Type", http.DetectContentType(f))
	http.ServeFile(w, r, imagePath)
}

func isExist(path string) (r bool) {
	_, err := os.Stat(path)
	if err != nil {
		r = os.IsExist(err)
	}

	r = true
	return
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	fileInfoArr, err := ioutil.ReadDir(UPLOAD_DIR)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		check(err)
	}

	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images

	if err := renderHtml(w, "list", locals); err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		check(err)
	}
}

// DRY (Don`t Repeat Yourself)
// 将模板渲染分离出来，单独编写一个处理函数，以便其他业务逻辑处理函数可以使用
func renderHtml(w http.ResponseWriter, temp string, locals map[string]interface{}) (err error) {
	err = templates[temp].Execute(w, locals)

	return
}

// 统一捕获 50x 系列的服务端错误
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// 绶存所有模板的内容
var templates = make(map[string]*template.Template)

func init() {
	fileInfoArr, err := ioutil.ReadDir(TEMPLATE_DIR)
	if err != nil {
		panic(err)
		return
	}

	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()

		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}

		templatePath = TEMPLATE_DIR + "/" + templateName
		log.Println("Loading template: ", templatePath)
		t := template.Must(template.ParseFiles(templatePath))
		templates[templateName[:strings.LastIndex(templateName, ".")]] = t
	}
}

func main() {
	
	http.HandleFunc("/", listHandler)
	http.HandleFunc("/views", viewHandler)
	http.HandleFunc("/upload", uploadHandler)

	log.Fatal(http.ListenAndServe(":3002", nil))
}
