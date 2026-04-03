package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Intermediate data type for unmarshaling json webhook payload
type jsonData struct {
	ButtonId uint `json:"button_id"`
}

type Data struct {
	Timestamp time.Time
	ButtonId  uint
}

// Start a webserver to listen for webhooks.
//
// This function will block indefinitely, so it should be run in a goroutine.
func StartListening(addr string, webhookId string, webhookCh chan<- Data) {
	http.Handle("POST /webhook/{id}", Handler{ID: webhookId, Ch: webhookCh})
	log.Print(http.ListenAndServe(addr, nil))
}

// http.Handler to listen for webhook requests and send them to a channel
type Handler struct {
	ID string
	Ch chan<- Data
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	webhookId := r.PathValue("id")
	if webhookId != h.ID {
		log.Printf("error: invalid webhook ID: %s\n", webhookId)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, "invalid webhook ID")
		return
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var jd jsonData
	if err := decoder.Decode(&jd); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%v\n", err)
		return
	}
	if jd.ButtonId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "invalid button_id")
		return
	}
	data := Data{
		ButtonId: jd.ButtonId,
	}
	log.Printf("data: %v\n", data)

	select {
	case h.Ch <- data:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusTeapot)
	}
}
