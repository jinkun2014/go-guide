package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type Guide struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Color   string `json:"color"`
	Link    string `json:"link"`
	Time    string `json:"time"`
}

type Tab struct {
	Title  string  `json:"title"`
	Guides []Guide `json:"guides"`
}

type Data struct {
	Title string `json:"title"`
	Tabs  []Tab  `json:"tabs"`
}

var token string
var dataName string
var password string
var h bool
var title string
var port int

func init() {
	flag.BoolVar(&h, "h", false, "帮助")
	flag.StringVar(&dataName, "f", "data.json", "设置数据存储文件名")
	flag.StringVar(&title, "t", "Go导航", "设置标题")
	flag.StringVar(&password, "pwd", "123456", "设置登录密码")
	flag.IntVar(&port, "p", 8080, "设置端口")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintln(os.Stderr, "Guide version: Guide/1.0.0")
	fmt.Fprintln(os.Stderr, "Usage: Guide [-h 帮助] [-f 设置数据存储文件名] [-t 设置标题] [-pwd 设置登录密码] [-p 设置端口]")
	fmt.Fprintln(os.Stderr, "启动成功后,打开浏览器访问端口即可，例如:http://127.0.0.1:8080")
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if h {
		flag.Usage()
		return
	}

	// 初始化数据文件
	data := readJsonFromFile(dataName)
	data.Title = title
	writeJsonToFile(dataName, data)

	// 初始化Gin
	r := gin.Default()
	// 跨域
	r.Use(Cors())

	// 加载模板
	t, err := loadTemplate()
	if err != nil {
		panic(err)
	}
	r.SetHTMLTemplate(t)
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "/templates/index.html", gin.H{
			"Title": title,
		})
	})
	r.GET("/static/*file", func(c *gin.Context) {
		file, err := Assets.Open("/templates/static/" + c.Param("file"))
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Data(http.StatusOK, "", content)
	})

	// 首页
	r.GET("/home", home)
	// 登录
	r.GET("/login/:password", login)
	// 保存
	r.POST("/save/:token", save)
	// 启动
	r.Run(fmt.Sprintf(":%d", port))
}

// 静态文件
func loadTemplate() (*template.Template, error) {
	t := template.New("")
	for name, file := range Assets.Files {
		if file.IsDir() || !strings.HasSuffix(name, ".html") {
			continue
		}
		h, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		t, err = t.New(name).Parse(string(h))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// 跨域
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		// 放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// 处理请求
		c.Next()
	}
}

// 首页
func home(c *gin.Context) {
	c.JSON(http.StatusOK, Result{http.StatusOK, "成功!", readJsonFromFile(dataName)})
}

// 登录
func login(c *gin.Context) {
	p := c.Param("password")
	if p != password {
		c.JSON(http.StatusOK, Result{http.StatusUnauthorized, "密码错误!", nil})
		return
	}

	data := make(map[string]interface{})
	token = RandString(10)
	data["token"] = token
	c.JSON(http.StatusOK, Result{http.StatusOK, "成功!", data})
}

// 保存
func save(c *gin.Context) {
	t := c.Param("token")
	if token == "" || token != t {
		c.JSON(http.StatusOK, Result{http.StatusUnauthorized, "登录失效,请重新登录!", ""})
		return
	}

	var data Data
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusOK, Result{http.StatusBadRequest, "请求参数错误!", ""})
		return
	}
	writeJsonToFile(dataName, data)
	c.JSON(http.StatusOK, Result{http.StatusOK, "成功!", data})
}

// 判断文件或文件夹是否存在
func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println(err)
		return false
	}
	return true
}

// 读取数据文件
func readJsonFromFile(filename string) Data {

	if !isExist(filename) {
		return Data{}
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Open file failed [Err:%s]\n", err.Error())
		return Data{}
	}
	defer file.Close()

	var data Data

	// 创建json解码器
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		fmt.Println("Decoder failed", err.Error())
		return Data{}
	}
	return data
}

// 写入数据文件
func writeJsonToFile(filename string, data Data) {

	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Create file failed", err.Error())
		return
	}
	defer file.Close()

	// 创建Json编码器
	encoder := json.NewEncoder(file)

	err = encoder.Encode(data)
	if err != nil {
		fmt.Println("Encoder failed", err.Error())
	} else {
		fmt.Println("Encoder success")
	}
}

// 随机字符串
func RandString(len int) string {
	base := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano() / 1e6))
	result := ""
	for i := 0; i < len; i++ {
		start := r.Intn(46)
		result = result + base[start:start+1]
	}
	return result
}
