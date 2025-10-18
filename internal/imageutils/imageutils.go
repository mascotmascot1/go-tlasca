// Package imageutils предоставляет вспомогательные функции для загрузки, сохранения,
// конвертации и извлечения информации из файлов изображений.
package imageutils

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ExtractNumber извлекает числовое значение из имени файла (например, "10.png").
//
// Принимает:
//
//	filename string: путь к файлу.
//
// Возвращает:
//
//	int: числовое значение, извлеченное из имени файла.
//	error: ошибку, если имя файла имеет неверный формат или не содержит числа.
func ExtractNumber(filename string) (int, error) {
	filename = filepath.Base(filename)
	filename = strings.TrimSuffix(filename, ".png")
	number, err := strconv.Atoi(filename)
	if err != nil {
		return 0, err
	}
	return number, nil
}

// LoadImage загружает изображение из файла.
//
// Принимает:
// filename string: путь к изображению.
//
// Возвращает:
// image.Image: загруженное изображение.
// error: ошибку, если не удалось загрузить изображение.
func LoadImage(filename string) (img image.Image, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}()
	img, _, err = image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// СonvertToGray преобразует изображение в градации серого.
//
// Принимает:
// img image.Image: входное изображение.
//
// Возвращает:
// *image.Gray: изображение в градациях серого.
func ConvertToGray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)
	draw.Draw(grayImg, bounds, img, bounds.Min, draw.Src)
	return grayImg
}

// saveImage сохраняет изображение в формате PNG.
//
// Принимает:
// filename string: путь для сохранения.
// img *image.Gray: изображение в градациях серого.
//
// Возвращает:
// error: ошибку, если не удалось сохранить файл.
func SaveImage(filename string, img *image.Gray) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}()
	err = png.Encode(file, img)
	if err != nil {
		return err
	}
	return nil
}
