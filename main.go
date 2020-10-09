package main

import (
	"flag"
	"fmt"
	"github.com/tealeg/xlsx"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var codes string
var images string
var startRow = 1
var processor int

var wg sync.WaitGroup
var imageConf = make([]map[string]string, 0)
var imageDir = "images"

func init() {
	runtime.GOMAXPROCS(2)
	flag.StringVar(&codes, "codes", "./codes.xlsx", "商品清单")
	flag.StringVar(&images, "images", "./images.xlsx", "图片前后缀及尺寸配置，标题不能修改")
	flag.IntVar(&startRow, "startRow", 1, "商品开始行")
	flag.IntVar(&processor, "processor", 3, "进程数")
	confirm()
}

//确认所需资源
func confirm() {
	if _, err := os.Stat(imageDir); err == nil {
		imageDir = "images_" + strconv.Itoa(rand.Int())
	}
	os.Mkdir(imageDir, 0777)

	log("images will be store in " + imageDir)

	if _, err := os.Stat(codes); err != nil {
		panic(err)
	}
	if _, err := os.Stat(images); err != nil {
		panic(err)
	}
}

func main() {
	parseImageConfig()
	signalChan := make(chan int, processor)
	file, err := xlsx.OpenFile(codes)
	if err != nil {
		panic(err)
	}

	sheet := file.Sheets[0]
	for i := startRow; i < sheet.MaxRow; i++ {
		code := strings.TrimSpace(sheet.Cell(i, 0).Value)
		if code == "" {
			log(fmt.Sprintf("data in %d is empty", i))
			continue
		}
		wg.Add(1)
		signalChan <- i
		go getImage(code, signalChan)
	}

	wg.Wait()
	log("Done")
}

func getImage(code string, signalChan chan int) {
	//log("begin get " + code)
	defer func() {
		//log("end get " + code)
		<- signalChan
		wg.Done()
	}()
	urlTemplate := "https://underarmour.scene7.com/is/image/Underarmour/%s%s%s?qty=%s&size=%s&wid=%s&hei=%s&fmt=%s&extend=%s&%s&%s"
	for _, conf := range imageConf {
		url := fmt.Sprintf(urlTemplate,
			conf["prefix"],
			code,
			conf["suffix"],
			conf["qty"],
			conf["size"],
			conf["wid"],
			conf["hei"],
			conf["fmt"],
			conf["extend"],
			conf["column1"],
			conf["column2"],
		)
		imageTemplate := "%s/%s%s%s.jpg"
		if conf["fmt"] != "jpg" {
			imageTemplate = "%s/%s%s%s.png"
		}
		imageName := fmt.Sprintf(imageTemplate, imageDir, conf["prefix"], code, conf["suffix"])

		response, err := http.Get(url)
		if err != nil {
			log("500 " + url + " " + err.Error())
			continue
		}

		if response.StatusCode == 200 {
			imageContent, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log("500 " + url + " image write error : " + err.Error())
				continue
			}

			if err = ioutil.WriteFile(imageName, imageContent, 0777); err != nil{
				log("500 " + url + " image write error: " + err.Error())
			}
		}else {
			log(strconv.Itoa(response.StatusCode) + " " + url + " " + response.Status)
		}
	}
}

func parseImageConfig() {
	configFile, err := xlsx.OpenFile(images)
	if err != nil {
		panic(err)
	}

	sheet := configFile.Sheets[0]
	index := getIndex(sheet)

	for i:=1; i<sheet.MaxRow; i++ {
		if isGet := strings.TrimSpace(sheet.Cell(i, index["get"]).Value);  isGet != "1" {
			continue
		}

		imageConf = append(imageConf, map[string]string{
			"prefix": strings.TrimSpace(sheet.Cell(i, index["prefix"]).Value),
			"suffix": strings.TrimSpace(sheet.Cell(i, index["suffix"]).Value),
			"fmt": strings.TrimSpace(sheet.Cell(i, index["fmt"]).Value),
			"qty": strings.TrimSpace(sheet.Cell(i, index["qty"]).Value),
			"wid": strings.TrimSpace(sheet.Cell(i, index["wid"]).Value),
			"hei": strings.TrimSpace(sheet.Cell(i, index["hei"]).Value),
			"size": strings.TrimSpace(sheet.Cell(i, index["size"]).Value),
			"extend": strings.TrimSpace(sheet.Cell(i, index["extend"]).Value),
			"column1": strings.TrimSpace(sheet.Cell(i, index["column1"]).Value),
			"column2": strings.TrimSpace(sheet.Cell(i, index["column2"]).Value),
		})
	}
}

func getIndex(sheet *xlsx.Sheet) map[string]int {
	var index = map[string]int{
		"id": -1,
		"category": -1,
		"prefix": -1,
		"suffix": -1,
		"get": -1,
		"desc": -1,
		"fmt": -1,
		"qty": -1,
		"wid": -1,
		"hei": -1,
		"size": -1,
		"extend": -1,
		"column1": -1,
		"column2": -1,
	}

	for k, _ := range index {
		for i, cell := range sheet.Row(0).Cells {
			if strings.TrimSpace(cell.Value) == k {
				index[k] = i
				break
			}
		}
		if index[k] == -1 {
			panic("图片配置异常， 缺少 " + k)
		}
	}

	return index
}

func log(str string)  {
	fmt.Println(str)
}
