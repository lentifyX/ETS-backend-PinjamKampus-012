# PROSES

## 1. Persona dan masalahnya
Pengguna persona saya adalah Admin Himpunan yang bertanggung jawab mengelola dan mengawasi penggunaan ruang himpunan agar tidak digunakan secara sembarangan. Masalah yang dihadapi adalah adanya peminjaman ruang yang langsung aktif tanpa pengawasan, sehingga fitur khas ini menyelesaikannya dengan menyediakan alur persetujuan (approval workflow) di mana setiap pengajuan berstatus awal "diajukan", serta mewajibkan persetujuan (peminjaman berstatus "disetujui") sebelum ruang terpakai atau penolakan berdisertai alasan mendasar.   

## 2. Rancangan struktur data
* **`SumberDaya` struct**: Menyimpan informasi `id` (int), `nama` (string), `jenis` (string - "ruang" atau "alat"), dan `stok` (int)[cite: 2]. Disimpan dalam slice `[]SumberDaya` di dalam struct `Store`.
* **`Peminjaman` struct**: Menyimpan data `id`, `sumber_daya_id`, `peminjam`, `tanggal`, `slot_mulai`, `slot_selesai`, `jumlah`, `status` (`diajukan`, `disetujui`, `ditolak`, `selesai`, `dibatalkan`), dan `alasan` (string). Disimpan dalam slice `[]Peminjaman`.
* **Sinkronisasi**: Seluruh operasi baca dan tulis dilindungi menggunakan `sync.Mutex` di dalam struct `Store` untuk menjamin keamanan akses konkuren (*thread-safe*) saat ribuan request masuk bersamaan.

## 3. Satu keputusan desain
* **Integrasi Status dengan Fungsi `menahanSlot`**: Alih-alih membuat logika terpisah untuk pengecekan slot dan pembatalan, fungsi `menahanSlot(status)` digunakan secara terpusat untuk menentukan status apa saja yang membebani kapasitas slot (`diajukan` dan `disetujui`). Hal ini memastikan bahwa status seperti `ditolak`, `selesai`, dan `dibatalkan` secara otomatis melepaskan slot tanpa risiko inkonsistensi data[cite: 3].

## 4. Bug atau kendala (minimal dua)
* **Kendala 1 (Urutan Validasi Error 404 vs 400 pada Endpoint Tolak)**: 
  * Masalah: Awalnya validasi body JSON dijalankan sebelum pengecekan keberadaan ID peminjaman, sehingga request ke ID yang tidak ada dengan body rusak mengembalikan kode `400` alih-alih `404`.
  * Solusi: Memastikan pemanggilan `s.store.AmbilPeminjaman(id)` dieksekusi lebih dulu sebelum fungsi `decodeJSON` dijalankan pada handler `tolak`.
* **Kendala 2 (Race Condition pada Aksi Serentak)**:
  * Masalah: Pengujian menggunakan *race detector* (`-race`) sempat mendeteksi potensi konflik memori saat beberapa goroutine mengubah status peminjaman secara paralel.
  * *Solusi*: Membungkus seluruh logika perubahan status di dalam method `Store` dengan penguncian mutex (`s.mu.Lock()` dan `defer s.mu.Unlock()`) secara atomik.

## 5. Change request
 Change Request menghendaki agar endpoint DELETE /peminjaman/{id} berfungsi sebagai pembatalan, di mana status peminjaman berubah menjadi "dibatalkan", data tetap disimpan, dan mengembalikan respons 204 tanpa body. Khusus untuk Persona C, pembatalan hanya diizinkan pada status "diajukan" atau "disetujui", sedangkan status lainnya menghasilkan error 409. Implementasi ini melibatkan perubahan pada file store.go untuk penambahan konstanta statusDibatalkan dan method BatalkanPeminjaman, file handler.go untuk memperbarui rute DELETE ke handler pembatalan, serta file api_test.go untuk menyesuaikan dan menambah test suite pengujian. Adapun keputusan desain yang diambil adalah menggunakan kembali fungsi menahanSlot secara terpusat untuk memvalidasi status yang sah dibatalkan agar selaras dengan logika pelepasan kapasitas slot.
