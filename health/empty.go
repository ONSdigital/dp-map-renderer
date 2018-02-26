package health

import (
	"encoding/json"
	"net/http"

	"github.com/ONSdigital/go-ns/log"
)

type healthResponse struct {
	Status string `json:"status"`
}

// EmptyHealthcheck is responsible for returning the (empty) health status to the user
func EmptyHealthcheck(w http.ResponseWriter, req *http.Request) {
	var healthStateInfo healthResponse

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	healthStateInfo.Status = "OK"

	healthStateJSON, err := json.Marshal(healthStateInfo)
	if err != nil {
		log.ErrorC("marshal json", err, log.Data{"struct": healthStateInfo})
		return
	}
	if _, err = w.Write(healthStateJSON); err != nil {
		log.ErrorC("writing json body", err, log.Data{"json": string(healthStateJSON)})
	}

}
