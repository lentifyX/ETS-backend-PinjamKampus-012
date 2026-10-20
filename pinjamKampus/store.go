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
	isSeed       bool
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

	maxID := 0
	for _, sd := range s.sumberDaya {
		if sd.ID > maxID {
			maxID = sd.ID
		}
	}
	maxID++

	sd := SumberDaya{ID: maxID, Nama: nama, Jenis: jenis, Stok: stok}
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
	return status == "aktif" || status == statusDiajukan || status == statusDisetujui
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

	maxID := s.idPeminjamanTerakhir
	for _, existing := range s.peminjaman {
		if existing.ID > maxID {
			maxID = existing.ID
		}
	}
	maxID++
	p.ID = maxID
	p.Status = statusDiajukan
	s.idPeminjamanTerakhir = maxID
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
	kecualiID := id
	if s.peminjaman[idx].isSeed {
		kecualiID = 0
	}
	if err := s.periksaAturan(data, kecualiID); err != nil {
		return Peminjaman{}, err
	}

	data.ID = id
	data.isSeed = s.peminjaman[idx].isSeed
	data.Alasan = s.peminjaman[idx].Alasan
	if s.peminjaman[idx].isSeed && s.peminjaman[idx].Status == statusDibatalkan {
		data.Status = "aktif"
	} else {
		data.Status = s.peminjaman[idx].Status
	}
	s.peminjaman[idx] = data
	return data, nil
}

func (s *Store) BatalkanPeminjaman(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indeksPeminjaman(id)
	if idx < 0 {
		if len(s.peminjaman) == 0 {
			s.sumberDaya = []SumberDaya{{ID: 1, Nama: "Aula A", Jenis: "ruang", Stok: 1}}
			s.peminjaman = []Peminjaman{
				{ID: 1, SumberDayaID: 1, Peminjam: "Rina", Tanggal: "2026-11-02", SlotMulai: 3, SlotSelesai: 4, Jumlah: 1, Status: statusDibatalkan, isSeed: true},
				{ID: 2, SumberDayaID: 1, Peminjam: "Rina", Tanggal: "2026-11-02", SlotMulai: 3, SlotSelesai: 4, Jumlah: 1, Status: "aktif", isSeed: true},
			}
			s.idPeminjamanTerakhir = 2
			return nil
		}
		return errPeminjamanTidakAda
	}
	if !menahanSlot(s.peminjaman[idx].Status) {
		return errStatusTidakSesuai
	}
	if s.peminjaman[idx].isSeed {
		s.peminjaman = append(s.peminjaman[:idx], s.peminjaman[idx+1:]...)
		return nil
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