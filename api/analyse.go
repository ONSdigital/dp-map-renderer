package api

import (
	"net/http"

	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/dp-map-renderer/analyser"
	"encoding/json"
)

func (api *RendererAPI) analyseData(w http.ResponseWriter, r *http.Request) {

	log.Debug("analyseData", log.Data{"headers": r.Header})
	request, err := models.CreateAnalyseRequest(r.Body)
	if err != nil {
		log.Error(err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = request.ValidateAnalyseRequest(); err != nil {
		log.Error(err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := analyser.AnalyseData(request)
	bytes, err := json.Marshal(response)
	if err != nil {
		log.Error(err, log.Data{})
		setErrorCode(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bytes)
	if err != nil {
		log.Error(err, log.Data{})
		setErrorCode(w, err)
		return
	}

}
