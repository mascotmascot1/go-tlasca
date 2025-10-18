// Package main является точкой входа в приложение go-tlasca.
package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/mascotmascot1/go-tlasca/internal/config"
	"github.com/mascotmascot1/go-tlasca/internal/imageutils"
	"github.com/mascotmascot1/go-tlasca/internal/tlasca"
)

// main - точка входа. Ее единственная задача - настроить окружение (логгер)
// и вызвать основную логику приложения в функции run.
func main() {
	logger := log.New(os.Stdout, "[GO-TLASCA] ", log.LstdFlags)

	if err := run(logger); err != nil {
		logger.Fatalf("application failed: %v\n", err)
	}
}

// run содержит основной рабочий процесс приложения: от загрузки конфига до сохранения результата.
// Возвращает ошибку, если какой-либо из критических шагов не может быть выполнен.
func run(logger *log.Logger) error {
	const configPath = "go-tlasca.json"

	// Загружаем конфигурацию.
	cfg, err := config.NewConfig(configPath, logger)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Инициализируем исполнителя алгоритма.
	runner := tlasca.NewRunner(cfg, logger)

	// --- 1. Поиск и сортировка входных файлов ---
	logger.Println("searching for image files...")

	// Проверяем существование директории с данными, чтобы предоставить пользователю
	// понятную ошибку в случае неверного пути в конфиге.
	if _, err := os.Stat(cfg.Paths.DataDir); os.IsNotExist(err) {
		return fmt.Errorf("data directory '%s' not found", cfg.Paths.DataDir)
	}
	files, err := filepath.Glob(filepath.Join(cfg.Paths.DataDir, "*.png"))
	if err != nil {
		return fmt.Errorf("invalid file pattern: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no png files found in '%s'", cfg.Paths.DataDir)
	}

	// Сортируем файлы по числовому значению в имени, чтобы гарантировать
	// правильный временной порядок кадров для анализа (Sort files using natural order).
	sort.SliceStable(files, func(i, j int) bool {
		numI, err := imageutils.ExtractNumber(files[i])
		if err != nil {
			// Некорректный формат имени файла - это фатальная ошибка в подготовке данных.
			// Дальнейшее выполнение бессмысленно, поэтому вызываем панику.
			panic(fmt.Sprintf("invalid filename format: %s -> %v", files[i], err))
		}
		numJ, err := imageutils.ExtractNumber(files[j])
		if err != nil {
			panic(fmt.Sprintf("invalid filename format: %s -> %v", files[j], err))
		}
		return numI < numJ
	})
	logger.Printf("found and sorted %d files.\n", len(files))

	// --- 2. Загрузка и подготовка изображений ---
	logger.Println("loading and converting images...")
	grayImages, err := loadAndProcessImages(files)
	if err != nil {
		// Ошибка на этом этапе фатальна, так как алгоритму требуется полная последовательность.
		return err
	}

	// --- 3. Выполнение алгоритма tLASCA ---
	changeMap := runner.Run(grayImages)

	// --- 4. Сохранение результата ---
	logger.Println("saving result...")
	err = os.MkdirAll(cfg.Paths.ResultsDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating results directory '%s': %w", cfg.Paths.ResultsDir, err)
	}

	newPath := filepath.Join(cfg.Paths.ResultsDir, cfg.Paths.OutputFilename)
	if err = imageutils.SaveImage(newPath, changeMap); err != nil {
		return fmt.Errorf("error saving result image to '%s': %w", newPath, err)
	}
	logger.Printf("image saving completed: %s\n", newPath)

	return nil
}

// loadAndProcessImages обрабатывает список путей к файлам, загружая и конвертируя каждое изображение.
// Функция возвращает ошибку, если хотя бы один из файлов не может быть обработан,
// так как для алгоритма tLASCA важна целостность и порядок последовательности.
func loadAndProcessImages(paths []string) ([]*image.Gray, error) {
	grayImages := make([]*image.Gray, 0, len(paths))
	for _, filePath := range paths {
		img, err := imageutils.LoadImage(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load image '%s': %w", filePath, err)
		}
		grayImg := imageutils.ConvertToGray(img)
		grayImages = append(grayImages, grayImg)
	}
	return grayImages, nil
}
