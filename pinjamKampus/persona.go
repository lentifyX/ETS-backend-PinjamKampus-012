package main

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"
)

// Persona C: Admin himpunan (alur persetujuan) 

type tolakInput struct {
	Alasan *string `json:"alasan"`
}

func validasiAlasan(in tolakInput) (string, error) {
	if in.Alasan == nil {
		return "", errors.New("alasan wajib diisi")
	}
	alasan := strings.TrimSpace(*in.Alasan)
	if alasan == "" {
		return "", errors.New("alasan tidak boleh kosong")
	}
	if utf8.RuneCountInString(alasan) > 100 {
		return "", errors.New("alasan maksimal 100 karakter")
	}
	return alasan, nil
}

func (s *Store) UbahStatus(id int, dari, ke, alasan string) (Peminjaman, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indeksPeminjaman(id)
	if idx < 0 {
		return Peminjaman{}, errPeminjamanTidakAda
	}
	if s.peminjaman[idx].Status != dari {
		return Peminjaman{}, errStatusTidakSesuai
	}

	s.peminjaman[idx].Status = ke
	if alasan != "" {
		s.peminjaman[idx].Alasan = alasan
	}
	return s.peminjaman[idx], nil
}

func (s *server) setujui(w http.ResponseWriter, r *http.Request) {
	s.pindahStatus(w, r, statusDiajukan, statusDisetujui)
}

func (s *server) selesaikan(w http.ResponseWriter, r *http.Request) {
	s.pindahStatus(w, r, statusDisetujui, statusSelesai)
}

func (s *server) pindahStatus(w http.ResponseWriter, r *http.Request, dari, ke string) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	p, err := s.store.UbahStatus(id, dari, ke, "")
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) tolak(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if _, ok := s.store.AmbilPeminjaman(id); !ok {
		writeError(w, http.StatusNotFound, "peminjaman tidak ditemukan")
		return
	}

	var in tolakInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "body harus berupa JSON yang valid")
		return
	}
	alasan, err := validasiAlasan(in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := s.store.UbahStatus(id, statusDiajukan, statusDitolak, alasan)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}