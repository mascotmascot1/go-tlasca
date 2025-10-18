// Package config определяет структуру конфигурации приложения
// и предоставляет функцию для загрузки настроек из JSON-файла.
package config

import (
	"encoding/json"
	"log"
	"os"
)

// PathsConfig содержит настройки, связанные с путями файловой системы.
type PathsConfig struct {
	// DataDir указывает директорию, содержащую входную последовательность изображений.
	DataDir string `json:"data_dir"`
	// ResultsDir указывает директорию, куда будет сохранено выходное изображение.
	ResultsDir string `json:"results_dir"`
	// OutputFilename указывает имя файла для сгенерированной карты контраста.
	OutputFilename string `json:"output_filename"`
}

// AlgorithmConfig содержит параметры, специфичные для алгоритма tLASCA.
type AlgorithmConfig struct {
	// WindowSize определяет размер стороны (в пикселях) квадратного скользящего окна,
	// используемого для пространственного усреднения при вычислении контраста.
	WindowSize int `json:"window_size"`
}

// Config является корневой структурой конфигурации, включающей все остальные секции.
type Config struct {
	Paths     PathsConfig     `json:"paths"`
	Algorithm AlgorithmConfig `json:"algorithm"`
}

// NewConfig пытается загрузить конфигурацию из указанного JSON-файла.
// Если файл не существует, логируется предупреждение и возвращается конфигурация по умолчанию.
// Возвращает ошибку, если файл существует, но не может быть прочитан или распарсен,
// или при любых других ошибках файловой системы.
func NewConfig(path string, logger *log.Logger) (*Config, error) {
	// Инициализация значениями по умолчанию, которые будут использованы, если файл не найден.
	var cfg = Config{
		Paths: PathsConfig{
			DataDir:        "data",
			ResultsDir:     "results",
			OutputFilename: "result.png",
		},
		Algorithm: AlgorithmConfig{
			// WindowSize: 1 по умолчанию означает отсутствие пространственного усреднения.
			// Контраст рассчитывается только по временным изменениям каждого пикселя.
			WindowSize: 1,
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("warn: config file '%s' not found, using default settings.\n", path)
			// Возвращаем конфиг по умолчанию; отсутствие файла не считается фатальной ошибкой.
			return &cfg, nil
		}
		// Все другие ошибки (например, нет прав) считаются фатальными.
		return nil, err
	}

	// Если файл успешно прочитан, пытаемся десериализовать JSON
	// поверх значений по умолчанию.
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Возвращаем загруженную из файла конфигурацию.
	return &cfg, nil
}
