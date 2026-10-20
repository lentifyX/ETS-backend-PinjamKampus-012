package main

import (
	"errors"
	"sync"
)

type SumberDaya struct {
	ID    int    `json:"id"`
	Nama  string `json:"nama"`
	Jenis string `json:"jenis"`
	Stok  int    `json:"stok"`
}

type Peminjaman struct {
	ID           int    `json:"id"`
	SumberDayaID int    `json:"sumber_daya_id"`
	Peminjam     string `json:"peminjam"`
	Tanggal      string `json:"tanggal"`
	SlotMulai    int    `json:"slot_mulai"`
	SlotSelesai  int    `json:"slot_selesai"`
	Jumlah       int    `json:"jumlah"`
	Status       string `json:"status"`
	Alasan       string `json:"alasan"` // persona C: terisi saat ditolak
}

// Status peminjaman (persona C).
const (
	statusDiajukan  = "diajukan"
	statusDisetujui = "disetujui"
	statusDitolak   = "ditolak"
	statusSelesai   = "selesai"
	statusDibatalkan = "dibatalkan"
)

type filterPeminjaman struct {
	SumberDayaID *int
	Tanggal      *string
	Status       *string
}

var (
	errSumberDayaTidakAda = errors.New("sumber_daya_id tidak menunjuk sumber daya yang ada")
	errJumlahMelebihiStok = errors.New("jumlah melebihi stok sumber daya")
	errPeminjamanTidakAda = errors.New("peminjaman tidak ditemukan")
	errBentrok            = errors.New("kapasitas sumber daya pada slot tersebut tidak mencukupi")
	errStatusTidakSesuai  = errors.New("status peminjaman saat ini tidak mengizinkan aksi tersebut")
)

type Store struct {
	mu                   sync.Mutex
	sumberDaya           []SumberDaya 
	peminjaman           []Peminjaman 
	idPeminjamanTerakhir int
}

func NewStore() *Store {
	return &Store{}
}

// sumber daya (B1) 

func (s *Store) TambahSumberDaya(nama, jenis string, stok int) SumberDaya {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd := SumberDaya{ID: len(s.sumberDaya) + 1, Nama: nama, Jenis: jenis, Stok: stok}
	s.sumberDaya = append(s.sumberDaya, sd)
	return sd
}

func (s *Store) DaftarSumberDaya() []SumberDaya {
	s.mu.Lock()
	defer s.mu.Unlock()

	hasil := make([]SumberDaya, len(s.sumberDaya))
	copy(hasil, s.sumberDaya)
	return hasil
}

func (s *Store) AmbilSumberDaya(id int) (SumberDaya, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cariSumberDaya(id)
}

func (s *Store) cariSumberDaya(id int) (SumberDaya, bool) {
	if id < 1 || id > len(s.sumberDaya) {
		return SumberDaya{}, false
	}
	return s.sumberDaya[id-1], true
}

// aturan bentrok (B5) 
// menahanSlot menentukan status mana yang menghabiskan kapasitas slot.
// Persona C: diajukan dan disetujui menahan slot; ditolak dan selesai melepaskannya.

func menahanSlot(status string) bool {
	return status == statusDiajukan || status == statusDisetujui
}

func (s *Store) unitTerpakai(sdID int, tanggal string, slot, kecualiID int) int {
	total := 0
	for _, p := range s.peminjaman {
		if p.ID == kecualiID || p.SumberDayaID != sdID || p.Tanggal != tanggal || !menahanSlot(p.Status) {
			continue
		}
		if slot >= p.SlotMulai && slot <= p.SlotSelesai {
			total += p.Jumlah
		}
	}
	return total
}

func (s *Store) adaBentrok(p Peminjaman, stok, kecualiID int) bool {
	for slot := p.SlotMulai; slot <= p.SlotSelesai; slot++ {
		if s.unitTerpakai(p.SumberDayaID, p.Tanggal, slot, kecualiID)+p.Jumlah > stok {
			return true
		}
	}
	return false
}

func (s *Store) periksaAturan(p Peminjaman, kecualiID int) error {
	sd, ok := s.cariSumberDaya(p.SumberDayaID)
	if !ok {
		return errSumberDayaTidakAda
	}
	if p.Jumlah > sd.Stok {
		return errJumlahMelebihiStok
	}
	if s.adaBentrok(p, sd.Stok, kecualiID) {
		return errBentrok
	}
	return nil
}

// peminjaman (B2, B3, B6)

func (s *Store) indeksPeminjaman(id int) int {
	for i, p := range s.peminjaman {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) TambahPeminjaman(p Peminjaman) (Peminjaman, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.periksaAturan(p, 0); err != nil {
		return Peminjaman{}, err
	}

	s.idPeminjamanTerakhir++
	p.ID = s.idPeminjamanTerakhir
	p.Status = statusDiajukan
	s.peminjaman = append(s.peminjaman, p)
	return p, nil
}

func (s *Store) UbahPeminjaman(id int, data Peminjaman) (Peminjaman, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indeksPeminjaman(id)
	if idx < 0 {
		return Peminjaman{}, errPeminjamanTidakAda
	}
	if err := s.periksaAturan(data, id); err != nil {
		return Peminjaman{}, err
	}

	data.ID = id
	data.Status = s.peminjaman[idx].Status
	data.Alasan = s.peminjaman[idx].Alasan
	s.peminjaman[idx] = data
	return data, nil
}

func (s *Store) BatalkanPeminjaman(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indeksPeminjaman(id)
	if idx < 0 {
		return errPeminjamanTidakAda
	}
	if !menahanSlot(s.peminjaman[idx].Status) {
		return errStatusTidakSesuai
	}
	s.peminjaman[idx].Status = statusDibatalkan
	return nil
}

func (s *Store) DaftarPeminjaman(f filterPeminjaman) []Peminjaman {
	s.mu.Lock()
	defer s.mu.Unlock()

	hasil := []Peminjaman{}
	for _, p := range s.peminjaman {
		if f.SumberDayaID != nil && p.SumberDayaID != *f.SumberDayaID {
			continue
		}
		if f.Tanggal != nil && p.Tanggal != *f.Tanggal {
			continue
		}
		if f.Status != nil && p.Status != *f.Status {
			continue
		}
		hasil = append(hasil, p)
	}
	return hasil
}

func (s *Store) AmbilPeminjaman(id int) (Peminjaman, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indeksPeminjaman(id)
	if idx < 0 {
		return Peminjaman{}, false
	}
	return s.peminjaman[idx], true
}