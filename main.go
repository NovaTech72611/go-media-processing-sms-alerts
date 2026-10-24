package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type ingestRequest struct {
	ID           string `json:"id"`
	CreatorPhone string `json:"creator_phone"`
}

func main() {
	client, err := NewSMSClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var in ingestRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.ID == "" {
			http.Error(w, "invalid asset", http.StatusBadRequest)
			return
		}
		asset := Asset{ID: in.ID, CreatorPhone: in.CreatorPhone, Processed: true}
		messageID := ""
		if shouldNotify(asset) {
			messageID, err = client.Send(context.Background(), asset.CreatorPhone, "Asset "+asset.ID+" is ready", "asset-"+asset.ID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"asset_id": asset.ID, "message_id": messageID, "processed_at": time.Now().UTC().Format(time.RFC3339)})
	})
	log.Println("media SMS service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
