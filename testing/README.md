# QRUPI API integration testing

Runner ini menguji API melalui Gateway dan membuat laporan JSON serta HTML. Semua service harus berjalan dan database development harus sudah dimigrasikan.

> Jalankan hanya pada environment development/testing. Runner melakukan insert, update, archive, dan permanent delete. Jika service berhenti di tengah pengujian, sebagian data dengan prefix `API Test` mungkin perlu dibersihkan manual.

```bash
go run ./testing
```

Konfigurasi opsional:

```env
TEST_API_BASE_URL=http://localhost:3000/api/v1
TEST_EMAIL=admin@admin.com
TEST_PASSWORD=Admin11234
TEST_PROVINCE_CODE=13
TEST_REGENCY_CODE=13.71
```

Skenario membuat data dengan prefix `API Test`, menjalankan get/insert/update/archive/permanent delete, kemudian membersihkannya. Skenario learning resource membuat kombinasi Kelas 1–2 dan mata pelajaran Matematika–IPA pada institution target, menugaskan satu modul ke seluruh group, lalu memverifikasi jumlah assignment.

Laporan dibuat pada:

```text
testing/reports/report-YYYYMMDD-HHMMSS.json
testing/reports/report-YYYYMMDD-HHMMSS.html
testing/reports/latest.json
```
