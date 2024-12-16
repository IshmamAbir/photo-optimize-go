package main

import (
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/h2non/bimg"
	"github.com/nfnt/resize"
)

const MAX_UPLOAD_SIZE = 20 * 1024 * 1024 // 20 mb limit

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
		http.Error(w, "The uploaded file is too big.Put less than 20 MB.", http.StatusBadRequest)
		return
	}

	// match name of formfile
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// add watermark using "image/draw"
	// we are using library specific methods for adding watermark. so, we won't use it here. 
	// file = AddWatermark(file)

	// create upload folder if not exist
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}


	// start: using nfnt library
	// destinationPath := "./uploads/" + fileHeader.Filename
	// currentPath, err := os.Getwd()
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// logoPath := currentPath + "/logo.png"
	// if err := ProcessImage(file, destinationPath, logoPath); err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// end: using nfnt library


	// start: using bimg library
	_, err = ImageProcessing(file, 1000, "./uploads/bimg.jpeg")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// end: using bimg library

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

// using bimg module from https://github.com/h2non/bimg
func ImageProcessing(file io.Reader, quality int, destinationPath string) (string, error) {

	buffer, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	converted, err := bimg.NewImage(buffer).Convert(bimg.JPEG)
	if err != nil {
		return destinationPath, err
	}

	resized, err := bimg.NewImage(converted).Process(bimg.Options{Quality: quality})
	if err != nil {
		return destinationPath, err
	}

	watermark := bimg.WatermarkImage{
		Left:    10,
		Top:     10,
		Buf:     []byte("./logo.png"),
		Opacity: 0.5,
	}
	finalImage, err := bimg.NewImage(resized).WatermarkImage(watermark)
	if err != nil {
		return destinationPath, err
	}

	if err := bimg.Write(destinationPath, finalImage); err != nil {
		return destinationPath, err
	}
	return destinationPath, nil
}

// using "github.com/nfnt/resize" library. this library is deprecated.
func ProcessImage(imageFile io.Reader, destinationPath string, watermarkPath string) error {
	srcImage, _, err := image.Decode(imageFile)
	if err != nil {
		return err
	}

	width := 1000
	height := 0 // 0 maintains the aspect ratio
	resizedImage := resize.Resize(uint(width), uint(height), srcImage, resize.Lanczos3)

	destinationImageFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationImageFile.Close()

	jpeg.Encode(destinationImageFile, resizedImage, nil)

	watermarkFile, err := os.Open(watermarkPath)
	if err != nil {
		return err
	}
	watermarkImage, _, err := image.Decode(watermarkFile)
	if err != nil {
		return err
	}

	finalImage := addWatermark(resizedImage, watermarkImage)

	jpeg.Encode(destinationImageFile, finalImage, nil)

	return nil
}

func addWatermark(srcImage image.Image, watermarkImage image.Image) image.Image {
	// Create a new RGBA image for the final result
	b := srcImage.Bounds()
	finalImage := image.NewRGBA(b)

	// Draw the source image onto the final image
	draw.Draw(finalImage, b, srcImage, image.Point{}, draw.Over)

	// Calculate the position to place the watermark (e.g., bottom right corner)
	watermarkX := finalImage.Bounds().Dx() - watermarkImage.Bounds().Dx() - 10
	watermarkY := finalImage.Bounds().Dy() - watermarkImage.Bounds().Dy() - 10
	watermarkPos := image.Point{watermarkX, watermarkY}

	// Draw the watermark onto the final image
	draw.Draw(finalImage, watermarkImage.Bounds().Add(watermarkPos), watermarkImage, image.Point{}, draw.Over)

	return finalImage
}
