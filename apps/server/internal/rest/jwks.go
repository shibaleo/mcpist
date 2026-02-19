package rest

import (
	"encoding/base64"
	"net/http"

	"mcpist/server/internal/auth"
)

// RegisterJWKS registers the JWKS endpoint on the mux.
// Called separately since it doesn't need the Handler struct.
func RegisterJWKS(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/jwks.json", handleJWKS)
}

func handleJWKS(w http.ResponseWriter, r *http.Request) {
	kp := auth.GetKeyPair()
	if kp == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"keys": []interface{}{},
		})
		return
	}

	jwk := map[string]interface{}{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(kp.PublicKey),
		"kid": kp.KID,
		"use": "sig",
		"alg": "EdDSA",
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keys": []interface{}{jwk},
	})
}
