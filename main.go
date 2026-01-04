package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	wordsFile     = "words.json"
	usedWordsFile = "used_words.json"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Запрашиваем флаг для парсинга
	fmt.Print("Парсить сайт? (0 - нет, 1 - да): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Ошибка при чтении ввода: %v\n", err)
		return
	}
	input = strings.TrimSpace(input)

	parseFlag, err := strconv.Atoi(input)
	if err != nil || (parseFlag != 0 && parseFlag != 1) {
		fmt.Println("Некорректный ввод. Используется значение по умолчанию: 0")
		parseFlag = 0
	}

	// Парсим сайт, если указан флаг
	if parseFlag == 1 {
		fmt.Println("Запуск парсинга сайта...")
		err := parseSite()
		if err != nil {
			fmt.Printf("Ошибка при парсинге: %v\n", err)
			return
		}
		fmt.Println("Парсинг завершен")
	}

	// Загружаем слова из файла
	wordsDict, err := loadWords()
	if err != nil {
		fmt.Printf("Ошибка при загрузке слов: %v\n", err)
		return
	}

	// Загружаем использованные слова
	usedWords, err := loadUsedWords()
	if err != nil {
		fmt.Printf("Ошибка при загрузке использованных слов: %v\n", err)
		usedWords = make(map[string]bool)
	}

	// Получаем 5 случайных слов
	randomWords, err := getRandomWords(wordsDict, usedWords, 5)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}

	// Выводим слова
	fmt.Println("\n=== 5 случайных слов ===")
	for num, wordData := range randomWords {
		for word, translation := range wordData {
			fmt.Printf("%s. %s - %s\n", num, word, translation)
		}
	}
}

// parseSite парсит сайт и сохраняет слова в words.json
func parseSite() error {
	url := "https://skyeng.ru/articles/samye-populyarnye-slova-v-anglijskom-yazyke/"

	// Скачиваем страницу
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("ошибка при загрузке страницы: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка: статус код %d", resp.StatusCode)
	}

	// Парсим HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка при парсинге HTML: %w", err)
	}

	// Словарь для хранения результатов
	wordsDict := make(map[string]map[string]string)
	var lastTranslation string

	// Ищем все таблицы на странице
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		table.Find("tbody tr, tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			cellCount := cells.Length()

			if cellCount >= 3 {
				numStr := strings.TrimSpace(cells.Eq(0).Text())
				word := strings.TrimSpace(cells.Eq(1).Text())

				var translation string
				if cellCount >= 4 {
					translation = strings.TrimSpace(cells.Eq(3).Text())
					if translation == "" {
						translation = lastTranslation
					}
				} else if cellCount == 3 {
					translation = lastTranslation
				}

				if _, err := strconv.Atoi(numStr); err == nil && word != "" && translation != "" {
					lastTranslation = translation
					wordMap := make(map[string]string)
					wordMap[word] = translation
					wordsDict[numStr] = wordMap
				}
			}
		})
	})

	// Сохраняем в файл
	jsonData, err := json.MarshalIndent(wordsDict, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при создании JSON: %w", err)
	}

	err = os.WriteFile(wordsFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении файла: %w", err)
	}

	fmt.Printf("Успешно спарсено %d слов\n", len(wordsDict))
	return nil
}

// loadWords загружает слова из words.json
func loadWords() (map[string]map[string]string, error) {
	data, err := os.ReadFile(wordsFile)
	if err != nil {
		return nil, fmt.Errorf("файл %s не найден. Запустите с флагом --parse=1", wordsFile)
	}

	var wordsDict map[string]map[string]string
	err = json.Unmarshal(data, &wordsDict)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении JSON: %w", err)
	}

	return wordsDict, nil
}

// loadUsedWords загружает использованные слова из used_words.json
func loadUsedWords() (map[string]bool, error) {
	data, err := os.ReadFile(usedWordsFile)
	if err != nil {
		// Если файл не существует, создаем пустой словарь
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	var usedWords map[string]bool
	err = json.Unmarshal(data, &usedWords)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении JSON: %w", err)
	}

	return usedWords, nil
}

// saveUsedWords сохраняет использованные слова в used_words.json
func saveUsedWords(usedWords map[string]bool) error {
	jsonData, err := json.MarshalIndent(usedWords, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при создании JSON: %w", err)
	}

	err = os.WriteFile(usedWordsFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении файла: %w", err)
	}

	return nil
}

// getRandomWords получает n случайных слов из словаря, исключая использованные
func getRandomWords(wordsDict map[string]map[string]string, usedWords map[string]bool, n int) (map[string]map[string]string, error) {
	// Получаем список неиспользованных слов
	unusedNumbers := make([]string, 0)
	for num := range wordsDict {
		if !usedWords[num] {
			unusedNumbers = append(unusedNumbers, num)
		}
	}

	// Проверяем, есть ли доступные слова
	if len(unusedNumbers) == 0 {
		return nil, fmt.Errorf("все слова уже использованы")
	}

	// Если доступных слов меньше, чем нужно, используем все доступные
	if len(unusedNumbers) < n {
		n = len(unusedNumbers)
		fmt.Printf("Внимание: доступно только %d неиспользованных слов\n", n)
	}

	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())

	// Перемешиваем список
	rand.Shuffle(len(unusedNumbers), func(i, j int) {
		unusedNumbers[i], unusedNumbers[j] = unusedNumbers[j], unusedNumbers[i]
	})

	// Выбираем n случайных слов
	selectedWords := make(map[string]map[string]string)
	selectedNumbers := make([]string, 0)
	for i := 0; i < n; i++ {
		num := unusedNumbers[i]
		selectedWords[num] = wordsDict[num]
		selectedNumbers = append(selectedNumbers, num)
		usedWords[num] = true
	}

	// Сохраняем обновленный список использованных слов
	err := saveUsedWords(usedWords)
	if err != nil {
		return nil, fmt.Errorf("ошибка при сохранении использованных слов: %w", err)
	}

	return selectedWords, nil
}
