package allocator

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
)

// GET /svc-info?name=exam_svc
// GET /counter
func httpServerStart(port int) {
	http.HandleFunc("/svc-info", getSvcConfig)
	http.HandleFunc("/counter", getCounterInfo)
	http.HandleFunc("/modify-group", modifyGroupNumber)

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

func getCounterInfo(w http.ResponseWriter, r *http.Request) {
	if jsonData, err := getHandlerJSONData(); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(jsonData)
		if err != nil {
			return
		}
	} else {
		http.Error(w, "to json error", http.StatusInternalServerError)
	}
}

// groupData: svcName -> [group1 number, group2 number, ...]
func modifyGroupNumber(w http.ResponseWriter, r *http.Request) {
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// 解析 JSON 数据
	var groupData map[string][]int
	if err := json.Unmarshal(body, &groupData); err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	// 处理接收到的数据
	configMu.Lock()
	for svcName, numbers := range groupData {
		_, ok := config[svcName]
		if !ok {
			// 这个服务不存在
			continue
		}
		groupNames := make([]string, 0, len(config[svcName].Group))
		for groupName := range config[svcName].Group {
			groupNames = append(groupNames, groupName)
		}
		// 对 group 进行排序
		sort.Strings(groupNames)

		// 如果 numbers 长度小于组数，后面的组不会修改；反之多余的 numbers 不会被用到
		minLen := min(len(numbers), len(groupNames))
		for i := 0; i < minLen; i++ {
			// config[svcName].Group[groupNames[i]].Number = numbers[i]
			// serviceConfig 结构体是值类型，无法直接赋值，需要创建一个新的；groupInfo 也是这样
			sc := config[svcName]
			gi := sc.Group[groupNames[i]]
			gi.Number = numbers[i]
			// 重新赋值
			sc.Group[groupNames[i]] = gi
			config[svcName] = sc
		}
	}
	configMu.Unlock()
	// 返回响应
	w.WriteHeader(http.StatusOK)
}
