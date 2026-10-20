package main

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	slotMin = 1
	slotMax = 8
)
type sumberDayaInput struct {
	Nama  *string `json:"nama"`
	Jenis *string `json:"jenis"`
	Stok  *int    `json:"stok"`
}

type peminjamanInput struct {
	SumberDayaID *int    `json:"sumber_daya_id"`
	Peminjam     *string `json:"peminjam"`
	Tanggal      *string `json:"tanggal"`
	SlotMulai    *int    `json:"slot_mulai"`
	SlotSelesai  *int    `json:"slot_selesai"`
	Jumlah       *int    `json:"jumlah"`
}

func validasiSumberDaya(in sumberDayaInput) (nama, jenis string, stok int, err error) {
	if in.Nama == nil || in.Jenis == nil || in.Stok == nil {
		return "", "", 0, errors.New("nama, jenis, dan stok wajib diisi")
	}

	nama = strings.TrimSpace(*in.Nama)
	if nama == "" {
		return "", "", 0, errors.New("nama tidak boleh kosong")
	}
	if utf8.RuneCountInString(nama) > 50 {
		return "", "", 0, errors.New("nama maksimal 50 karakter")
	}

	jenis = strings.TrimSpace(*in.Jenis)
	if jenis != "ruang" && jenis != "alat" {
		return "", "", 0, errors.New(`jenis harus "ruang" atau "alat"`)
	}

	stok = *in.Stok
	if stok < 1 {
		return "", "", 0, errors.New("stok minimal 1")
	}

	return nama, jenis, stok, nil
}

func validasiPeminjaman(in peminjamanInput) (Peminjaman, error) {
	if in.SumberDayaID == nil || in.Peminjam == nil || in.Tanggal == nil ||
		in.SlotMulai == nil || in.SlotSelesai == nil || in.Jumlah == nil {
		return Peminjaman{}, errors.New("sumber_daya_id, peminjam, tanggal, slot_mulai, slot_selesai, dan jumlah wajib diisi")
	}

	peminjam := strings.TrimSpace(*in.Peminjam)
	if peminjam == "" {
		return Peminjaman{}, errors.New("peminjam tidak boleh kosong")
	}
	if utf8.RuneCountInString(peminjam) > 50 {
		return Peminjaman{}, errors.New("peminjam maksimal 50 karakter")
	}

	tanggal := strings.TrimSpace(*in.Tanggal)
	if _, err := time.Parse("2006-01-02", tanggal); err != nil {
		return Peminjaman{}, errors.New("tanggal harus berformat YYYY-MM-DD dan berupa tanggal yang ada")
	}

	mulai, selesai := *in.SlotMulai, *in.SlotSelesai
	if mulai < slotMin || mulai > slotMax || selesai < slotMin || selesai > slotMax {
		return Peminjaman{}, errors.New("slot harus bernilai 1 sampai 8")
	}
	if mulai > selesai {
		return Peminjaman{}, errors.New("slot_mulai tidak boleh lebih besar dari slot_selesai")
	}

	if *in.Jumlah < 1 {
		return Peminjaman{}, errors.New("jumlah minimal 1")
	}

	return Peminjaman{
		SumberDayaID: *in.SumberDayaID,
		Peminjam:     peminjam,
		Tanggal:      tanggal,
		SlotMulai:    mulai,
		SlotSelesai:  selesai,
		Jumlah:       *in.Jumlah,
	}, nil
}