package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 嵌入前端静态文件
//
//go:embed static/*
var staticFS embed.FS

// LogFile 文件信息结构体
type LogFile struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime string `json:"mtime"`
}

// SearchMatch 搜索匹配结果
type SearchMatch struct {
	LineNo  int    `json:"lineNo"`
	Content string `json:"content"`
}

// ReadChunkResp 读取返回结构
type ReadChunkResp struct {
	Lines      []string `json:"lines"`
	TotalLines int      `json:"totalLines"`
}

// 列出当前目录所有 .log 文件
func listLogFiles(w http.ResponseWriter, r *http.Request) {
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	logDir := fmt.Sprintf("%s/logs", wd)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		http.Error(w, "读取logs目录失败: "+err.Error(), 500)
		return
	}
	var files []LogFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, LogFile{
			Name:  name,
			Size:  info.Size(),
			MTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	// 按修改时间倒序
	sort.Slice(files, func(i, j int) bool {
		t1, _ := time.Parse("2006-01-02 15:04:05", files[i].MTime)
		t2, _ := time.Parse("2006-01-02 15:04:05", files[j].MTime)
		return t1.After(t2)
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

// 读取整个日志文件，一次性返回所有行
func readLogChunk(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, "获取工作目录失败", 500)
		return
	}
	filePath := fmt.Sprintf("%s/logs/%s", wd, fileName)
	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "打开文件失败:"+err.Error(), 500)
		return
	}
	defer f.Close()

	// 使用 bufio.Scanner 按行读取，支持超长行
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	resp := ReadChunkResp{
		Lines:      lines,
		TotalLines: len(lines),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 全文搜索关键字
func searchLogFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"matches": []SearchMatch{}})
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, "获取目录失败", 500)
		return
	}
	filePath := fmt.Sprintf("%s/logs/%s", wd, fileName)
	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var matches []SearchMatch
	lineNo := 0
	lowerKey := strings.ToLower(keyword)
	for scanner.Scan() {
		lineNo++
		text := scanner.Text()
		if strings.Contains(strings.ToLower(text), lowerKey) {
			matches = append(matches, SearchMatch{
				LineNo:  lineNo,
				Content: text,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"matches": matches})
}

// 新增：日志文件下载接口
func downloadLogFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		http.Error(w, "缺少文件名称参数file", 400)
		return
	}
	// 简单路径防护，禁止上级目录跳转
	if strings.Contains(fileName, "..") {
		http.Error(w, "非法文件名", 403)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, "获取工作目录失败", 500)
		return
	}
	filePath := fmt.Sprintf("%s/logs/%s", wd, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "文件不存在或打开失败: "+err.Error(), 404)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "读取文件信息失败", 500)
		return
	}

	// 设置下载响应头
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	// 流式输出文件内容，支持超大日志
	_, err = io.Copy(w, file)
	if err != nil {
		fmt.Printf("文件下载流出错: %v\n", err)
	}
}

func main() {
	// 静态资源路由
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 根路由返回前端页面
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	// API路由
	http.HandleFunc("/api/log/files", listLogFiles)       // 列出日志文件
	http.HandleFunc("/api/log/read", readLogChunk)        // 读取日志文件
	http.HandleFunc("/api/log/search", searchLogFile)     // 关键字搜索
	http.HandleFunc("/api/log/download", downloadLogFile) // 下载接口

	fmt.Println("日志查看服务启动成功,地址:http://127.0.0.1:8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		fmt.Println("服务启动失败：", err)
		os.Exit(1)
	}
}
