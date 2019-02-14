package main

import (
	"bufio"
	"github.com/BurntSushi/graphics-go/graphics"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"gocv.io/x/gocv"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

var fontsList map[string]*truetype.Font
var charList map[int]string
var running chan int
var trainlist *os.File
var valilist *os.File

func main() {
	trainlist, err := os.Create("train.txt")
	if err != nil {
		log.Fatal("Failed to open the file", err.Error())
	}

	valist, err := os.Create("val.txt")
	if err != nil {
		log.Fatal("Failed to open the file", err.Error())
	}

	running = make(chan int, 10)
	for num := 1; num <= 8; num++ {
		running <- 1
	}
	ReadChar()
	ReadFont()
	i := 0
	for id, thisStr := range charList {
		i += 1
		log.Printf("Processing[%d/%d]:%s", i, len(charList), thisStr)
		<-running
		go MakeBaseImg(id)
	}
	for len(running) < 8 {
		time.Sleep(time.Second)
	}
	trainlist.Close()
	valist.Close()
}

func MakeBaseImg(id int) {
	thisStr := charList[id]
	count := 0
	for _, fontstyle := range fontsList {
		face := truetype.NewFace(fontstyle, &truetype.Options{
			Size:              70,
			DPI:               0,
			Hinting:           0,
			GlyphCacheEntries: 0,
			SubPixelsX:        0,
			SubPixelsY:        0,
		})
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		draw.Draw(img, img.Bounds(), image.Black, image.ZP, draw.Src)
		dr := &font.Drawer{
			Dst:  img,
			Src:  image.White,
			Face: face,
			Dot:  fixed.Point26_6{},
		}
		dr.Dot.X = (fixed.I(100) - dr.MeasureString(thisStr)) / 2
		dr.Dot.Y = fixed.I(75)
		dr.DrawString(thisStr)
		for degree := -90.0; degree <= 90; degree++ {
			newimg := image.NewRGBA(image.Rect(0, 0, 100, 100))
			graphics.Rotate(newimg, img, &graphics.RotateOptions{math.Pi / 180 * degree})
			thisMat, err := gocv.ImageToMatRGBA(newimg)
			if err != nil {
				log.Fatal(err)
			}
			thisMat = RGBA2Binary(thisMat)
			SaveFile(thisMat, id, count)
			count++
			thisMat1 := Dilate(thisMat)
			SaveFile(thisMat1, id, count)
			count++
			thisMat2 := Erode(thisMat)
			SaveFile(thisMat2, id, count)
			count++
			thisMat3 := Erode(thisMat1)
			SaveFile(thisMat3, id, count)
			count++
		}
	}
	running <- 1
}

func RGBA2Binary(imgmat gocv.Mat) gocv.Mat {
	NewMat := gocv.NewMat()
	gocv.CvtColor(imgmat, &NewMat, gocv.ColorBGRAToGray)
	gocv.Threshold(NewMat, &imgmat, 100, 255, gocv.ThresholdBinary)
	return NewMat
}

func Dilate(imgmat gocv.Mat) gocv.Mat {
	newImgmat := gocv.NewMat()
	Elepoint := image.Point{X: 3, Y: 3}
	Element := gocv.GetStructuringElement(gocv.MorphRect, Elepoint)
	gocv.Dilate(imgmat, &newImgmat, Element)
	return newImgmat
}

func Erode(imgmat gocv.Mat) gocv.Mat {
	newImgmat := gocv.NewMat()
	Elepoint := image.Point{X: 3, Y: 3}
	Element := gocv.GetStructuringElement(gocv.MorphRect, Elepoint)
	gocv.Erode(imgmat, &newImgmat, Element)
	return newImgmat
}

func SaveFile(img gocv.Mat, id int, count int) {
	thisStr := charList[id]
	image, err := img.ToImage()
	if err != nil {
		log.Println("生成文件出错")
		log.Fatal(err)
	}
	if count%5 == 0 {
		valilist.WriteString(strconv.Itoa(id) + "_" + strconv.Itoa(count) + ".png " + thisStr + "\n")
		valilist.Sync()
	} else {
		trainlist.WriteString(strconv.Itoa(id) + "_" + strconv.Itoa(count) + ".png " + thisStr + "\n")
		trainlist.Sync()
	}
	imgout, err := os.Create(strconv.Itoa(id) + "_" + strconv.Itoa(count) + ".png")
	defer imgout.Close()
	if err != nil {
		log.Println("生成文件出错")
		log.Fatal(err)
	}
	err = png.Encode(imgout, image)
	if err != nil {
		log.Println("生成图片出错")
		log.Fatal(err)
	}
}

func ReadChar() {
	charList = make(map[int]string)
	f, err := os.Open("./charlist.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	i := 1
	for {
		line, err := reader.ReadString('\n') //以'\n'为结束符读入一行
		if err != nil || io.EOF == err {
			break
		}
		charList[i] = strings.Split(line, "")[0]
		i += 1
	}
}
func ReadFont() {
	fontsList = make(map[string]*truetype.Font)
	rd, err := ioutil.ReadDir("./fonts/")
	if err != nil {
		log.Println("请建立fonts文件夹，并放入字体")
		log.Println(err)
		return
	}
	for _, fi := range rd {
		if strings.Contains(fi.Name(), "TTF") || strings.Contains(fi.Name(), "ttf") {
			fontBytes, err := ioutil.ReadFile("./fonts/" + fi.Name())
			if err != nil {
				log.Println("读取字体数据出错")
				log.Println(err)
				return
			}
			fontstyle, err := freetype.ParseFont(fontBytes)
			if err != nil {
				log.Println("转换字体样式出错")
				log.Println(err)
				return
			}
			fontsList[fi.Name()] = fontstyle
		}
	}
}
