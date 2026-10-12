package handlers

import (
	"net/http"
	"time"

	"github.com/Pondaaaaaa/GO/practices/pz02-myapp/utils"
)

func Ping(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ping" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	payload := map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	if err := utils.WriteJSON(w, http.StatusOK, payload); err != nil {
		utils.Error("write ping response", "error", err)
	}
}
