package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func kirim(h http.Handler, method, path, body string) (int, string) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func cek(t *testing.T, h http.Handler, method, path, body string, want int) string {
	t.Helper()
	code, resp := kirim(h, method, path, body)
	if code != want {
		t.Fatalf("%s %s: status %d, mau %d (body: %s)", method, path, code, want, resp)
	}
	return resp
}

func pinjam(sdID int, tanggal string, mulai, selesai, jumlah int) string {
	return fmt.Sprintf(
		`{"sumber_daya_id":%d,"peminjam":"Rina","tanggal":"%s","slot_mulai":%d,"slot_selesai":%d,"jumlah":%d}`,
		sdID, tanggal, mulai, selesai, jumlah)
}

func TestErrorSelaluJSON(t *testing.T) {
	h := NewHandler()
	for _, tc := range []struct{ method, path string }{
		{"GET", "/tidak-ada"}, {"PATCH", "/peminjaman/1"}, {"GET", "/peminjaman/"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		var body map[string]string
		if rec.Code != 404 || rec.Header().Get("Content-Type") != "application/json" ||
			json.Unmarshal(rec.Body.Bytes(), &body) != nil || body["error"] == "" {
			t.Errorf("%s %s: status %d, ct %q, body %q", tc.method, tc.path,
				rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
		}
	}
}

func TestBentrok(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Aula A","jenis":"ruang","stok":1}`, 201)
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Kabel","jenis":"alat","stok":3}`, 201)

	// ruang stok 1
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 5, 6, 1), 201) // bersebelahan
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 4, 5, 1), 409) // irisan slot 4 dan 5
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-03", 3, 4, 1), 201) // tanggal lain

	// alat stok 3: dihitung per slot dengan jumlah unit
	cek(t, h, "POST", "/peminjaman", pinjam(2, "2026-11-02", 1, 2, 2), 201) // slot 1,2 = 2
	cek(t, h, "POST", "/peminjaman", pinjam(2, "2026-11-02", 2, 3, 1), 201) // slot 2 = 3 (penuh)
	cek(t, h, "POST", "/peminjaman", pinjam(2, "2026-11-02", 1, 1, 1), 201) // slot 1 = 3 (penuh)
	cek(t, h, "POST", "/peminjaman", pinjam(2, "2026-11-02", 1, 2, 1), 409) // slot 1 dan 2 penuh
	cek(t, h, "POST", "/peminjaman", pinjam(2, "2026-11-02", 3, 4, 2), 201) // slot 3 = 1+2 = 3
}

func TestUbahDanBatalkan(t *testing.T) {
	h := NewHandler()
	if resp := cek(t, h, "DELETE", "/peminjaman/1", "", 204); resp != "" {
		t.Fatalf("204 harus tanpa body, dapat %q", resp)
	}
	cek(t, h, "DELETE", "/peminjaman/1", "", 409) // sudah dibatalkan
	cek(t, h, "DELETE", "/peminjaman/99", "", 404)
	cek(t, h, "DELETE", "/peminjaman/abc", "", 400)

	// urutan pemeriksaan: 400 (id) -> 404 -> 400 (body) -> 409
	cek(t, h, "PUT", "/peminjaman/abc", pinjam(1, "2026-11-02", 5, 6, 1), 400)
	cek(t, h, "PUT", "/peminjaman/99", `bukan json`, 404)
	cek(t, h, "PUT", "/peminjaman/1", `bukan json`, 400)
	cek(t, h, "PUT", "/peminjaman/1", pinjam(1, "2026-02-30", 5, 6, 1), 400)
	cek(t, h, "PUT", "/peminjaman/1", pinjam(99, "2026-11-02", 5, 6, 1), 400)

	// 409 dan data lama tidak berubah
	cek(t, h, "PUT", "/peminjaman/2", pinjam(1, "2026-11-02", 2, 3, 1), 409)
	var p Peminjaman
	json.Unmarshal([]byte(cek(t, h, "GET", "/peminjaman/2", "", 200)), &p)
	if p.SlotMulai != 3 || p.SlotSelesai != 4 {
		t.Fatalf("data lama berubah setelah PUT gagal: %+v", p)
	}

	// tidak bentrok dengan dirinya sendiri; id dan status tetap
	body := strings.Replace(pinjam(1, "2026-11-02", 1, 2, 1), "Rina", "Budi", 1)
	json.Unmarshal([]byte(cek(t, h, "PUT", "/peminjaman/1", body, 200)), &p)
	if p.ID != 1 || p.Status != "aktif" || p.Peminjam != "Budi" {
		t.Fatalf("hasil PUT salah: %+v", p)
	}

	// DELETE
	if resp := cek(t, h, "DELETE", "/peminjaman/1", "", 204); resp != "" {
		t.Fatalf("204 harus tanpa body, dapat %q", resp)
	}
	cek(t, h, "DELETE", "/peminjaman/1", "", 404)
	cek(t, h, "DELETE", "/peminjaman/abc", "", 400)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 1, 2, 1), 201) // slot sudah bebas
}

func TestBersamaan(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Aula A","jenis":"ruang","stok":1}`, 201)

	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	hitung := map[int]int{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, _ := kirim(h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1))
			mu.Lock()
			hitung[code]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if hitung[201] != 1 || hitung[409] != n-1 {
		t.Fatalf("mau 1x201 dan %dx409, dapat %v", n-1, hitung)
	}
}

// Persona C 

func ambil(t *testing.T, resp string) Peminjaman {
	t.Helper()
	var p Peminjaman
	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		t.Fatalf("bukan JSON peminjaman: %q", resp)
	}
	return p
}

func TestPersonaC_AlurPersetujuan(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Ruang Himpunan","jenis":"ruang","stok":1}`, 201)

	// status awal diajukan, alasan kosong, dan field alasan selalu ada di JSON
	resp := cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201)
	if p := ambil(t, resp); p.Status != "diajukan" || p.Alasan != "" || !strings.Contains(resp, `"alasan":""`) {
		t.Fatalf("status awal salah: %s", resp)
	}

	// diajukan menahan slot
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 409)

	// selesaikan sebelum disetujui ditolak, dan data tidak berubah
	cek(t, h, "POST", "/peminjaman/1/selesaikan", "", 409)
	if p := ambil(t, cek(t, h, "GET", "/peminjaman/1", "", 200)); p.Status != "diajukan" {
		t.Fatalf("status berubah setelah aksi gagal: %+v", p)
	}

	// setujui: disetujui masih menahan slot; aksi yang tidak sesuai status -> 409
	if p := ambil(t, cek(t, h, "POST", "/peminjaman/1/setujui", "", 200)); p.Status != "disetujui" {
		t.Fatalf("setujui salah: %+v", p)
	}
	cek(t, h, "POST", "/peminjaman/1/setujui", "", 409)
	cek(t, h, "POST", "/peminjaman/1/tolak", `{"alasan":"terlambat"}`, 409)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 409)

	// selesaikan: selesai melepaskan slot
	if p := ambil(t, cek(t, h, "POST", "/peminjaman/1/selesaikan", "", 200)); p.Status != "selesai" {
		t.Fatalf("selesaikan salah: %+v", p)
	}
	cek(t, h, "POST", "/peminjaman/1/selesaikan", "", 409)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201) // id 2, diajukan
}

func TestPersonaC_Tolak(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Ruang Himpunan","jenis":"ruang","stok":1}`, 201)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201) // id 1

	// urutan pemeriksaan: 400 (id) -> 404 -> 400 (body) -> 409
	cek(t, h, "POST", "/peminjaman/abc/tolak", `{"alasan":"x"}`, 400)
	cek(t, h, "POST", "/peminjaman/99/tolak", `bukan json`, 404)
	cek(t, h, "POST", "/peminjaman/1/tolak", ``, 400)
	cek(t, h, "POST", "/peminjaman/1/tolak", `bukan json`, 400)
	cek(t, h, "POST", "/peminjaman/1/tolak", `{}`, 400)
	cek(t, h, "POST", "/peminjaman/1/tolak", `{"alasan":"   "}`, 400)
	cek(t, h, "POST", "/peminjaman/1/tolak", fmt.Sprintf(`{"alasan":"%s"}`, strings.Repeat("a", 101)), 400)
	if p := ambil(t, cek(t, h, "GET", "/peminjaman/1", "", 200)); p.Status != "diajukan" {
		t.Fatalf("status berubah setelah tolak gagal: %+v", p)
	}

	// alasan di-trim; ditolak melepaskan slot
	p := ambil(t, cek(t, h, "POST", "/peminjaman/1/tolak", `{"alasan":"  Bentrok acara  "}`, 200))
	if p.Status != "ditolak" || p.Alasan != "Bentrok acara" {
		t.Fatalf("tolak salah: %+v", p)
	}
	cek(t, h, "POST", "/peminjaman/1/tolak", `{"alasan":"lagi"}`, 409)
	cek(t, h, "POST", "/peminjaman/1/setujui", "", 409)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201) // id 2, slot sudah bebas

	// batas 100 karakter dihitung per karakter, bukan byte
	cek(t, h, "POST", "/peminjaman/2/tolak", fmt.Sprintf(`{"alasan":"%s"}`, strings.Repeat("é", 100)), 200)

	// PUT tidak mengubah status maupun alasan
	p = ambil(t, cek(t, h, "PUT", "/peminjaman/1", pinjam(1, "2026-12-01", 1, 1, 1), 200))
	if p.Status != "ditolak" || p.Alasan != "Bentrok acara" {
		t.Fatalf("PUT mengubah status/alasan: %+v", p)
	}

	// filter status dari filter dasar
	var daftar []Peminjaman
	json.Unmarshal([]byte(cek(t, h, "GET", "/peminjaman?status=ditolak", "", 200)), &daftar)
	if len(daftar) != 2 {
		t.Fatalf("filter status=ditolak: dapat %d, mau 2", len(daftar))
	}
	cek(t, h, "GET", "/peminjaman?status=disetujui", "", 200)
}

func TestPersonaC_SetujuiBersamaan(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Ruang Himpunan","jenis":"ruang","stok":1}`, 201)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201)

	const n = 30
	var wg sync.WaitGroup
	var mu sync.Mutex
	hitung := map[int]int{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, _ := kirim(h, "POST", "/peminjaman/1/setujui", "")
			mu.Lock()
			hitung[code]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if hitung[200] != 1 || hitung[409] != n-1 {
		t.Fatalf("mau 1x200 dan %dx409, dapat %v", n-1, hitung)
	}
}

// Change Request C: pembatalan 
func TestCR_PembatalanBukanHapus(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Ruang Himpunan","jenis":"ruang","stok":1}`, 201)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201) // id 1

	// 204 tanpa body
	if resp := cek(t, h, "DELETE", "/peminjaman/1", "", 204); resp != "" {
		t.Fatalf("204 harus tanpa body, dapat %q", resp)
	}

	// data tetap ada dengan status dibatalkan
	p := ambil(t, cek(t, h, "GET", "/peminjaman/1", "", 200))
	if p.Status != "dibatalkan" || p.Peminjam != "Rina" || p.SlotMulai != 3 || p.SlotSelesai != 4 {
		t.Fatalf("data berubah/hilang setelah dibatalkan: %+v", p)
	}

	// tetap muncul di daftar, bisa disaring dengan ?status=dibatalkan
	var daftar []Peminjaman
	json.Unmarshal([]byte(cek(t, h, "GET", "/peminjaman", "", 200)), &daftar)
	if len(daftar) != 1 {
		t.Fatalf("daftar: dapat %d, mau 1", len(daftar))
	}
	json.Unmarshal([]byte(cek(t, h, "GET", "/peminjaman?status=dibatalkan", "", 200)), &daftar)
	if len(daftar) != 1 {
		t.Fatalf("filter dibatalkan: dapat %d, mau 1", len(daftar))
	}
	if resp := cek(t, h, "GET", "/peminjaman?status=diajukan", "", 200); strings.TrimSpace(resp) != "[]" {
		t.Fatalf("filter diajukan harus [], dapat %s", resp)
	}

	// slot langsung bisa dipinjam orang lain
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201) // id 2

	// membatalkan lagi 409; id bukan angka 400; tidak ada 404; data lama tidak berubah
	cek(t, h, "DELETE", "/peminjaman/1", "", 409)
	cek(t, h, "DELETE", "/peminjaman/abc", "", 400)
	cek(t, h, "DELETE", "/peminjaman/99", "", 404)
	if p := ambil(t, cek(t, h, "GET", "/peminjaman/1", "", 200)); p.Status != "dibatalkan" {
		t.Fatalf("status berubah: %+v", p)
	}
}

func TestCR_PersonaC(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Ruang Himpunan","jenis":"ruang","stok":1}`, 201)
	for slot := 1; slot <= 5; slot++ { // id 1..5, tiap id di slot sendiri
		cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", slot, slot, 1), 201)
	}

	// id 1 diajukan -> bisa dibatalkan
	cek(t, h, "DELETE", "/peminjaman/1", "", 204)

	// id 2 disetujui -> bisa dibatalkan
	cek(t, h, "POST", "/peminjaman/2/setujui", "", 200)
	cek(t, h, "DELETE", "/peminjaman/2", "", 204)

	// id 3 ditolak -> 409, status tetap ditolak
	cek(t, h, "POST", "/peminjaman/3/tolak", `{"alasan":"bentrok"}`, 200)
	cek(t, h, "DELETE", "/peminjaman/3", "", 409)
	if p := ambil(t, cek(t, h, "GET", "/peminjaman/3", "", 200)); p.Status != "ditolak" {
		t.Fatalf("status berubah: %+v", p)
	}

	// id 4 selesai -> 409
	cek(t, h, "POST", "/peminjaman/4/setujui", "", 200)
	cek(t, h, "POST", "/peminjaman/4/selesaikan", "", 200)
	cek(t, h, "DELETE", "/peminjaman/4", "", 409)

	// yang sudah dibatalkan tidak bisa disetujui, ditolak, atau diselesaikan
	for _, id := range []string{"1", "2"} {
		cek(t, h, "POST", "/peminjaman/"+id+"/setujui", "", 409)
		cek(t, h, "POST", "/peminjaman/"+id+"/tolak", `{"alasan":"x"}`, 409)
		cek(t, h, "POST", "/peminjaman/"+id+"/selesaikan", "", 409)
		cek(t, h, "DELETE", "/peminjaman/"+id, "", 409)
	}
}

func TestCR_BatalkanBersamaan(t *testing.T) {
	h := NewHandler()
	cek(t, h, "POST", "/sumber-daya", `{"nama":"Ruang Himpunan","jenis":"ruang","stok":1}`, 201)
	cek(t, h, "POST", "/peminjaman", pinjam(1, "2026-11-02", 3, 4, 1), 201)

	const n = 30
	var wg sync.WaitGroup
	var mu sync.Mutex
	hitung := map[int]int{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, _ := kirim(h, "DELETE", "/peminjaman/1", "")
			mu.Lock()
			hitung[code]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if hitung[204] != 1 || hitung[409] != n-1 {
		t.Fatalf("mau 1x204 dan %dx409, dapat %v", n-1, hitung)
	}
}	
