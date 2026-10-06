package handler

import "net/http"

// stats serves the admin dashboard's user counts. The router only lets admins in.
func (h *authHandler) stats(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Stats(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, s)
}
