package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nfnt/resize"
	"golang.org/x/image/bmp"
)

const MAX_UPLOAD_SIZE = 20 * 1024 * 1024

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	http.ServeFile(w, r, "index.html")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check total file size
	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(w, "The uploaded file is too big.Put less than 1MB.", http.StatusBadRequest)
		return
	}

	// match name of formfile
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// file = AddWatermark(file)

	// create upload folder if not exist
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	destinationPath := "./uploads/" + fileHeader.Filename
	currentPath, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logoPath := currentPath + "/logo.png"
	if err := ProcessImage(file, destinationPath, logoPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	println("File uploaded successfully")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/upload", uploadHandler)
	println("Server started on port 4500")
	println("-----------------------------")
	if err := http.ListenAndServe(":4500", mux); err != nil {
		log.Fatal(err)
	}
}

// using "github.com/nfnt/resize" library. this library is deprecated.
func ProcessImage(imageFile io.Reader, destinationPath string, watermarkPath string) error {
	srcImage, imageExtension, err := image.Decode(imageFile)
	if err != nil {
		return err
	}
	println("image extension: ", imageExtension)

	width := 2000
	height := 0 // 0 maintains the aspect ratio
	resizedImage := resize.Resize(uint(width), uint(height), srcImage, resize.Lanczos3)

	destinationImageFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationImageFile.Close()

	// jpeg.Encode(destinationImageFile, resizedImage, nil)

	watermarkFilePng, err := os.Open(watermarkPath)
	if err != nil {
		return err
	}
	watermarkImage, err := png.Decode(watermarkFilePng)
	if err != nil {
		return err
	}

	finalImage := addWatermark(resizedImage, watermarkImage)

	switch imageExtension {
	case "png":
		err = png.Encode(destinationImageFile, finalImage)
	case "gif":
		err = gif.Encode(destinationImageFile, finalImage, nil)
	case "bmp":
		err = bmp.Encode(destinationImageFile, finalImage)
	default:
		err = jpeg.Encode(destinationImageFile, finalImage, nil)
	}
	if err != nil {
		return err
	}

	return nil
}

func addWatermark(srcImage image.Image, watermarkImage image.Image) image.Image {
	srcBound := srcImage.Bounds()
	watermarkBound := watermarkImage.Bounds()

	if watermarkBound.Dx() > srcBound.Dx()/8 {
		width := 255
		height := 0 // 0 maintains the aspect ratio
		resizedWatermark := resize.Resize(uint(width), uint(height), watermarkImage, resize.Lanczos3)
		watermarkBound = resizedWatermark.Bounds()
		watermarkImage = resizedWatermark
	}

	offset := image.Pt(
		srcBound.Dx()-watermarkBound.Dx()-20,
		srcBound.Dy()-watermarkBound.Dy()-20,
	)

	outputImage := image.NewRGBA(srcBound)
	draw.Draw(outputImage, outputImage.Bounds(), srcImage, image.Point{}, draw.Over)
	draw.DrawMask(outputImage, watermarkBound.Add(offset), watermarkImage, image.Point{}, image.NewUniform(color.Alpha{200}), image.Point{}, draw.Over)

	return outputImage
}
