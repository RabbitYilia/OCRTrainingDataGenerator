package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"github.com/BurntSushi/graphics-go/graphics"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"gocv.io/x/gocv"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"image"
	"image/draw"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var fontsList map[string]*truetype.Font
var charList map[int]string
var running chan int
var starttime int64
var filecount int64
var FileNameChan chan *ImgFile

type ImgFile struct {
	Filename string
	Data     *gocv.Mat
}

func main() {
	FileNameChan = make(chan *ImgFile, 100)
	filecount = 0
	starttime = time.Now().Unix()
	running = make(chan int, 10)
	for num := 1; num <= 8; num++ {
		running <- 1
	}
	ReadChar()
	ReadFont()
	i := 0
	go SaveProcess()
	go StaticProcess()
	for id, thisStr := range charList {
		i += 1
		log.Printf("Processing[%d/%d][%d]:%s", i, len(charList), id, thisStr)
		<-running
		go MakeBaseImg(id)
	}
	for len(running) < 8 && len(FileNameChan) == 0 {
		time.Sleep(time.Second)
	}
	log.Println(time.Now().Unix() - starttime)
}
func StaticProcess() {
	for {
		time.Sleep(time.Second * 5)
		log.Println(atomic.LoadInt64(&filecount) / (time.Now().Unix() - starttime))
	}
}

func SaveProcess() {
	fw, err := os.Create("output.zip")
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	outputWriter := zip.NewWriter(fw)
	var buf []byte
	for {
		atomic.AddInt64(&filecount, 1)
		File := <-FileNameChan
		buf, err = gocv.IMEncode(".png", *File.Data)
		if err != nil {
			log.Fatal(err)
		}
		f, err := outputWriter.Create(File.Filename)
		if err != nil {
			log.Fatal(err)
		}
		defer File.Data.Close()
		f.Write(buf)
		outputWriter.Flush()
	}
	defer outputWriter.Close()
	defer fw.Close()
}

func MakeBaseImg(id int) {
	count := 0
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	dr := &font.Drawer{
		Dst: img,
		Src: image.White,
		Dot: fixed.Point26_6{},
	}
	op := truetype.Options{
		Size:              70,
		DPI:               0,
		Hinting:           0,
		GlyphCacheEntries: 0,
		SubPixelsX:        0,
		SubPixelsY:        0,
	}
	for _, fontstyle := range fontsList {
		dr.Face = truetype.NewFace(fontstyle, &op)
		dr.Dot.X = (fixed.I(100) - dr.MeasureString(charList[id])) / 2
		dr.Dot.Y = fixed.I(75)
		draw.Draw(img, img.Bounds(), image.Black, image.ZP, draw.Src)
		dr.DrawString(charList[id])
		newimg := image.NewRGBA(image.Rect(0, 0, 100, 100))
		for degree := -90.0; degree <= 90; degree++ {
			graphics.Rotate(newimg, img, &graphics.RotateOptions{math.Pi / 180 * degree})
			thisMat, err := gocv.ImageToMatRGBA(newimg)
			if err != nil {
				log.Fatal(err)
			}
			thisMatp := RGBA2Binary(&thisMat)
			FileNameChan <- &ImgFile{Filename: strconv.Itoa(id) + "_" + charList[id] + "_" + strconv.Itoa(count) + ".png", Data: thisMatp}
			count++
			thisMat1 := Dilate(thisMatp)
			FileNameChan <- &ImgFile{Filename: strconv.Itoa(id) + "_" + charList[id] + "_" + strconv.Itoa(count) + ".png", Data: thisMat1}
			count++
			FileNameChan <- &ImgFile{Filename: strconv.Itoa(id) + "_" + charList[id] + "_" + strconv.Itoa(count) + ".png", Data: Erode(thisMatp)}
			count++
			FileNameChan <- &ImgFile{Filename: strconv.Itoa(id) + "_" + charList[id] + "_" + strconv.Itoa(count) + ".png", Data: Erode(thisMat1)}
			count++
			defer thisMat.Close()
		}
	}
	running <- 1
}

func RGBA2Binary(imgmat *gocv.Mat) *gocv.Mat {
	NewMat := gocv.NewMat()
	gocv.CvtColor(*imgmat, &NewMat, gocv.ColorBGRAToGray)
	gocv.Threshold(NewMat, imgmat, 100, 255, gocv.ThresholdBinary)
	defer NewMat.Close()
	return imgmat
}

func Dilate(imgmat *gocv.Mat) *gocv.Mat {
	newImgmat := gocv.NewMat()
	gocv.Dilate(*imgmat, &newImgmat, gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3}))
	return &newImgmat
}

func Erode(imgmat *gocv.Mat) *gocv.Mat {
	newImgmat := gocv.NewMat()
	gocv.Erode(*imgmat, &newImgmat, gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3}))
	return &newImgmat
}
func ReadChar() {
	charList = make(map[int]string)
	f, err := os.Open("./charlist.txt")
	if err != nil {
		log.Fatal(err)
	}
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
	defer f.Close()
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
