package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
	"github.com/joho/godotenv"
)

type AccessLog struct {
	IP           string
	Port         string
	Timestamp    string
	Method       string
	Path         string
	Protocol     string
	StatusCode   int
	BodyBytes    int
	Referer      string
	UserAgent    string
	ResponseTime int // マイクロ秒
}

type Stats struct {
	Count        int
	TotalTime    int64
	MaxTime      int
	MinTime      int
	SlowRequests int // 1秒以上
}

var stats = make(map[string]*Stats)
var globalStats = &Stats{MinTime: 999999999}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	logFile := os.Getenv("LOGDIR")
	if logFile == "" {
		log.Fatal("Not read ENV")
	}

	file, err := os.Open(logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// ファイルの最後まで移動（既存のログをスキップ）
	file.Seek(0, os.SEEK_END)

	// 修正した正規表現パターン
	// IPアドレス ポート - - [日時] "メソッド パス プロトコル" ステータス バイト数 "Referer" "User-Agent" レスポンスタイム
	pattern := `^(\S+) (\d+) - - \[([^\]]+)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "([^"]*)" "([^"]*)" (\d+)`
	re := regexp.MustCompile(pattern)

	fmt.Println("ログファイルを監視中... (Ctrl+C で終了)")
	fmt.Println("==========================================")

	// 統計情報を定期的に表示
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			printStats()
		}
	}()

	scanner := bufio.NewScanner(file)
	for {
		for scanner.Scan() {
			line := scanner.Text()
			matches := re.FindStringSubmatch(line)

			if matches != nil {
				statusCode, _ := strconv.Atoi(matches[7])
				bodyBytes, _ := strconv.Atoi(matches[8])
				responseTime, _ := strconv.Atoi(matches[11])

				accessLog := AccessLog{
					IP:           matches[1],
					Port:         matches[2],
					Timestamp:    matches[3],
					Method:       matches[4],
					Path:         matches[5],
					Protocol:     matches[6],
					StatusCode:   statusCode,
					BodyBytes:    bodyBytes,
					Referer:      matches[9],
					UserAgent:    matches[10],
					ResponseTime: responseTime,
				}

				// 統計更新
				updateStats(accessLog.Path, accessLog.ResponseTime)

				// リアルタイム表示
				responseTimeMs := float64(accessLog.ResponseTime) / 1000.0

				color := "🟢"
				if responseTimeMs > 1000 {
					color = "🔴"
				} else if responseTimeMs > 500 {
					color = "🟡"
				}

				fmt.Printf("[%s] %s %s %s:%s %s - Status: %d, Response: %.2f ms (%d μs)\n",
					time.Now().Format("15:04:05"),
					color,
					accessLog.Method,
					accessLog.IP,
					accessLog.Port,
					accessLog.Path,
					accessLog.StatusCode,
					responseTimeMs,
					accessLog.ResponseTime)
			}
		}

		time.Sleep(100 * time.Millisecond)
		scanner = bufio.NewScanner(file)
	}
}

func updateStats(path string, responseTime int) {
	// パス別統計
	if _, ok := stats[path]; !ok {
		stats[path] = &Stats{MinTime: 999999999}
	}
	s := stats[path]
	s.Count++
	s.TotalTime += int64(responseTime)
	if responseTime > s.MaxTime {
		s.MaxTime = responseTime
	}
	if responseTime < s.MinTime {
		s.MinTime = responseTime
	}
	if responseTime > 1000000 { // 1秒以上
		s.SlowRequests++
	}

	// 全体統計
	globalStats.Count++
	globalStats.TotalTime += int64(responseTime)
	if responseTime > globalStats.MaxTime {
		globalStats.MaxTime = responseTime
	}
	if responseTime < globalStats.MinTime {
		globalStats.MinTime = responseTime
	}
	if responseTime > 1000000 {
		globalStats.SlowRequests++
	}
}

func printStats() {
	if globalStats.Count == 0 {
		return
	}

	fmt.Println("\n========== 統計情報 (直近30秒) ==========")
	fmt.Printf("全体: リクエスト数=%d, 平均=%.2f ms, 最大=%.2f ms, 最小=%.2f ms, 遅延リクエスト=%d\n",
		globalStats.Count,
		float64(globalStats.TotalTime)/float64(globalStats.Count)/1000.0,
		float64(globalStats.MaxTime)/1000.0,
		float64(globalStats.MinTime)/1000.0,
		globalStats.SlowRequests)

	fmt.Println("\nパス別統計 (Top 10):")

	// パス別でソートして上位10件を表示
	type PathStat struct {
		Path string
		Stat *Stats
	}
	var pathStats []PathStat
	for path, stat := range stats {
		pathStats = append(pathStats, PathStat{path, stat})
	}

	// リクエスト数でソート
	for i := 0; i < len(pathStats); i++ {
		for j := i + 1; j < len(pathStats); j++ {
			if pathStats[j].Stat.Count > pathStats[i].Stat.Count {
				pathStats[i], pathStats[j] = pathStats[j], pathStats[i]
			}
		}
	}

	limit := 10
	if len(pathStats) < limit {
		limit = len(pathStats)
	}

	for i := 0; i < limit; i++ {
		path := pathStats[i].Path
		stat := pathStats[i].Stat
		avg := float64(stat.TotalTime) / float64(stat.Count) / 1000.0
		fmt.Printf("  %s: 回数=%d, 平均=%.2f ms, 最大=%.2f ms, 最小=%.2f ms\n",
			path, stat.Count, avg,
			float64(stat.MaxTime)/1000.0,
			float64(stat.MinTime)/1000.0)
	}
	fmt.Println("==========================================\n")

	// 統計をリセット
	stats = make(map[string]*Stats)
	globalStats = &Stats{MinTime: 999999999}
}
