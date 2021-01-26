package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/disintegration/imaging"
	"gopkg.in/gcfg.v1"
)

type config struct {
	Section struct {
		Port    string
		Timeout string
	}
}

// 自定义返回
type JsonRes struct {
	Status    int         `json:"status"`
	Data      interface{} `json:"data"`
	Msg       string      `json:"msg"`
	TimeStamp int64       `json:"timestmap"`
}

var listenAddr string

// 图片信息
type SliderInfo struct {
	BacW    int    `json:"BacW"`
	BacH    int    `json:"BacH"`
	SliderW int    `json:"SliderW"`
	SliderH int    `json:"SliderH"`
	Dx      int    `json:"Dx"`
	Dy      int    `json:"Dy"`
	Src     string `json:"Src"`
	Time    int64  `json:"Time"`
}

// 全局变量
const (
	bacH = 200                // 图片高度
	bacW = 400                // 图片宽度
	key  = "ABCDEFGHIJKLMNO1" // 16位
)

// 主方法
func main() {
	path, err := os.Getwd()
	if err != nil {
		fmt.Println("没有权限获取到该目录")
		return
	}

	// 获取配置文件
	config := config{}
	inifile := path + "/conf/system.ini"
	err = gcfg.ReadFileInto(&config, inifile)
	if err != nil {
		fmt.Println("没有找到配置文件:", inifile)
		return
	}
	if len(config.Section.Port) == 0 {
		fmt.Println("端口号（port）不存在，或者不正确:", inifile)
		return
	}
	if len(config.Section.Timeout) == 0 {
		fmt.Println("失效时间（timeout）不存在，或者不正确:", inifile)
		return
	}

	flag.StringVar(&listenAddr, "listen-addr", config.Section.Port, "server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// signal.Notify(quit, os.Interrupt)

	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	server := newWebserver(logger)
	go gracefullShutdown(server, logger, quit, done)

	logger.Println("Server is ready to handle requests at", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	logger.Println("Server stopped")
}

// 关闭 server
// quit: 接收关闭信号
// done: 发出已经关闭信号
func gracefullShutdown(server *http.Server, logger *log.Logger, quit <-chan os.Signal, done chan<- bool) {
	<-quit

	logger.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	close(done)
}

// 初始化 server
func newWebserver(logger *log.Logger) *http.Server {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defaultHttp(w, r)
	})

	//http 服务配置:
	return &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
}

// 获取当前可执行文件目录
func getCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}

// 主方法
func test() {
	// 获取配置文件
	config := config{}
	inifile := "./conf/system.ini"
	err := gcfg.ReadFileInto(&config, inifile)
	if err != nil {
		fmt.Println("没有找到配置文件:", inifile)
		return
	}
	if len(config.Section.Port) == 0 {
		fmt.Println("端口号（port）不存在，或者不正确:", inifile)
		return
	}
	if len(config.Section.Timeout) == 0 {
		fmt.Println("失效时间（timeout）不存在，或者不正确:", inifile)
		return
	}

	s, err := strconv.Atoi(config.Section.Timeout)
	if err != nil {
		s = 20
	}
	timeout := time.Duration(s) * time.Second

	// 监控端口
	srv := http.Server{
		Addr:    config.Section.Port,
		Handler: http.TimeoutHandler(http.HandlerFunc(defaultHttp), timeout, "Timeout!!!"),
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Printf("服务器%s端口已被占用，无法使用", config.Section.Port)
	}
}

// 接口返回地址
func apiResult(w http.ResponseWriter, status int, data interface{}, msg string) {
	boDy, _ := json.Marshal(JsonRes{
		Status: status,
		Data:   data,
		Msg:    msg,
		// 获取时间戳
		TimeStamp: time.Now().Unix(),
	})
	w.Write(boDy)
}

// 动态获取背景图
func readAllPngFiles(path string) (fileName []string, err error) {

	var files []os.FileInfo
	// 读取文件
	files, err = ioutil.ReadDir(path)
	if err != nil {
		err = errors.New("读取文件失败: " + path)
		return
	}

	var reg = regexp.MustCompile(`.\.{1}png$`)
	for _, file := range files {
		if file.IsDir() {
			continue
		} else {
			if reg.MatchString(file.Name()) {
				fileName = append(fileName, path+"/"+file.Name())
			}
		}
	}
	return
}

// 获取访问s值以及宽高
func getCode(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	rbacW := r.PostFormValue("width")
	// 转化成int型
	width, err := strconv.Atoi(rbacW)
	if err != nil {
		width = 400
	}

	// 获取滑块尺寸
	sliderInt := getSliderSize(width)

	// 获取背景高
	hight := bacH * width / bacW

	// 获取滑块位置
	rand.Seed(time.Now().UnixNano())
	dx := rand.Intn(width-2*sliderInt) + sliderInt
	dy := rand.Intn(hight - sliderInt)

	// 动态加载图片
	src, err := getPic()
	if err != nil {
		apiResult(w, '0', nil, "服务器图片无法加载，请及时联系管理人员")
		return
	}

	slider := SliderInfo{
		BacW:    width,             // 背景图宽度
		BacH:    hight,             // 背景图高度
		SliderW: sliderInt,         // 滑块宽度
		SliderH: sliderInt,         // 滑块高度
		Dx:      dx,                // 滑块位置x坐标
		Dy:      dy,                // 滑块位置y坐标
		Src:     src,               // 图片地址
		Time:    time.Now().Unix(), // 时间戳
	}

	source, err := json.Marshal(slider)
	if err != nil {
		apiResult(w, '0', nil, "数据错误")
		return
	}
	sign := AesEncryptCBC(string(source), key)

	// 处理为urlget请求
	s := url.QueryEscape(sign)

	// 封装返回
	var res map[string]string
	res = make(map[string]string)
	res["x"] = strconv.Itoa(dx)
	res["y"] = strconv.Itoa(dy)
	res["sign"] = s

	apiResult(w, 1, res, "调用成功")
}

// 根据背景图大小，获取滑块实际大小
func getSliderSize(w int) int {

	var sliderW int

	switch {
	case w < 200:
		sliderW = 30
	case w < 300:
		sliderW = 40
	case w < 400:
		sliderW = 50
	default:
		sliderW = 50
	}
	return sliderW
}

// 域名处理
func defaultHttp(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
	w.Header().Set("content-type", "application/json")             //返回数据格式是json

	path, httpMethod := r.URL.Path, r.Method

	if path == "/" {
		w.Write([]byte("欢迎使用本系统"))
		return
	}

	if path == "/getCode" && httpMethod == "POST" {
		getCode(w, r)
		return
	}

	if path == "/slider" {
		responseSlider(w, r)
		return
	}

	if path == "/sliderBac" {
		responseSliderBac(w, r)
		return
	}

	if path == "/file" {
		putFile(w, r)
		return
	}

	// 自定义404
	http.Error(w, "请求方式不合法或者没有找到该路由", http.StatusNotFound)
}

// 获取随机图片地址
func getPic() (file string, err error) {

	path, err := os.Getwd()
	if err != nil {
		fmt.Println("没有权限获取到该目录")
		return
	}

	files, err := readAllPngFiles(path + "/img")
	if err != nil {
		return
	}
	randInt := rand.Intn(len(files) - 1)

	file = files[randInt]
	return
}

// 获取滑动验证详情
func getSliderInfo(r *http.Request) (slider SliderInfo, err error) {
	// 获取请求参数
	s := r.FormValue("s")

	// 解密转换
	sign := AesDecryptCBC(s, key)
	slider = SliderInfo{}
	err = myUnmarshal([]byte(sign), &slider)
	if err != nil {
		return
	}
	return
}

// json 解码
func myUnmarshal(input []byte, target interface{}) error {
	if len(input) == 0 {
		return nil
	}

	return json.Unmarshal(input, target)
}

// 返回背景图片
func responseSliderBac(w http.ResponseWriter, r *http.Request) {

	// 获取图片参数
	slider, err := getSliderInfo(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 获取文件
	img, err := getImg(slider.Src)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 压缩图片大小
	img = imaging.Resize(img, slider.BacW, slider.BacH, imaging.Lanczos)

	// // 设置滑块大小
	alpha := image.NewAlpha(image.Rect(0, 0, slider.SliderW, slider.SliderH))
	for x := 0; x < slider.SliderW; x++ {
		for y := 0; y < slider.SliderH; y++ {
			alpha.Set(x, y, color.Alpha{uint8(100 % 256)}) //设定alpha图片的透明度
		}
	}

	// 绘图的背景图。
	dist := image.NewRGBA(image.Rect(0, 0, slider.BacW, slider.BacH))
	siiderRect := image.Rect(slider.Dx, slider.Dy, slider.Dx+slider.SliderW, slider.Dy+slider.SliderH)

	draw.Draw(dist, dist.Bounds(), img, image.ZP, draw.Src)
	draw.Draw(dist, siiderRect, alpha, image.ZP, draw.Over)

	w.Header().Set("text/plain", "charset=utf-8")
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, dist)
	return
}

// 返回滑动小块图片
func responseSlider(w http.ResponseWriter, r *http.Request) {

	slider, err := getSliderInfo(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 获取文件
	img, err := getImg(slider.Src)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	img = imaging.Resize(img, slider.BacW, slider.BacH, imaging.Lanczos)

	rgba := image.NewRGBA(image.Rect(0, 0, slider.SliderW, slider.SliderH))
	draw.Draw(rgba, rgba.Bounds(), img, image.Pt(slider.Dx, slider.Dy), draw.Src)

	w.Header().Set("text/plain", "charset=utf-8")
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, rgba)
	return
}

// 获取文件并转码
func getImg(SrcStr string) (img image.Image, err error) {

	// 打开文件
	fileObj, err := os.Open(SrcStr)
	if err != nil {
		return
	}
	defer fileObj.Close()

	img, _, err = image.Decode(fileObj)
	if err != nil {
		return
	}
	return
}

// 获取文件并返回
func putFile(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	request := r.URL.Query()
	pathStr := request.Get("path")

	// 打开文件
	fileObj, e := os.Open(pathStr)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer fileObj.Close()

	// 切片
	// data := make([]byte, 1024)
	// var strContent string = ""
	// for {
	// 	n, _:= fileObj.Read(data)
	// 	if n == 0 {
	// 		break
	// 	}
	// 	strContent += string(data[0:n])
	// }
	// b :=  []byte(strContent)

	// 读取文件，返回
	buff, err := ioutil.ReadAll(fileObj)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("text/plain", "charset=utf-8")
	w.Header().Set("Content-Type", "image/png")

	w.Write(buff)
	return
}
