package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ---------------- Config ----------------
type Config struct {
	URLs                []string `yaml:"urls"`
	MinSpeedMbps        float64  `yaml:"min_speed"`
	MaxSpeedMbps        float64  `yaml:"max_speed"`
	MaxConcurrency      int      `yaml:"max_concurrency"`
	LogFile             string   `yaml:"log_file"`
	MinBytesPerDownload int64    `yaml:"min_bytes_per_download"` // 最小单次下载
	MaxBytesPerDownload int64    `yaml:"max_bytes_per_download"` // 最大单次下载
	MinIntervalSec      int      `yaml:"min_interval_sec"`       // 最小循环间隔（秒）
	MaxIntervalSec      int      `yaml:"max_interval_sec"`       // 最大循环间隔（秒）
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ---------------- Global Stats ----------------
var (
	totalBytes uint64
	statsLock  sync.Mutex
)

// ---------------- Rate-Limited Reader ----------------
type rateLimitedReader struct {
	reader  io.Reader
	limiter *rate.Limiter
}

func (r *rateLimitedReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		_ = r.limiter.WaitN(context.Background(), n)
	}
	return n, err
}

// ---------------- Downloader ----------------
func downloadFile(url string, limiter *rate.Limiter) error {
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// 伪造常见浏览器请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", url)
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 限速读取
	reader := &rateLimitedReader{
		reader:  resp.Body,
		limiter: limiter,
	}

	n, err := io.Copy(io.Discard, reader) // 读完整个文件
	statsLock.Lock()
	totalBytes += uint64(n)
	statsLock.Unlock()

	fmt.Printf("[%s] Downloaded %d bytes from %s (elapsed %s)\n",
		time.Now().Format("15:04:05"), n, url, time.Since(start))

	if err != nil {
		return fmt.Errorf("download error: %v", err)
	}
	return nil
}

// ---------------- Logger ----------------
func startLogger(logPath string) {
	os.MkdirAll(filepath.Dir(logPath), 0755)

	for {
		// 计算距离下一个整点的时间
		now := time.Now()
		next := now.Truncate(time.Hour).Add(time.Hour)
		//next := now.Add(time.Second * 10)
		sleepDuration := next.Sub(now)
		time.Sleep(sleepDuration)

		// 到整点写入日志
		statsLock.Lock()
		totalMB := float64(totalBytes) / (1024 * 1024)
		totalBytes = 0
		statsLock.Unlock()

		appendLog(logPath, fmt.Sprintf("[%s] Hourly traffic: %.2f MB\n",
			time.Now().Format("2006-01-02 15:04:05"), totalMB))
	}
}

func appendLog(path string, line string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("log open error:", err)
		return
	}
	defer f.Close()

	f.WriteString(line)

	content, _ := os.ReadFile(path)
	lines := 0
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	if lines > 1000 {
		newPath := fmt.Sprintf("%s.%d", path, time.Now().Unix())
		os.Rename(path, newPath)
		fmt.Printf("Log rotated: %s -> %s\n", path, newPath)
	}
}

func runDownloader() {
	rand.Seed(time.Now().UnixNano())

	cfg, err := loadConfig("config/conf.yaml")
	if err != nil {
		panic(err)
	}

	for k, v := range cfg.URLs {
		log.Printf("URL %d: %s", k+1, v)
	}
	go startLogger(cfg.LogFile)

	sem := make(chan struct{}, cfg.MaxConcurrency)

	for {
		if len(cfg.URLs) < 2 {
			fmt.Println("need at least 2 URLs")
			return
		}

		idx1 := rand.Intn(len(cfg.URLs))
		idx2 := rand.Intn(len(cfg.URLs))
		for idx2 == idx1 {
			idx2 = rand.Intn(len(cfg.URLs))
		}
		urls := []string{cfg.URLs[idx1], cfg.URLs[idx2]}

		for _, url := range urls {
			sem <- struct{}{}
			go func(u string) {
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("Recovered from panic:", r)
					}
				}()

				// 随机限速
				speed := cfg.MinSpeedMbps + rand.Float64()*(cfg.MaxSpeedMbps-cfg.MinSpeedMbps)
				bytesPerSec := (speed * 1024 * 1024) / 8 // Mbps -> B/s
				limiter := rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))

				fmt.Printf("[%s] Start %s @ %.1f Mbps\n",
					time.Now().Format("15:04:05"), u, speed)

				err := downloadFile(u, limiter)
				if err != nil {
					fmt.Printf("[%s] %s error: %v\n", time.Now().Format("15:04:05"), u, err)
				} else {
					fmt.Printf("[%s] %s done\n", time.Now().Format("15:04:05"), u)
				}
			}(url)
		}

		// 随机间隔
		durTime := time.Duration(cfg.MinIntervalSec+rand.Intn(cfg.MaxIntervalSec-cfg.MinIntervalSec+1)) * time.Second
		log.Printf("Sleep %s\n", durTime)
		time.Sleep(durTime)
	}
}

// ---------------- Main ----------------
func main() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Main panic recovered:", r)
					fmt.Println("Sleeping 30s before restart...")
					time.Sleep(30 * time.Second)
				}
			}()

			runDownloader()
		}()
	}
}
