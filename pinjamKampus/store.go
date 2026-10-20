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
}

type filterPeminjaman struct {
	SumberDayaID *int
	Tanggal      *string
	Status       *string
}

var (
	errSumberDayaTidakAda = errors.New("sumber_daya_id tidak menunjuk sumber daya yang ada")
	errJumlahMelebihiStok = errors.New("jumlah melebihi stok sumber daya")
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

// Sumber Daya (B1)

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

// peminjaman (B2, B3)

func (s *Store) TambahPeminjaman(p Peminjaman) (Peminjaman, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd, ok := s.cariSumberDaya(p.SumberDayaID)
	if !ok {
		return Peminjaman{}, errSumberDayaTidakAda
	}
	if p.Jumlah > sd.Stok {
		return Peminjaman{}, errJumlahMelebihiStok
	}

	s.idPeminjamanTerakhir++
	p.ID = s.idPeminjamanTerakhir
	p.Status = "aktif"
	s.peminjaman = append(s.peminjaman, p)
	return p, nil
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

	for _, p := range s.peminjaman {
		if p.ID == id {
			return p, true
		}
	}
	return Peminjaman{}, false
}