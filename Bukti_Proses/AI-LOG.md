# AI-LOG

Catat setiap penggunaan AI yang menghasilkan kode. Jika tidak memakai AI sama sekali, tulis satu baris: **tidak memakai AI**.

Log yang jujur tidak mengurangi nilai, sebanyak apa pun AI dipakai. Yang mengurangi nilai adalah log yang tidak sesuai dengan kode, atau isi log yang tidak bisa dijelaskan.

## Struktur Folder 
- Prompt: Buatkan saya struktur folder untuk mengimplemetasi tugas ini
- Dipakai: handler.go, main.go, persona.go, store.go
- Diubah: Awalnya AI memberikan struktur yang kompleks dengan jumlah file +- 10 file. Yang dipakai sesuai dengan yang diatas

## Struktur Dasar Store dan Validasi Aturan Bentrok 
* **Prompt**: "Buatkan fungsi Go untuk memeriksa apakah rentang slot peminjaman beririsan dengan data peminjaman lain dengan mempertimbangkan kapasitas stok sumber daya."
* **Bagian Kode yang Dipakai**: Logika perulangan iterasi slot pada method `adaBentrok` dan `unitTerpakai` di `store.go`.
* **Perbaikan/Modifikasi**: Hasil kode dari AI awalnya hanya mencocokkan irisan tanggal secara umum tanpa memperhitungkan rentang jam per slot dan akumulasi jumlah unit stok. Saya mengubahnya menjadi pengecekan per satuan slot (`slot_mulai` hingga `slot_selesai`) dan menjumlahkan beban unit yang aktif.

## Fitur Khas Persona C
* **Prompt**: "Bagaimana cara merancang endpoint persetujuan status di Go net/http dengan enkapsulasi mutex di struct Store agar aman dari race condition?"
* **Bagian Kode yang Dipakai**: Pola method `UbahStatus` dan handler `pindahStatus` di `persona.go`.
* **Perbaikan/Modifikasi**: Menyesuaikan aturan transisi status spesifik Persona C (hanya status `"diajukan"` yang bisa disetujui/ditolak, dan hanya `"disetujui"` yang bisa diselesaikan) serta memastikan pengembalian error `409` ketika syarat status tidak terpenuhi.

## Entri 3: Change Request C (Pembatalan Peminjaman via DELETE)
* **Prompt**: "Bagaimana mengubah fungsi DELETE di Go agar tidak menghapus elemen dari slice, melainkan mengubah statusnya menjadi 'dibatalkan' dan mengembalikan status HTTP 204 tanpa body?"
* **Bagian Kode yang Dipakai**: Implementasi method `BatalkanPeminjaman` di `store.go` dan handler `batalkanPeminjaman` di `handler.go`.
* **Perbaikan/Modifikasi**: Menambahkan validasi agar peminjaman yang sudah berstatus selain `"diajukan"` atau `"disetujui"` ditolak dengan kode `409 Conflict`, serta memastikan respons mengembalikan `w.WriteHeader(http.StatusNoContent)` tanpa mengirim body apa pun
