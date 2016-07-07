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
	"runtime/debug"
)

const (
	LIST_DIR = 0x0001
	UPLOAD_DIR = "./uploads"
	TEMPLATE_DIR = "./views"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "Get" {
		log.Debug("Get: upload image ")
		renderHtml(w, "upload", nil)
	}

	if r.Method == "POST" {
		mf, mfh, err := r.FormFile("image")
		check(err)

		filename := mfh.Filename
		defer mf.Close()

		t, os_err := os.Create(UPLOAD_DIR + "/" + filename)
		check(os_err)
		defer t.Close()

		_, io_err := io.Copy(t, mf)
		check(io_err)

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
	check(err)

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
	check(err)

	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}

	locals["images"] = images
	renderHtml(w, "list", locals)
}

// DRY (Don`t Repeat Yourself)
// 将模板渲染分离出来，单独编写一个处理函数，以便其他业务逻辑处理函数可以使用
func renderHtml(w http.ResponseWriter, temp string, locals map[string]interface{}) {
	err := templates[temp].Execute(w, locals)
	check(err)
}

// 统一捕获 50x 系列的服务端错误
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// 使用闭包避免程序运行时出错崩溃
func safeHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				//http.Error(w, e.Error(), http.StatusInternalServerError)
				// or output 自定义的 50x error html
				w.WriteHeader(http.StatusInternalServerError)
				errors := make(map[string]interface{})
				errors["errors"] = e
				renderHtml(w, "error", errors)
				log.Warnf("%v - %v", fn, e)
				log.Debug(string(debug.Stack()))
			}
		}()

		log.Debugf("safe handler, use function %+v", fn)
		fn(w, r)
	}
}

// prefix -> staticDir
func staticHandler(mux *http.ServeMux, prefix string, staticDir string, flags int) {
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		file := staticDir + r.URL.Path[len(prefix) -1:]
		if (flags & LIST_DIR) == 0 {
			if exists := isExist(file); !exists {
				http.NotFound(w, r)
				return
			}
		}

		http.ServeFile(w, r, file)
	})
}

// 绶存所有模板的内容
var templates = make(map[string]*template.Template)

func init() {
	log.Println("set log output level to ", log.Ldebug)
	log.SetOutputLevel(log.Ldebug)

	fileInfoArr, err := ioutil.ReadDir(TEMPLATE_DIR)
	check(err)

	var templateName, templatePath string
	for _, fileInfo := range fileInfoArr {
		templateName = fileInfo.Name()

		if ext := path.Ext(templateName); ext != ".html" {
			continue
		}

		templatePath = TEMPLATE_DIR + "/" + templateName
		log.Println("Loading template: ", templatePath)
		t := template.Must(template.ParseFiles(templatePath))
		log.Println("Loading template name: ", templateName[:strings.LastIndex(templateName, ".")])
		templates[templateName[:strings.LastIndex(templateName, ".")]] = t
	}
}

func main() {
	// 静态资源与动态请求的分离
	mux := http.NewServeMux();

	// Static resources
	staticHandler(mux, "/bee/", "./public", 0)

	// Dynamic request
	mux.HandleFunc("/", safeHandler(listHandler))
	mux.HandleFunc("/views", safeHandler(viewHandler))
	mux.HandleFunc("/upload", safeHandler(uploadHandler))

	log.Fatal(http.ListenAndServe(":3002", mux))
}
