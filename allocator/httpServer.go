package allocator

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

// GET /counter
func httpServerStart(port int) {
	http.HandleFunc("/counter", getCounterInfo)
	http.HandleFunc("/updateConfigMap", updateConfigMapHandler)

	log.Info().Msgf("Server is running on: %d", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		log.Error().Msgf("HTTP Listen error: %v", err)
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

// 修改configMap的HTTP处理函数
func updateConfigMapHandler(w http.ResponseWriter, r *http.Request) {
	var newConfigMap ServiceConfigMap

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&newConfigMap); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	// 在修改configMap之前加锁
	configMapMutex.Lock()
	defer configMapMutex.Unlock()

	// 更新configMap
	configMap = newConfigMap

	log.Info().Msg("ConfigMap updated successfully")

	w.WriteHeader(http.StatusOK)
}
