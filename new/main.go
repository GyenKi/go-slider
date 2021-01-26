package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"example.com/m/middlewares"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"gopkg.in/gcfg.v1"
)

type config struct {
	Section struct {
		Port string
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

func main() {

	path, err := os.Executable()
	if err != nil {
		fmt.Println("路径获取不正确", err)
	}
	dir := filepath.Dir(path)

	// 获取配置文件
	config := config{}
	inifile := dir + "/conf/system.ini"
	err = gcfg.ReadFileInto(&config, inifile)
	if err != nil {
		fmt.Println("没有找到配置文件:", inifile)
		return
	}
	if len(config.Section.Port) == 0 {
		fmt.Println("端口号（port）不存在，或者不正确:", inifile)
		return
	}

	r := gin.Default()
	r.Use(middlewares.Cors())

	r.POST("/getCode", getCode)
	r.GET("/slider", responseSlider)
	r.GET("/sliderBac", responseSliderBac)

	r.Run(config.Section.Port)
}

// 获取访问s值以及宽高
func getCode(c *gin.Context) {

	// 获取请求参数
	rbacW := c.PostForm("width")
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
		responseJson(c, '0', nil, "服务器图片无法加载，请及时联系管理人员")
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
		responseJson(c, '0', nil, "数据错误")
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

	responseJson(c, 1, res, "调用成功")
}

// 返回滑动小块图片
func responseSlider(c *gin.Context) {

	slider, err := getSliderInfo(c)
	if err != nil {
		log.Println(err)
		responseJson(c, 0, nil, "请求参数s签名不正确")
		return
	}

	// 获取文件
	img, err := getImg(slider.Src)
	if err != nil {
		log.Println(err)
		responseJson(c, 0, nil, "文件查询不到")
		return
	}

	img = imaging.Resize(img, slider.BacW, slider.BacH, imaging.Lanczos)

	rgba := image.NewRGBA(image.Rect(0, 0, slider.SliderW, slider.SliderH))
	draw.Draw(rgba, rgba.Bounds(), img, image.Pt(slider.Dx, slider.Dy), draw.Src)

	png.Encode(c.Writer, rgba)
}

// 返回背景图片
func responseSliderBac(c *gin.Context) {

	// 获取图片参数
	slider, err := getSliderInfo(c)
	if err != nil {
		responseJson(c, 0, nil, "请求参数s签名不正确")
		return
	}

	// 获取文件
	img, err := getImg(slider.Src)
	if err != nil {
		log.Println(err)
		responseJson(c, 0, nil, "文件查询不到")
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

	png.Encode(c.Writer, dist)
}

// 获取文件并转码
func getImg(SrcStr string) (img image.Image, err error) {

	// 打开文件
	fileObj, err := os.Open(SrcStr)
	if err != nil {
		return
	}
	defer fileObj.Close()

	img, err = png.Decode(fileObj)
	if err != nil {
		return
	}
	return
}

// 获取滑动验证详情
func getSliderInfo(c *gin.Context) (slider SliderInfo, err error) {
	// 获取请求参数
	s := c.Query("s")

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

// 返回json数据
func responseJson(c *gin.Context, status int, data interface{}, msg string) {
	c.JSON(200, JsonRes{
		Status: status,
		Data:   data,
		Msg:    msg,
		// 获取时间戳
		TimeStamp: time.Now().Unix(),
	})
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

// 获取随机图片地址
func getPic() (file string, err error) {

	path, err := os.Executable()
	if err != nil {
		fmt.Println("路径获取不正确", err)
	}
	dir := filepath.Dir(path)
	log.Println(dir)

	files, err := readAllPngFiles(dir + "/img")
	if err != nil {
		return
	}
	randInt := rand.Intn(len(files) - 1)

	file = files[randInt]
	return
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
