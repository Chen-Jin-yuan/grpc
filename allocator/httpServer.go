package allocator

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strconv"
)

// GET /svc-info?name=exam_svc
// GET /counter?key=request-type
func httpServerStart(port int) {
	http.HandleFunc("/svc-info", getSvcConfig)
	//http.HandleFunc("/counter", getCounterInfo)

	log.Info().Msgf("Server is running on: %d", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		log.Error().Msgf("HTTP Listen error: %v", err)
		return
	}
}

func getSvcConfig(w http.ResponseWriter, r *http.Request) {
	// 从查询参数中获取文件名参数
	svcName := r.URL.Query().Get("name")

	if svcName == "" {
		http.Error(w, "Missing name parameter", http.StatusBadRequest)
		return
	}

	fileName := svcName + ".json"

	// 检查文件是否存在
	_, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "target service not found", http.StatusNotFound)
			return
		}
	}

	// 读取 JSON 文件
	file, err := os.Open(fileName)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		http.Error(w, "Error decoding JSON", http.StatusInternalServerError)
		return
	}

	// 格式化 JSON 数据
	formattedJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}

	// 构建 HTML 页面，包括标题
	//	htmlPage := `
	//<!DOCTYPE html>
	//<html>
	//<head>
	//	<title>` + svcName + ` JSON Data</title>
	//</head>
	//<body>
	//	<h1>` + svcName + `</h1>
	//	<pre>` + string(formattedJSON) + `</pre>
	//</body>
	//</html>`
	//
	//	// 返回 HTML 响应
	//	w.Header().Set("Content-Type", "text/html")
	//	w.WriteHeader(http.StatusOK)
	//	_, err = w.Write([]byte(htmlPage))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(formattedJSON)
	if err != nil {
		return
	}
}

//func getCounterInfo(w http.ResponseWriter, r *http.Request) {
//	if rcs == nil {
//		//w.Header().Set("Content-Type", "text/html")
//		//w.WriteHeader(http.StatusOK)
//		//_, _ = w.Write([]byte("The counter is not enabled or there are still no requests."))
//		//return
//		rcs = monitor.NewRequestCounters()
//	}
//	// 获取查询参数 key
//	key := r.URL.Query().Get("key")
//
//	// 如果 key 不为空，返回指定 key 的计数信息
//	if key != "" {
//		if counterData, err := rcs.GetCounter(key).ToJSON(); err == nil {
//			jsonData, err := json.Marshal(counterData)
//			if err != nil {
//				http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
//				return
//			}
//			w.Header().Set("Content-Type", "application/json")
//			w.WriteHeader(http.StatusOK)
//			_, err = w.Write(jsonData)
//			if err != nil {
//				return
//			}
//		} else {
//			http.Error(w, "Key of Counter, to json error", http.StatusInternalServerError)
//		}
//	} else {
//		// 返回整个 counterMap
//		if allCountersData, err := rcs.ToJSON(); err == nil {
//			jsonData, err := json.Marshal(allCountersData)
//			if err != nil {
//				http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
//				return
//			}
//			w.Header().Set("Content-Type", "application/json")
//			w.WriteHeader(http.StatusOK)
//			_, err = w.Write(jsonData)
//			if err != nil {
//				return
//			}
//		} else {
//			http.Error(w, "all Counters, to json error", http.StatusInternalServerError)
//		}
//
//	}
//}
