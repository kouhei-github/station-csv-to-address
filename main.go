package main

import (
	"encoding/csv"
	"os"
	"station-to-prefecture/utils"
	"strings"
	"sync"
)

// 結果を格納する構造体
type Result struct {
	index   int
	address string
	err     error
}

func main() {
	records, err := readCSV()
	if err != nil {
		return
	}
	http := utils.NewRequest()

	// ワーカー数の設定（同時実行数）
	const numWorkers = 10

	// 結果を保存するスライス
	results := make([]string, len(records)-1)

	// ジョブチャネルの作成
	jobs := make(chan struct {
		index  int
		record []string
	}, len(records)-1)

	// 結果チャネルの作成
	resultsChan := make(chan Result, len(records)-1)

	// ワーカープールの作成
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(http, jobs, resultsChan, &wg)
	}

	// ジョブの投入
	for i, record := range records[1:] {
		jobs <- struct {
			index  int
			record []string
		}{i, record}
	}
	close(jobs)

	// 別のgoroutineで結果の収集
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// 結果の収集
	for result := range resultsChan {
		if result.err != nil {
			results[result.index] = "ERROR"
		} else {
			results[result.index] = result.address
		}
	}

	// 結果をCSVフォーマットに変換
	var prefs [][]string
	for _, result := range results {
		prefs = append(prefs, []string{result})
	}

	if err := writeCSV(prefs); err != nil {
		return
	}
}

// ワーカー関数
func worker(http utils.Request, jobs <-chan struct {
	index  int
	record []string
}, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		station, area := regex(job.record[0])
		stationRes, err := http.StationToAddress("https://express.heartrails.com/api/json?method=getStations&name=" + station)
		if err != nil {
			results <- Result{index: job.index, err: err}
			continue
		}

		var postal string
		if area != "" {
			ourStation, err := stationRes.Response.GetLine(area)
			if err != nil || ourStation == nil {
				results <- Result{index: job.index, address: "ERROR"}
				continue
			}
			postal = ourStation.Postal
		} else {
			if len(stationRes.Response.Stations) == 0 {
				results <- Result{index: job.index, address: "ERROR"}
				continue
			}
			postal = stationRes.Response.Stations[0].Postal
		}

		addressRes, err := http.PostalCodeToPref("https://jp-postal-code-api.ttskch.com/api/v1/" + postal + ".json")
		if err != nil {
			results <- Result{index: job.index, err: err}
			continue
		}

		address := addressRes.GetFormattedJapaneseAddress()
		results <- Result{
			index:   job.index,
			address: address,
		}
	}
}

// readCSV CSVディレクトリからinput.csvファイルを読み込む
func readCSV() ([][]string, error) {
	file, err := os.Open("csv/input.csv")
	if err != nil {
		return nil, &utils.CustomError{Message: err.Error()}
	}
	defer file.Close()

	reader := csv.NewReader(file)
	return reader.ReadAll()
}

// regex splits the input string into two parts at the first occurrence of '(', removes surrounding spaces and also trims end ')'.
func regex(content string) (string, string) {
	splits := strings.Split(content, "(")
	if len(splits) == 2 {
		return strings.TrimSpace(splits[0]), strings.TrimSpace(strings.ReplaceAll(splits[1], ")", ""))
	}
	return strings.TrimSpace(splits[0]), ""
}

// writeCSV writes a 2D string slice into a CSV format and returns an error if writing fails.
func writeCSV(records [][]string) error {
	// Create or truncate the output file
	file, err := os.Create("csv/output.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	// Write BOM for UTF-8
	_, err = file.Write([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		return err
	}

	// Create a new CSV writer
	writer := csv.NewWriter(file)
	writer.UseCRLF = true // Windows対応
	defer writer.Flush()

	// Write all records to the CSV file
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
