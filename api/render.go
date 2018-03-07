package api

import (
	"net/http"

	"errors"

	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/dp-map-renderer/renderer"
	"github.com/ONSdigital/go-ns/log"
	"github.com/gorilla/mux"
)

// Error types
var (
	internalError     = "Failed to process the request due to an internal error"
	badRequest        = "Bad request - Invalid request body"
	unknownRenderType = "Unknown render type"
	statusBadRequest  = "bad request"
)

// Content types
var (
	contentSVG  = "image/svg+xml"
	contentHTML = "text/html"
)

func (api *RendererAPI) renderMap(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	renderType := vars["render_type"]

	log.Debug("renderTable", log.Data{"headers": r.Header})
	renderRequest, err := models.CreateRenderRequest(r.Body)
	if err != nil {
		log.Error(err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = renderRequest.ValidateRenderRequest(); err != nil {
		log.Error(err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var bytes []byte

	switch renderType {
	case "html":
		bytes, err = renderer.RenderHTML(renderRequest)
		setContentType(w, contentHTML)
	default:
		log.Error(errors.New("Unknown render type"), log.Data{"render_type": renderType})
		http.Error(w, unknownRenderType, http.StatusNotFound)
		return
	}

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

func setContentType(w http.ResponseWriter, contentType string) {
	w.Header().Set("Content-Type", contentType)
}

func setErrorCode(w http.ResponseWriter, err error) {
	log.Debug("error is", log.Data{"error": err})
	switch err.Error() {
	case "Bad request":
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	default:
		http.Error(w, internalError, http.StatusInternalServerError)
		return
	}
}
