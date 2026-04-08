package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/eamonburns/gameshow-button-dashboard/internal/config"
)

// Intermediate data type for unmarshaling json webhook payload
type jsonData struct {
	ButtonId int `json:"button_id"`
}

type Data struct {
	Timestamp time.Time
	ButtonId  int
}

// Start an HTTP server to listen for webhooks.
//
// This function will block indefinitely, so it should be run in a goroutine.
func StartListening(addr string, webhookId string, cfg *config.Config, webhookCh chan<- Data) {
	http.Handle("POST /webhook/{id}", Handler{
		ID:  webhookId,
		Cfg: cfg,
		Ch:  webhookCh,
	})
	log.Fatalf("error: %v", http.ListenAndServe(addr, nil))
}

// http.Handler to listen for webhook requests and send them to a channel
type Handler struct {
	ID  string
	Cfg *config.Config
	Ch  chan<- Data
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
	if _, ok := h.Cfg.PlayerForButtonId(jd.ButtonId); !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "no player with button_id %d\n", jd.ButtonId)
		return
	}
	data := Data{
		ButtonId: jd.ButtonId,
	}
	log.Printf("data: %v\n", data)

	// TODO: Either validate that the player has not already buzzed-in, or try
	// to send the data to the channel until a timeout (rather than using the
	// `default` case).
	// Currently, there is a very small amount of time where player A
	// buzzes-in, the data is sent, player B buzzes-in and is rejected, and
	// then player A is found to have already buzzed-in
	select {
	case h.Ch <- data:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusTeapot)
	}
}
