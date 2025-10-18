// Package tlasca содержит основную логику для вычисления
// карты контраста на основе алгоритма Temporal Laser Speckle Contrast Analysis.
package tlasca

import (
	"image"
	"image/color"
	"log"
	"math"
	"runtime"
	"sync"

	"github.com/mascotmascot1/go-tlasca/internal/config"
)

// Runner инкапсулирует основную логику и зависимости (конфигурацию, логгер)
// для выполнения алгоритма tLASCA.
type Runner struct {
	algorithm config.AlgorithmConfig
	logger    *log.Logger
}

// NewRunner является конструктором для Runner. Он создает и инициализирует
// новый экземпляр со всеми необходимыми зависимостями.
func NewRunner(cfg *config.Config, logger *log.Logger) *Runner {
	return &Runner{
		algorithm: cfg.Algorithm,
		logger:    logger,
	}
}

// Run является главной публичной точкой входа для запуска вычислений.
// Он оркестрирует весь процесс анализа, вызывая внутренние методы для расчетов.
func (r *Runner) Run(grayImages []*image.Gray) *image.Gray {
	r.logger.Println("starting contrast map calculation...")
	changeMap := r.calculateContrastMap(grayImages)
	r.logger.Println("calculation finished.")
	return changeMap
}

// temporalWindowContrast вычисляет временной контраст в окне размером windowSize x windowSize
// на основе последовательности изображений.
//
// Принимает:
//
//	images []*image.Gray: срез последовательных изображений в градациях серого (кадры по времени).
//	x, y int: координаты верхнего левого угла окна в изображении.
//
// Возвращает:
//
//	float64: усреднённый временной контраст в пределах окна.
//
// Алгоритм:
// 1. Для каждого пикселя в окне собирается временной ряд его интенсивности (по кадрам).
// 2. Для этого временного ряда рассчитываются:
//   - Среднее значение интенсивности по времени (mean).
//   - **Выборочная дисперсия (sample variance)**, используя (N-1) в знаменателе.
//     Это критически важно, так как мы работаем с ограниченной выборкой кадров,
//     а не со всей генеральной совокупностью возможных спекл-паттернов.
//   - Стандартное отклонение (stdDev) как корень из дисперсии.
//
// 3. Вычисляется контраст для пикселя как отношение `stdDev / mean` (если mean > 0).
// 4. Результат — среднее значение контраста по всем пикселям окна.
func (r *Runner) temporalWindowContrast(images []*image.Gray, x, y int) float64 {
	// накапливаем общий контраст по окну
	var sumVar float64

	pixelCount := float64(r.algorithm.WindowSize * r.algorithm.WindowSize)

	for dy := 0; dy < r.algorithm.WindowSize; dy++ {
		for dx := 0; dx < r.algorithm.WindowSize; dx++ {

			// временной ряд интенсиностей для пикселя
			values := make([]float64, 0, len(images))
			for _, img := range images {
				values = append(values, float64(img.GrayAt(x+dx, y+dy).Y))
			}

			var mean float64
			for _, v := range values {
				mean += v
			}
			// среднее по времени
			mean /= float64(len(values))

			var sumDiff2 float64
			for _, v := range values {
				diff := v - mean
				sumDiff2 += diff * diff
			}

			variance := sumDiff2 / float64(len(values)-1)
			stdDev := math.Sqrt(variance)
			if mean > 0 {
				// контраст
				sumVar += stdDev / mean
			}
		}
	}
	return sumVar / pixelCount // усреднение по всем пикселям окна (относительное измерение изменчивости)
}

// calculateContrastMap вычисляет карту контраста изображения параллельно,
// используя скользящее окно.
//
// Принимает:
//
//	grayImages []*image.Gray: слайс последовательных изображений в градациях серого.
//
// Возвращает:
//
//	*image.Gray: изображение, где интенсивность пикселя соответствует усредненному
//	             временному контрасту в соответствующей области исходных изображений.
//
// Алгоритм:
// 1. Изображение делится на горизонтальные полосы по числу доступных логических ядер CPU.
// 2. Для каждой полосы запускается отдельная горутина.
// 3. Внутри горутины:
//   - Для каждого возможного положения окна (верхнего левого угла) размером WindowSize x WindowSize
//     вычисляется усредненный временной контраст с помощью temporalWindowContrast.
//   - Результаты для одной строки записываются во временный срез.
//   - Заполненный срез-строка записывается в соответствующую строку общего среза результатов listContrast.
//
// 4. После завершения всех горутин (wg.Wait()):
//   - Значения контраста из listContrast масштабируются в диапазон [0, 255].
//   - Генерируется финальное изображение *image.Gray с полученными значениями интенсивности.
func (r *Runner) calculateContrastMap(grayImages []*image.Gray) *image.Gray {
	bounds := grayImages[0].Bounds()
	// Вычисляем размеры итогового изображения контраста.
	widthNew, heightNew := bounds.Dx()-r.algorithm.WindowSize+1, bounds.Dy()-r.algorithm.WindowSize+1
	// Предварительно выделяем память под внешний срез для строк результатов.
	listContrast := make([][]float64, heightNew)

	// --- Параллельное вычисление контраста для каждой строки ---
	numWorkers := runtime.NumCPU() // Используем все доступные логические ядра CPU.
	var wg sync.WaitGroup

	rowsPerWorker := heightNew / numWorkers // Делим изображение на горизонтальные полосы.
	wg.Add(numWorkers)                      // Сообщаем WaitGroup, сколько горутин ожидать.
	for i := 0; i < numWorkers; i++ {
		// Определяем диапазон строк (startY, endY) для текущей горутины.
		startY := i * rowsPerWorker
		endY := (i + 1) * rowsPerWorker

		// Последняя горутина забирает остаток строк, если не делится нацело.
		if i == numWorkers-1 {
			endY = heightNew
		}

		// Запускаем горутину для обработки своей полосы.
		go func(startY, endY int) {
			defer wg.Done() // Сообщаем WaitGroup о завершении работы при выходе из горутины.

			// Итерируемся по строкам (y), назначенным этой горутине.
			for y := startY; y < endY; y++ {
				// Создаем и заполняем срез для текущей строки.
				row := make([]float64, 0, widthNew)
				for x := 0; x < widthNew; x++ {
					contrast := r.temporalWindowContrast(grayImages, x, y)
					row = append(row, contrast)
				}
				// Записываем готовую строку в общий срез результатов.
				// Запись безопасна, так как каждая горутина пишет в свой уникальный индекс 'y'.
				listContrast[y] = row
			}
		}(startY, endY)
	}
	wg.Wait() // Ожидаем завершения всех горутин.

	// --- Сборка финального изображения из среза контрастов ---
	changeMap := image.NewGray(image.Rect(0, 0, widthNew, heightNew))
	for y := 0; y < heightNew; y++ {
		for x := 0; x < widthNew; x++ {
			// Масштабируем значение контраста (float64) в яркость пикселя (byte [0-255]).
			// math.Min используется для ограничения сверху значением 255.
			intensity := byte(math.Min(listContrast[y][x]*255, 255))
			changeMap.SetGray(x, y, color.Gray{Y: intensity})
		}
	}
	return changeMap
}
