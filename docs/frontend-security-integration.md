# Panduan Integrasi Frontend: Keamanan Auth dan Iframe

Dokumen ini menjelaskan perubahan keamanan backend yang perlu diikuti oleh seluruh frontend QRUPI.

## 1. Autentikasi menggunakan cookie

Endpoint berikut membuat session autentikasi:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/student/verify-pin`
- `POST /api/v1/auth/student/switch`

Backend mengirim cookie bernama `qrupi_auth` dengan atribut:

- `HttpOnly`: token tidak dapat dibaca JavaScript.
- `SameSite=Lax`: mengurangi risiko pengiriman cookie lintas situs.
- `Secure`: aktif di production; local development dapat memakai `false`.
- `Path=/`.

Token tidak lagi tersedia di response JSON login. Frontend tidak perlu menyimpan token di `localStorage`, `sessionStorage`, atau state yang dipersistenkan.

## 2. Konfigurasi request frontend

Request ke API harus mengikutsertakan credentials.

### Fetch

```ts
await fetch(`${API_URL}/api/v1/quizzes`, {
  method: "GET",
  credentials: "include",
});
```

### Axios

```ts
const api = axios.create({
  baseURL: API_URL,
  withCredentials: true,
});
```

Frontend tidak perlu lagi menambahkan header `Authorization: Bearer ...` untuk alur cookie baru. Backend masih menerima header tersebut sementara waktu untuk mendukung frontend lama selama masa migrasi.

## 3. CORS

Origin frontend harus didaftarkan di backend:

```env
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
```

Karena request menggunakan cookie, backend menggunakan `AllowCredentials=true`. Jangan menggunakan wildcard `*` sebagai origin ketika credentials aktif.

## 4. Logout

Gunakan endpoint berikut:

```http
POST /api/v1/auth/logout
```

Request harus menggunakan `credentials: "include"` dan header CSRF. Backend akan mencabut token aktif di database serta menghapus cookie autentikasi dan cookie CSRF.

Penghapusan cookie HttpOnly tidak dapat dilakukan oleh JavaScript frontend.

## 4a. CSRF untuk request mutasi

Setelah login, backend mengirim cookie `qrupi_csrf` yang dapat dibaca frontend. Untuk setiap request `POST`, `PUT`, `PATCH`, atau `DELETE` berbasis cookie, kirim nilainya sebagai header `X-CSRF-Token`.

```ts
const csrf = getCookie("qrupi_csrf");
await fetch(`${API_URL}/api/v1/auth/logout`, {
  method: "POST",
  credentials: "include",
  headers: { "X-CSRF-Token": csrf ?? "" },
});
```

Untuk berpindah ke akun anak, kirim `POST /api/v1/auth/student/switch` dengan body `{ "user_id": "child-user-id" }` dan header CSRF. Backend mengganti cookie `qrupi_auth` ke session student serta menerbitkan `qrupi_csrf` baru; session parent tetap aktif di backend. Saat ini validasi menggunakan akun non-student aktif dan institusi yang sama karena model belum memiliki relasi parent-child eksplisit.

## 5. CSP dan iframe

Gateway mengirim header `Content-Security-Policy` dengan aturan utama:

- `default-src 'self'`
- `frame-src` berdasarkan `IFRAME_ALLOWED_ORIGINS`
- `frame-ancestors 'self'`
- `object-src 'none'`
- `base-uri 'self'`

Contoh konfigurasi production:

```env
IFRAME_ALLOWED_ORIGINS='self',https://player.example.com,https://video.example.com
```

Nilai harus berupa origin/domain yang memang digunakan iframe. Jangan memasukkan domain yang tidak dipercaya.

## 6. Environment local dan production

Local development:

```env
AUTH_COOKIE_NAME=qrupi_auth
CSRF_COOKIE_NAME_PELAJAR=qrupi_pelajar_csrf
AUTH_COOKIE_SECURE=false
AUTH_COOKIE_PERSIST=false
CORS_ALLOWED_ORIGINS=http://localhost:5173
IFRAME_ALLOWED_ORIGINS='self'
```

Production:

```env
AUTH_COOKIE_NAME=qrupi_auth
AUTH_COOKIE_SECURE=true
AUTH_COOKIE_PERSIST=false
CORS_ALLOWED_ORIGINS=https://app.example.com
IFRAME_ALLOWED_ORIGINS='self',https://trusted-player.example.com
```

Production wajib menggunakan HTTPS. Jangan menyalin nilai rahasia dari file `.env` development ke repository atau dokumentasi.

## 7. Perubahan yang masih dalam tahap berikutnya

Tim frontend perlu mengantisipasi perubahan API berikut:

1. Endpoint quiz student akan menghilangkan `correct_answer` sebelum quiz selesai.
2. Backend akan menghitung ulang `is_correct`, `points_earned`, dan `score`; frontend tidak boleh dianggap sebagai sumber kebenaran nilai.
3. Endpoint quiz, session, dan resource akan memvalidasi ownership serta membership learning group.
4. Authorization role dan learning group akan diterapkan secara konsisten pada endpoint detail, update, archive, dan delete.
5. Frontend wajib menggunakan endpoint logout dan header CSRF untuk request mutasi.

## 8. Checklist migrasi frontend

- [ ] Hapus pembacaan token dari `localStorage`.
- [ ] Hapus penyimpanan token JWT di browser.
- [ ] Tambahkan `credentials: "include"` atau `withCredentials: true`.
- [ ] Pastikan origin frontend terdaftar di `CORS_ALLOWED_ORIGINS`.
- [ ] Jangan mengandalkan `correct_answer` untuk tampilan student sebelum submit/penilaian resmi.
- [ ] Jangan mengirim `score`, `is_correct`, atau `points_earned` sebagai nilai yang dipercaya backend.
- [ ] Uji aplikasi melalui HTTPS sebelum production.
