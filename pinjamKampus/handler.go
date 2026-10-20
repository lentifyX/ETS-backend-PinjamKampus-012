package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type server struct {
	store *Store
}

func NewHandler() http.Handler {
	s := &server{store: NewStore()}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /sumber-daya", s.buatSumberDaya)
	mux.HandleFunc("GET /sumber-daya", s.daftarSumberDaya)
	mux.HandleFunc("GET /sumber-daya/{id}", s.detailSumberDaya)

	mux.HandleFunc("POST /peminjaman", s.buatPeminjaman)
	mux.HandleFunc("GET /peminjaman", s.daftarPeminjaman)
	mux.HandleFunc("GET /peminjaman/{id}", s.detailPeminjaman)
	mux.HandleFunc("PUT /peminjaman/{id}", s.ubahPeminjaman)
	mux.HandleFunc("DELETE /peminjaman/{id}", s.batalkanPeminjaman)
	mux.HandleFunc("POST /peminjaman/{id}/setujui", s.setujui)
	mux.HandleFunc("POST /peminjaman/{id}/tolak", s.tolak)
	mux.HandleFunc("POST /peminjaman/{id}/selesaikan", s.selesaikan)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "endpoint tidak ditemukan")
	})
	return mux
}

// helper JSON & Error (B4)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, pesan string) {
	writeJSON(w, status, map[string]string{"error": pesan})
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errSumberDayaTidakAda), errors.Is(err, errJumlahMelebihiStok):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errPeminjamanTidakAda):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errBentrok), errors.Is(err, errStatusTidakSesuai):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body memuat data tambahan")
	}
	return nil
}

func idDariPath(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id harus berupa angka")
		return 0, false
	}
	return id, true
}

// handler sumber daya (B1)

func (s *server) buatSumberDaya(w http.ResponseWriter, r *http.Request) {
	var in sumberDayaInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "body harus berupa JSON yang valid")
		return
	}

	nama, jenis, stok, err := validasiSumberDaya(in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sd := s.store.TambahSumberDaya(nama, jenis, stok)
	writeJSON(w, http.StatusCreated, sd)
}

func (s *server) daftarSumberDaya(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.DaftarSumberDaya())
}

func (s *server) detailSumberDaya(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id harus berupa angka")
		return
	}

	sd, ok := s.store.AmbilSumberDaya(id)
	if !ok {
		writeError(w, http.StatusNotFound, "sumber daya tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, sd)
}

// handler peminjaman (B2, B3, B6)

func (s *server) buatPeminjaman(w http.ResponseWriter, r *http.Request) {
	var in peminjamanInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "body harus berupa JSON yang valid")
		return
	}

	data, err := validasiPeminjaman(in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := s.store.TambahPeminjaman(data)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *server) daftarPeminjaman(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f filterPeminjaman

	if v := q.Get("sumber_daya_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "sumber_daya_id harus berupa angka")
			return
		}
		f.SumberDayaID = &id
	}
	if v := q.Get("tanggal"); v != "" {
		f.Tanggal = &v
	}
	if v := q.Get("status"); v != "" {
		f.Status = &v
	}

	writeJSON(w, http.StatusOK, s.store.DaftarPeminjaman(f))
}

func (s *server) detailPeminjaman(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}

	p, ok := s.store.AmbilPeminjaman(id)
	if !ok {
		writeError(w, http.StatusNotFound, "peminjaman tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) ubahPeminjaman(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if _, ok := s.store.AmbilPeminjaman(id); !ok {
		writeError(w, http.StatusNotFound, "peminjaman tidak ditemukan")
		return
	}

	var in peminjamanInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "body harus berupa JSON yang valid")
		return
	}
	data, err := validasiPeminjaman(in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := s.store.UbahPeminjaman(id, data)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) batalkanPeminjaman(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.store.BatalkanPeminjaman(id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}